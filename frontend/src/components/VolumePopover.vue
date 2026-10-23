<script setup lang="ts">
import { onMounted, useTemplateRef } from 'vue'

defineProps<{ volume: number; muted: boolean }>()
const emit = defineEmits<{ volume: [value: number]; muted: [value: boolean] }>()

const slider = useTemplateRef<HTMLInputElement>('slider')
onMounted(() => slider.value?.focus())

function onInput(e: Event) {
  emit('volume', Number((e.target as HTMLInputElement).value))
}
</script>

<template>
  <div class="popover">
    <button
      class="icon-button"
      type="button"
      :aria-label="muted ? 'Включить звук' : 'Выключить звук'"
      :aria-pressed="muted"
      @click="emit('muted', !muted)"
    >
      <svg
        aria-hidden="true"
        width="17"
        height="17"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.75"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path d="M11 5L6 9H3v6h3l5 4z" />
        <template v-if="muted || volume === 0">
          <path d="M16 9l5 6M21 9l-5 6" />
        </template>
        <template v-else>
          <path d="M15.5 8.5a5 5 0 0 1 0 7" />
          <path d="M18.5 5.5a9 9 0 0 1 0 13" />
        </template>
      </svg>
    </button>
    <input
      ref="slider"
      class="slider"
      type="range"
      min="0"
      max="100"
      :value="volume"
      :style="{ '--fill': `${volume}%` }"
      aria-label="Громкость"
      @input="onInput"
    />
    <span class="value" aria-hidden="true">{{ volume }}</span>
  </div>
</template>

<style scoped>
/* Above the music row and below the player (SPEC §2). */
.popover {
  position: absolute;
  right: 12px;
  top: 346px;
  width: 240px;
  height: 52px;
  padding: 0 14px 0 6px;
  border-radius: 12px;
  background: var(--popover);
  border: 1px solid var(--glass-border);
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  gap: 8px;
  z-index: 10;
}

.slider {
  flex: 1 1 auto;
  min-width: 0;
  height: 20px;
  margin: 0;
  appearance: none;
  background: transparent;
  cursor: pointer;
}

.slider::-webkit-slider-runnable-track {
  height: 4px;
  border-radius: 2px;
  background: linear-gradient(to right, var(--accent-text) var(--fill), var(--track) var(--fill));
}

.slider::-webkit-slider-thumb {
  appearance: none;
  width: 14px;
  height: 14px;
  margin-top: -5px;
  border-radius: 50%;
  background: var(--accent);
  border: 2px solid #fff;
}

.slider:focus-visible {
  outline: none;
}

.slider:focus-visible::-webkit-slider-thumb {
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.value {
  width: 26px;
  text-align: right;
  font-size: 12px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--text-title);
}
</style>
