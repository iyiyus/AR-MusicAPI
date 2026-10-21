<!-- Cookie 池管理 -->
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
            <ElButton type="primary" @click="openAdd" v-ripple>新增 Cookie</ElButton>
            <ElButton :disabled="!selectedRows.length" @click="batchToggle" v-ripple>
              启用 / 禁用
            </ElButton>
            <ElButton :disabled="!selectedRows.length" @click="batchReset" v-ripple>
              重置计数
            </ElButton>
            <ElButton
              type="danger"
              plain
              :disabled="!selectedRows.length"
              @click="batchDelete"
              v-ripple
            >
              批量删除
            </ElButton>
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
        empty-text="暂无 Cookie，点击左上角新增"
        @selection-change="handleSelectionChange"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      />
    </ElCard>

    <!-- 新增 / 编辑 -->
    <ElDialog
      v-model="dialogVisible"
      :title="dialogType === 'add' ? '新增 Cookie' : '编辑 Cookie'"
      width="640px"
      destroy-on-close
    >
      <ElForm :model="form" label-width="90px">
        <ElFormItem label="音乐源" required>
          <ElSelect v-model="form.source" style="width: 100%" placeholder="请选择音乐源">
            <ElOption
              v-for="(label, key) in SOURCE_LABELS"
              :key="key"
              :label="label"
              :value="key"
            />
          </ElSelect>
        </ElFormItem>

        <ElFormItem label="备注">
          <ElInput v-model="form.remark" placeholder="如：主账号 / 备用账号" maxlength="100" />
        </ElFormItem>

        <ElFormItem label="Cookie" required>
          <ElInput
            v-model="form.cookie"
            type="textarea"
            :rows="dialogType === 'add' ? 8 : 5"
            :placeholder="
              dialogType === 'add'
                ? '粘贴完整的 Cookie；支持批量，每行一条'
                : '修改该条 Cookie 内容'
            "
          />
        </ElFormItem>

        <ElFormItem v-if="dialogType === 'edit'" label="状态">
          <ElSwitch v-model="form.enabled" />
        </ElFormItem>

        <ElAlert
          v-if="dialogType === 'add'"
          type="info"
          :closable="false"
          show-icon
          title="获取方式"
          description="登录对应音乐网站后按 F12，在 Network 中任选一个请求，复制请求头里完整的 Cookie 值。"
        />
      </ElForm>

      <template #footer>
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="saving" @click="submit">保存</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { h, ref } from 'vue'
  import { ElButton, ElMessage, ElMessageBox, ElSwitch, ElTag } from 'element-plus'
  import ArtButtonTable from '@/components/core/forms/art-button-table/index.vue'
  import { useTable } from '@/hooks/core/useTable'
  import {
    addCookies,
    deleteCookies,
    fetchCookieList,
    resetCookies,
    toggleCookies,
    updateCookie,
    type CookieItem
  } from '@/api/music'
  import {
    SOURCE_COLORS,
    SOURCE_LABELS,
    copyText,
    formatTime,
    sourceName,
    truncate
  } from '@/utils/music-format'

  defineOptions({ name: 'MusicCookie' })

  type CookieSearchParams = {
    source?: string
    status?: string
    keyword?: string
  }

  const searchForm = ref<CookieSearchParams>({
    source: '',
    status: '',
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
      label: '状态',
      key: 'status',
      type: 'select',
      props: {
        placeholder: '全部状态',
        clearable: true,
        options: [
          { label: '已启用', value: '1' },
          { label: '已禁用', value: '0' }
        ]
      }
    },
    {
      label: '关键词',
      key: 'keyword',
      type: 'input',
      props: { placeholder: '备注或 Cookie 内容', clearable: true }
    }
  ])

  const selectedRows = ref<CookieItem[]>([])

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
      apiFn: fetchCookieList,
      apiParams: { current: 1, size: 10, ...searchForm.value },
      columnsFactory: () => [
        { type: 'selection' },
        { prop: 'id', label: 'ID', width: 80 },
        {
          prop: 'source',
          label: '音乐源',
          width: 110,
          formatter: (row: CookieItem) =>
            h(
              ElTag,
              { size: 'small', effect: 'plain', style: { color: SOURCE_COLORS[row.source] } },
              () => sourceName(row.source)
            )
        },
        {
          prop: 'cookie',
          label: 'Cookie',
          minWidth: 300,
          formatter: (row: CookieItem) =>
            h('div', { class: 'flex items-center gap-2' }, [
              h('span', { class: 'font-mono text-xs break-all' }, truncate(row.cookie, 70)),
              h(
                ElButton,
                {
                  link: true,
                  type: 'primary',
                  size: 'small',
                  onClick: () => copyCookie(row)
                },
                () => '复制'
              )
            ])
        },
        {
          prop: 'remark',
          label: '备注',
          minWidth: 130,
          showOverflowTooltip: true,
          formatter: (row: CookieItem) => row.remark || '-'
        },
        {
          prop: 'status',
          label: '状态',
          width: 80,
          formatter: (row: CookieItem) =>
            h(ElSwitch, {
              modelValue: row.status === 1,
              size: 'small',
              'onUpdate:modelValue': (value: string | number | boolean) =>
                toggleRow(row, Boolean(value))
            })
        },
        {
          prop: 'fail_count',
          label: '失败 / 调用',
          width: 120,
          formatter: (row: CookieItem) =>
            h('span', [
              h('span', { class: row.fail_count > 0 ? 'text-error' : '' }, String(row.fail_count)),
              h('span', { class: 'text-g-500' }, ` / ${row.call_count}`)
            ])
        },
        {
          prop: 'last_used',
          label: '最后使用',
          width: 170,
          formatter: (row: CookieItem) => formatTime(row.last_used)
        },
        {
          prop: 'operation',
          label: '操作',
          width: 130,
          fixed: 'right',
          formatter: (row: CookieItem) =>
            h('div', [
              h(ArtButtonTable, { type: 'edit', onClick: () => openEdit(row) }),
              h(ArtButtonTable, {
                type: 'view',
                icon: 'ri:refresh-line',
                onClick: () => resetOne(row)
              }),
              h(ArtButtonTable, { type: 'delete', onClick: () => removeOne(row) })
            ])
        }
      ]
    }
  })

  /**
   * 搜索
   */
  const handleSearch = (params: CookieSearchParams) => {
    replaceSearchParams(params)
    getData()
  }

  /**
   * 选中行
   */
  const handleSelectionChange = (rows: CookieItem[]) => {
    selectedRows.value = rows
  }

  // ---- 弹窗 ----
  const dialogVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const saving = ref(false)
  const form = ref({
    id: 0,
    source: 'qqmusic',
    remark: '',
    cookie: '',
    enabled: true
  })

  const openAdd = () => {
    dialogType.value = 'add'
    form.value = { id: 0, source: 'qqmusic', remark: '', cookie: '', enabled: true }
    dialogVisible.value = true
  }

  const openEdit = (row: CookieItem) => {
    dialogType.value = 'edit'
    form.value = {
      id: row.id,
      source: row.source,
      remark: row.remark,
      cookie: row.cookie,
      enabled: row.status === 1
    }
    dialogVisible.value = true
  }

  const submit = async () => {
    if (!form.value.source) {
      ElMessage.warning('请选择音乐源')
      return
    }
    if (!form.value.cookie.trim()) {
      ElMessage.warning('请填写 Cookie 内容')
      return
    }

    saving.value = true
    try {
      if (dialogType.value === 'add') {
        const cookies = form.value.cookie
          .split('\n')
          .map((line) => line.trim())
          .filter(Boolean)
        await addCookies({
          source: form.value.source,
          remark: form.value.remark,
          cookies
        })
      } else {
        await updateCookie({
          id: form.value.id,
          source: form.value.source,
          cookie: form.value.cookie.trim(),
          remark: form.value.remark,
          status: form.value.enabled ? 1 : 0
        })
      }
      dialogVisible.value = false
      refreshData()
    } catch {
      // 错误提示由请求层统一处理
    } finally {
      saving.value = false
    }
  }

  // ---- 行操作 ----
  const toggleRow = async (row: CookieItem, enabled: boolean) => {
    try {
      await updateCookie({
        id: row.id,
        source: row.source,
        cookie: row.cookie,
        remark: row.remark,
        status: enabled ? 1 : 0
      })
      refreshData()
    } catch {
      refreshData()
    }
  }

  const resetOne = async (row: CookieItem) => {
    await resetCookies([row.id])
    refreshData()
  }

  const removeOne = async (row: CookieItem) => {
    try {
      await ElMessageBox.confirm('确定删除这条 Cookie 吗？', '删除确认', { type: 'warning' })
    } catch {
      return
    }
    await deleteCookies([row.id])
    refreshData()
  }

  const copyCookie = async (row: CookieItem) => {
    const ok = await copyText(row.cookie)
    ElMessage[ok ? 'success' : 'error'](ok ? '已复制到剪贴板' : '复制失败')
  }

  // ---- 批量操作 ----
  const batchToggle = async () => {
    await toggleCookies(selectedRows.value.map((item) => item.id))
    refreshData()
  }

  const batchReset = async () => {
    await resetCookies(selectedRows.value.map((item) => item.id))
    refreshData()
  }

  const batchDelete = async () => {
    try {
      await ElMessageBox.confirm(
        `确定删除选中的 ${selectedRows.value.length} 条 Cookie 吗？`,
        '批量删除',
        { type: 'warning' }
      )
    } catch {
      return
    }
    await deleteCookies(selectedRows.value.map((item) => item.id))
    refreshData()
  }
</script>
