<!-- 公开首页：面向普通用户的在线搜歌与接口说明 -->
<template>
  <div class="home-page">
    <!-- 背景光晕 -->
    <span class="glow glow-1"></span>
    <span class="glow glow-2"></span>

    <!-- 顶部导航 -->
    <header class="home-nav">
      <div class="nav-inner">
        <div class="flex-c gap-2.5">
          <ArtLogo size="32" />
          <span class="brand-name">{{ siteName }}</span>
        </div>
        <button class="nav-link" @click="scrollToApi">
          <ArtSvgIcon icon="ri:book-open-line" />
          <span>接口文档</span>
        </button>
      </div>
    </header>

    <main class="home-main">
      <!-- Hero -->
      <section class="hero">
        <div class="hero-inner">
          <p class="hero-badge">
            <ArtSvgIcon icon="ri:sparkling-line" />
            多音源聚合 · 无需登录
          </p>

          <h1 class="hero-title">
            在线搜歌，一条请求拿到<span class="hero-accent">播放直链</span>
          </h1>
          <p class="hero-sub">
            QQ音乐、网易云、酷狗、汽水音乐，一次检索即可试听并复制链接
          </p>

          <!-- 搜索面板 -->
          <div class="search-panel">
            <div class="search-row">
              <div class="search-field">
                <ArtSvgIcon icon="ri:search-line" class="search-icon" />
                <input
                  v-model="keyword"
                  class="search-input"
                  type="text"
                  placeholder="输入歌名，例如：晴天"
                  @keyup.enter="doSearch"
                />
              </div>

              <div class="search-field search-field-singer">
                <ArtSvgIcon icon="ri:user-voice-line" class="search-icon" />
                <input
                  v-model="singer"
                  class="search-input"
                  type="text"
                  placeholder="歌手（可选）"
                  @keyup.enter="doSearch"
                />
              </div>

              <ElButton type="primary" class="search-btn" :loading="loading" @click="doSearch">
                搜索
              </ElButton>
            </div>

            <div class="search-chips">
              <button
                v-for="item in sourceOptions"
                :key="item.key"
                class="chip"
                :class="{ active: source === item.key }"
                @click="source = item.key"
              >
                <i class="chip-dot" :style="{ backgroundColor: item.color }"></i>
                {{ item.name }}
              </button>
            </div>
          </div>

          <!-- 结果 -->
          <transition name="rise">
            <div v-if="loading || result || errorMsg" class="result-card art-card">
              <div v-if="loading" class="result-loading">
                <span class="dot"></span>
                <span class="dot"></span>
                <span class="dot"></span>
                <span class="ml-1">正在搜索…</span>
              </div>

              <ElAlert
                v-else-if="errorMsg"
                :title="errorMsg"
                type="warning"
                :closable="false"
                show-icon
              />

              <template v-else-if="result">
                <div class="result-body">
                  <img
                    v-if="result.cover && !coverFailed"
                    class="result-cover"
                    :src="result.cover"
                    alt="cover"
                    referrerpolicy="no-referrer"
                    loading="lazy"
                    @error="coverFailed = true"
                  />
                  <div v-else class="result-cover result-cover-empty">
                    <ArtSvgIcon icon="ri:music-2-line" />
                  </div>

                  <div class="min-w-0 flex-1">
                    <div class="flex-c gap-2">
                      <h3 class="result-title">{{ result.title }}</h3>
                      <ElTag size="small" effect="light" round>{{ sourceName(result.source) }}</ElTag>
                    </div>
                    <p class="result-singer">{{ result.singer }}</p>

                    <audio
                      v-if="result.music_url"
                      ref="audioRef"
                      class="result-audio"
                      :src="result.music_url"
                      controls
                      @timeupdate="onTimeUpdate"
                      @seeked="onTimeUpdate"
                    />

                    <div class="result-actions">
                      <ElButton size="small" @click="copy(result.music_url, '播放直链已复制')">
                        复制直链
                      </ElButton>
                      <ElButton
                        size="small"
                        @click="copy(JSON.stringify(result, null, 2), '返回 JSON 已复制')"
                      >
                        复制 JSON
                      </ElButton>
                      <a
                        v-if="result.link"
                        class="result-link"
                        :href="result.link"
                        target="_blank"
                        rel="noreferrer"
                      >
                        查看原曲
                        <ArtSvgIcon icon="ri:arrow-right-up-line" />
                      </a>
                      <span v-if="duration" class="result-duration">{{ duration }}</span>
                    </div>
                  </div>
                </div>

                <!-- 歌词 -->
                <div v-if="lyricLines.length" class="result-lyric">
                  <div class="lyric-head">
                    <span>歌词</span>
                    <span v-if="hasTimeline" class="lyric-tip">播放时自动滚动，点击歌词可跳转</span>
                  </div>
                  <div ref="lyricBox" class="lyric-box">
                    <p
                      v-for="(line, index) in lyricLines"
                      :key="index"
                      class="lyric-line"
                      :class="{
                        active: hasTimeline && index === activeLine,
                        clickable: hasTimeline
                      }"
                      @click="seekTo(line)"
                    >
                      {{ line.text }}
                    </p>
                  </div>
                </div>
              </template>
            </div>
          </transition>
        </div>
      </section>

      <!-- 特性 -->
      <section class="features">
        <div v-for="item in features" :key="item.title" class="feature-item art-card">
          <div class="feature-icon">
            <ArtSvgIcon :icon="item.icon" />
          </div>
          <h3>{{ item.title }}</h3>
          <p>{{ item.desc }}</p>
        </div>
      </section>

      <!-- 接口说明 -->
      <section id="api" class="api-section">
        <div class="section-head">
          <h2>接口文档</h2>
          <p>一条 GET 请求即可返回播放直链，可直接用于网页、小程序或自建播放器。</p>
        </div>

        <div class="api-card art-card">
          <div class="api-line">
            <span class="method">GET</span>
            <code>{{ apiExample }}</code>
            <button class="copy-btn" @click="copy(apiExample, '示例地址已复制')">
              <ArtSvgIcon icon="ri:file-copy-line" />
              复制
            </button>
          </div>

          <table class="param-table">
            <thead>
              <tr>
                <th>参数</th>
                <th>类型</th>
                <th>说明</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in params" :key="row.name">
                <td><code>{{ row.name }}</code></td>
                <td>{{ row.type }}</td>
                <td>{{ row.desc }}</td>
              </tr>
            </tbody>
          </table>

          <p class="code-title">返回示例</p>
          <pre class="api-code">{{ sampleJson }}</pre>
        </div>
      </section>
    </main>

    <footer class="home-footer">
      仅供个人学习与研究使用 · {{ siteName }} · 开发者：逸雨网络
    </footer>
  </div>
</template>

<script setup lang="ts">
  import { ElMessage } from 'element-plus'
  import AppConfig from '@/config'
  import { searchMusic, type MusicResult } from '@/api/music'
  import { copyText, sourceName } from '@/utils/music-format'

  defineOptions({ name: 'Home' })

  const siteName = AppConfig.systemInfo.name
  const apiBase = window.location.origin

  const sourceOptions = [
    { key: 'all', name: '全部音源', color: 'var(--el-color-primary)' },
    { key: 'qqmusic', name: 'QQ音乐', color: '#5D87FF' },
    { key: 'wyy', name: '网易云音乐', color: '#FA8A6C' },
    { key: 'kugou', name: '酷狗音乐', color: '#FFAF20' },
    { key: 'qishui', name: '汽水音乐', color: '#14DEBA' }
  ]

  const features = [
    {
      icon: 'ri:flashlight-line',
      title: '一次请求',
      desc: '标准 GET 调用，直接返回可播放的地址，无需多轮跳转解析'
    },
    {
      icon: 'ri:shield-check-line',
      title: 'Cookie 池调度',
      desc: '多账号自动轮换，失效自动切换，稳定提升取链成功率'
    },
    {
      icon: 'ri:code-box-line',
      title: '统一 JSON',
      desc: '四个音源字段一致，任何语言都能快速接入自己的项目'
    }
  ]

  const params = [
    {
      name: 'source',
      type: 'string',
      desc: 'qqmusic / wyy / kugou / qishui，缺省使用默认音源'
    },
    { name: 'id', type: 'string', desc: '汽水音乐歌曲 ID 或分享链接' },
    { name: 'name', type: 'string', desc: '歌曲名（必填，汽水音乐请改用 id）' },
    { name: 'singer', type: 'string', desc: '歌手名，可选，用于避开翻唱、精准命中原唱' }
  ]

  const keyword = ref('晴天')
  const singer = ref('')
  const source = ref('all')
  const loading = ref(false)
  const result = ref<MusicResult | null>(null)
  const errorMsg = ref('')
  const coverFailed = ref(false)

  const duration = computed(() => {
    const ms = result.value?.song_duration_ms || 0
    if (!ms) return ''
    const total = Math.round(ms / 1000)
    return `${Math.floor(total / 60)}:${String(total % 60).padStart(2, '0')}`
  })

  // ---------- 歌词 ----------
  interface LyricLine {
    /** 开始时间（秒），-1 表示无时间轴 */
    time: number
    text: string
  }

  const audioRef = ref<HTMLAudioElement>()
  const lyricBox = ref<HTMLElement>()
  const currentTime = ref(0)

  const lyricTimeRe = /\[(\d{1,2}):(\d{1,2})(?:[.:](\d{1,3}))?\]/g

  /**
   * 解析 LRC，得到「时间 + 文本」列表
   */
  const parseLrc = (raw: string): LyricLine[] => {
    if (!raw) return []

    const lines: LyricLine[] = []

    raw.split('\n').forEach((row) => {
      const tags = Array.from(row.matchAll(lyricTimeRe))
      const text = row.replace(lyricTimeRe, '').trim()
      if (!tags.length || !text) return

      tags.forEach((tag) => {
        const minutes = Number(tag[1] || 0)
        const seconds = Number(tag[2] || 0)
        const millis = Number((tag[3] || '0').padEnd(3, '0'))
        lines.push({ time: minutes * 60 + seconds + millis / 1000, text })
      })
    })

    return lines.sort((a, b) => a.time - b.time)
  }

  // 优先使用带时间轴的歌词，否则按纯文本逐行展示
  const lyricLines = computed<LyricLine[]>(() => {
    const synced = parseLrc(result.value?.lyric_lrc || '')
    if (synced.length) return synced

    return (result.value?.lyric || '')
      .split('\n')
      .map((text) => text.trim())
      .filter(Boolean)
      .map((text) => ({ time: -1, text }))
  })

  const hasTimeline = computed(() => lyricLines.value.some((line) => line.time >= 0))

  // 当前播放到第几句
  const activeLine = computed(() => {
    if (!hasTimeline.value) return -1

    let current = -1
    for (let i = 0; i < lyricLines.value.length; i++) {
      if (lyricLines.value[i].time <= currentTime.value + 0.15) {
        current = i
      } else {
        break
      }
    }
    return current
  })

  const onTimeUpdate = () => {
    currentTime.value = audioRef.value?.currentTime || 0
  }

  /**
   * 点击歌词跳转到对应位置
   */
  const seekTo = (line: LyricLine) => {
    if (line.time < 0 || !audioRef.value) return
    audioRef.value.currentTime = line.time
    audioRef.value.play?.().catch(() => undefined)
  }

  // 当前行变化时把歌词滚到中间
  watch(activeLine, async (index) => {
    if (index < 0) return
    await nextTick()

    const box = lyricBox.value
    const line = box?.querySelectorAll<HTMLElement>('.lyric-line')[index]
    if (!box || !line) return

    // 让当前行停在可视区中部，offsetTop 以 .lyric-box（position: relative）为基准
    const top = line.offsetTop - (box.clientHeight - line.clientHeight) / 2
    box.scrollTo({ top: Math.max(0, top), behavior: 'smooth' })
  })

  const apiExample = `${apiBase}/api/music/search?source=qqmusic&name=晴天`

  const sampleJson = `{
  "code": 200,
  "msg": "success",
  "source": "qqmusic",
  "song_id": "0039MnYb0qxYhV",
  "title": "晴天",
  "singer": "周杰伦",
  "cover": "https://.../cover.jpg",
  "link": "https://y.qq.com/n/ryqq/songDetail/...",
  "music_url": "https://.../晴天.mp3",
  "lyric": "[00:00.00] 晴天 - 周杰伦",
  "duration_ms": 386,
  "song_duration_ms": 269000
}`

  /**
   * 搜索歌曲
   */
  const doSearch = async () => {
    const value = keyword.value.trim()
    if (!value) {
      ElMessage.warning(source.value === 'qishui' ? '请输入歌曲 ID' : '请输入歌名')
      return
    }

    loading.value = true
    errorMsg.value = ''
    coverFailed.value = false
    currentTime.value = 0
    result.value = null

    try {
      const singerValue = singer.value.trim()
      const res = await searchMusic({
        ...(source.value === 'qishui'
          ? { id: value }
          : { name: value, singer: singerValue || undefined }),
        source: source.value === 'all' ? undefined : source.value
      })

      if (res.code !== 200 || !res.music_url) {
        errorMsg.value = res.msg || '没有找到可用的播放地址，请换个关键词或音源试试'
        return
      }

      result.value = res

      // 指定歌手未命中时给出提示，避免用户以为搜错了
      if (singerValue && !res.singer.includes(singerValue)) {
        ElMessage.warning(`没找到「${singerValue}」的版本，已返回名称最接近的结果`)
      }
    } catch (error: any) {
      errorMsg.value = error?.response?.data?.msg || error?.message || '请求失败，请稍后再试'
    } finally {
      loading.value = false
    }
  }

  /**
   * 复制文本
   */
  const copy = async (text: string, tip: string) => {
    if (!text) return
    const ok = await copyText(text)
    ElMessage[ok ? 'success' : 'error'](ok ? tip : '复制失败')
  }

  /**
   * 滚动到接口文档（hash 路由下避免改地址）
   */
  const scrollToApi = () => {
    document.getElementById('api')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
</script>

<style lang="scss" scoped>
  .home-page {
    position: relative;
    display: flex;
    flex-direction: column;
    min-height: 100vh;
    overflow-x: hidden;
    color: var(--art-gray-900);
    background:
      radial-gradient(1200px 520px at 50% -12%, var(--el-color-primary-light-9), transparent 72%),
      var(--default-bg-color);
  }

  /* ---------- 顶部导航 ---------- */
  .home-nav {
    position: sticky;
    top: 0;
    z-index: 10;
    background: color-mix(in srgb, var(--default-bg-color) 70%, transparent);
    backdrop-filter: blur(14px);
  }

  .nav-inner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    max-width: 960px;
    height: 62px;
    padding: 0 20px;
    margin: 0 auto;
  }

  .brand-name {
    font-size: 17px;
    font-weight: 600;
    letter-spacing: -0.2px;
  }

  .nav-link {
    display: inline-flex;
    gap: 5px;
    align-items: center;
    padding: 6px 12px;
    font-size: 13px;
    color: var(--art-gray-600);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: 8px;
    transition: all 0.2s;

    &:hover {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
    }
  }

  /* ---------- 主体 ---------- */
  .home-main {
    position: relative;
    z-index: 1;
    width: 100%;
    max-width: 960px;
    padding: 0 20px;
    margin: 0 auto;
  }

  .hero {
    position: relative;
    padding: 76px 0 8px;
    text-align: center;
  }

  .glow {
    position: absolute;
    z-index: 0;
    width: 560px;
    height: 560px;
    pointer-events: none;
    border-radius: 50%;
    filter: blur(120px);
    opacity: 0.45;
  }

  .glow-1 {
    top: -260px;
    left: -80px;
    background: var(--el-color-primary-light-7);
  }

  .glow-2 {
    top: -220px;
    right: -60px;
    background: var(--el-color-primary-light-8);
  }

  .hero-inner {
    position: relative;
    z-index: 1;
  }

  .hero-badge {
    display: inline-flex;
    gap: 6px;
    align-items: center;
    padding: 5px 13px;
    margin-bottom: 20px;
    font-size: 12px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border: 1px solid var(--el-color-primary-light-8);
    border-radius: 999px;
  }

  .hero-title {
    font-size: 42px;
    font-weight: 600;
    line-height: 1.25;
    letter-spacing: -1px;
  }

  .hero-accent {
    color: transparent;
    background: linear-gradient(100deg, var(--el-color-primary), var(--el-color-primary-light-3));
    background-clip: text;
    -webkit-background-clip: text;
  }

  .hero-sub {
    margin-top: 14px;
    font-size: 15px;
    color: var(--art-gray-500);
  }

  /* ---------- 搜索面板 ---------- */
  .search-panel {
    max-width: 720px;
    padding: 6px;
    margin: 34px auto 0;
    text-align: left;
    background: var(--default-box-color);
    border: 1px solid var(--art-card-border);
    border-radius: 18px;
    box-shadow: 0 20px 45px -28px color-mix(in srgb, var(--el-color-primary) 70%, transparent);
    transition: box-shadow 0.25s;

    &:focus-within {
      box-shadow: 0 22px 50px -26px color-mix(in srgb, var(--el-color-primary) 85%, transparent);
    }
  }

  .search-row {
    display: flex;
    gap: 10px;
    align-items: center;
    padding: 8px;
  }

  .search-field {
    display: flex;
    flex: 1;
    gap: 8px;
    align-items: center;
    min-width: 0;
    height: 42px;
    padding: 0 12px;
    background: var(--art-gray-100);
    border-radius: 10px;
    transition:
      background-color 0.2s,
      box-shadow 0.2s;

    &:focus-within {
      background: var(--default-box-color);
      box-shadow: inset 0 0 0 1px var(--el-color-primary);
    }
  }

  .search-field-singer {
    flex: 0 1 30%;
    max-width: 30%;
  }

  .search-icon {
    flex: none;
    font-size: 17px;
    color: var(--art-gray-500);
  }

  .search-input {
    flex: 1;
    min-width: 0;
    height: 100%;
    font-size: 15px;
    color: var(--art-gray-900);
    background: transparent;
    border: 0;
    outline: none;

    &::placeholder {
      color: var(--art-gray-500);
    }
  }

  .search-btn {
    height: 42px;
    padding: 0 26px;
    font-size: 14px;
    border-radius: 10px;
  }

  .search-chips {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    padding: 0 12px 10px 14px;
  }

  .chip {
    display: inline-flex;
    gap: 6px;
    align-items: center;
    padding: 5px 12px;
    font-size: 12px;
    color: var(--art-gray-600);
    cursor: pointer;
    background: var(--art-gray-100);
    border: 1px solid transparent;
    border-radius: 999px;
    transition: all 0.2s;

    &:hover {
      color: var(--el-color-primary);
    }

    &.active {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
      border-color: var(--el-color-primary-light-7);
    }
  }

  .chip-dot {
    width: 6px;
    height: 6px;
    border-radius: 999px;
  }

  /* ---------- 结果 ---------- */
  .result-card {
    max-width: 720px;
    padding: 18px;
    margin: 18px auto 0;
    text-align: left;
  }

  .result-loading {
    display: flex;
    gap: 5px;
    align-items: center;
    justify-content: center;
    padding: 22px 0;
    font-size: 13px;
    color: var(--art-gray-500);

    .dot {
      width: 6px;
      height: 6px;
      background: var(--el-color-primary);
      border-radius: 50%;
      animation: bounce 1s infinite ease-in-out;

      &:nth-child(2) {
        animation-delay: 0.15s;
      }

      &:nth-child(3) {
        animation-delay: 0.3s;
      }
    }
  }

  .result-body {
    display: flex;
    gap: 18px;
    align-items: flex-start;
  }

  .result-cover {
    flex: none;
    width: 124px;
    height: 124px;
    background: var(--art-gray-200);
    border-radius: 14px;
    object-fit: cover;
  }

  .result-cover-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 32px;
    color: var(--art-gray-500);
  }

  .result-title {
    overflow: hidden;
    font-size: 18px;
    font-weight: 600;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .result-singer {
    margin-top: 6px;
    font-size: 13px;
    color: var(--art-gray-500);
  }

  .result-audio {
    width: 100%;
    height: 36px;
    margin-top: 14px;
  }

  .result-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    align-items: center;
    margin-top: 12px;
  }

  .result-link {
    display: inline-flex;
    gap: 3px;
    align-items: center;
    font-size: 12px;
    color: var(--art-gray-500);
    text-decoration: none;
    transition: color 0.2s;

    &:hover {
      color: var(--el-color-primary);
    }
  }

  .result-duration {
    margin-left: auto;
    font-size: 12px;
    color: var(--art-gray-500);
  }

  /* ---------- 歌词 ---------- */
  .result-lyric {
    padding-top: 14px;
    margin-top: 16px;
    border-top: 1px dashed var(--art-gray-300);
  }

  .lyric-head {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
    font-size: 12px;
    color: var(--art-gray-500);
  }

  .lyric-tip {
    font-size: 11px;
    color: var(--art-gray-500);
    opacity: 0.8;
  }

  .lyric-box {
    position: relative;
    max-height: 176px;
    overflow-y: auto;
    text-align: center;
    scrollbar-width: thin;
    mask-image: linear-gradient(180deg, transparent 0, #000 14%, #000 86%, transparent 100%);
  }

  .lyric-line {
    padding: 5px 8px;
    font-size: 13px;
    line-height: 1.7;
    color: var(--art-gray-600);
    transition:
      color 0.2s,
      transform 0.2s;

    &.active {
      font-weight: 500;
      color: var(--el-color-primary);
      transform: scale(1.02);
    }

    &.clickable {
      cursor: pointer;

      &:hover {
        color: var(--el-color-primary);
      }
    }
  }

  /* ---------- 特性 ---------- */
  .features {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 16px;
    margin-top: 52px;
  }

  .feature-item {
    padding: 22px;
    border-radius: 14px;
    transition:
      transform 0.25s,
      box-shadow 0.25s;

    &:hover {
      box-shadow: 0 14px 30px -22px color-mix(in srgb, var(--el-color-primary) 80%, transparent);
      transform: translateY(-2px);
    }

    h3 {
      margin-top: 14px;
      font-size: 15px;
      font-weight: 600;
    }

    p {
      margin-top: 6px;
      font-size: 12px;
      line-height: 1.7;
      color: var(--art-gray-500);
    }
  }

  .feature-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 38px;
    height: 38px;
    font-size: 19px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-radius: 11px;
  }

  /* ---------- 接口文档 ---------- */
  .api-section {
    margin-top: 56px;
    scroll-margin-top: 80px;
  }

  .section-head {
    h2 {
      font-size: 22px;
      font-weight: 600;
      letter-spacing: -0.4px;
    }

    p {
      margin-top: 8px;
      font-size: 13px;
      color: var(--art-gray-500);
    }
  }

  .api-card {
    padding: 20px;
    margin-top: 18px;
    border-radius: 14px;
  }

  .api-line {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    align-items: center;
    padding: 12px 14px;
    background: var(--art-gray-100);
    border-radius: 10px;

    code {
      font-size: 13px;
      color: var(--art-gray-700);
      word-break: break-all;
    }
  }

  .method {
    padding: 2px 8px;
    font-size: 11px;
    font-weight: 600;
    color: #fff;
    background: var(--el-color-success);
    border-radius: 6px;
  }

  .copy-btn {
    display: inline-flex;
    gap: 4px;
    align-items: center;
    padding: 4px 10px;
    margin-left: auto;
    font-size: 12px;
    color: var(--el-color-primary);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-radius: 6px;
    transition: background 0.2s;

    &:hover {
      background: var(--el-color-primary-light-9);
    }
  }

  .param-table {
    width: 100%;
    margin-top: 18px;
    font-size: 13px;
    border-collapse: collapse;

    th,
    td {
      padding: 10px 12px;
      text-align: left;
      border-bottom: 1px solid var(--art-gray-200);
    }

    th {
      font-weight: 500;
      color: var(--art-gray-500);
      background: var(--art-gray-100);
    }

    td {
      color: var(--art-gray-700);

      &:first-child code {
        font-size: 12px;
        color: var(--el-color-primary);
      }
    }

    tr:last-child td {
      border-bottom: 0;
    }
  }

  .code-title {
    margin-top: 20px;
    margin-bottom: 10px;
    font-size: 13px;
    font-weight: 500;
  }

  .api-code {
    padding: 16px;
    overflow: auto;
    font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 12px;
    line-height: 1.7;
    color: #c9d1d9;
    background: #1b1b24;
    border-radius: 12px;
  }

  /* ---------- 页脚 ---------- */
  .home-footer {
    position: relative;
    z-index: 1;
    padding: 24px;
    margin-top: 60px;
    font-size: 12px;
    color: var(--art-gray-500);
    text-align: center;
    border-top: 1px solid var(--art-card-border);
  }

  /* ---------- 动画 ---------- */
  .rise-enter-active {
    transition: all 0.28s ease;
  }

  .rise-enter-from {
    opacity: 0;
    transform: translateY(10px);
  }

  @keyframes bounce {
    0%,
    80%,
    100% {
      opacity: 0.35;
      transform: translateY(0);
    }

    40% {
      opacity: 1;
      transform: translateY(-4px);
    }
  }

  /* ---------- 响应式 ---------- */
  @media (width <= 768px) {
    .nav-inner {
      height: 56px;
      padding: 0 16px;
    }

    .hero {
      padding-top: 52px;
    }

    .hero-title {
      font-size: 26px;
      letter-spacing: -0.4px;
    }

    .hero-sub {
      font-size: 13px;
    }

    .search-row {
      flex-wrap: wrap;
      gap: 6px;
      padding: 6px;
    }

    .search-field {
      flex: 1 1 100%;
      height: 40px;
    }

    .search-field-singer {
      flex: 1 1 auto;
      max-width: none;
    }

    .search-input {
      font-size: 14px;
    }

    .search-btn {
      flex: none;
      height: 40px;
      padding: 0 16px;
    }

    .result-body {
      flex-direction: column;
    }

    .result-cover {
      width: 88px;
      height: 88px;
    }

    .features {
      grid-template-columns: 1fr;
      margin-top: 36px;
    }

    .api-section {
      margin-top: 40px;
    }

    .param-table {
      font-size: 12px;

      th,
      td {
        padding: 8px 10px;
      }
    }
  }
</style>
