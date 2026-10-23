<script setup lang="ts">
import PlayerPlaceholder from './components/PlayerPlaceholder.vue'
import TimerControls from './components/TimerControls.vue'
import TimerView from './components/TimerView.vue'
import TitleBar from './components/TitleBar.vue'
import { useTimer } from './composables/useTimer'

const { state, start, pause, resume, skip, resetPhase } = useTimer()
</script>

<template>
  <div class="widget" :data-phase="state?.phase ?? 'focus'">
    <TitleBar />
    <PlayerPlaceholder />
    <template v-if="state">
      <TimerView :state="state" />
      <!-- T7: music row goes here -->
      <TimerControls
        :state="state"
        @start="start"
        @pause="pause"
        @resume="resume"
        @skip="skip"
        @reset-phase="resetPhase"
      />
    </template>
  </div>
</template>

<style scoped>
/* The glass fills the whole window: Windows draws the corners and shadow. */
.widget {
  height: 100%;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: var(--glass);
  border: 1px solid var(--glass-border);
}
</style>
