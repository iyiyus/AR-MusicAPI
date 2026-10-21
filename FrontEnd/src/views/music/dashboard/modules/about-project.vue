<!-- 关于项目 -->
<template>
  <div class="art-card flex-b p-5 mb-5 max-sm:mb-4">
    <div>
      <h2 class="text-2xl font-medium">关于项目</h2>
      <p class="mt-1 text-g-700">
        AR-musicAPI 是一套开箱即用的多平台音乐解析接口服务，自带管理后台
      </p>
      <p class="mt-1 text-g-700">
        支持 QQ 音乐、网易云音乐、酷狗音乐与汽水音乐，可按歌名或歌手搜索，并返回播放直链与歌词
      </p>
      <p class="mt-1 text-sm text-g-500">
        后端 Gin + MySQL · 前端 Vue3 + TypeScript + Vite + Element Plus
      </p>

      <!-- 开发者信息 -->
      <div class="flex flex-wrap gap-3 mt-5">
        <div class="dev-badge">
          <ArtSvgIcon icon="ri:user-star-line" />
          <span class="dev-label">开发者</span>
          <span class="dev-value">逸雨网络</span>
        </div>

        <div class="dev-badge dev-badge-copy" @click="copyQQ">
          <ArtSvgIcon icon="ri:chat-3-line" />
          <span class="dev-label">QQ</span>
          <span class="dev-value">204325164</span>
          <ArtSvgIcon icon="ri:file-copy-line" class="dev-copy" />
        </div>
      </div>

      <div class="mt-9 flex flex-wrap gap-3.5 max-w-150">
        <div
          v-for="link in linkList"
          :key="link.label"
          class="w-60 flex-cb h-12.5 px-3.5 border border-g-300 c-p rounded-lg text-sm bg-g-100 duration-300 hover:-translate-y-1 max-sm:w-full"
          @click="goPage(link)"
        >
          <span class="text-g-700">{{ link.label }}</span>
          <ArtSvgIcon icon="ri:arrow-right-s-line" class="text-lg text-g-600" />
        </div>
      </div>
    </div>

    <img class="w-75 max-md:!hidden" src="@imgs/draw/draw1.png" alt="关于项目" />
  </div>
</template>

<script setup lang="ts">
  import { ElMessage } from 'element-plus'
  import { useRouter } from 'vue-router'
  import { copyText } from '@/utils/music-format'

  defineOptions({ name: 'MusicAboutProject' })

  interface AboutLink {
    /** 按钮文字 */
    label: string
    /** 目标路由或地址 */
    url: string
    /** 是否在新标签页打开 */
    external?: boolean
  }

  const router = useRouter()

  const linkList: AboutLink[] = [
    { label: '在线体验', url: '/', external: true },
    { label: 'Cookie 管理', url: '/cookie' },
    { label: '调用日志', url: '/log' },
    { label: '系统配置', url: '/settings' }
  ]

  /**
   * 打开入口：站内跳转，外部页面新标签打开
   */
  const goPage = (link: AboutLink): void => {
    if (link.external) {
      window.open(link.url, '_blank', 'noopener,noreferrer')
      return
    }
    router.push(link.url)
  }

  /**
   * 复制开发者 QQ
   */
  const copyQQ = async (): Promise<void> => {
    const ok = await copyText('204325164')
    ElMessage[ok ? 'success' : 'error'](ok ? 'QQ 号已复制' : '复制失败')
  }
</script>

<style lang="scss" scoped>
  .dev-badge {
    display: inline-flex;
    gap: 7px;
    align-items: center;
    height: 38px;
    padding: 0 13px;
    font-size: 16px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border: 1px solid var(--el-color-primary-light-8);
    border-radius: 10px;

    :deep(.art-svg-icon) {
      flex: none;
      font-size: 16px;
    }
  }

  .dev-label {
    font-size: 13px;
    color: var(--art-gray-600);
  }

  .dev-value {
    font-size: 14px;
    font-weight: 600;
    letter-spacing: 0.2px;
  }

  .dev-badge-copy {
    cursor: pointer;
    transition:
      background-color 0.2s,
      transform 0.2s;

    &:hover {
      background: var(--el-color-primary-light-8);
      transform: translateY(-1px);
    }
  }

  .dev-copy {
    font-size: 13px;
    opacity: 0.65;
  }
</style>
