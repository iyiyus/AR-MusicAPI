<!-- 歌曲播放弹窗：封面 + 歌名 + 歌手 + 滚动歌词 -->
<template>
  <ElDialog
    v-model="visible"
    width="520px"
    title="正在播放"
    class="song-player-dialog"
    append-to-body
    destroy-on-close
  >
    <div v-if="record" class="player">
      <div class="player-head">
        <img
          v-if="record.album_img"
          class="player-cover"
          :src="record.album_img"
          alt="cover"
          referrerpolicy="no-referrer"
        />
        <div v-else class="player-cover player-cover-empty">
          <ArtSvgIcon icon="ri:music-2-line" />
        </div>

        <div class="min-w-0 flex-1">
          <h3 class="player-title">{{ record.song_name || '-' }}</h3>
          <p class="player-singer">{{ record.singer || '未知歌手' }}</p>
          <div class="flex-c gap-2 mt-1.5">
            <ElTag size="small" effect="plain" :style="{ color: SOURCE_COLORS[record.source] }">
              {{ sourceName(record.source) }}
            </ElTag>
            <span class="text-xs text-g-500">{{ formatTime(record.created_at) }}</span>
          </div>
        </div>
      </div>

      <audio
        v-if="record.music_url"
        ref="audioRef"
        class="player-audio"
        :src="record.music_url"
        controls
        autoplay
        @timeupdate="onTimeUpdate"
        @seeked="onTimeUpdate"
      />
      <ElAlert
        v-else
        class="mt-4"
        type="warning"
        :closable="false"
        title="该记录没有可用的播放直链"
        description="历史日志未保存直链，或音源本次没有返回地址，重新搜索一次即可。"
      />

      <!-- 歌词 -->
      <div v-if="lyricLines.length" ref="lyricBox" class="lyric-box">
        <p
          v-for="(line, index) in lyricLines"
          :key="index"
          class="lyric-line"
          :class="{ active: index === activeLine, clickable: hasTimeline }"
          @click="seekTo(line)"
        >
          {{ line.text }}
        </p>
      </div>
      <ElEmpty v-else class="mt-2" description="暂无歌词" :image-size="50" />
    </div>
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed, nextTick, ref, watch } from 'vue'
  import type { LogItem } from '@/api/music'
  import { SOURCE_COLORS, formatTime, sourceName } from '@/utils/music-format'
  import { findActiveLine, toLyricLines, type LyricLine } from '@/utils/lyric'

  defineOptions({ name: 'MusicSongPlayer' })

  interface Props {
    /** 弹窗显隐 */
    modelValue: boolean
    /** 当前播放的日志记录 */
    record: LogItem | null
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

  const visible = computed({
    get: () => props.modelValue,
    set: (value: boolean) => emit('update:modelValue', value)
  })

  const audioRef = ref<HTMLAudioElement>()
  const lyricBox = ref<HTMLElement>()
  const currentTime = ref(0)

  const lyricLines = computed<LyricLine[]>(() => toLyricLines(props.record?.lyric_lrc))
  const hasTimeline = computed(() => lyricLines.value.some((line) => line.time >= 0))
  const activeLine = computed(() => findActiveLine(lyricLines.value, currentTime.value))

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

    const top = line.offsetTop - (box.clientHeight - line.clientHeight) / 2
    box.scrollTo({ top: Math.max(0, top), behavior: 'smooth' })
  })

  // 每次打开弹窗从头计时
  watch(
    () => props.modelValue,
    (open) => {
      if (open) currentTime.value = 0
    }
  )
</script>

<style lang="scss" scoped>
  .player-head {
    display: flex;
    gap: 14px;
    align-items: center;
  }

  .player-cover {
    flex: none;
    width: 88px;
    height: 88px;
    object-fit: cover;
    background: var(--art-gray-200);
    border-radius: 10px;
  }

  .player-cover-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 26px;
    color: var(--art-gray-500);
  }

  .player-title {
    overflow: hidden;
    font-size: 18px;
    font-weight: 600;
    color: var(--art-gray-900);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .player-singer {
    margin-top: 2px;
    font-size: 13px;
    color: var(--art-gray-500);
  }

  .player-audio {
    width: 100%;
    height: 38px;
    margin-top: 16px;
  }

  .lyric-box {
    max-height: 220px;
    margin-top: 16px;
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
</style>
