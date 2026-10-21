/**
 * 歌词解析：支持 LRC 时间轴，也兼容纯文本歌词。
 */

export interface LyricLine {
  /** 开始时间（秒），-1 表示无时间轴 */
  time: number
  text: string
}

const TIME_RE = /\[(\d{1,2}):(\d{1,2})(?:[.:](\d{1,3}))?\]/g

/**
 * 解析 LRC，返回按时间排序的歌词行
 */
export function parseLrc(raw: string): LyricLine[] {
  if (!raw) return []

  const lines: LyricLine[] = []

  raw.split('\n').forEach((row) => {
    const tags = Array.from(row.matchAll(TIME_RE))
    const text = row.replace(TIME_RE, '').trim()
    if (!tags.length || !text) return

    tags.forEach((tag) => {
      const minutes = Number(tag[1] || 0)
      const seconds = Number(tag[2] || 0)
      const millis = Number((tag[3] || '0').padEnd(3, '0'))
      lines.push({ time: minutes * 60 + seconds + millis / 1000, text })
    })
  })

  return lines.sort((a, b) => a.time - b.time)
}

/**
 * 优先使用带时间轴的歌词，否则按纯文本逐行展示
 */
export function toLyricLines(lrc?: string, plain?: string): LyricLine[] {
  const synced = parseLrc(lrc || '')
  if (synced.length) return synced

  return (plain || '')
    .split('\n')
    .map((text) => text.trim())
    .filter(Boolean)
    .map((text) => ({ time: -1, text }))
}

/**
 * 根据播放进度计算当前行下标，无时间轴返回 -1
 */
export function findActiveLine(lines: LyricLine[], currentTime: number): number {
  if (!lines.some((line) => line.time >= 0)) return -1

  let active = -1
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].time <= currentTime + 0.15) {
      active = i
    } else {
      break
    }
  }
  return active
}
