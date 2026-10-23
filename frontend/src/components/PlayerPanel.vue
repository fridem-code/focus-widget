<script setup lang="ts">
import { onMounted, useTemplateRef } from 'vue'
import type { MusicPlayer } from '../lib/musicPlayer'
import PlayerError from './PlayerError.vue'
import PlayerPlaceholder from './PlayerPlaceholder.vue'

const props = defineProps<{ music: MusicPlayer }>()
const emit = defineEmits<{ chooseGenre: [] }>()

const host = useTemplateRef<HTMLElement>('host')
onMounted(() => props.music.attach(host.value!))
</script>

<!-- The iframe lives in `host` once created and is only hidden (never covered)
     while an error card takes its place (SPEC §2, §3.4). -->
<template>
  <div class="panel">
    <div v-show="music.state.created && !music.state.error" ref="host" class="host"></div>
    <PlayerError
      v-if="music.state.error"
      :kind="music.state.error"
      :code="music.state.errorCode"
      :genre-name="music.state.genreName"
      @retry="music.retry()"
      @choose-genre="emit('chooseGenre')"
    />
    <PlayerPlaceholder v-else-if="!music.state.created" />
  </div>
</template>

<style scoped>
.panel {
  width: 356px;
  height: 204px;
  flex-shrink: 0;
}

.host {
  width: 100%;
  height: 100%;
  border-radius: var(--radius-l);
  overflow: hidden;
  background: var(--player-bg);
}

.host :deep(iframe) {
  display: block;
}
</style>
