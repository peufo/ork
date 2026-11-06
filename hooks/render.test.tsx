import { expect, test } from 'claude-code/testing'
import type { TestBody } from 'claude-code/testing'

const TEXT = [
  '[OK] `t12-a` -> `main`',
  '',
  '### ⏸ [t14-b](.ork/tickets/t14-b.md) · ouvrier en pause',
  '**Question ?**',
  '',
  '`1` : oui *(recommandé · 6/10)*',
  '`2` : non',
].join('\n')

const message = (text: string) =>
  ({
    plugin: 'ork',
    surface: 'desktop',
    component: 'AssistantMessage',
    requestId: 'message',
    props: { text, isFirstOfReply: true },
  }) as const

test('draws a lead message on the desktop, and a button answers', async ($, on) => {
  const submitted: string[] = []
  on('prompt.submit', (_, e) => {
    submitted.push(e.text)
    return { text: e.text }
  })

  const ui = await $.ui.mount(message(TEXT))

  expect((await ui.find({ type: 'Text', text: /^OK$/ }))?.text).toBe('OK')
  expect(await ui.find({ key: '1' })).toBeDefined()
  await ui.press({ key: '2' })
  expect(submitted).toEqual(['2'])
  await ui.unmount()

  const again = await $.ui.mount(message(TEXT))

  expect(await again.find({ key: '1' })).toBeUndefined()
  await again.unmount()
})

test('a typed reply also retires the buttons', async ($, on) => {
  on('prompt.submit', (_, e) => ({ text: e.text }))

  const ui = await $.ui.mount(message(TEXT))
  await ui.unmount()
  await $.prompt.submit({ text: 'une réponse libre', wait: false, origin: { kind: 'composer' } })

  const again = await $.ui.mount(message(TEXT))

  expect(await again.find({ key: '1' })).toBeUndefined()
  await again.unmount()
})

test('leaves ordinary text to the engine', async ($, on) => {
  on('ui.render', ($, e) => {
    const { Text } = $.ui.resolve(e)

    return <Text>engine</Text>
  })

  const ui = await $.ui.mount(message('Une réponse normale.'))

  expect((await ui.find({ type: 'Text', text: /engine/ }))?.text).toBe('engine')
  await ui.unmount()
})

const BOARD = {
  isStopping: false,
  waiting: [
    {
      file: 't14-a.md',
      id: 't14',
      title: 'Un ticket bloqué',
      note: 'question',
      action: { label: 'répondre', text: 'présente t14' },
    },
  ],
  running: [{ file: 't15-a.md', id: 't15', title: 'Un ouvrier', note: 'exécution' }],
  queued: [],
  ideas: [],
}

const PANE = {
  plugin: 'ork',
  component: 'Pane',
  requestId: 'ork',
  surface: 'desktop',
  props: {
    title: 'ork',
    isFocused: false,
    bodyColumns: 40,
    placement: 'dock',
    scroll: { offset: 0, bodyRows: 20 },
    view: {},
  },
} as const

test('the panel shows the board, and a row answers the lead', async ($, on) => {
  const submitted: string[] = []
  const state: Record<string, unknown> = {
    board: BOARD,
    focus: '.ork/tickets/t14-a.md',
    activity: {
      a1: { ticket: 't15', name: 'worker', description: 't15 exécution', text: 'Lit a.ts' },
      a2: { ticket: 't15', name: 'Explore', description: 'Trouver', text: 'Cherche dans le code' },
      a3: { ticket: null, name: 'Explore', description: 'Trier une remarque', text: 'Lit b.ts' },
    },
  }
  on('state.get', (_, e) => ({ value: { value: state[e.key], version: 1 } }))
  on('prompt.submit', (_, e) => {
    submitted.push(e.text)
    return { text: e.text }
  })

  const ui = await $.ui.mount(PANE)

  expect((await ui.find({ type: 'Text', text: /T'attend \(1\)/ }))?.text).toBe("T'attend (1)")
  expect((await ui.find({ type: 'Text', text: /En cours/ }))?.text).toBe('En cours (1)')
  expect(await ui.find({ type: 'Text', text: /Prêts/ })).toBeUndefined()
  expect(await ui.find({ key: 'focused-t14' })).toBeDefined()
  expect(await ui.find({ key: 'focused-t15' })).toBeUndefined()
  expect((await ui.find({ type: 'Text', text: /Lit a\.ts/ }))?.text).toBe('worker · Lit a.ts')
  expect((await ui.find({ type: 'Text', text: /Cherche/ }))?.text).toBe(
    'Explore · Cherche dans le code',
  )
  expect((await ui.find({ type: 'Text', text: /Autres agents/ }))?.text).toBe('Autres agents (1)')
  expect((await ui.find({ type: 'Text', text: /Lit b\.ts/ }))?.text).toBe(
    'Trier une remarque · Lit b.ts',
  )
  expect(await ui.find({ key: 'a1' })).toBeDefined()
  expect(await ui.find({ key: 'a2' })).toBeDefined()
  await ui.press({ key: 'présente t14' })
  expect(submitted).toEqual(['présente t14'])
  await ui.press({ key: 'stop' })
  expect(submitted).toEqual(['présente t14', 'arrête-toi au prochain état stable'])
  await ui.unmount()
})

test('the panel says when a clean stop is under way, and hides its button', async ($, on) => {
  on('state.get', (_, e) => ({
    value: {
      value: e.key === 'board' ? { ...BOARD, isStopping: true } : e.key === 'activity' ? {} : null,
      version: 1,
    },
  }))

  const ui = await $.ui.mount(PANE)

  expect(await ui.find({ type: 'Text', text: /Arrêt propre en cours/ })).toBeDefined()
  expect(await ui.find({ key: 'stop' })).toBeUndefined()
  await ui.unmount()
})

const orkEngine = (on: Parameters<TestBody>[1]) => {
  const opened: string[] = []
  let answer: (text: string) => void = () => {}
  const submitted = new Promise<string>(resolve => {
    answer = resolve
  })
  const state = new Map<string, unknown>()
  let version = 0
  on('command.register', () => ({ value: { command: 'ork' } }))
  on('session.start', (_, e) => ({ cwd: e.cwd }))
  on('fs.exists', () => ({ value: true }))
  on('fs.list', () => ({ value: [] }))
  on('state.get', (_, e) => ({ value: { value: state.get(e.key) ?? null, version } }))
  on('state.set', (_, e) => {
    state.set(e.key, e.value)
    version += 1
    return { value: { isSet: true, version } }
  })
  on('clock.after', () => ({ value: undefined }))
  on('ui.open', (_, e) => {
    opened.push(e.id)
    return { value: { isPlaced: true } }
  })
  on('prompt.submit', (_, e) => {
    answer(e.text)
    return { text: e.text }
  })

  return { opened, submitted }
}

test('a new session stays normal: the panel waits for /ork', async ($, on) => {
  const { opened } = orkEngine(on)

  await $.session.start({ cwd: '/', surface: 'desktop', isInteractive: true })
  expect(opened).toEqual([])
})

test('/ork opens the panel, then starts the lead once the command is over', async ($, on) => {
  const { opened, submitted } = orkEngine(on)

  await $.session.start({ cwd: '/', surface: 'desktop', isInteractive: true })
  await $.command.run({
    command: 'ork',
    args: '',
    origin: { kind: 'composer' },
    presentation: { isFullscreen: false, columns: 80 },
  })
  expect(opened).toEqual(['ork'])
  expect(await submitted).toBe('démarre ork')
})

test('/ork:start also opens the panel', async ($, on) => {
  const { opened } = orkEngine(on)

  await $.prompt.submit({
    text: '/ork:start',
    command: { name: 'ork:start', args: '' },
    wait: false,
    origin: { kind: 'composer' },
  })
  expect(opened).toEqual(['ork'])
})
