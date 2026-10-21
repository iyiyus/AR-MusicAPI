import { AppRouteRecord } from '@/types/router'

/**
 * 音乐 API 管理模块
 *
 * 四个页面均为顶级单页路由，模板会自动用 Layout 包裹并作为一级菜单展示。
 * 放在模块列表首位，因此第一个页面会成为默认首页。
 */
export const musicRoutes: AppRouteRecord[] = [
  {
    path: '/dashboard',
    name: 'MusicDashboard',
    component: '/music/dashboard',
    meta: {
      title: 'menus.music.dashboard',
      icon: 'ri:dashboard-line',
      keepAlive: true,
      fixedTab: true,
      roles: ['R_SUPER', 'R_ADMIN']
    }
  },
  {
    path: '/cookie',
    name: 'MusicCookie',
    component: '/music/cookie',
    meta: {
      title: 'menus.music.cookie',
      icon: 'ri:coupon-3-line',
      keepAlive: true,
      roles: ['R_SUPER', 'R_ADMIN']
    }
  },
  {
    path: '/log',
    name: 'MusicLog',
    component: '/music/log',
    meta: {
      title: 'menus.music.log',
      icon: 'ri:file-list-3-line',
      keepAlive: true,
      roles: ['R_SUPER', 'R_ADMIN']
    }
  },
  {
    path: '/settings',
    name: 'MusicSettings',
    component: '/music/settings',
    meta: {
      title: 'menus.music.settings',
      icon: 'ri:settings-3-line',
      keepAlive: true,
      roles: ['R_SUPER', 'R_ADMIN']
    }
  }
]
