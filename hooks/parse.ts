export type StatusLine = {
  tag: string
  text: string
}

export type Subject = {
  icon: string
  title: string
  href: string
  step: string
}

export type Action = {
  word: string
  effect: string
  isRecommended: boolean
  confidence?: number
}

export type Parsed = {
  statusLines: StatusLine[]
  intro: string
  subject?: Subject
  body: string
  actions: Action[]
}

const TAGS = 'WARN|FAIL|STOP|OK|AUTO|NEW|NOTE|HINT|WT'
const STATUS_LINE = new RegExp(`^\\[(${TAGS})\\](?:\\s+(.*))?$`)
const SUBJECT = /^###\s+(\S+)\s+\[([^\]]+)\]\(([^)]+)\)\s+·\s+(.+)$/
const ACTION = /^`([^`]+)`\s+:\s+(.+)$/
const RECOMMENDED = /\s*\*\(recommandé(?:\s+·\s+(\d+)\/10)?\)\*\s*$/

function parseAction(line: string): Action | undefined {
  const [, word, effect] = ACTION.exec(line) ?? []

  if (word === undefined || effect === undefined) {
    return undefined
  }

  const recommended = RECOMMENDED.exec(effect)
  const confidence = recommended?.[1]

  return {
    word,
    effect: effect.replace(RECOMMENDED, ''),
    isRecommended: recommended !== null,
    ...(confidence === undefined ? {} : { confidence: Number(confidence) }),
  }
}

function parseSubject(line: string): Subject | undefined {
  const [, icon, title, href, step] = SUBJECT.exec(line) ?? []

  if (icon === undefined || title === undefined || href === undefined || step === undefined) {
    return undefined
  }

  return { icon, title, href, step }
}

/**
 * Reads a lead message. Answers undefined when it is not in the lead's format,
 * so the caller leaves the text as it is. While a message is still being
 * written, its last line may be incomplete: it stays in the body until it ends.
 */
export function parseMessage(text: string): Parsed | undefined {
  const lines = text.trim().split('\n').map(line => line.trim())
  const statusLines: StatusLine[] = []
  let index = 0

  for (const line of lines) {
    const [, tag, rest] = STATUS_LINE.exec(line) ?? []

    if (tag === undefined) {
      break
    }

    statusLines.push({ tag, text: rest ?? '' })
    index += 1
  }

  const subjectIndex = lines.findIndex((line, position) => position >= index && parseSubject(line))
  const subject = subjectIndex === -1 ? undefined : parseSubject(lines[subjectIndex] ?? '')

  if (statusLines.length === 0 && !subject) {
    return undefined
  }

  if (!subject) {
    return { statusLines, intro: '', body: lines.slice(index).join('\n').trim(), actions: [] }
  }

  const intro = lines.slice(index, subjectIndex).join('\n').trim()
  const rest = lines.slice(subjectIndex + 1)
  const actions: Action[] = []
  let end = rest.length

  while (end > 0) {
    const line = rest[end - 1] ?? ''

    if (line === '' && actions.length === 0) {
      end -= 1
      continue
    }

    const action = parseAction(line)

    if (!action) {
      break
    }

    actions.unshift(action)
    end -= 1
  }

  return {
    statusLines,
    intro,
    subject,
    body: rest.slice(0, end).join('\n').trim(),
    actions,
  }
}
