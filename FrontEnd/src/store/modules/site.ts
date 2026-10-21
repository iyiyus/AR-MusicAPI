/**
 * 站点信息（名称、图标）
 *
 * 图标由后台「系统设置」上传后保存到 settings 表，
 * 通过公开接口 /api/site/info 提供给登录页与前台页面使用。
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchSiteInfo } from '@/api/music'

// 记录 index.html 中声明的默认图标，便于恢复
const defaultFaviconHref =
  document.querySelector<HTMLLinkElement>('link[rel="icon"]')?.getAttribute('href') || ''

/**
 * 应用浏览器标签页图标
 */
function applyFavicon(url: string) {
  let link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }
  link.href = url || defaultFaviconHref
}

export const useSiteStore = defineStore('siteStore', () => {
  /** 站点名称（后台可配置，未配置时使用本地默认值） */
  const siteName = ref('')
  /** 站点图标（空表示使用内置 logo） */
  const siteIcon = ref('')

  let loading = false
  let loaded = false

  /**
   * 更新站点信息
   */
  const setInfo = (info: { site_name?: string; site_icon?: string }) => {
    if (info.site_name !== undefined) {
      siteName.value = info.site_name
    }
    if (info.site_icon !== undefined) {
      siteIcon.value = info.site_icon
      applyFavicon(info.site_icon)
    }
  }

  /**
   * 拉取站点信息（失败时保持默认图标，不影响页面）
   */
  const load = async () => {
    if (loaded || loading) return
    loading = true
    try {
      const info = await fetchSiteInfo()
      if (info) {
        setInfo(info)
      }
      loaded = true
    } catch {
      // 忽略：使用内置图标
    } finally {
      loading = false
    }
  }

  return { siteName, siteIcon, setInfo, load }
})
