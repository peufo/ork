export type BoardRow = {
  file: string
  id: string
  title: string
  note?: string
  reason?: string
  isPaused?: boolean
  isBlocked?: boolean
  action?: { label: string; text: string }
}

/** What the panel adds to a ticket's row: it replaces the row's own button when it has one. */
export type TicketAlert = {
  text: string
  action?: { label: string; text: string }
}

export type GitRow = {
  key: string
  name: string
  detail: string
  action?: { label: string; text: string }
}

/** The integration branch, and what hangs from it: history is linear, so no real graph. */
export type GitView = {
  trunk: { name: string; detail: string }
  rows: readonly GitRow[]
  hiddenBranches: number
}

export type Health = {
  alerts: Readonly<Record<string, TicketAlert>>
  git: GitView | null
}

export type AgentActivity = {
  ticket: string | null
  name: string
  description: string
  text: string
}

export type Board = {
  isStopping: boolean
  waiting: readonly BoardRow[]
  running: readonly BoardRow[]
  queued: readonly BoardRow[]
  ideas: readonly BoardRow[]
}

declare module 'claude-code' {
  interface PluginState {
    ork: {
      board: Board | null
      /** Null until the lead has started: the git checks wait for it. */
      health: Health | null
      focus: string | null
      activity: Readonly<Record<string, AgentActivity>>
      answered: readonly string[]
    }
  }
}
