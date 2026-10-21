<!-- API 调用日志 -->
<template>
  <div class="art-full-height">
    <!-- 搜索栏 -->
    <ArtSearchBar
      v-model="searchForm"
      :items="searchItems"
      @search="handleSearch"
      @reset="resetSearchParams"
    />

    <ElCard class="art-table-card">
      <!-- 表格头部 -->
      <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData">
        <template #left>
          <ElSpace wrap>
            <ElDropdown trigger="click" @command="handleClear">
              <ElButton type="danger" plain v-ripple>
                清理日志
                <ArtSvgIcon icon="ri:arrow-down-s-line" class="ml-1" />
              </ElButton>
              <template #dropdown>
                <ElDropdownMenu>
                  <ElDropdownItem command="7">保留最近 7 天</ElDropdownItem>
                  <ElDropdownItem command="30">保留最近 30 天</ElDropdownItem>
                  <ElDropdownItem command="0" divided>清空全部日志</ElDropdownItem>
                </ElDropdownMenu>
              </template>
            </ElDropdown>
          </ElSpace>
        </template>
      </ArtTableHeader>

      <!-- 表格 -->
      <ArtTable
        :loading="loading"
        :data="data"
        :columns="columns"
        :pagination="pagination"
        row-key="id"
        empty-text="暂无调用日志"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      />
    </ElCard>

    <!-- 详情 -->
    <ElDrawer v-model="detailVisible" title="调用详情" size="520px">
      <div v-if="current" class="space-y-4">
        <div class="flex flex-wrap items-center gap-2">
          <ElTag :type="current.success ? 'success' : 'danger'" effect="dark">
            {{ current.success ? '成功' : '失败' }}
          </ElTag>
          <ElTag effect="plain" :style="{ color: SOURCE_COLORS[current.source] }">
            {{ sourceName(current.source) }}
          </ElTag>
          <span class="text-sm text-g-500">{{ formatTime(current.created_at) }}</span>
        </div>

        <div v-if="current.album_img" class="flex-c gap-3">
          <img
            :src="current.album_img"
            alt="专辑封面"
            referrerpolicy="no-referrer"
            class="object-cover w-24 h-24 rounded-lg"
          />
          <div class="text-sm text-g-500">
            <p>{{ current.song_name || '-' }}</p>
            <p class="mt-1">专辑封面</p>
          </div>
        </div>

        <ElDescriptions :column="2" border>
          <ElDescriptionsItem label="歌曲">{{ current.song_name || '-' }}</ElDescriptionsItem>
          <ElDescriptionsItem label="记录 ID">#{{ current.id }}</ElDescriptionsItem>
          <ElDescriptionsItem label="状态码">{{ current.response_code }}</ElDescriptionsItem>
          <ElDescriptionsItem label="耗时">{{ current.duration_ms }} ms</ElDescriptionsItem>
          <ElDescriptionsItem label="IP">{{ current.ip || '-' }}</ElDescriptionsItem>
          <ElDescriptionsItem label="User-Agent" :span="2">
            {{ current.user_agent || '-' }}
          </ElDescriptionsItem>
          <ElDescriptionsItem v-if="current.error_msg" label="错误信息" :span="2">
            <span class="text-error">{{ current.error_msg }}</span>
          </ElDescriptionsItem>
        </ElDescriptions>

        <div>
          <p class="mb-2 text-sm text-g-500">请求参数</p>
          <pre class="art-card-xs p-3 text-xs whitespace-pre-wrap break-all">{{
            current.request_params || '-'
          }}</pre>
        </div>
      </div>
    </ElDrawer>

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
  import { h, ref } from 'vue'
  import { ElButton, ElMessage, ElMessageBox, ElTag } from 'element-plus'
  import ArtButtonTable from '@/components/core/forms/art-button-table/index.vue'
  import ArtSvgIcon from '@/components/core/base/art-svg-icon/index.vue'
  import SongPlayer from '../components/song-player.vue'
  import { useTable } from '@/hooks/core/useTable'
  import { clearLogs, fetchLogList, type LogItem } from '@/api/music'
  import { SOURCE_COLORS, SOURCE_LABELS, formatTime, sourceName } from '@/utils/music-format'

  defineOptions({ name: 'MusicLog' })

  type LogSearchParams = {
    source?: string
    success?: string
    keyword?: string
  }

  const searchForm = ref<LogSearchParams>({
    source: '',
    success: '',
    keyword: ''
  })

  const searchItems = computed(() => [
    {
      label: '音乐源',
      key: 'source',
      type: 'select',
      props: {
        placeholder: '全部音源',
        clearable: true,
        options: Object.entries(SOURCE_LABELS).map(([value, label]) => ({ label, value }))
      }
    },
    {
      label: '结果',
      key: 'success',
      type: 'select',
      props: {
        placeholder: '全部结果',
        clearable: true,
        options: [
          { label: '仅成功', value: 'true' },
          { label: '仅失败', value: 'false' }
        ]
      }
    },
    {
      label: '歌曲',
      key: 'keyword',
      type: 'input',
      props: { placeholder: '歌曲名关键词', clearable: true }
    }
  ])

  const current = ref<LogItem | null>(null)
  const detailVisible = ref(false)

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

  const {
    columns,
    columnChecks,
    data,
    loading,
    pagination,
    getData,
    replaceSearchParams,
    resetSearchParams,
    handleSizeChange,
    handleCurrentChange,
    refreshData
  } = useTable({
    core: {
      apiFn: fetchLogList,
      apiParams: { current: 1, size: 20, ...searchForm.value },
      columnsFactory: () => [
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
        {
          prop: 'song_name',
          label: '歌曲',
          minWidth: 190,
          formatter: (row: LogItem) => songCell(row)
        },
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
        { prop: 'response_code', label: '状态码', width: 90 },
        {
          prop: 'duration_ms',
          label: '耗时',
          width: 90,
          formatter: (row: LogItem) =>
            h(
              'span',
              { class: row.duration_ms > 1500 ? 'text-warning' : '' },
              `${row.duration_ms} ms`
            )
        },
        { prop: 'ip', label: 'IP', width: 140, showOverflowTooltip: true },
        {
          prop: 'error_msg',
          label: '错误信息',
          minWidth: 200,
          showOverflowTooltip: true,
          formatter: (row: LogItem) =>
            h(
              'span',
              { class: row.error_msg ? 'text-error' : 'text-g-500' },
              row.error_msg || '-'
            )
        },
        {
          prop: 'operation',
          label: '操作',
          width: 90,
          fixed: 'right',
          formatter: (row: LogItem) =>
            h(ArtButtonTable, {
              type: 'view',
              onClick: () => {
                current.value = row
                detailVisible.value = true
              }
            })
        }
      ]
    }
  })

  /**
   * 搜索
   */
  const handleSearch = (params: LogSearchParams) => {
    replaceSearchParams(params)
    getData()
  }

  /**
   * 清理日志
   */
  const handleClear = async (command: string) => {
    const days = Number(command)
    const text =
      days === 0 ? '确定清空全部日志吗？该操作不可恢复。' : `确定清理 ${days} 天之前的日志吗？`
    try {
      await ElMessageBox.confirm(text, '清理确认', { type: 'warning' })
    } catch {
      return
    }
    try {
      await clearLogs(days)
      refreshData()
    } catch {
      ElMessage.error('清理失败')
    }
  }
</script>

