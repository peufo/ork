import { expect, test } from 'claude-code/testing'

import {
  buildHealth,
  buildStatus,
  countMainDirty,
  findBrokenDeps,
  formatAge,
  parseAheadBehind,
  parseRefs,
  parseWorktrees,
  readSection,
  renderSnapshot,
  watchWorktrees,
} from './status'
import type { GitFacts, StatusInput, WorktreeState } from './status'
import type { TicketHead } from './tickets'

const NOW = 1_000_000

const ticket = (id: string, status: string, more: Partial<TicketHead> = {}): TicketHead => ({
  file: `${id}-a.md`,
  id,
  title: `Titre ${id}`,
  status,
  priority: 'normal',
  dependsOn: [],
  branch: '',
  agent: '',
  paused: '',
  ...more,
})

const MAIN: WorktreeState = {
  path: '/repo',
  branch: 'main',
  isLocked: false,
  isMissing: false,
  ahead: null,
  behind: null,
  dirty: null,
  lastCommit: null,
}

const worktree = (name: string, more: Partial<WorktreeState> = {}): WorktreeState => ({
  ...MAIN,
  path: `/repo/.claude/worktrees/${name}`,
  branch: name,
  ahead: 0,
  behind: 0,
  dirty: 0,
  lastCommit: NOW - 7200,
  ...more,
})

const input = (more: Partial<StatusInput> = {}): StatusInput => ({
  tickets: [],
  doneIds: [],
  agents: [],
  hasStopFile: false,
  integrationBranch: 'main',
  mainBranch: 'main',
  worktrees: [MAIN],
  gitFacts: null,
  nowSeconds: NOW,
  ...more,
})

const file = (head: TicketHead, body = '') => ({ head, text: `---\nid: ${head.id}\n---\n${body}` })

test('reads worktrees, whatever their path holds', () => {
  expect(
    parseWorktrees(
      [
        'worktree /repo',
        'HEAD a',
        'branch refs/heads/main',
        '',
        'worktree /repo/mon dossier',
        'HEAD b',
        'branch refs/heads/ticket/t1-a',
        'locked claude agent',
        '',
        'worktree /gone',
        'HEAD c',
        'detached',
        'prunable gitdir file points to non-existent location',
        '',
      ].join('\n'),
    ),
  ).toEqual([
    { path: '/repo', branch: 'main', isLocked: false, isMissing: false },
    { path: '/repo/mon dossier', branch: 'ticket/t1-a', isLocked: true, isMissing: false },
    { path: '/gone', branch: null, isLocked: false, isMissing: true },
  ])
  expect(parseWorktrees('')).toEqual([])
})

test('reads behind and ahead, and says when git gave nothing', () => {
  expect(parseAheadBehind('1\t2\n')).toEqual({ ahead: 2, behind: 1 })
  expect(parseAheadBehind(null)).toEqual({ ahead: null, behind: null })
  expect(parseAheadBehind('')).toEqual({ ahead: null, behind: null })
})

test('says an age in French', () => {
  expect(formatAge(NOW - 5, NOW)).toBe("à l'instant")
  expect(formatAge(NOW - 300, NOW)).toBe('il y a 5 min')
  expect(formatAge(NOW - 7200, NOW)).toBe('il y a 2 h')
  expect(formatAge(NOW - 172800, NOW)).toBe('il y a 2 j')
})

test('finds a dependency that exists nowhere, and accepts a closed one', () => {
  expect(
    findBrokenDeps([ticket('t2', 'ready', { dependsOn: ['t1', 't99'] })], ['t1']).map(
      broken => broken.problem,
    ),
  ).toEqual(['dépendance t99 introuvable'])
})

test('finds a cycle once', () => {
  const tickets = [
    ticket('t1', 'ready', { dependsOn: ['t2'] }),
    ticket('t2', 'ready', { dependsOn: ['t1'] }),
    ticket('t3', 'ready', { dependsOn: ['t3'] }),
    ticket('t4', 'ready', { dependsOn: ['t1'] }),
  ]

  expect(findBrokenDeps(tickets, []).map(broken => broken.problem)).toEqual([
    'cycle t1 -> t2 -> t1',
    'cycle t3 -> t3',
  ])
})

test('reads one section of a ticket', () => {
  const text = '## Context\nici\n\n## Questions\nQ1 ?\n- oui\n\n## Log\n- 2026'

  expect(readSection(text, 'Questions')).toBe('Q1 ?\n- oui')
  expect(readSection(text, 'Log')).toBe('- 2026')
  expect(readSection(text, 'Plan')).toBe('')
})

test('calls a worker stopped when it is gone or finished, not while it lives', () => {
  const stopped = (tickets: TicketHead[], agents: StatusInput['agents']) =>
    buildStatus(input({ tickets: tickets.map(one => file(one)), agents })).stopped.map(
      one => `${one.ticket.id} ${one.mode}`,
    )

  expect(stopped([ticket('t1', 'planning', { agent: 'a1' })], [])).toEqual(['t1 plan'])
  expect(stopped([ticket('t1', 'in-progress')], [])).toEqual(['t1 exécution'])
  expect(
    stopped([ticket('t1', 'in-progress', { agent: 'a1' })], [{ id: 'a1', status: 'completed' }]),
  ).toEqual(['t1 exécution'])
  expect(
    stopped([ticket('t1', 'in-progress', { agent: 'a1' })], [{ id: 'a1', status: 'running' }]),
  ).toEqual([])
  expect(stopped([ticket('t1', 'blocked', { agent: 'a1' })], [])).toEqual([])
})

test('gives the last line of a stopped ticket log', () => {
  const report = buildStatus(
    input({
      tickets: [file(ticket('t1', 'planning'), '## Log\n- 2026-11-01 créé\n- 2026-11-02 parti\n')],
    }),
  )

  expect(report.stopped[0]?.log).toBe('2026-11-02 parti')
})

test('offers to integrate only a worktree outside any ticket that has commits', () => {
  const watched = (one: WorktreeState) =>
    watchWorktrees([MAIN, one], [], [], NOW).map(row => [row.detail, row.action?.text])

  expect(watched(worktree('x', { ahead: 2, behind: 1, dirty: 3, isLocked: true }))).toEqual([
    ['2 en avance · 1 en retard · 3 fichiers sales · il y a 2 h · verrouillé', 'intègre x'],
  ])
  expect(watched(worktree('x'))).toEqual([['rien à intégrer', undefined]])
  expect(watched(worktree('x', { dirty: 1 }))).toEqual([['1 fichier sale', undefined]])
})

test('concludes nothing when git could not answer', () => {
  const rows = watchWorktrees([MAIN, worktree('x', { ahead: null, dirty: null })], [], [], NOW)

  expect(rows[0]?.action).toBeUndefined()
  expect(rows[0]?.detail).toBe('il y a 2 h')
})

test('leaves out the worktree of a live agent before its branch is renamed', () => {
  const fresh = worktree('agent-a1', { branch: 'worktree-agent-a1', isLocked: true })
  const live = [{ id: 'a1', status: 'running' }]

  expect(watchWorktrees([MAIN, fresh], [], live, NOW)).toEqual([])
  expect(watchWorktrees([MAIN, fresh], [ticket('t1', 'in-progress', { agent: 'a1' })], live, NOW)).toEqual([])
  expect(watchWorktrees([MAIN, fresh], [], [], NOW).map(row => row.name)).toEqual(['agent-a1'])
})

test('never lists the main repository, and says when a folder is gone', () => {
  expect(watchWorktrees([MAIN], [], [], NOW)).toEqual([])
  expect(watchWorktrees([MAIN, worktree('x', { isMissing: true })], [], [], NOW)).toEqual([
    { name: 'x', branch: 'x', detail: 'dossier absent' },
  ])
})

test('leaves out the worktree of a worker that works or waits for an answer', () => {
  const branch = 'ticket/t1-a'
  const dirty = worktree('t1-a', { branch, dirty: 4 })
  const clean = worktree('t1-a', { branch })
  const alive = [{ id: 'a1', status: 'running' }]
  const running = ticket('t1', 'in-progress', { agent: 'a1' })

  expect(watchWorktrees([MAIN, dirty], [running], alive, NOW)).toEqual([])
  expect(watchWorktrees([MAIN, clean], [running], alive, NOW)).toEqual([])
  expect(watchWorktrees([MAIN, clean], [ticket('t1', 'plan-review')], [], NOW)).toEqual([])
  expect(watchWorktrees([MAIN, dirty], [ticket('t1', 'blocked')], [], NOW)).toEqual([])
})

test('shows a ticket worktree that needs a look, without a button', () => {
  const dirty = worktree('t1-a', { branch: 'ticket/t1-a', dirty: 1, ahead: 1 })
  const stopped = watchWorktrees([MAIN, dirty], [ticket('t1', 'in-progress')], [], NOW)

  expect(stopped.map(row => [row.name, row.action])).toEqual([['t1-a', undefined]])

  const linkedByField = worktree('y', { branch: 'mine', ahead: 1 })

  expect(watchWorktrees([MAIN, linkedByField], [ticket('t1', 'in-review', { branch: 'mine' })], [], NOW)).toEqual([])
})

test('orders the queue, and gives the excerpt of the first subject', () => {
  const report = buildStatus(
    input({
      tickets: [
        file(ticket('t5', 'idea'), '## Context\nune idée\n\n## Questions\nlaquelle ?\n'),
        file(ticket('t4', 'in-review'), '## Review\n- [ ] point\n'),
        file(ticket('t3', 'plan-review'), '## Plan\nle plan\n'),
        file(ticket('t2', 'blocked'), '## Questions\nmoins urgent\n'),
        file(ticket('t9', 'blocked', { priority: 'high' }), '## Questions\nla question ?\n'),
        file(ticket('t8', 'ready')),
      ],
    }),
  )

  expect(report.queue.map(one => one.id)).toEqual(['t9', 't2', 't3', 't4', 't5'])
  expect(report.firstSubject?.excerpt).toBe('## Questions\nla question ?')
})

test('takes the section that suits each status', () => {
  const excerpt = (head: TicketHead, body: string) =>
    buildStatus(input({ tickets: [file(head, body)] })).firstSubject?.excerpt

  expect(excerpt(ticket('t1', 'plan-review'), '## Plan\nle plan\n## Log\n- x')).toBe('## Plan\nle plan')
  expect(excerpt(ticket('t1', 'in-review'), '## Review\n- [ ] p\n')).toBe('## Review\n- [ ] p')
  expect(excerpt(ticket('t1', 'idea'), '## Context\nc\n\n## Questions\nq ?\n')).toBe(
    '## Context\nc\n## Questions\nq ?',
  )
})

test('cuts a long excerpt, and says so', () => {
  const body = `## Plan\n${Array.from({ length: 60 }, (_, index) => `ligne ${index}`).join('\n')}\n`
  const report = buildStatus(input({ tickets: [file(ticket('t1', 'plan-review'), body)] }))

  expect(report.firstSubject?.isTruncated).toBe(true)
  expect(report.firstSubject?.excerpt.split('\n').length).toBe(40)
  expect(renderSnapshot(report)).toContain('truncated: read the file')
})

test('a review under way comes first: no excerpt, and its ticket offers to go back to it', () => {
  const report = buildStatus(
    input({
      mainBranch: 'review/t3',
      tickets: [file(ticket('t3', 'in-review')), file(ticket('t1', 'blocked'), '## Questions\nq ?\n')],
    }),
  )

  expect(report.review?.ticket?.id).toBe('t3')
  expect(report.firstSubject).toBeNull()
  expect(buildHealth(report).alerts.t3).toEqual({
    text: 'relecture ouverte sur review/t3',
    action: { label: 'reprendre', text: 'relisons t3' },
  })
  expect(renderSnapshot(report)).toContain('review_open: t3-a on review/t3')
})

test('a stop file is stale once no worker lives', () => {
  const stopFile = (agents: StatusInput['agents']) =>
    buildStatus(input({ hasStopFile: true, agents })).stopFile

  expect(buildStatus(input()).stopFile).toBe('no')
  expect(stopFile([])).toBe('stale')
  expect(stopFile([{ id: 'a', status: 'completed' }])).toBe('stale')
  expect(stopFile([{ id: 'a', status: 'running' }])).toBe('active')
})

test('an alert goes on the row of its ticket, and adds up', () => {
  const { alerts } = buildHealth(
    buildStatus(
      input({
        tickets: [
          file(ticket('t1', 'in-progress', { dependsOn: ['t98'] })),
          file(ticket('t2', 'ready', { dependsOn: ['t99'] })),
          file(ticket('t3', 'ready')),
        ],
      }),
    ),
  )

  expect(alerts).toEqual({
    t1: {
      text: 'ouvrier arrêté · dépendance t98 introuvable',
      action: { label: 'reprendre', text: 'reprends t1' },
    },
    t2: { text: 'dépendance t99 introuvable' },
  })
})

const FACTS: GitFacts = {
  upstream: { remote: 'origin', track: '' },
  reference: { name: 'main', ahead: 13, behind: 0 },
  mainDirty: 0,
  otherBranches: [],
  hiddenBranches: 0,
}

const gitView = (more: Partial<StatusInput>) =>
  buildStatus(
    input({
      integrationBranch: 'dev',
      mainBranch: 'dev',
      worktrees: [{ ...MAIN, branch: 'dev' }],
      gitFacts: FACTS,
      ...more,
    }),
  ).git

test('has no git view when git could not answer', () => {
  expect(buildStatus(input()).git).toBeNull()
})

test('says where the integration branch stands', () => {
  const trunk = (facts: Partial<GitFacts>) => gitView({ gitFacts: { ...FACTS, ...facts } })?.trunk

  expect(trunk({})).toEqual({ name: 'dev', detail: '13 en avance sur main · à jour avec origin' })
  expect(
    trunk({
      reference: { name: 'main', ahead: 0, behind: 2 },
      upstream: { remote: 'origin', track: '[ahead 3, behind 1]' },
      mainDirty: 2,
    })?.detail,
  ).toBe('2 en retard sur main · 3 non poussés · 1 à récupérer · 2 fichiers sales')
  expect(trunk({ reference: null, upstream: null })?.detail).toBe('')
  expect(trunk({ reference: null, upstream: { remote: 'origin', track: '[gone]' } })?.detail).toBe(
    'disparue de origin',
  )
})

test('shows every worktree, a worker being one', () => {
  const view = gitView({
    tickets: [
      file(ticket('t1', 'in-progress', { agent: 'a1' })),
      file(ticket('t2', 'plan-review', { branch: 'ticket/t2-a' })),
    ],
    agents: [{ id: 'a1', status: 'running' }],
    worktrees: [
      { ...MAIN, branch: 'dev' },
      worktree('agent-a1', { branch: 'worktree-agent-a1', isLocked: true }),
      worktree('t2-a', { branch: 'ticket/t2-a' }),
      worktree('x', { ahead: 1, dirty: 2 }),
      worktree('y'),
      worktree('z', { isMissing: true, branch: null }),
    ],
  })

  expect(view?.rows).toEqual([
    {
      key: 'worktree-agent-a1',
      name: 'worktree-agent-a1',
      detail: 't1 · ouvrier actif · aucun commit · verrouillé',
    },
    {
      key: 'worktree-t2-a',
      name: 'ticket/t2-a',
      detail: 't2 · en attente · aucun commit',
    },
    {
      key: 'worktree-x',
      name: 'x',
      detail: '1 en avance · 2 fichiers sales · il y a 2 h',
      action: { label: 'intégrer', text: 'intègre x' },
    },
    { key: 'worktree-y', name: 'y', detail: 'rien à intégrer' },
    { key: 'worktree-z', name: 'z', detail: 'dossier absent' },
  ])
})

test('shows a live agent without a ticket, and offers it nothing', () => {
  const view = gitView({
    agents: [{ id: 'a1', status: 'running' }],
    worktrees: [{ ...MAIN, branch: 'dev' }, worktree('agent-a1', { branch: 'worktree-agent-a1', ahead: 1 })],
  })

  expect(view?.rows[0]?.detail).toBe('ouvrier actif · 1 en avance · il y a 2 h')
  expect(view?.rows[0]?.action).toBeUndefined()
})

test('shows the main repository only when a review moved it off the branch', () => {
  const view = gitView({
    mainBranch: 'review/t3',
    worktrees: [{ ...MAIN, branch: 'review/t3', ahead: 2, behind: 0 }],
  })

  expect(view?.rows).toEqual([
    { key: 'main-repository', name: 'review/t3', detail: 'dépôt principal · 2 en avance' },
  ])
})

test('adds the branches with commits that have no worktree', () => {
  const view = gitView({
    gitFacts: { ...FACTS, otherBranches: [{ name: 'vieux', ahead: 3, behind: 40 }], hiddenBranches: 5 },
  })

  expect(view?.rows).toEqual([
    { key: 'branch-vieux', name: 'vieux', detail: '3 en avance · 40 en retard · sans worktree' },
  ])
  expect(view?.hiddenBranches).toBe(5)
})

test('counts the dirty files, without those of ork', () => {
  expect(countMainDirty('')).toBe(0)
  expect(
    countMainDirty(
      ' M .ork/tickets/t1-a.md\n?? .ork/config.md\n?? .claude/launch.json\n M src/a.ts\n?? "b c.ts"\n',
    ),
  ).toBe(2)
})

test('reads refs, with or without an upstream', () => {
  expect(parseRefs('dev\torigin/dev\t[ahead 1]\nlocal\t\t\n')).toEqual([
    { name: 'dev', upstream: 'origin/dev', track: '[ahead 1]' },
    { name: 'local', upstream: '', track: '' },
  ])
})

test('gives the lead a short snapshot', () => {
  const report = buildStatus(
    input({
      doneIds: ['t1'],
      tickets: [
        file(ticket('t2', 'blocked'), '## Questions\nla question ?\n'),
        file(ticket('t3', 'ready', { dependsOn: ['t1'] })),
        file(ticket('t4', 'ready', { dependsOn: ['t2'] })),
      ],
    }),
  )

  expect(renderSnapshot(report)).toBe(
    [
      'ork_status: ok',
      'integration_branch: main',
      'stop_file: no',
      'review_open: none',
      'stopped: none',
      'broken_deps: none',
      'worktrees: none',
      'queue: t2-a blocked normal',
      'ready: t3-a launchable; t4-a waits t2',
      'paused: none',
      'first_subject: t2-a (blocked)',
      '## Questions',
      'la question ?',
    ].join('\n'),
  )
})

test('a paused ticket is no subject, and the lead knows why', () => {
  const report = buildStatus(
    input({
      tickets: [
        file(ticket('t1', 'idea', { paused: "jusqu'à t3" }), '## Context\nici\n'),
        file(ticket('t2', 'ready', { paused: 'plus tard' })),
      ],
    }),
  )

  expect(report.queue).toEqual([])
  expect(report.firstSubject).toBeNull()
  expect(report.ready).toEqual([])

  const snapshot = renderSnapshot(report)

  expect(snapshot).toContain("paused: t1-a (jusqu'à t3); t2-a (plus tard)")
  expect(snapshot).toContain('first_subject: none')
})

test('the snapshot of an empty project has no subject', () => {
  const snapshot = renderSnapshot(buildStatus(input()))

  expect(snapshot).toContain('queue: none')
  expect(snapshot).toContain('first_subject: none')
})
