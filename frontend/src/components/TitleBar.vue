<script setup lang="ts">
import { Application } from '@wailsio/runtime'

defineProps<{ title: string }>()

// T10 replaces this with hide-to-tray; until then the close button quits.
function close() {
  void Application.Quit()
}
</script>

<template>
  <header class="titlebar">
    <span class="title">{{ title }}</span>
    <button class="icon" type="button" aria-label="Закрыть" @click="close">✕</button>
  </header>
</template>

<style scoped>
.titlebar {
  --wails-draggable: drag;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 32px;
  padding: 0 4px 0 12px;
  background: var(--surface);
}

.title {
  font-weight: 600;
}

.icon {
  --wails-draggable: no-drag;
  width: 28px;
  height: 24px;
  background: transparent;
}
</style>
