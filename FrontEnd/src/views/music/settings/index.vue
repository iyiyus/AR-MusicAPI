<!-- 系统设置 -->
<template>
  <div>
    <ElRow :gutter="20">
      <!-- 基础设置 -->
      <ElCol :xs="24" :lg="16">
        <ElCard class="art-card-sm">
          <template #header>
            <div class="flex-cb">
              <span class="font-semibold">基础设置</span>
              <ElButton link type="primary" :loading="loading" @click="load">重新读取</ElButton>
            </div>
          </template>

          <ElForm v-loading="loading" :model="form" label-width="150px">
            <ElFormItem label="站点名称">
              <ElInput v-model="form.site_name" maxlength="40" show-word-limit />
            </ElFormItem>

            <ElFormItem label="站点图标">
              <div class="flex-c gap-4">
                <div
                  class="flex-cc w-12 h-12 p-1.5 overflow-hidden rounded-[10px] border border-g-300 bg-[var(--art-gray-100)]"
                >
                  <img
                    v-if="form.site_icon"
                    :src="form.site_icon"
                    alt="站点图标"
                    class="object-contain w-full h-full"
                  />
                  <ArtLogo v-else size="30" />
                </div>
                <ElUpload
                  :show-file-list="false"
                  accept=".png,.jpg,.jpeg,.webp,.gif,.ico,.svg"
                  :http-request="handleIconUpload"
                >
                  <ElButton :loading="iconUploading">上传图片</ElButton>
                </ElUpload>
                <ElButton v-if="form.site_icon" link type="danger" @click="resetIcon">
                  恢复默认
                </ElButton>
              </div>
              <p class="w-full mt-1 text-xs text-g-500">
                支持 png / jpg / webp / svg / ico，大小不超过 2MB，上传后点击保存生效
              </p>
            </ElFormItem>

            <ElFormItem label="默认音源">
              <ElSelect v-model="form.default_source" class="w-60">
                <ElOption
                  v-for="source in sources"
                  :key="source.key"
                  :label="source.name"
                  :value="source.key"
                />
              </ElSelect>
              <span class="ml-3 text-sm text-g-500">调用方未指定 source 时使用</span>
            </ElFormItem>

            <ElFormItem label="日志保留天数">
              <ElInputNumber v-model="form.log_retention_days" :min="1" :max="365" />
              <span class="ml-3 text-sm text-g-500">1 - 365 天</span>
            </ElFormItem>

            <ElFormItem label="Cookie 失败阈值">
              <ElInputNumber v-model="form.cookie_fail_threshold" :min="1" :max="20" />
              <span class="ml-3 text-sm text-g-500">连续失败达到该次数后自动降权</span>
            </ElFormItem>

            <ElFormItem label="自动切换 Cookie">
              <ElSwitch v-model="form.auto_switch_cookie" />
              <span class="ml-3 text-sm text-g-500">请求失败时自动换用下一条 Cookie</span>
            </ElFormItem>

            <ElFormItem label="开放公开接口">
              <ElSwitch v-model="form.allow_public_api" />
              <span class="ml-3 text-sm text-g-500">
                关闭后 /api/music/search 仅允许携带令牌访问
              </span>
            </ElFormItem>

            <ElFormItem>
              <ElButton type="primary" :loading="saving" @click="save" v-ripple>保存设置</ElButton>
              <ElButton @click="load" v-ripple>还原</ElButton>
            </ElFormItem>
          </ElForm>
        </ElCard>
      </ElCol>

      <!-- 账号信息 -->
      <ElCol :xs="24" :lg="8">
        <ElCard class="art-card-sm mb-5">
          <template #header>
            <span class="font-semibold">当前账号</span>
          </template>

          <ElDescriptions :column="1" size="small">
            <ElDescriptionsItem label="用户名">{{ account?.username || '-' }}</ElDescriptionsItem>
            <ElDescriptionsItem label="昵称">{{ account?.nickname || '-' }}</ElDescriptionsItem>
            <ElDescriptionsItem label="角色">
              <ElTag v-for="role in account?.roles || []" :key="role" size="small" class="mr-1">
                {{ role }}
              </ElTag>
            </ElDescriptionsItem>
            <ElDescriptionsItem label="最近登录">
              {{ formatTime(account?.last_login_at) }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="登录 IP">
              {{ account?.last_login_ip || '-' }}
            </ElDescriptionsItem>
          </ElDescriptions>
        </ElCard>

        <ElCard class="art-card-sm">
          <template #header>
            <span class="font-semibold">修改密码</span>
          </template>

          <ElForm :model="pwdForm" label-position="top">
            <ElFormItem label="当前密码">
              <ElInput v-model="pwdForm.oldPassword" type="password" show-password />
            </ElFormItem>
            <ElFormItem label="新密码">
              <ElInput v-model="pwdForm.newPassword" type="password" show-password />
            </ElFormItem>
            <ElFormItem label="确认新密码">
              <ElInput v-model="pwdForm.confirm" type="password" show-password />
            </ElFormItem>
            <ElButton type="primary" :loading="pwdSaving" @click="submitPassword" v-ripple>
              修改密码
            </ElButton>
          </ElForm>
        </ElCard>
      </ElCol>
    </ElRow>

    <!-- 接口说明 -->
    <ElCard class="art-card-sm mt-5">
      <template #header>
        <div class="flex-cb">
          <span class="font-semibold">对外点歌接口</span>
          <ElButton link type="primary" @click="copyApi">复制地址</ElButton>
        </div>
      </template>

      <div class="space-y-3">
        <div class="flex-wrap gap-2 flex-c">
          <ElTag type="success" effect="dark" size="small">GET</ElTag>
          <code class="art-card-xs px-2 py-1 text-xs break-all">
            {{ apiBase }}/api/music/search?source=qqmusic&amp;name=晴天
          </code>
        </div>

        <ElDescriptions :column="1" border size="small">
          <ElDescriptionsItem label="source">
            qqmusic / wyy / kugou / qishui，缺省使用默认音源
          </ElDescriptionsItem>
          <ElDescriptionsItem label="id">
            汽水音乐歌曲 ID 或分享链接
          </ElDescriptionsItem>
          <ElDescriptionsItem label="name">歌曲名（必填，汽水音乐请改用 id）</ElDescriptionsItem>
        </ElDescriptions>

        <pre class="art-card-xs p-3 text-xs whitespace-pre-wrap">{{
          sampleResponse
        }}</pre>
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { onMounted, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    changePassword,
    fetchSettings,
    updateSettings,
    uploadSiteIcon,
    type MusicSource,
    type SettingsPayload
  } from '@/api/music'
  import { copyText, formatTime } from '@/utils/music-format'
  import { useSiteStore } from '@/store/modules/site'

  defineOptions({ name: 'MusicSettings' })

  const siteStore = useSiteStore()

  const loading = ref(false)
  const saving = ref(false)
  const pwdSaving = ref(false)
  const iconUploading = ref(false)
  const sources = ref<MusicSource[]>([])
  const account = ref<SettingsPayload['account']>()

  const form = ref({
    site_name: 'AR-musicAPI',
    site_icon: '',
    default_source: 'qqmusic',
    log_retention_days: 30,
    cookie_fail_threshold: 3,
    auto_switch_cookie: true,
    allow_public_api: true
  })

  const pwdForm = ref({ oldPassword: '', newPassword: '', confirm: '' })

  const apiBase = window.location.origin

  const sampleResponse = `{
  "code": 200,
  "msg": "success",
  "source": "qqmusic",
  "title": "晴天",
  "singer": "周杰伦",
  "cover": "https://...",
  "link": "https://y.qq.com/...",
  "music_url": "https://...",
  "lyric": "故事的小黄花..."
}`

  const toBool = (value?: string) => value === '1' || value === 'true'

  /**
   * 读取配置
   */
  const load = async () => {
    loading.value = true
    try {
      const data = await fetchSettings()
      if (!data) return

      sources.value = data.sources || []
      account.value = data.account
      form.value = {
        site_name: data.site_name || 'AR-musicAPI',
        site_icon: data.site_icon || '',
        default_source: data.default_source || 'qqmusic',
        log_retention_days: Number(data.log_retention_days || 30),
        cookie_fail_threshold: Number(data.cookie_fail_threshold || 3),
        auto_switch_cookie: toBool(data.auto_switch_cookie),
        allow_public_api: toBool(data.allow_public_api)
      }
    } catch {
      // 错误提示由请求层统一处理
    } finally {
      loading.value = false
    }
  }

  /**
   * 保存配置
   */
  const save = async () => {
    if (!form.value.site_name.trim()) {
      ElMessage.warning('站点名称不能为空')
      return
    }

    saving.value = true
    try {
      await updateSettings({
        site_name: form.value.site_name.trim(),
        site_icon: form.value.site_icon,
        default_source: form.value.default_source,
        log_retention_days: Number(form.value.log_retention_days),
        cookie_fail_threshold: Number(form.value.cookie_fail_threshold),
        auto_switch_cookie: form.value.auto_switch_cookie,
        allow_public_api: form.value.allow_public_api
      })

      // 同步到全局，图标与名称立即生效
      siteStore.setInfo({
        site_name: form.value.site_name.trim(),
        site_icon: form.value.site_icon
      })
      load()
    } catch {
      // 错误提示由请求层统一处理
    } finally {
      saving.value = false
    }
  }

  /**
   * 修改密码
   */
  const submitPassword = async () => {
    if (!pwdForm.value.oldPassword || !pwdForm.value.newPassword) {
      ElMessage.warning('请填写完整的密码信息')
      return
    }
    if (pwdForm.value.newPassword.length < 6) {
      ElMessage.warning('新密码至少 6 位')
      return
    }
    if (pwdForm.value.newPassword !== pwdForm.value.confirm) {
      ElMessage.warning('两次输入的新密码不一致')
      return
    }

    pwdSaving.value = true
    try {
      await changePassword({
        oldPassword: pwdForm.value.oldPassword,
        newPassword: pwdForm.value.newPassword
      })
      pwdForm.value = { oldPassword: '', newPassword: '', confirm: '' }
    } catch {
      // 错误提示由请求层统一处理
    } finally {
      pwdSaving.value = false
    }
  }

  /**
   * 复制接口地址
   */
  const copyApi = async () => {
    const ok = await copyText(`${apiBase}/api/music/search?source=qqmusic&name=晴天`)
    ElMessage[ok ? 'success' : 'error'](ok ? '接口地址已复制' : '复制失败')
  }

  /**
   * 上传站点图标
   */
  const handleIconUpload = async (options: any) => {
    const file: File = options?.file
    if (!file) return

    if (file.size > 2 * 1024 * 1024) {
      ElMessage.warning('图片大小不能超过 2MB')
      return
    }

    iconUploading.value = true
    try {
      const { url } = await uploadSiteIcon(file)
      form.value.site_icon = url
      ElMessage.success('图标已上传，点击保存后生效')
    } catch (error: any) {
      ElMessage.error(error?.response?.data?.msg || '图标上传失败')
    } finally {
      iconUploading.value = false
    }
  }

  /**
   * 恢复默认图标
   */
  const resetIcon = () => {
    form.value.site_icon = ''
  }

  onMounted(load)
</script>
