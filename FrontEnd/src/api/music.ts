/**
 * AR-musicAPI 业务接口
 *
 * 所有接口均由 Go 后端提供，统一返回 { code, msg, data } 结构，
 * request 封装会自动取出 data 字段。
 *
 * @module api/music
 */
import axios from 'axios'
import request from '@/utils/http'
import { useUserStore } from '@/store/modules/user'

/** 音乐源 */
export interface MusicSource {
  key: string
  name: string
  need_cookie: boolean
}

/** 仪表盘统计 */
export interface DashboardStats {
  total_calls: number
  today_calls: number
  success_rate: number
  avg_duration_ms: number
  total_cookies: number
  active_cookies: number
  cookies: Record<string, number>
  default_source: string
  site_name: string
  sources: MusicSource[]
  uptime: string
}

/** 趋势点 */
export interface TrendPoint {
  date: string
  full_date: string
  count: number
  success: number
  failed: number
}

/** Cookie 池记录 */
export interface CookieItem {
  id: number
  source: string
  cookie: string
  remark: string
  status: number
  fail_count: number
  call_count: number
  last_used: string | null
  created_at: string
  updated_at: string
}

/** 调用日志 */
export interface LogItem {
  id: number
  source: string
  song_name: string
  /** 歌手 */
  singer: string
  /** 歌曲 ID（音源内） */
  song_id: string
  album_img: string
  /** 播放直链，空表示不可播放（历史日志或未取到） */
  music_url: string
  /** 带时间轴的 LRC 歌词 */
  lyric_lrc: string
  request_params: string
  response_code: number
  success: boolean
  error_msg: string
  duration_ms: number
  ip: string
  user_agent: string
  created_at: string
}

/** 分页结果（与模板 useTable 的分页结构保持一致） */
export type PageResult<T> = Api.Common.PaginatedResponse<T>

/** 系统设置 */
export interface SettingsPayload {
  default_source: string
  site_name: string
  site_icon: string
  log_retention_days: string
  auto_switch_cookie: string
  cookie_fail_threshold: string
  allow_public_api: string
  sources: MusicSource[]
  account?: {
    id: number
    username: string
    nickname: string
    roles: string[]
    last_login_at: string
    last_login_ip: string
  }
}

/** 点歌结果（扁平结构，无 data 包装） */
export interface MusicResult {
  code: number
  msg: string
  source: string
  song_id?: string
  title: string
  singer: string
  cover: string
  link: string
  music_url: string
  lyric: string
  /** 带时间轴的原始歌词 */
  lyric_lrc?: string
  /** 本次请求耗时（毫秒） */
  duration_ms: number
  /** 歌曲时长（毫秒） */
  song_duration_ms?: number
}

/** 仪表盘汇总 */
export function fetchDashboardStats() {
  return request.get<DashboardStats>({ url: '/api/dashboard/stats' })
}

/** 调用趋势 */
export function fetchTrend(days = 7) {
  return request.get<TrendPoint[]>({ url: '/api/dashboard/trend', params: { days } })
}

/** 最近调用日志 */
export function fetchRecentLogs(limit = 8) {
  return request.get<LogItem[]>({ url: '/api/dashboard/recent-logs', params: { limit } })
}

/** Cookie 列表 */
export function fetchCookieList(params: {
  current?: number
  size?: number
  source?: string
  status?: string
  keyword?: string
}) {
  return request.get<PageResult<CookieItem>>({ url: '/api/cookie/list', params })
}

/** 新增 Cookie（支持批量） */
export function addCookies(payload: {
  source: string
  remark?: string
  cookie?: string
  cookies?: string[]
}) {
  return request.post<{ count: number }>({
    url: '/api/cookie/add',
    params: payload,
    showSuccessMessage: true
  })
}

/** 修改 Cookie */
export function updateCookie(payload: {
  id: number
  source: string
  cookie: string
  remark: string
  status: number
}) {
  return request.put({ url: '/api/cookie/update', params: payload, showSuccessMessage: true })
}

/** 删除 Cookie */
export function deleteCookies(ids: number[]) {
  return request.del<{ count: number }>({
    url: '/api/cookie/delete',
    params: { ids: ids.join(',') },
    showSuccessMessage: true
  })
}

/** 批量启用/禁用 */
export function toggleCookies(ids: number[]) {
  return request.post<{ count: number }>({
    url: '/api/cookie/toggle',
    params: { ids },
    showSuccessMessage: true
  })
}

/** 批量重置失败计数 */
export function resetCookies(ids: number[]) {
  return request.post<{ count: number }>({
    url: '/api/cookie/reset',
    params: { ids },
    showSuccessMessage: true
  })
}

/** 日志列表 */
export function fetchLogList(params: {
  current?: number
  size?: number
  source?: string
  success?: string
  keyword?: string
}) {
  return request.get<PageResult<LogItem>>({ url: '/api/log/list', params })
}

/** 清理日志，days=0 表示全部清空 */
export function clearLogs(days = 0) {
  return request.del<{ count: number }>({
    url: '/api/log/clear',
    params: { days },
    showSuccessMessage: true
  })
}

/** 读取系统设置 */
export function fetchSettings() {
  return request.get<SettingsPayload>({ url: '/api/settings' })
}

/** 保存系统设置 */
export function updateSettings(payload: Record<string, any>) {
  return request.put({ url: '/api/settings', params: payload, showSuccessMessage: true })
}

/** 修改密码 */
export function changePassword(payload: { oldPassword: string; newPassword: string }) {
  return request.put({ url: '/api/settings/password', params: payload, showSuccessMessage: true })
}

/**
 * 点歌测试
 *
 * 该接口返回的是扁平 JSON（兼容旧版 PHP 接口），没有 data 包装，
 * 因此不走 request 封装，直接用 axios。
 */
export async function searchMusic(params: {
  name?: string
  /** 歌手名，用于精确定位原唱 */
  singer?: string
  id?: string
  source?: string
}): Promise<MusicResult> {
  const { accessToken } = useUserStore()
  const res = await axios.get<MusicResult>('/api/music/search', {
    params,
    headers: accessToken ? { Authorization: accessToken } : {}
  })
  return res.data
}

/** 站点公开信息 */
export interface SiteInfo {
  site_name: string
  site_icon: string
}

/**
 * 读取站点公开信息（无需登录）
 */
export async function fetchSiteInfo(): Promise<SiteInfo> {
  const res = await axios.get<{ code: number; data: SiteInfo }>('/api/site/info')
  return res.data.data
}

/**
 * 上传站点图标
 */
export async function uploadSiteIcon(file: File): Promise<{ url: string }> {
  const { accessToken } = useUserStore()
  const form = new FormData()
  form.append('file', file)

  const res = await axios.post<{ code: number; data: { url: string } }>(
    '/api/settings/icon',
    form,
    { headers: accessToken ? { Authorization: accessToken } : {} }
  )
  return res.data.data
}
