import { expect, test } from 'claude-code/testing'

import { buildBoard, isFocused, parseTicketHead } from './tickets'
import type { TicketHead } from './tickets'

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

test('reads the frontmatter only', () => {
  expect(
    parseTicketHead(
      't14-a.md',
      '---\nid: t14\ntitle: Un titre : avec deux-points\nstatus: blocked\npriority: high\ndepends_on: [t1, t2]\nbranch: ticket/t14-a\nagent: abc\n---\n\nstatus: done\n',
    ),
  ).toEqual({
    file: 't14-a.md',
    id: 't14',
    title: 'Un titre : avec deux-points',
    status: 'blocked',
    priority: 'high',
    dependsOn: ['t1', 't2'],
    branch: 'ticket/t14-a',
    agent: 'abc',
    paused: '',
  })
  expect(parseTicketHead('a.md', 'pas de frontmatter')).toBeUndefined()
  expect(parseTicketHead('a.md', '---\ntitle: sans id\n---\n')).toBeUndefined()
})

test('puts what waits for the developer first, by urgency', () => {
  const board = buildBoard([
    ticket('t3', 'in-review'),
    ticket('t2', 'plan-review'),
    ticket('t1', 'blocked'),
    ticket('t9', 'blocked', { priority: 'high' }),
  ])

  expect(board.waiting.map(row => row.id)).toEqual(['t9', 't1', 't2', 't3'])
  expect(board.waiting[3]?.action).toEqual({ label: 'relire', text: 'relisons t3' })
})

test('sets a paused ticket apart, with its reason, and it no longer waits for anyone', () => {
  const board = buildBoard([
    ticket('t1', 'idea', { paused: "jusqu'à t3" }),
    ticket('t2', 'ready', { paused: 'plus tard' }),
    ticket('t3', 'idea'),
  ])

  expect(board.ideas.map(row => row.id)).toEqual(['t3'])
  expect(board.queued).toEqual([])
  expect(board.paused).toEqual([
    {
      file: 't1-a.md',
      id: 't1',
      title: 'Titre t1',
      reason: "jusqu'à t3",
      action: { label: 'reprendre', text: 'sors t1 de la pause' },
    },
    {
      file: 't2-a.md',
      id: 't2',
      title: 'Titre t2',
      reason: 'plus tard',
      action: { label: 'reprendre', text: 'sors t2 de la pause' },
    },
  ])
})

test('keeps in sight a ticket under way or waiting, whatever its field says', () => {
  const board = buildBoard([
    ticket('t1', 'blocked', { paused: 'oubli' }),
    ticket('t2', 'in-progress', { paused: 'oubli' }),
  ])

  expect(board.waiting.map(row => row.id)).toEqual(['t1'])
  expect(board.running.map(row => row.id)).toEqual(['t2'])
  expect(board.paused).toEqual([])
})

test('says when a clean stop is under way', () => {
  expect(buildBoard([]).isStopping).toBe(false)
  expect(buildBoard([], true).isStopping).toBe(true)
})

test('finds the rows of the current subject', () => {
  const row = { file: 't14-a.md', id: 't14', title: 'T' }

  expect(isFocused(row, '.ork/tickets', '.ork/tickets/t14-a.md')).toBe(true)
  expect(isFocused(row, '.ork/tickets', './.ork/tickets/t14-a.md')).toBe(true)
  expect(isFocused(row, '.ork/tickets', 'doc/story/a.md')).toBe(false)
  expect(isFocused(row, '.ork/tickets', '.ork/tickets/t15-b.md')).toBe(false)
  expect(isFocused(row, '.ork/tickets', null)).toBe(false)
})

test('says what a ready ticket waits for, and offers to start the others', () => {
  const board = buildBoard([
    ticket('t1', 'in-progress'),
    ticket('t2', 'ready', { dependsOn: ['t1', 't0'] }),
    ticket('t3', 'ready'),
    ticket('t4', 'idea'),
  ])

  expect(board.running.map(row => row.note)).toEqual(['exécution'])
  expect(board.queued[0]?.note).toBe('attend t1')
  expect(board.queued[0]?.action).toBeUndefined()
  expect(board.queued[1]?.action).toEqual({ label: 'lancer', text: 'lance t3' })
  expect(board.ideas[0]?.action).toEqual({ label: 'clarifier', text: 'clarifions t4' })
})
