import type { GitRow, GitView, Health, TicketAlert } from '../types'

import { isPaused, openBlockers, subjectQueue, ticketName } from './tickets'
import type { TicketHead } from './tickets'

export type TicketFile = {
  head: TicketHead
  text: string
}

export type WorktreeEntry = {
  path: string
  branch: string | null
  isLocked: boolean
  isMissing: boolean
}

/** A count is null when git could not answer: nothing is concluded from it. */
export type WorktreeState = WorktreeEntry & {
  ahead: number | null
  behind: number | null
  dirty: number | null
  lastCommit: number | null
}

export type AgentState = {
  id: string
  status: string
}

export type BranchState = {
  name: string
  ahead: number | null
  behind: number | null
}

/** What git says beyond the worktrees. Everything comes from local refs: nothing is fetched. */
export type GitFacts = {
  upstream: { remote: string; track: string } | null
  reference: BranchState | null
  mainDirty: number | null
  otherBranches: readonly BranchState[]
  hiddenBranches: number
}

export type StatusInput = {
  tickets: readonly TicketFile[]
  doneIds: readonly string[]
  agents: readonly AgentState[]
  hasStopFile: boolean
  integrationBranch: string
  mainBranch: string | null
  worktrees: readonly WorktreeState[]
  gitFacts: GitFacts | null
  nowSeconds: number
}

type Action = { label: string; text: string }

export type StopFile = 'no' | 'active' | 'stale'

export type Stopped = { ticket: TicketHead; mode: string; log: string }

export type BrokenDep = { ticket: TicketHead; problem: string }

export type WatchedWorktree = {
  name: string
  branch: string | null
  detail: string
  action?: Action
}

export type FirstSubject = { ticket: TicketHead; excerpt: string; isTruncated: boolean }

export type StatusReport = {
  integrationBranch: string
  stopFile: StopFile
  review: { branch: string; ticket?: TicketHead } | null
  stopped: readonly Stopped[]
  brokenDeps: readonly BrokenDep[]
  worktrees: readonly WatchedWorktree[]
  queue: readonly TicketHead[]
  ready: readonly { ticket: TicketHead; blockers: readonly string[] }[]
  paused: readonly TicketHead[]
  firstSubject: FirstSubject | null
  git: GitView | null
}

export const NO_CONFIG_SNAPSHOT = 'ork_status: no_config'

const LIVE_AGENT_STATUSES = ['pending', 'running', 'waiting', 'idle']
const RUNNING_STATUSES = ['planning', 'in-progress']
const WAITING_WORKER_STATUSES = ['plan-review', 'blocked']
const TOLERATED_DIRTY_PATHS = ['.ork/', '.claude/launch.json']
const EXCERPT_MAX_LINES = 40
const LOG_MAX_LENGTH = 120
const SUBJECT_SECTIONS: Record<string, readonly string[]> = {
  blocked: ['Questions'],
  'plan-review': ['Plan'],
  'in-review': ['Review'],
  idea: ['Context', 'Questions'],
}

function plural(count: number, one: string, many: string): string {
  return `${count} ${count === 1 ? one : many}`
}

/** Reads `git worktree list --porcelain`. The first entry is the main repository. */
export function parseWorktrees(porcelain: string): WorktreeEntry[] {
  return porcelain
    .split(/\n\s*\n/)
    .map(block => block.split('\n').filter(line => line !== ''))
    .filter(lines => lines[0]?.startsWith('worktree '))
    .map(lines => ({
      path: (lines[0] ?? '').slice('worktree '.length),
      branch:
        lines
          .find(line => line.startsWith('branch '))
          ?.slice('branch '.length)
          .replace(/^refs\/heads\//, '') ?? null,
      isLocked: lines.some(line => line === 'locked' || line.startsWith('locked ')),
      isMissing: lines.some(line => line === 'prunable' || line.startsWith('prunable ')),
    }))
}

/** Reads the `<behind>\t<ahead>` of `git rev-list --left-right --count`. */
export function parseAheadBehind(output: string | null): { ahead: number | null; behind: number | null } {
  const [behind, ahead] = (output ?? '').trim().split(/\s+/).map(Number)

  return behind === undefined || ahead === undefined || Number.isNaN(behind) || Number.isNaN(ahead)
    ? { ahead: null, behind: null }
    : { ahead, behind }
}

export function formatAge(lastCommit: number, nowSeconds: number): string {
  const seconds = Math.max(0, nowSeconds - lastCommit)

  if (seconds < 60) {
    return "à l'instant"
  }

  if (seconds < 3600) {
    return `il y a ${Math.floor(seconds / 60)} min`
  }

  if (seconds < 86400) {
    return `il y a ${Math.floor(seconds / 3600)} h`
  }

  return `il y a ${Math.floor(seconds / 86400)} j`
}

function findCycles(tickets: readonly TicketHead[]): BrokenDep[] {
  const byId = new Map(tickets.map(ticket => [ticket.id, ticket]))
  const done = new Set<string>()
  const path: string[] = []
  const cycles: BrokenDep[] = []

  const visit = (ticket: TicketHead) => {
    if (done.has(ticket.id)) {
      return
    }

    const start = path.indexOf(ticket.id)

    if (start !== -1) {
      const cycle = [...path.slice(start), ticket.id].join(' -> ')
      cycles.push({ ticket, problem: `cycle ${cycle}` })

      return
    }

    path.push(ticket.id)

    for (const id of ticket.dependsOn) {
      const dependency = byId.get(id)

      if (dependency) {
        visit(dependency)
      }
    }

    path.pop()
    done.add(ticket.id)
  }

  tickets.forEach(visit)

  return cycles
}

export function findBrokenDeps(
  tickets: readonly TicketHead[],
  doneIds: readonly string[],
): BrokenDep[] {
  const known = new Set([...tickets.map(ticket => ticket.id), ...doneIds])
  const unknown = tickets.flatMap(ticket =>
    ticket.dependsOn
      .filter(id => !known.has(id))
      .map(id => ({ ticket, problem: `dépendance ${id} introuvable` })),
  )

  return [...unknown, ...findCycles(tickets)]
}

function isLive(agent: AgentState): boolean {
  return LIVE_AGENT_STATUSES.includes(agent.status)
}

/** Whether a running ticket's worker is gone. A ticket without an agent has none left. */
function isStopped(ticket: TicketHead, agents: readonly AgentState[]): boolean {
  return (
    RUNNING_STATUSES.includes(ticket.status) &&
    !agents.some(agent => agent.id === ticket.agent && isLive(agent))
  )
}

/** The text of a `## <heading>` section, without its heading. */
export function readSection(text: string, heading: string): string {
  const lines = text.split('\n')
  const start = lines.findIndex(line => line.trim() === `## ${heading}`)

  if (start === -1) {
    return ''
  }

  const rest = lines.slice(start + 1)
  const end = rest.findIndex(line => line.startsWith('## '))

  return (end === -1 ? rest : rest.slice(0, end)).join('\n').trim()
}

function lastLogLine(text: string): string {
  const entries = readSection(text, 'Log')
    .split('\n')
    .filter(line => line.trim() !== '')
  const last = (entries[entries.length - 1] ?? '').replace(/^-\s*/, '')

  return last.length > LOG_MAX_LENGTH ? `${last.slice(0, LOG_MAX_LENGTH)}…` : last
}

function buildFirstSubject(file: TicketFile): FirstSubject {
  const sections = SUBJECT_SECTIONS[file.head.status] ?? []
  const lines = sections
    .map(heading => ({ heading, body: readSection(file.text, heading) }))
    .filter(section => section.body !== '')
    .flatMap(section => [`## ${section.heading}`, ...section.body.split('\n')])

  return {
    ticket: file.head,
    excerpt: lines.slice(0, EXCERPT_MAX_LINES).join('\n'),
    isTruncated: lines.length > EXCERPT_MAX_LINES,
  }
}

function worktreeDetail(worktree: WorktreeState, nowSeconds: number, isBusy = false): string {
  if (worktree.isMissing) {
    return 'dossier absent'
  }

  const { ahead, behind, dirty, lastCommit } = worktree
  const isEmpty = dirty === 0 && ahead === 0

  return [
    isEmpty && !isBusy ? 'rien à intégrer' : '',
    isBusy && ahead === 0 ? 'aucun commit' : '',
    ahead ? `${ahead} en avance` : '',
    behind ? `${behind} en retard` : '',
    dirty ? plural(dirty, 'fichier sale', 'fichiers sales') : '',
    // Without a commit of its own, the last commit is the base's: it says nothing of this worktree.
    lastCommit === null || ahead === 0 ? '' : formatAge(lastCommit, nowSeconds),
    worktree.isLocked ? 'verrouillé' : '',
  ]
    .filter(part => part !== '')
    .join(' · ')
}

/**
 * The agent a worktree was created for, from its name: `agent-<id>`, on the branch
 * `worktree-agent-<id>` until the worker renames it.
 */
function worktreeAgentId(worktree: WorktreeState): string | undefined {
  return /(?:^|\/)agent-([\w]+)$/.exec(worktree.path)?.[1]
}

function linkedTicket(
  worktree: WorktreeState,
  tickets: readonly TicketHead[],
): TicketHead | undefined {
  const { branch } = worktree
  const agentId = worktreeAgentId(worktree)

  return tickets.find(
    ticket =>
      (agentId !== undefined && ticket.agent === agentId) ||
      (branch !== null && (ticket.branch === branch || branch.startsWith(`ticket/${ticket.id}-`))),
  )
}

function hasLiveAgent(worktree: WorktreeState, agents: readonly AgentState[]): boolean {
  return agents.some(agent => agent.id === worktreeAgentId(worktree) && isLive(agent))
}

/** Whether a ticket's worker works or waits for an answer, which leaves its worktree as it is. */
function isTicketBusy(ticket: TicketHead, agents: readonly AgentState[]): boolean {
  return (
    WAITING_WORKER_STATUSES.includes(ticket.status) ||
    (RUNNING_STATUSES.includes(ticket.status) && !isStopped(ticket, agents))
  )
}

/**
 * Picks the worktrees to show. A worker's own worktree is normal while it works
 * or waits for an answer, dirty or without commits: it stays out of the list, even
 * before the lead has noted its agent in the ticket.
 */
export function watchWorktrees(
  worktrees: readonly WorktreeState[],
  tickets: readonly TicketHead[],
  agents: readonly AgentState[],
  nowSeconds: number,
): WatchedWorktree[] {
  return worktrees.slice(1).flatMap(worktree => {
    const name = worktree.path.split('/').filter(Boolean).pop() ?? worktree.path
    const ticket = linkedTicket(worktree, tickets)
    const row = (action?: Action): WatchedWorktree => ({
      name,
      branch: worktree.branch,
      detail: worktreeDetail(worktree, nowSeconds),
      ...(action && { action }),
    })
    const isDirty = (worktree.dirty ?? 0) > 0
    const hasNothingToIntegrate = worktree.dirty === 0 && worktree.ahead === 0

    if (worktree.isMissing) {
      return [row()]
    }

    if (hasLiveAgent(worktree, agents)) {
      return []
    }

    if (ticket === undefined) {
      return [(worktree.ahead ?? 0) > 0 ? row({ label: 'intégrer', text: `intègre ${name}` }) : row()]
    }

    return !isTicketBusy(ticket, agents) && (isDirty || hasNothingToIntegrate) ? [row()] : []
  })
}

/** Counts the changes that would stop an integration: ork's own files don't. */
export function countMainDirty(porcelain: string): number {
  return porcelain
    .split('\n')
    .filter(line => line.trim() !== '')
    .map(line => line.slice(3).replace(/^"/, ''))
    .filter(path => !TOLERATED_DIRTY_PATHS.some(tolerated => path.startsWith(tolerated))).length
}

/** Reads the `<name>\t<upstream>\t<track>` lines of `git for-each-ref`. */
export function parseRefs(output: string): { name: string; upstream: string; track: string }[] {
  return output
    .split('\n')
    .filter(line => line !== '')
    .map(line => {
      const [name = '', upstream = '', track = ''] = line.split('\t')

      return { name, upstream, track }
    })
}

function upstreamDetail({ remote, track }: { remote: string; track: string }): string {
  if (track.includes('gone')) {
    return `disparue de ${remote}`
  }

  const ahead = Number(/ahead (\d+)/.exec(track)?.[1] ?? 0)
  const behind = Number(/behind (\d+)/.exec(track)?.[1] ?? 0)

  if (ahead === 0 && behind === 0) {
    return `à jour avec ${remote}`
  }

  return [
    ahead > 0 ? plural(ahead, 'non poussé', 'non poussés') : '',
    behind > 0 ? `${behind} à récupérer` : '',
  ]
    .filter(part => part !== '')
    .join(' · ')
}

function aheadBehindParts({ ahead, behind }: { ahead: number | null; behind: number | null }): string[] {
  return [ahead ? `${ahead} en avance` : '', behind ? `${behind} en retard` : '']
}

function joinDetail(parts: readonly string[]): string {
  return parts.filter(part => part !== '').join(' · ')
}

function worktreeGitRow(
  worktree: WorktreeState,
  tickets: readonly TicketHead[],
  agents: readonly AgentState[],
  nowSeconds: number,
): GitRow {
  const name = worktree.path.split('/').filter(Boolean).pop() ?? worktree.path
  const ticket = linkedTicket(worktree, tickets)
  const isWaiting = ticket !== undefined && WAITING_WORKER_STATUSES.includes(ticket.status)
  const isActive =
    hasLiveAgent(worktree, agents) || (ticket !== undefined && !isWaiting && isTicketBusy(ticket, agents))
  const canIntegrate = ticket === undefined && !isActive && !worktree.isMissing && (worktree.ahead ?? 0) > 0

  return {
    key: `worktree-${name}`,
    name: worktree.branch ?? name,
    detail: joinDetail([
      ticket?.id ?? '',
      isActive ? 'ouvrier actif' : '',
      isWaiting ? 'en attente' : '',
      worktreeDetail(worktree, nowSeconds, isActive || isWaiting),
    ]),
    ...(canIntegrate && { action: { label: 'intégrer', text: `intègre ${name}` } }),
  }
}

function buildGitView(input: StatusInput): GitView | null {
  const { gitFacts, integrationBranch, worktrees } = input

  if (gitFacts === null) {
    return null
  }

  const heads = input.tickets.map(file => file.head)
  const { mainDirty, reference, upstream } = gitFacts
  const main = worktrees[0]
  const mainRows: GitRow[] =
    main === undefined || main.branch === integrationBranch
      ? []
      : [
          {
            key: 'main-repository',
            name: main.branch ?? 'HEAD détachée',
            detail: joinDetail(['dépôt principal', ...aheadBehindParts(main)]),
          },
        ]

  return {
    trunk: {
      name: integrationBranch,
      detail: joinDetail([
        ...(reference === null
          ? []
          : [
              reference.ahead ? `${reference.ahead} en avance sur ${reference.name}` : '',
              reference.behind ? `${reference.behind} en retard sur ${reference.name}` : '',
            ]),
        upstream === null ? '' : upstreamDetail(upstream),
        mainDirty ? plural(mainDirty, 'fichier sale', 'fichiers sales') : '',
      ]),
    },
    rows: [
      ...mainRows,
      ...worktrees.slice(1).map(one => worktreeGitRow(one, heads, input.agents, input.nowSeconds)),
      ...gitFacts.otherBranches.map(branch => ({
        key: `branch-${branch.name}`,
        name: branch.name,
        detail: joinDetail([...aheadBehindParts(branch), 'sans worktree']),
      })),
    ],
    hiddenBranches: gitFacts.hiddenBranches,
  }
}

/** Works out what the lead and the panel need to know, from what was read. */
export function buildStatus(input: StatusInput): StatusReport {
  const heads = input.tickets.map(file => file.head)
  const reviewId = /^review\/(.+)$/.exec(input.mainBranch ?? '')?.[1]
  const reviewTicket = heads.find(ticket => ticket.id === reviewId)
  const queue = subjectQueue(heads)
  const firstFile = input.tickets.find(file => file.head.id === queue[0]?.id)
  const hasLiveAgent = input.agents.some(isLive)

  return {
    integrationBranch: input.integrationBranch,
    stopFile: !input.hasStopFile ? 'no' : hasLiveAgent ? 'active' : 'stale',
    review:
      reviewId === undefined
        ? null
        : {
            branch: input.mainBranch ?? '',
            ...(reviewTicket && { ticket: reviewTicket }),
          },
    stopped: input.tickets
      .filter(file => isStopped(file.head, input.agents))
      .map(file => ({
        ticket: file.head,
        mode: file.head.status === 'planning' ? 'plan' : 'exécution',
        log: lastLogLine(file.text),
      })),
    brokenDeps: findBrokenDeps(heads, input.doneIds),
    worktrees: watchWorktrees(input.worktrees, heads, input.agents, input.nowSeconds),
    queue,
    ready: heads
      .filter(ticket => ticket.status === 'ready' && !isPaused(ticket))
      .map(ticket => ({ ticket, blockers: openBlockers(ticket, heads) })),
    paused: heads.filter(isPaused),
    firstSubject: reviewId === undefined && firstFile ? buildFirstSubject(firstFile) : null,
    git: buildGitView(input),
  }
}

/** What the panel shows: an alert on the row of its ticket, and the state of git. */
export function buildHealth(report: StatusReport): Health {
  const alerts: Record<string, TicketAlert> = {}
  const add = (ticket: TicketHead, text: string, action?: TicketAlert['action']) => {
    const known = alerts[ticket.id]
    const merged = known?.action ?? action

    alerts[ticket.id] = {
      text: known ? `${known.text} · ${text}` : text,
      ...(merged && { action: merged }),
    }
  }

  if (report.review?.ticket) {
    add(report.review.ticket, `relecture ouverte sur ${report.review.branch}`, {
      label: 'reprendre',
      text: `relisons ${report.review.ticket.id}`,
    })
  }

  report.stopped.forEach(({ ticket }) => {
    add(ticket, 'ouvrier arrêté', { label: 'reprendre', text: `reprends ${ticket.id}` })
  })
  report.brokenDeps.forEach(({ ticket, problem }) => {
    add(ticket, problem)
  })

  return { alerts, git: report.git }
}

function listOrNone(items: readonly string[]): string {
  return items.length === 0 ? 'none' : items.join('; ')
}

/** The text the lead reads under the status skill. */
export function renderSnapshot(report: StatusReport): string {
  const { firstSubject, review } = report
  const reviewName = review?.ticket ? ticketName(review.ticket) : 'unknown ticket'
  const lines = [
    'ork_status: ok',
    `integration_branch: ${report.integrationBranch}`,
    `stop_file: ${report.stopFile}`,
    `review_open: ${review === null ? 'none' : `${reviewName} on ${review.branch}`}`,
    `stopped: ${listOrNone(
      report.stopped.map(({ ticket, mode, log }) => `${ticketName(ticket)} (${mode}) | log: ${log}`),
    )}`,
    `broken_deps: ${listOrNone(
      report.brokenDeps.map(({ ticket, problem }) => `${ticketName(ticket)}: ${problem}`),
    )}`,
    `worktrees: ${listOrNone(
      report.worktrees.map(({ name, branch, detail, action }) =>
        [`${name} (${branch ?? 'detached'})`, detail, action?.text]
          .filter(part => part !== undefined && part !== '')
          .join(' | '),
      ),
    )}`,
    `queue: ${listOrNone(
      report.queue.map(ticket => `${ticketName(ticket)} ${ticket.status} ${ticket.priority}`),
    )}`,
    `ready: ${listOrNone(
      report.ready.map(({ ticket, blockers }) =>
        blockers.length > 0
          ? `${ticketName(ticket)} waits ${blockers.join(', ')}`
          : `${ticketName(ticket)} launchable`,
      ),
    )}`,
    `paused: ${listOrNone(report.paused.map(ticket => `${ticketName(ticket)} (${ticket.paused})`))}`,
    `first_subject: ${
      firstSubject === null
        ? 'none'
        : `${ticketName(firstSubject.ticket)} (${firstSubject.ticket.status})`
    }`,
  ]

  if (firstSubject !== null && firstSubject.excerpt !== '') {
    lines.push(firstSubject.excerpt)
  }

  if (firstSubject?.isTruncated) {
    lines.push('truncated: read the file')
  }

  return lines.join('\n')
}
