<script setup lang="ts">
import { Application } from '@wailsio/runtime'

// Settings (T12) and compact mode (T9) are not wired yet, so their buttons are
// disabled. T10 turns close into hide-to-tray; until then it quits.
function close() {
  void Application.Quit()
}
</script>

<template>
  <header class="titlebar">
    <span class="name"><span class="dot" aria-hidden="true"></span>Focus</span>
    <div class="actions">
      <button class="icon-button" type="button" aria-label="Настройки" disabled>
        <svg
          aria-hidden="true"
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
        >
          <path d="M4 6h10M18 6h2M4 12h4M12 12h8M4 18h12" />
          <circle cx="16" cy="6" r="2" />
          <circle cx="10" cy="12" r="2" />
          <circle cx="18" cy="18" r="2" />
        </svg>
      </button>
      <button class="icon-button" type="button" aria-label="Свернуть в компактный режим" disabled>
        <svg
          aria-hidden="true"
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M4 14h6v6M20 10h-6V4M14 10l7-7M3 21l7-7" />
        </svg>
      </button>
      <button class="icon-button" type="button" aria-label="Закрыть" @click="close">
        <svg
          aria-hidden="true"
          width="18"
          height="18"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.75"
          stroke-linecap="round"
        >
          <path d="M6 6l12 12M18 6L6 18" />
        </svg>
      </button>
    </div>
  </header>
</template>

<style scoped>
.titlebar {
  --wails-draggable: drag;
  height: 36px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 2px 0 6px;
}

.name {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: var(--text-title);
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent);
}

.actions {
  --wails-draggable: no-drag;
  display: flex;
  gap: 2px;
}
</style>
