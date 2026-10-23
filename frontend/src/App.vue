<script setup lang="ts">
import TitleBar from './components/TitleBar.vue'
</script>

<template>
  <div class="widget">
    <TitleBar title="Focus Widget" />
    <main class="content">
      <!-- Placeholders until T5 (timer) and T6 (player). Player viewport must stay ≥200×200 (SPEC §2). -->
      <div class="player-slot">YouTube-плеер · 356×200</div>
      <div class="timer">25:00</div>
    </main>
  </div>
</template>

<style scoped>
.widget {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.content {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 12px;
}

.player-slot {
  width: 356px;
  height: 200px;
  display: grid;
  place-items: center;
  color: var(--text-muted);
  background: var(--surface);
  border-radius: var(--radius);
}

.timer {
  font-size: 48px;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}
</style>
