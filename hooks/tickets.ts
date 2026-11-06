import type { Board, BoardRow } from '../types'

export type TicketHead = {
  file: string
  id: string
  title: string
  status: string
  priority: string
  dependsOn: readonly string[]
}

const WAITING_ORDER = ['blocked', 'plan-review', 'in-review']
const PRIORITY_ORDER = ['high', 'normal', 'low']

const WAITING_ACTIONS: Record<string, { label: string; prefix: string; note: string }> = {
  blocked: { label: 'répondre', prefix: 'présente', note: 'question' },
  'plan-review': { label: 'valider', prefix: 'présente', note: 'plan' },
  'in-review': { label: 'relire', prefix: 'relisons', note: 'à relire' },
}

function parseList(value: string | undefined): string[] {
  return (value ?? '')
    .replace(/^\[|\]$/g, '')
    .split(',')
    .map(item => item.trim())
    .filter(item => item !== '')
}

/** Reads a ticket's frontmatter. `file` is the file's name in the tickets folder. */
export function parseTicketHead(file: string, text: string): TicketHead | undefined {
  const lines = text.split('\n')

  if (lines[0]?.trim() !== '---') {
    return undefined
  }

  const fields = new Map<string, string>()

  for (const line of lines.slice(1)) {
    if (line.trim() === '---') {
      break
    }

    const separator = line.indexOf(':')

    if (separator > 0) {
      fields.set(line.slice(0, separator).trim(), line.slice(separator + 1).trim())
    }
  }

  const id = fields.get('id')
  const status = fields.get('status')

  if (!id || !status) {
    return undefined
  }

  return {
    file,
    id,
    status,
    title: fields.get('title') ?? '',
    priority: fields.get('priority') ?? 'normal',
    dependsOn: parseList(fields.get('depends_on')),
  }
}

function rank(order: readonly string[], value: string): number {
  const index = order.indexOf(value)

  return index === -1 ? order.length : index
}

function ticketNumber(id: string): number {
  return Number(id.replace(/\D/g, '')) || 0
}

function byUrgency(a: TicketHead, b: TicketHead): number {
  return (
    rank(WAITING_ORDER, a.status) - rank(WAITING_ORDER, b.status) ||
    rank(PRIORITY_ORDER, a.priority) - rank(PRIORITY_ORDER, b.priority) ||
    ticketNumber(a.id) - ticketNumber(b.id)
  )
}

function row(ticket: TicketHead): BoardRow {
  return { file: ticket.file, id: ticket.id, title: ticket.title }
}

function normalizePath(path: string): string {
  return path.replace(/^\.\//, '')
}

/** Whether a row is the ticket of the current subject. A story's subject focuses no row. */
export function isFocused(row: BoardRow, ticketsDir: string, focus: string | null): boolean {
  if (focus === null) {
    return false
  }

  return normalizePath(focus) === `${ticketsDir}/${row.file}`
}

/** Sorts the open tickets into what the panel shows. */
export function buildBoard(tickets: readonly TicketHead[], isStopping = false): Board {
  const sorted = [...tickets].sort(byUrgency)
  const open = new Set(tickets.map(ticket => ticket.id))
  const withStatus = (...statuses: string[]) =>
    sorted.filter(ticket => statuses.includes(ticket.status))

  return {
    isStopping,
    waiting: withStatus(...WAITING_ORDER).map(ticket => {
      const action = WAITING_ACTIONS[ticket.status]

      return {
        ...row(ticket),
        ...(action && {
          note: action.note,
          action: { label: action.label, text: `${action.prefix} ${ticket.id}` },
        }),
      }
    }),
    running: withStatus('planning', 'in-progress').map(ticket => ({
      ...row(ticket),
      note: ticket.status === 'planning' ? 'plan' : 'exécution',
    })),
    queued: withStatus('ready').map(ticket => {
      const blockers = ticket.dependsOn.filter(id => open.has(id))

      return blockers.length > 0
        ? { ...row(ticket), note: `attend ${blockers.join(', ')}` }
        : { ...row(ticket), action: { label: 'lancer', text: `lance ${ticket.id}` } }
    }),
    ideas: withStatus('idea').map(ticket => ({
      ...row(ticket),
      action: { label: 'clarifier', text: `clarifions ${ticket.id}` },
    })),
  }
}
