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
  ...more,
})

test('reads the frontmatter only', () => {
  expect(
    parseTicketHead(
      't14-a.md',
      '---\nid: t14\ntitle: Un titre : avec deux-points\nstatus: blocked\npriority: high\ndepends_on: [t1, t2]\n---\n\nstatus: done\n',
    ),
  ).toEqual({
    file: 't14-a.md',
    id: 't14',
    title: 'Un titre : avec deux-points',
    status: 'blocked',
    priority: 'high',
    dependsOn: ['t1', 't2'],
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
