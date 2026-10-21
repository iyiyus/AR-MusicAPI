/**
 * 快速入口配置
 * 包含：应用列表、快速链接等配置
 */
import type { FastEnterConfig } from '@/types/config'

const fastEnterConfig: FastEnterConfig = {
  // 显示条件（屏幕宽度）
  minWidth: 1200,
  // 应用列表：仅保留音乐 API 相关页面
  applications: [
    {
      name: '运行概览',
      description: '调用统计与音源状态',
      icon: 'ri:dashboard-line',
      iconColor: '#377dff',
      enabled: true,
      order: 1,
      routeName: 'MusicDashboard'
    },
    {
      name: 'Cookie 池',
      description: '账号凭证与启停管理',
      icon: 'ri:coupon-3-line',
      iconColor: '#13DEB9',
      enabled: true,
      order: 2,
      routeName: 'MusicCookie'
    },
    {
      name: '调用日志',
      description: '接口调用明细查询',
      icon: 'ri:file-list-3-line',
      iconColor: '#ffb100',
      enabled: true,
      order: 3,
      routeName: 'MusicLog'
    },
    {
      name: '系统设置',
      description: '默认音源与日志策略',
      icon: 'ri:settings-3-line',
      iconColor: '#7A7FFF',
      enabled: true,
      order: 4,
      routeName: 'MusicSettings'
    }
  ],
  // 快速链接：无额外入口
  quickLinks: []
}

export default Object.freeze(fastEnterConfig)
