/** 音乐源展示名 */
export const SOURCE_LABELS: Record<string, string> = {
  qqmusic: 'QQ音乐',
  wyy: '网易云音乐',
  kugou: '酷狗音乐',
  qishui: '汽水音乐'
}

/** 音乐源标识色（同属主色系，仅以深浅区分，避免与状态色混淆） */
export const SOURCE_COLORS: Record<string, string> = {
  qqmusic: 'var(--el-color-primary)',
  wyy: 'var(--el-color-primary-dark-2)',
  kugou: 'var(--el-color-primary-light-3)',
  qishui: 'var(--el-color-primary-light-5)'
}

/** 取音乐源名称 */
export function sourceName(key?: string): string {
  if (!key) return '-'
  return SOURCE_LABELS[key] || key
}

/** 格式化时间 */
export function formatTime(value?: string | null): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return String(value)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(
    date.getHours()
  )}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

/** 截断字符串 */
export function truncate(value: string, max = 60): string {
  if (!value) return ''
  return value.length > max ? `${value.slice(0, max)}...` : value
}

/** 复制到剪贴板 */
export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    try {
      const el = document.createElement('textarea')
      el.value = text
      el.style.position = 'fixed'
      el.style.opacity = '0'
      document.body.appendChild(el)
      el.select()
      document.execCommand('copy')
      document.body.removeChild(el)
      return true
    } catch {
      return false
    }
  }
}
