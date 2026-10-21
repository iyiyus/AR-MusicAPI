import { AppRouteRecord } from '@/types/router'
import { musicRoutes } from './music'

/**
 * 导出所有模块化路由
 *
 * 本项目只保留音乐 API 相关的业务菜单，模板自带的演示菜单已移除。
 */
export const routeModules: AppRouteRecord[] = [...musicRoutes]
