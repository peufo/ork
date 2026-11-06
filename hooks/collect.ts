import type { EngineInterface } from 'claude-code'

import { buildStatus, countMainDirty, parseAheadBehind, parseRefs, parseWorktrees } from './status'
import type {
  BranchState,
  GitFacts,
  StatusReport,
  TicketFile,
  WorktreeEntry,
  WorktreeState,
} from './status'
import { parseTicketHead, readFrontmatter } from './tickets'

export const CONFIG = '.ork/config.md'
export const TICKETS_DIR = '.ork/tickets'
export const STOP_FILE = `${TICKETS_DIR}/.stop`

const DONE_DIR = `${TICKETS_DIR}/done`
const REFERENCE_BRANCHES = ['main', 'master']
const MAX_OTHER_BRANCHES = 8

export async function isOrkProject($: EngineInterface): Promise<boolean> {
  return (await $.fs.exists(CONFIG)) && (await $.fs.exists(TICKETS_DIR))
}

export async function readTickets($: EngineInterface): Promise<TicketFile[]> {
  const entries = await $.fs.list(TICKETS_DIR)
  const files = await Promise.all(
    entries
      .filter(entry => entry.kind === 'file' && entry.name.endsWith('.md'))
      .map(async entry => {
        const text = await $.fs.read(`${TICKETS_DIR}/${entry.name}`)
        const head = parseTicketHead(entry.name, text)

        return head && { head, text }
      }),
  )

  return files.filter((file): file is TicketFile => file !== undefined)
}

/** The ids of the closed tickets, from their files' names: there can be many to open. */
async function readDoneIds($: EngineInterface): Promise<string[]> {
  if (!(await $.fs.exists(DONE_DIR))) {
    return []
  }

  const entries = await $.fs.list(DONE_DIR)

  return entries
    .filter(entry => entry.kind === 'file' && entry.name.endsWith('.md'))
    .map(entry => entry.name.replace(/\.md$/, '').split('-')[0] ?? '')
}

async function git(
  $: EngineInterface,
  args: readonly string[],
  cwd?: string,
): Promise<string | null> {
  try {
    const { exitCode, stdout } = await $.process.run(['git', ...args], cwd ? { cwd } : {})

    return exitCode === 0 ? stdout : null
  } catch {
    return null
  }
}

async function inspectWorktree(
  $: EngineInterface,
  entry: WorktreeEntry,
  base: string,
): Promise<WorktreeState> {
  if (entry.isMissing) {
    return { ...entry, ahead: null, behind: null, dirty: null, lastCommit: null }
  }

  const [counts, status, log] = await Promise.all([
    git($, ['rev-list', '--left-right', '--count', `${base}...HEAD`], entry.path),
    git($, ['status', '--porcelain'], entry.path),
    git($, ['log', '-1', '--format=%ct'], entry.path),
  ])
  const lastCommit = Number(log?.trim() ?? Number.NaN)

  return {
    ...entry,
    ...parseAheadBehind(counts),
    dirty: status === null ? null : status.split('\n').filter(line => line !== '').length,
    lastCommit: Number.isNaN(lastCommit) ? null : lastCommit,
  }
}

async function inspectBranch(
  $: EngineInterface,
  name: string,
  range: string,
): Promise<BranchState> {
  return {
    name,
    ...parseAheadBehind(await git($, ['rev-list', '--left-right', '--count', range])),
  }
}

/** Its changes are counted apart, without ork's own files. Only a review moves it off the base. */
async function inspectMainRepository(
  $: EngineInterface,
  entry: WorktreeEntry,
  base: string,
): Promise<WorktreeState> {
  const counts =
    entry.branch === base ? null : await git($, ['rev-list', '--left-right', '--count', `${base}...HEAD`], entry.path)

  return { ...entry, ...parseAheadBehind(counts), dirty: null, lastCommit: null }
}

/** Reads local refs only: a fetch would be slow, and isn't ours to do. */
async function collectGitFacts(
  $: EngineInterface,
  entries: readonly WorktreeEntry[],
  base: string,
): Promise<GitFacts> {
  const [refsOutput, unmergedOutput, mainStatus] = await Promise.all([
    git($, ['for-each-ref', '--format=%(refname:short)%09%(upstream:short)%09%(upstream:track)', 'refs/heads']),
    git($, ['for-each-ref', `--no-merged=${base}`, '--format=%(refname:short)', 'refs/heads']),
    git($, ['status', '--porcelain'], entries[0]?.path),
  ])
  const refs = parseRefs(refsOutput ?? '')
  const baseRef = refs.find(ref => ref.name === base)
  const referenceName = REFERENCE_BRANCHES.find(
    name => name !== base && refs.some(ref => ref.name === name),
  )
  const withWorktree = new Set(entries.map(entry => entry.branch))
  const others = (unmergedOutput ?? '')
    .split('\n')
    .filter(name => name !== '' && !withWorktree.has(name))
  const [reference, otherBranches] = await Promise.all([
    referenceName === undefined ? null : inspectBranch($, referenceName, `${referenceName}...${base}`),
    Promise.all(
      others
        .slice(0, MAX_OTHER_BRANCHES)
        .map(name => inspectBranch($, name, `${base}...${name}`)),
    ),
  ])

  return {
    upstream: baseRef?.upstream
      ? { remote: baseRef.upstream.split('/')[0] ?? baseRef.upstream, track: baseRef.track }
      : null,
    reference,
    mainDirty: mainStatus === null ? null : countMainDirty(mainStatus),
    otherBranches,
    hiddenBranches: Math.max(0, others.length - MAX_OTHER_BRANCHES),
  }
}

/** Reads the tickets, git and the agents. Answers null outside an ork project. */
export async function collectStatus($: EngineInterface): Promise<StatusReport | null> {
  if (!(await isOrkProject($))) {
    return null
  }

  const [config, tickets, doneIds, hasStopFile, agents, porcelain, nowMs] = await Promise.all([
    $.fs.read(CONFIG),
    readTickets($),
    readDoneIds($),
    $.fs.exists(STOP_FILE),
    $.agent.list(),
    git($, ['worktree', 'list', '--porcelain']),
    $.clock.now(),
  ])
  const entries = parseWorktrees(porcelain ?? '')
  const mainBranch = entries[0]?.branch ?? null
  const integrationBranch =
    readFrontmatter(config)?.get('integration_branch') || mainBranch || 'HEAD'
  const worktrees = await Promise.all(
    entries.map((entry, index) =>
      index === 0 ? inspectMainRepository($, entry, integrationBranch) : inspectWorktree($, entry, integrationBranch),
    ),
  )
  const gitFacts = porcelain === null ? null : await collectGitFacts($, entries, integrationBranch)

  return buildStatus({
    tickets,
    doneIds,
    agents,
    hasStopFile,
    integrationBranch,
    mainBranch,
    worktrees,
    gitFacts,
    nowSeconds: Math.floor(nowMs / 1000),
  })
}
