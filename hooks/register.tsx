import { atom, read, update } from 'claude-code'
import type { EngineInterface, Register } from 'claude-code'

import type { AgentActivity, BoardRow } from '../types'

import { agentName, describeToolCall, ticketOfDescription } from './activity'
import { parseMessage } from './parse'
import type { Action } from './parse'
import { buildBoard, isFocused, parseTicketHead } from './tickets'
import type { TicketHead } from './tickets'

const PANE = 'ork'
const CONFIG = '.ork/config.md'
const TICKETS_DIR = '.ork/tickets'
const STOP_FILE = `${TICKETS_DIR}/.stop`
const START_SKILL = 'ork:start'
const START_PROMPT = 'démarre ork'
const STOP_PROMPT = 'arrête-toi au prochain état stable'

const boardAtom = atom({ plugin: 'ork', key: 'board' } as const, null)
const focusAtom = atom({ plugin: 'ork', key: 'focus' } as const, null)
const activityAtom = atom({ plugin: 'ork', key: 'activity' } as const, {})
const answeredAtom = atom({ plugin: 'ork', key: 'answered' } as const, [])

const drawnWithActions = new Set<string>()
const agentMeta = new Map<string, Omit<AgentActivity, 'text'>>()

const PULSE = `<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8" viewBox="0 0 8 8"><circle cx="4" cy="4" r="3" fill="#8a8a85" opacity="0.3"><animate attributeName="opacity" values="0.2;0.6;0.2" dur="2.4s" repeatCount="indefinite"/></circle></svg>`

const TAG_COLORS: Record<string, string> = {
  WARN: '#b7791f',
  FAIL: '#d64545',
  STOP: '#d64545',
  OK: '#2f9e60',
  AUTO: '#2f9e60',
  NEW: '#3b82c4',
  NOTE: '#8a8a85',
  HINT: '#8a8a85',
  WT: '#8a8a85',
}

function actionLabel(action: Action): string {
  const effect = action.effect.replaceAll('`', '')
  const note = action.confidence === undefined ? '' : ` · ${action.confidence}/10`

  return `${action.word} : ${effect}${note}`
}

function keepLineBreaks(markdown: string): string {
  return markdown.replace(/([^\n])\n(?=[^\n])/g, '$1  \n')
}

async function refreshBoard($: EngineInterface): Promise<void> {
  const isOrkProject = (await $.fs.exists(CONFIG)) && (await $.fs.exists(TICKETS_DIR))

  if (!isOrkProject) {
    await update($, boardAtom, () => null)

    return
  }

  const entries = await $.fs.list(TICKETS_DIR)
  const heads = await Promise.all(
    entries
      .filter(entry => entry.kind === 'file' && entry.name.endsWith('.md'))
      .map(async entry =>
        parseTicketHead(entry.name, await $.fs.read(`${TICKETS_DIR}/${entry.name}`)),
      ),
  )
  const board = buildBoard(
    heads.filter((head): head is TicketHead => head !== undefined),
    await $.fs.exists(STOP_FILE),
  )

  await update($, boardAtom, () => board)
}

async function openPanel($: EngineInterface): Promise<boolean> {
  await refreshBoard($)

  if ((await read($, boardAtom)) === null) {
    return false
  }

  await $.ui.open({ id: PANE, title: 'ork' })

  return true
}

async function findAgent(
  $: EngineInterface,
  agentId: string,
): Promise<Omit<AgentActivity, 'text'> | undefined> {
  const known = agentMeta.get(agentId)

  if (known !== undefined) {
    return known
  }

  const agents = await $.agent.list()
  const agent = agents.find(one => one.id === agentId)

  if (agent === undefined) {
    return undefined
  }

  let root = agent

  while (root.parentId !== undefined) {
    const parentId = root.parentId
    const parent = agents.find(one => one.id === parentId)

    if (parent === undefined) {
      break
    }

    root = parent
  }

  const meta = {
    ticket: ticketOfDescription(root.description) ?? null,
    name: agentName(agent.type),
    description: agent.description,
  }
  agentMeta.set(agentId, meta)

  return meta
}

async function setActivity($: EngineInterface, agentId: string, text: string): Promise<void> {
  const meta = await findAgent($, agentId)

  if (meta !== undefined) {
    await update($, activityAtom, activity => ({ ...activity, [agentId]: { ...meta, text } }))
  }
}

async function clearActivity($: EngineInterface, agentId: string): Promise<void> {
  await update($, activityAtom, activity =>
    Object.fromEntries(Object.entries(activity).filter(([id]) => id !== agentId)),
  )
}

async function retireButtons($: EngineInterface): Promise<void> {
  const drawn = [...drawnWithActions]
  await update($, answeredAtom, () => drawn)
}

export const register: Register = on => {
  on('session.start', async ($, e, next) => {
    const started = await next(e)

    await $.command.register({
      name: 'ork',
      description: "Démarrer le chef d'ork dans cette session et ouvrir son panneau",
    })

    return started
  })

  on('command.run', { command: 'ork' }, async $ => {
    if (!(await openPanel($))) {
      return { text: "Ce dépôt n'a pas de .ork/config.md : lance /ork:init." }
    }

    // A command's hook cannot submit a prompt: it would wait on its own turn.
    $.clock.after(0, () => {
      void $.prompt.submit({ text: START_PROMPT, asUser: true })
    })

    return { text: "Panneau d'ork ouvert." }
  })

  on('agent.spawn', async ($, e, next) => {
    const spawned = await next(e)

    if (spawned.agentId !== undefined) {
      await setActivity($, spawned.agentId, 'Démarre')
    }

    return spawned
  }).catch(($, e, next) => next(e))

  on('tool.call', async ($, e, next) => {
    if (e.agentId !== undefined) {
      await setActivity($, e.agentId, describeToolCall(e.tool, e))
    }

    return next(e)
  }).catch(($, e, next) => next(e))

  on('turn.complete', async ($, e, next) => {
    if (e.agentId !== undefined) {
      await clearActivity($, e.agentId)
    }

    await refreshBoard($)

    const subject = e.agentId === undefined ? parseMessage(e.answer ?? '')?.subject : undefined

    if (subject) {
      await update($, focusAtom, () => subject.href)
    }

    return next(e)
  })

  on('tool.call', { tool: 'Write' }, async ($, e, next) => {
    const ran = await next(e)

    if (e.file_path.includes(TICKETS_DIR)) {
      await refreshBoard($)
    }

    return ran
  }).catch(($, e, next) => next(e))

  on('tool.call', { tool: 'Edit' }, async ($, e, next) => {
    const ran = await next(e)

    if (e.file_path.includes(TICKETS_DIR)) {
      await refreshBoard($)
    }

    return ran
  }).catch(($, e, next) => next(e))

  on('prompt.submit', async ($, e, next) => {
    await retireButtons($)

    if (e.command?.name === START_SKILL) {
      await openPanel($)
    }

    return next(e)
  }).catch(($, e, next) => next(e))

  on('ui.render', { component: 'Pane', requestId: PANE }, async ($, e, next) => {
    if (e.surface !== 'desktop') {
      return next(e)
    }

    const { Box, Button, Markdown, Svg, Text } = $.ui.resolve(e)
    const board = await read($, boardAtom)
    const focus = await read($, focusAtom)
    const activity = await read($, activityAtom)

    if (board === null) {
      return <Text dimColor>Pas de projet ork ici.</Text>
    }

    const agents = Object.entries(activity).map(([key, agent]) => ({ key, ...agent }))
    const others = agents.filter(
      agent =>
        agent.ticket === null ||
        ![...board.waiting, ...board.running, ...board.queued, ...board.ideas].some(
          row => row.id === agent.ticket,
        ),
    )
    const loader = (key: string, text: string) => (
      <Box key={key} flexDirection="row" columnGap={1} alignItems="center" marginTop={1}>
        <Svg source={PULSE} alt="En cours" width={8} height={8} isInteractive />
        <Text dimColor wrap="truncate-end">
          {text}
        </Text>
      </Box>
    )
    const section = (title: string, rows: readonly BoardRow[]) =>
      rows.length > 0 && (
        <Box key={title} flexDirection="column" rowGap={1}>
          <Text bold>{`${title} (${rows.length})`}</Text>
          {rows.map(row => {
            const focused = isFocused(row, TICKETS_DIR, focus)

            return (
              <Box key={focused ? `focused-${row.id}` : row.id} flexDirection="column">
                <Box flexDirection="row" columnGap={1} alignItems="center">
                  <Markdown text={`[${row.id}](${TICKETS_DIR}/${row.file})`} />
                  {focused && <Text color="suggestion">●</Text>}
                  <Box flexGrow={1} />
                  {row.note !== undefined && <Text dimColor>{row.note}</Text>}
                  {row.action && (
                    <Button
                      key={row.action.text}
                      label={row.action.label}
                      onPress={() => {
                        const text = row.action?.text

                        if (text !== undefined) {
                          void $.prompt.submit({ text, asUser: true })
                        }
                      }}
                    />
                  )}
                </Box>
                <Text dimColor={!focused}>{row.title}</Text>
                {agents
                  .filter(agent => agent.ticket === row.id)
                  .map(agent => loader(agent.key, `${agent.name} · ${agent.text}`))}
              </Box>
            )
          })}
        </Box>
      )
    const isEmpty =
      board.waiting.length +
        board.running.length +
        board.queued.length +
        board.ideas.length +
        others.length ===
      0

    return (
      <Box flexDirection="column" rowGap={2}>
        {board.isStopping && <Text color="warning">Arrêt propre en cours : rien de nouveau ne part.</Text>}
        {isEmpty && <Text dimColor>Aucun ticket ouvert.</Text>}
        {section("T'attend", board.waiting)}
        {section('En cours', board.running)}
        {others.length > 0 && (
          <Box key="others" flexDirection="column">
            <Text bold>{`Autres agents (${others.length})`}</Text>
            {others.map(agent => loader(agent.key, `${agent.description} · ${agent.text}`))}
          </Box>
        )}
        {section('Prêts', board.queued)}
        {section('Idées', board.ideas)}
        {!board.isStopping && agents.length + board.running.length > 0 && (
          <Button
            key="stop"
            label="Arrêt propre"
            onPress={() => {
              void $.prompt.submit({ text: STOP_PROMPT, asUser: true })
            }}
          />
        )}
      </Box>
    )
  })

  on('ui.render', { component: 'AssistantMessage' }, async ($, e, next) => {
    if (e.surface !== 'desktop') {
      return next(e)
    }

    const parsed = parseMessage(e.props.text)

    if (!parsed) {
      return next(e)
    }

    const { Box, Button, Markdown, Text } = $.ui.resolve(e)
    const answered = await read($, answeredAtom)
    const showsActions = parsed.actions.length > 0 && !answered.includes(e.requestId)

    if (showsActions) {
      drawnWithActions.add(e.requestId)
    }

    return (
      <Box flexDirection="column" rowGap={1}>
        {parsed.statusLines.length > 0 && (
          <Box flexDirection="column">
            {parsed.statusLines.map((line, index) => (
              <Box key={`status-${index}`} flexDirection="row" columnGap={1} alignItems="flex-start">
                <Text mono bold color={TAG_COLORS[line.tag]}>
                  {line.tag}
                </Text>
                <Markdown text={line.text} />
              </Box>
            ))}
          </Box>
        )}
        {parsed.intro !== '' && <Markdown text={keepLineBreaks(parsed.intro)} />}
        {parsed.subject && (
          <Box flexDirection="column" rowGap={1} padding={1} borderStyle="round" borderDimColor>
            <Box flexDirection="row" columnGap={1}>
              <Text>{parsed.subject.icon}</Text>
              <Markdown text={`[${parsed.subject.title}](${parsed.subject.href})`} />
              <Text dimColor>{parsed.subject.step}</Text>
            </Box>
            {parsed.body !== '' && <Markdown text={keepLineBreaks(parsed.body)} />}
            {showsActions && (
              <Box flexDirection="row" flexWrap="wrap" columnGap={1} rowGap={1}>
                {parsed.actions.map(action => (
                  <Button
                    key={action.word}
                    label={actionLabel(action)}
                    variant={action.isRecommended ? 'primary' : 'secondary'}
                    onPress={() => {
                      void retireButtons($).then(() =>
                        $.prompt.submit({ text: action.word, asUser: true }),
                      )
                    }}
                  />
                ))}
              </Box>
            )}
          </Box>
        )}
        {!parsed.subject && parsed.body !== '' && <Markdown text={parsed.body} />}
      </Box>
    )
  })
}
