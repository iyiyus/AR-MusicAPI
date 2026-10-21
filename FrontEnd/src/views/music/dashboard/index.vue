<!-- 音乐 API 运行概览 -->
<template>
  <div>
    <!-- 指标卡 -->
    <ElRow :gutter="20">
      <ElCol v-for="card in cards" :key="card.label" :xs="12" :sm="12" :md="6">
        <div class="art-card mb-5 p-5">
          <p class="text-sm text-g-500">{{ card.label }}</p>
          <p class="mt-2 text-2xl font-semibold" :class="card.className">{{ card.value }}</p>
          <p class="mt-1 text-xs text-g-500">{{ card.hint }}</p>
        </div>
      </ElCol>
    </ElRow>

    <ElRow :gutter="20">
      <!-- 调用趋势 -->
      <ElCol :xs="24" :lg="16">
        <div class="art-card mb-5 p-5">
          <div class="art-card-header">
            <div class="title">
              <h4>调用趋势</h4>
              <p>最近 {{ trendDays }} 天 · 平均耗时 {{ stats.avg_duration_ms }} ms</p>
            </div>
            <ElRadioGroup v-model="trendDays" size="small" @change="loadTrend">
              <ElRadioButton :value="7">7 天</ElRadioButton>
              <ElRadioButton :value="14">14 天</ElRadioButton>
              <ElRadioButton :value="30">30 天</ElRadioButton>
            </ElRadioGroup>
          </div>

          <ArtBarChart
            height="230px"
            :data="trendSeries"
            :xAxisData="trendLabels"
            :colors="trendColors"
            :loading="loading"
            bar-width="26%"
            :border-radius="[3, 3, 0, 0]"
            :show-axis-line="false"
            :show-legend="true"
            legend-position="bottom"
          />
        </div>
      </ElCol>

      <!-- Cookie 池 -->
      <ElCol :xs="24" :lg="8">
        <div class="art-card mb-5 p-5">
          <div class="art-card-header">
            <div class="title">
              <h4>Cookie 池</h4>
              <p>共 {{ stats.total_cookies }} 条，启用 {{ stats.active_cookies }} 条</p>
            </div>
            <RouterLink to="/cookie" class="text-sm text-theme">管理</RouterLink>
          </div>

          <div v-if="stats.sources.length" class="mt-4 space-y-4">
            <div v-for="source in stats.sources" :key="source.key">
              <div class="flex-cb text-sm">
                <span class="flex-c gap-2">
                  <i
                    class="source-dot"
                    :style="{ backgroundColor: SOURCE_COLORS[source.key] }"
                  ></i>
                  {{ source.name }}
                </span>
                <span class="text-g-500">{{ stats.cookies?.[source.key] || 0 }} 条</span>
              </div>
              <div class="source-track">
                <div
                  class="source-fill"
                  :style="{
                    width: `${percent(stats.cookies?.[source.key] || 0)}%`,
                    backgroundColor: SOURCE_COLORS[source.key]
                  }"
                ></div>
              </div>
            </div>

            <p v-if="!stats.total_cookies" class="text-xs text-g-500">
              QQ音乐、网易云需添加 Cookie 才能获取播放直链
            </p>
          </div>

          <ElEmpty v-else description="暂无音乐源" :image-size="60" />
        </div>
      </ElCol>
    </ElRow>

    <!-- 点歌测试 -->
    <ElCard class="art-card-sm mb-5">
      <template #header>
        <div class="flex-cb">
          <span class="font-semibold">点歌测试</span>
          <span class="text-sm text-g-500">直接调用 /api/music/search 验证音源可用性</span>
        </div>
      </template>

      <div class="flex-wrap gap-2 flex-c">
        <ElInput
          v-model="testForm.name"
          placeholder="歌曲名，如：晴天"
          class="w-52"
          clearable
          @keyup.enter="runTest"
        />
        <ElSelect v-model="testForm.source" class="w-40" placeholder="默认音源" clearable>
          <ElOption
            v-for="source in stats.sources"
            :key="source.key"
            :label="source.name"
            :value="source.key"
          />
        </ElSelect>
        <ElButton type="primary" :loading="testing" @click="runTest" v-ripple>测试</ElButton>
        <ElSelect v-model="testForm.mode" class="w-36" placeholder="查询方式">
          <ElOption label="按歌名" value="name" />
          <ElOption label="按歌曲ID" value="id" />
        </ElSelect>
      </div>

      <div v-if="testResult" class="mt-5">
        <div class="flex-wrap gap-4 flex-c">
          <ElImage
            v-if="testResult.cover"
            :src="testResult.cover"
            fit="cover"
            class="size-24 shrink-0 rounded-lg"
            :preview-src-list="[testResult.cover]"
            preview-teleported
          />
          <div class="min-w-0 flex-1">
            <div class="flex-wrap gap-2 flex-c">
              <span class="text-base font-medium">{{ testResult.title || '未获取到歌曲' }}</span>
              <ElTag size="small" effect="plain" :style="{ color: SOURCE_COLORS[testResult.source] }">
                {{ sourceName(testResult.source) }}
              </ElTag>
              <ElTag v-if="testResult.music_url" type="success" size="small" effect="dark">
                直链可用
              </ElTag>
              <ElTag v-else type="danger" size="small" effect="dark">无直链</ElTag>
            </div>

            <p class="mt-1 text-sm text-g-500">
              {{ testResult.singer || '-' }}
              <span v-if="testResult.duration_ms"> · {{ testResult.duration_ms }} ms</span>
            </p>

            <p v-if="testResult.msg && testResult.msg !== 'success'" class="mt-2 text-xs text-warning">
              {{ testResult.msg }}
            </p>

            <div v-if="testResult.music_url" class="mt-3 flex-wrap gap-2 flex-c">
              <audio :src="testResult.music_url" controls class="h-9 max-w-full" />
              <ElButton size="small" text type="primary" @click="copyMusicUrl">复制直链</ElButton>
            </div>

            <div class="mt-2 flex-wrap gap-3 text-xs flex-c">
              <a
                v-if="testResult.link"
                :href="testResult.link"
                target="_blank"
                class="text-theme"
              >
                查看详情页
              </a>
              <span v-if="testResult.lyric" class="text-g-500">
                歌词 {{ testResult.lyric.length }} 字
              </span>
            </div>
          </div>
        </div>
      </div>
    </ElCard>

    <!-- 最近调用 -->
    <ElCard class="art-card-sm mb-5">
      <template #header>
        <div class="flex-cb">
          <span class="font-semibold">最近调用</span>
          <RouterLink to="/log" class="text-sm text-theme">查看全部</RouterLink>
        </div>
      </template>

      <ArtTable :data="recent" :columns="recentColumns" row-key="id" empty-text="暂无调用记录" />
    </ElCard>

    <!-- 关于项目 -->
    <AboutProject />

    <!-- 封面大图预览 -->
    <ElImageViewer
      v-if="previewVisible"
      :url-list="[previewUrl]"
      teleported
      @close="previewVisible = false"
    />

    <!-- 播放弹窗 -->
    <SongPlayer v-model="playerVisible" :record="playerRecord" />
  </div>
</template>

<script setup lang="ts">
  import { h, onMounted, ref } from 'vue'
  import { ElButton, ElMessage, ElTag } from 'element-plus'
  import ArtSvgIcon from '@/components/core/base/art-svg-icon/index.vue'
  import type { ColumnOption } from '@/types/component'
  import {
    fetchDashboardStats,
    fetchRecentLogs,
    fetchTrend,
    searchMusic,
    type DashboardStats,
    type LogItem,
    type MusicResult,
    type TrendPoint
  } from '@/api/music'
  import { SOURCE_COLORS, copyText, formatTime, sourceName } from '@/utils/music-format'
  import AboutProject from './modules/about-project.vue'
  import SongPlayer from '../components/song-player.vue'

  defineOptions({ name: 'MusicDashboard' })

  const loading = ref(false)
  const trendDays = ref(7)
  const trend = ref<TrendPoint[]>([])
  const recent = ref<LogItem[]>([])

  const stats = ref<DashboardStats>({
    total_calls: 0,
    today_calls: 0,
    success_rate: 0,
    avg_duration_ms: 0,
    total_cookies: 0,
    active_cookies: 0,
    cookies: {},
    default_source: '',
    site_name: '',
    sources: [],
    uptime: ''
  })

  const cards = computed(() => [
    {
      label: '总调用次数',
      value: String(stats.value.total_calls),
      hint: `自上线以来 · 已运行 ${stats.value.uptime || '-'}`,
      className: 'text-theme'
    },
    {
      label: '今日调用',
      value: String(stats.value.today_calls),
      hint: '零点至现在',
      className: 'text-success'
    },
    {
      label: '成功率',
      value: `${stats.value.success_rate}%`,
      hint: '基于全部调用日志',
      className: 'text-warning'
    },
    {
      label: '平均耗时',
      value: `${stats.value.avg_duration_ms} ms`,
      hint: `默认音源 ${sourceName(stats.value.default_source)}`,
      className: 'text-secondary'
    }
  ])

  // 趋势图：成功 / 失败分组柱
  const trendLabels = computed(() => trend.value.map((item) => item.date))
  const trendSeries = computed(() => [
    { name: '成功', data: trend.value.map((item) => item.success) },
    { name: '失败', data: trend.value.map((item) => item.failed) }
  ])
  // 状态语义色：成功用系统 success 色，失败用 error 色，与音源的蓝色系明确区分
  const trendColors = ['#13DEB9', '#FA896B']

  const percent = (value: number) => {
    if (!stats.value.total_cookies) return 0
    return Math.round((value / stats.value.total_cookies) * 100)
  }

  // 表格单元格由 ArtTable 渲染，scoped 样式不生效，尺寸直接用内联样式
  const coverStyle = {
    flex: 'none',
    width: '30px',
    height: '30px',
    objectFit: 'cover',
    background: 'var(--art-gray-200)',
    borderRadius: '6px'
  } as const

  const coverEmptyStyle = {
    ...coverStyle,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontSize: '15px',
    color: 'var(--art-gray-500)'
  } as const

  // ---- 封面预览 / 播放弹窗 ----
  const previewUrl = ref('')
  const previewVisible = ref(false)
  const playerVisible = ref(false)
  const playerRecord = ref<LogItem | null>(null)

  const openPreview = (url: string) => {
    if (!url) return
    previewUrl.value = url
    previewVisible.value = true
  }

  const openPlayer = (row: LogItem) => {
    playerRecord.value = row
    playerVisible.value = true
  }

  /**
   * 歌曲单元格：封面（可放大）+ 歌名 + 播放按钮
   */
  const songCell = (row: LogItem) =>
    h('div', { class: 'flex-c gap-2.5 min-w-0' }, [
      row.album_img
        ? h('img', {
            src: row.album_img,
            alt: row.song_name,
            loading: 'lazy',
            title: '点击查看大图',
            referrerpolicy: 'no-referrer',
            style: { ...coverStyle, cursor: 'zoom-in' },
            onClick: () => openPreview(row.album_img),
            // 封面失效时保留灰色占位，避免出现破图
            onError: (e: Event) => {
              ;(e.target as HTMLImageElement).style.visibility = 'hidden'
            }
          })
        : h('div', { style: coverEmptyStyle }, [h(ArtSvgIcon, { icon: 'ri:music-2-line' })]),
      h('span', { class: 'truncate', title: row.song_name || '-' }, row.song_name || '-'),
      h(
        ElButton,
        {
          size: 'small',
          text: true,
          circle: true,
          title: '播放',
          onClick: (e: Event) => {
            e.stopPropagation()
            openPlayer(row)
          }
        },
        () => h(ArtSvgIcon, { icon: 'ri:play-circle-fill', class: 'text-theme', style: 'font-size: 17px' })
      )
    ])

  /**
   * 最近调用表格列
   */
  const recentColumns: ColumnOption[] = [
    {
      prop: 'created_at',
      label: '时间',
      width: 170,
      formatter: (row: LogItem) => formatTime(row.created_at)
    },
    {
      prop: 'source',
      label: '音源',
      width: 110,
      formatter: (row: LogItem) =>
        h(
          ElTag,
          { size: 'small', effect: 'plain', style: { color: SOURCE_COLORS[row.source] } },
          () => sourceName(row.source)
        )
    },
    { prop: 'song_name', label: '歌曲', minWidth: 190, formatter: (row: LogItem) => songCell(row) },
    {
      prop: 'success',
      label: '结果',
      width: 90,
      formatter: (row: LogItem) =>
        h(
          ElTag,
          { type: row.success ? 'success' : 'danger', size: 'small', effect: 'plain' },
          () => (row.success ? '成功' : '失败')
        )
    },
    {
      prop: 'duration_ms',
      label: '耗时',
      width: 90,
      formatter: (row: LogItem) => `${row.duration_ms} ms`
    },
    { prop: 'ip', label: 'IP', width: 140, showOverflowTooltip: true },
    {
      prop: 'error_msg',
      label: '错误信息',
      minWidth: 160,
      showOverflowTooltip: true,
      formatter: (row: LogItem) => row.error_msg || '-'
    }
  ]

  /**
   * 加载概览数据
   */
  const loadAll = async () => {
    loading.value = true
    try {
      const [statsData, trendData, logData] = await Promise.all([
        fetchDashboardStats(),
        fetchTrend(trendDays.value),
        fetchRecentLogs(8)
      ])
      if (statsData) stats.value = statsData
      trend.value = trendData || []
      recent.value = logData || []
    } catch {
      // 错误提示由请求层统一处理
    } finally {
      loading.value = false
    }
  }

  /**
   * 切换趋势天数
   */
  const loadTrend = async () => {
    try {
      trend.value = (await fetchTrend(trendDays.value)) || []
    } catch {
      trend.value = []
    }
  }

  // ---- 点歌测试 ----
  const testForm = ref({ name: '晴天', source: '', mode: 'name' })
  const testing = ref(false)
  const testResult = ref<MusicResult | null>(null)

  const runTest = async () => {
    const value = testForm.value.name.trim()
    if (!value) {
      ElMessage.warning(testForm.value.mode === 'id' ? '请输入歌曲 ID' : '请输入歌曲名')
      return
    }

    testing.value = true
    testResult.value = null
    try {
      const res = await searchMusic({
        ...(testForm.value.mode === 'id' ? { id: value } : { name: value }),
        source: testForm.value.source || stats.value.default_source
      })
      testResult.value = res
      if (res.code !== 200) {
        ElMessage.error(res.msg || '点歌失败')
      } else if (!res.music_url) {
        ElMessage.warning(res.msg || '未获取到播放直链')
      }
    } catch (error: any) {
      ElMessage.error(error?.message || '请求失败')
    } finally {
      testing.value = false
    }
  }

  const copyMusicUrl = async () => {
    if (!testResult.value?.music_url) return
    const ok = await copyText(testResult.value.music_url)
    ElMessage[ok ? 'success' : 'error'](ok ? '已复制播放直链' : '复制失败')
  }

  onMounted(loadAll)
</script>

<style lang="scss" scoped>
  .source-dot {
    width: 8px;
    height: 8px;
    border-radius: 999px;
  }

  .source-track {
    height: 4px;
    margin-top: 8px;
    overflow: hidden;
    background: var(--art-gray-200);
    border-radius: 999px;
  }

  .source-fill {
    height: 100%;
    border-radius: 999px;
    transition: width 0.3s ease;
  }

</style>
