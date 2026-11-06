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

test('the panel puts an alert on its ticket, and git at the end', async ($, on) => {
  const submitted: string[] = []
  on('prompt.submit', (_, e) => {
    submitted.push(e.text)
    return { text: e.text }
  })
  on('state.get', (_, e) => ({
    value: {
      value:
        e.key === 'board'
          ? BOARD
          : e.key === 'activity'
            ? {}
            : e.key === 'health'
              ? {
                  alerts: {
                    t15: {
                      text: 'ouvrier arrêté',
                      action: { label: 'reprendre', text: 'reprends t15' },
                    },
                  },
                  git: {
                    trunk: { name: 'dev', detail: 'à jour avec origin' },
                    rows: [
                      {
                        key: 'worktree-x',
                        name: 'x',
                        detail: '1 en avance',
                        action: { label: 'intégrer', text: 'intègre x' },
                      },
                    ],
                    hiddenBranches: 2,
                  },
                }
              : null,
      version: 1,
    },
  }))

  const ui = await $.ui.mount(PANE)

  expect((await ui.find({ type: 'Text', text: /ouvrier arrêté/ }))?.text).toBe('ouvrier arrêté')
  expect((await ui.find({ type: 'Text', text: /^Git$/ }))?.text).toBe('Git')
  expect((await ui.find({ type: 'Text', text: /● dev/ }))?.text).toBe('● dev')
  expect((await ui.find({ type: 'Text', text: /└ x/ }))?.text).toBe('└ x')
  expect((await ui.find({ type: 'Text', text: /autres branches/ }))?.text).toBe('+ 2 autres branches')
  await ui.press({ key: 'reprends t15' })
  await ui.press({ key: 'intègre x' })
  expect(submitted).toEqual(['reprends t15', 'intègre x'])
  await ui.unmount()
})

test('the panel shows the paused tickets and their reason', async ($, on) => {
  const submitted: string[] = []
  on('prompt.submit', (_, e) => {
    submitted.push(e.text)
    return { text: e.text }
  })
  on('state.get', (_, e) => ({
    value: {
      value:
        e.key === 'board'
          ? {
              ...BOARD,
              paused: [
                {
                  file: 't30-a.md',
                  id: 't30',
                  title: 'Importer',
                  reason: "jusqu'à t34",
                  action: { label: 'reprendre', text: 'sors t30 de la pause' },
                },
              ],
            }
          : e.key === 'activity'
            ? {}
            : null,
      version: 1,
    },
  }))

  const ui = await $.ui.mount(PANE)

  expect((await ui.find({ type: 'Text', text: /En pause/ }))?.text).toBe('En pause (1)')
  expect((await ui.find({ type: 'Text', text: /jusqu'à t34/ }))?.text).toBe("jusqu'à t34")
  await ui.press({ key: 'sors t30 de la pause' })
  expect(submitted).toEqual(['sors t30 de la pause'])
  await ui.unmount()
})

test('the status skill gets the state computed, typed or loaded by the tool', async ($, on) => {
  const TICKET = '---\nid: t1\ntitle: T\nstatus: blocked\n---\n\n## Questions\nla question ?\n'
  on('skill.prompt', () => ({ text: 'Le skill.' }))
  on('fs.exists', (_, e) => ({ value: !e.path.endsWith('.stop') }))
  on('fs.list', (_, e) => ({
    value: e.path.endsWith('.ork/tickets') ? [{ name: 't1-a.md', kind: 'file' }] : [],
  }))
  on('fs.read', (_, e) => ({
    value: e.path.endsWith('.ork/config.md') ? '---\nintegration_branch: dev\n---\n' : TICKET,
  }))
  on('agent.list', () => ({ value: [] }))
  on('clock.now', () => ({ value: 1_000_000_000 }))
  on('process.run', () => ({
    value: { exitCode: 0, stdout: 'worktree /repo\nbranch refs/heads/dev\n\n', stderr: '' },
  }))
  on('state.get', () => ({ value: { value: null, version: 1 } }))
  on('state.set', () => ({ value: { isSet: true, version: 2 } }))

  const { text } = await $.skill.prompt({ skill: 'ork:status', text: 'Le skill.', frontmatter: {} })

  expect(text).toContain('Le skill.\n\nork_status: ok')
  expect(text).toContain('integration_branch: dev')
  expect(text).toContain('first_subject: t1-a (blocked)')
  expect(text).toContain('la question ?')
})

test('the status skill says when the project has no config', async ($, on) => {
  on('skill.prompt', () => ({ text: 'Le skill.' }))
  on('fs.exists', () => ({ value: false }))
  on('state.set', () => ({ value: { isSet: true, version: 2 } }))
  on('state.get', () => ({ value: { value: null, version: 1 } }))

  const { text } = await $.skill.prompt({ skill: 'ork:status', text: 'Le skill.', frontmatter: {} })

  expect(text).toBe('Le skill.\n\nork_status: no_config')
})

test('other skills are left alone', async ($, on) => {
  on('skill.prompt', () => ({ text: 'Autre.' }))

  const { text } = await $.skill.prompt({ skill: 'ork:ticket', text: 'Autre.', frontmatter: {} })

  expect(text).toBe('Autre.')
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
  on('clock.every', () => ({ value: undefined }))
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
