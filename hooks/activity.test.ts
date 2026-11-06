import { expect, test } from 'claude-code/testing'

import { agentName, describeToolCall, ticketOfDescription } from './activity'

test('says what a tool call does', () => {
  expect(describeToolCall('Bash', { command: 'bun test', description: 'Run the tests' })).toBe(
    'Run the tests',
  )
  expect(describeToolCall('Bash', { command: 'ls' })).toBe('Lance une commande')
  expect(describeToolCall('Read', { file_path: '/a/b/c.ts' })).toBe('Lit c.ts')
  expect(describeToolCall('Edit', { file_path: '/a/b/c.ts' })).toBe('Modifie c.ts')
  expect(describeToolCall('Grep', { pattern: 'x' })).toBe('Cherche dans le code')
  expect(describeToolCall('mcp__x__y', {})).toBe('Utilise un outil externe')
})

test('shortens an agent type', () => {
  expect(agentName('ork:worker')).toBe('worker')
  expect(agentName('Explore')).toBe('Explore')
})

test('finds the ticket in an agent description', () => {
  expect(ticketOfDescription('t33 plan vélocités')).toBe('t33')
  expect(ticketOfDescription('T7 exécution')).toBe('t7')
  expect(ticketOfDescription('Chercher le bouton')).toBeUndefined()
  expect(ticketOfDescription('t33a plan')).toBeUndefined()
})
