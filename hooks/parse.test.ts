import { expect, test } from 'claude-code/testing'

import { parseMessage } from './parse'

const SUBJECT_MESSAGE = [
  '[OK] `t12-bouton` -> `main` : sans relecture',
  '[NEW] `t11-renommer-deck` ready',
  '',
  '### ⏸ [t14-export-pdf](.ork/tickets/t14-export-pdf.md) · ouvrier en pause',
  "L'ancien export CSV est encore appelé.",
  '**Garder l\'ancien export CSV ?**',
  '',
  '`1` : le garder, rien ne casse *(recommandé · 6/10)*',
  '`2` : le supprimer',
  '`sujet suivant` : passer à `t11-renommer-deck`',
].join('\n')

test('reads status lines, subject, body and actions', () => {
  const parsed = parseMessage(SUBJECT_MESSAGE)

  expect(parsed?.statusLines.map(line => line.tag)).toEqual(['OK', 'NEW'])
  expect(parsed?.subject?.title).toBe('t14-export-pdf')
  expect(parsed?.subject?.href).toBe('.ork/tickets/t14-export-pdf.md')
  expect(parsed?.subject?.step).toBe('ouvrier en pause')
  expect(parsed?.body).toContain('**Garder')
  expect(parsed?.actions.map(action => action.word)).toEqual(['1', '2', 'sujet suivant'])
  expect(parsed?.actions[0]).toEqual({
    word: '1',
    effect: 'le garder, rien ne casse',
    isRecommended: true,
    confidence: 6,
  })
})

test('reads a message of status lines only', () => {
  const parsed = parseMessage('[NEW] `t29-tri` ready')

  expect(parsed?.statusLines).toEqual([{ tag: 'NEW', text: '`t29-tri` ready' }])
  expect(parsed?.subject).toBeUndefined()
})

test('keeps the prose under status lines', () => {
  const parsed = parseMessage('[NOTE] `t1-a` : vu\n\nUne réponse de fond.')

  expect(parsed?.body).toBe('Une réponse de fond.')
})

test('reads a subject that follows a sentence', () => {
  const parsed = parseMessage(
    "Rien n'a bougé depuis tout à l'heure.\n\n### 💡 [Tapis roulant](doc/story/tapis-roulant.md) · en pause\n**On reprend ?**\n\n`1` : oui *(recommandé · 7/10)*\n`2` : non",
  )

  expect(parsed?.statusLines).toEqual([])
  expect(parsed?.intro).toBe("Rien n'a bougé depuis tout à l'heure.")
  expect(parsed?.subject?.title).toBe('Tapis roulant')
  expect(parsed?.body).toBe('**On reprend ?**')
  expect(parsed?.actions.map(action => action.word)).toEqual(['1', '2'])
})

test('leaves other text alone', () => {
  expect(parseMessage('Une réponse normale.')).toBeUndefined()
  expect(parseMessage('Voir [OK] plus loin.')).toBeUndefined()
  expect(parseMessage('[TODO] pas un tag ork')).toBeUndefined()
  expect(parseMessage('[WAIT] un ancien tag')).toBeUndefined()
  expect(parseMessage('### Un titre normal')).toBeUndefined()
})

test('keeps an incomplete action in the body while it is being written', () => {
  const parsed = parseMessage(
    '### ⏸ [t1-a](.ork/tickets/t1-a.md) · question\nCorps\n\n`1',
  )

  expect(parsed?.actions).toEqual([])
  expect(parsed?.body).toContain('`1')
})

test('does not read the action words inside the body', () => {
  const parsed = parseMessage(
    '### 👀 [t1-a](.ork/tickets/t1-a.md) · relecture\n`a.ts:1` : un point\nSuite du corps\n\n`ok` : valider',
  )

  expect(parsed?.body).toContain('`a.ts:1` : un point')
  expect(parsed?.actions.map(action => action.word)).toEqual(['ok'])
})
