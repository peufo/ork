function field(input: unknown, key: string): string | undefined {
  if (typeof input !== 'object' || input === null) {
    return undefined
  }

  const value = (input as Record<string, unknown>)[key]

  return typeof value === 'string' && value !== '' ? value : undefined
}

function fileName(path: string | undefined): string {
  return path?.split('/').at(-1) ?? 'un fichier'
}

/** A few words on what a tool call does, for the panel. */
export function describeToolCall(tool: string, input: unknown): string {
  switch (tool) {
    case 'Bash':
      return field(input, 'description') ?? 'Lance une commande'
    case 'Read':
      return `Lit ${fileName(field(input, 'file_path'))}`
    case 'Edit':
    case 'Write':
    case 'NotebookEdit':
      return `Modifie ${fileName(field(input, 'file_path'))}`
    case 'Grep':
    case 'Glob':
      return 'Cherche dans le code'
    case 'Skill':
      return `Lance ${field(input, 'skill') ?? 'un skill'}`
    case 'Agent':
    case 'Task':
      return field(input, 'description') ?? 'Délègue une recherche'
    case 'WebFetch':
    case 'WebSearch':
      return 'Cherche sur le web'
    default:
      return tool.startsWith('mcp__') ? 'Utilise un outil externe' : `Utilise ${tool}`
  }
}

/** The short name of an agent type: `ork:worker` becomes `worker`. */
export function agentName(type: string): string {
  return type.split(':').at(-1) ?? type
}

/** The ticket an agent works on: the lead starts its description with the ticket's id. */
export function ticketOfDescription(description: string): string | undefined {
  return /^(t\d+)\b/i.exec(description)?.[1]?.toLowerCase()
}
