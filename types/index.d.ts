export type BoardRow = {
  file: string
  id: string
  title: string
  note?: string
  action?: { label: string; text: string }
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
      focus: string | null
      activity: Readonly<Record<string, AgentActivity>>
      answered: readonly string[]
    }
  }
}
