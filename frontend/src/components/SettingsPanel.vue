<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { Genre } from '../../bindings/focuswidget/internal/catalog/models'
import {
  BreakBehavior,
  Theme,
  type Settings,
} from '../../bindings/focuswidget/internal/config/models'
import type { SettingsPatch } from '../../bindings/focuswidget/services/models'
import { NUMBER_SPECS, plural } from '../lib/settingsForm'
import SettingNumber from './settings/SettingNumber.vue'
import SettingSegmented from './settings/SettingSegmented.vue'
import SettingSwitch from './settings/SettingSwitch.vue'

const props = defineProps<{
  settings: Settings
  genres: Genre[]
  genreId: string
  /** Saves the changed fields; resolves to an error message if rejected. */
  save: (patch: SettingsPatch) => Promise<string | null>
  /** Starting with Windows (T19): kept by Windows, not in the settings. */
  autostart: { enabled: () => Promise<boolean>; set: (on: boolean) => Promise<void> }
}>()
const emit = defineEmits<{
  back: []
  hide: []
  genre: [id: string]
  alwaysOnTop: [on: boolean]
  resetCycle: []
}>()

const saveError = ref<string | null>(null)

// Read every time the screen opens: Task Manager can switch it too. Hidden
// until known, so it never shows a wrong state.
const autostartOn = ref<boolean | null>(null)
onMounted(async () => {
  autostartOn.value = await props.autostart.enabled().catch(() => null)
})

async function setAutostart(on: boolean) {
  try {
    await props.autostart.set(on)
    autostartOn.value = on
    saveError.value = null
  } catch (err) {
    saveError.value = err instanceof Error ? err.message : String(err)
  }
}

async function set(patch: SettingsPatch) {
  saveError.value = await props.save(patch)
}

const breakOptions = [
  { value: BreakBehavior.BreakPause, label: 'Пауза' },
  { value: BreakBehavior.BreakDuck, label: 'Тише' },
  { value: BreakBehavior.BreakContinue, label: 'Не трогать' },
]
const themeOptions = [
  { value: Theme.ThemeSystem, label: 'Системная' },
  { value: Theme.ThemeDark, label: 'Тёмная' },
  { value: Theme.ThemeLight, label: 'Светлая' },
]
const sessions = (n: number) => plural(n, 'сессия', 'сессии', 'сессий')
</script>

<template>
  <div class="settings">
    <header class="header">
      <button class="icon-button" type="button" aria-label="Назад к таймеру" @click="emit('back')">
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
          <path d="M15 5l-7 7 7 7" />
        </svg>
      </button>
      <h1>Настройки</h1>
      <button class="icon-button" type="button" aria-label="Скрыть в трей" @click="emit('hide')">
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
    </header>

    <div class="scroll">
      <p v-if="saveError" class="save-error" role="alert">Не сохранилось: {{ saveError }}</p>

      <section aria-labelledby="s-timer">
        <h2 id="s-timer">Таймер</h2>
        <SettingNumber
          label="Фокус"
          unit="мин"
          :value="settings.timer.focusMin"
          :spec="NUMBER_SPECS.focusMin"
          @commit="set({ focusMin: $event })"
        />
        <SettingNumber
          label="Короткий перерыв"
          unit="мин"
          :value="settings.timer.shortBreakMin"
          :spec="NUMBER_SPECS.shortBreakMin"
          @commit="set({ shortBreakMin: $event })"
        />
        <SettingNumber
          label="Длинный перерыв"
          unit="мин"
          :value="settings.timer.longBreakMin"
          :spec="NUMBER_SPECS.longBreakMin"
          @commit="set({ longBreakMin: $event })"
        />
        <SettingNumber
          label="Длинный перерыв каждые"
          :unit="sessions(settings.timer.longBreakEvery)"
          :value="settings.timer.longBreakEvery"
          :spec="NUMBER_SPECS.longBreakEvery"
          @commit="set({ longBreakEvery: $event })"
        />
        <SettingSwitch
          label="Автостарт перерывов"
          :value="settings.timer.autoStartBreaks"
          @toggle="set({ autoStartBreaks: $event })"
        />
        <SettingSwitch
          label="Автостарт фокуса"
          :value="settings.timer.autoStartFocus"
          @toggle="set({ autoStartFocus: $event })"
        />
      </section>

      <section aria-labelledby="s-music">
        <h2 id="s-music">Музыка</h2>
        <SettingSegmented
          label="На перерыве"
          :value="settings.music.breakBehavior"
          :options="breakOptions"
          @select="set({ breakBehavior: $event })"
        />
        <SettingNumber
          v-if="settings.music.breakBehavior === BreakBehavior.BreakDuck"
          label="Громкость «Тише»"
          unit="%"
          :value="settings.music.duckVolume"
          :spec="NUMBER_SPECS.duckVolume"
          @commit="set({ duckVolume: $event })"
        />
        <div class="row">
          <label for="s-genre">Жанр</label>
          <select
            id="s-genre"
            :value="genreId"
            @change="emit('genre', ($event.target as HTMLSelectElement).value)"
          >
            <option v-for="g in genres" :key="g.id" :value="g.id">{{ g.name }}</option>
          </select>
        </div>
        <SettingSwitch
          label="Перемешивать плейлисты"
          :value="settings.music.shuffle"
          @toggle="set({ shuffle: $event })"
        />
      </section>

      <section aria-labelledby="s-notify">
        <h2 id="s-notify">Уведомления</h2>
        <SettingSwitch
          label="Звук в конце фазы"
          :value="settings.notify.sound"
          @toggle="set({ sound: $event })"
        />
        <SettingSwitch
          label="Уведомление Windows"
          :value="settings.notify.toast"
          @toggle="set({ toast: $event })"
        />
      </section>

      <section aria-labelledby="s-window">
        <h2 id="s-window">Окно</h2>
        <SettingSwitch
          label="Всегда поверх окон"
          :value="settings.window.alwaysOnTop"
          @toggle="emit('alwaysOnTop', $event)"
        />
        <SettingSwitch
          v-if="autostartOn !== null"
          label="Запускать вместе с Windows"
          :value="autostartOn"
          @toggle="setAutostart"
        />
        <SettingSegmented
          label="Тема"
          :value="settings.window.theme"
          :options="themeOptions"
          @select="set({ theme: $event })"
        />
      </section>

      <section aria-labelledby="s-cycle">
        <h2 id="s-cycle">Цикл</h2>
        <button class="reset" type="button" @click="emit('resetCycle')">
          Сбросить цикл · счётчик сессий в 0
        </button>
      </section>
    </div>
  </div>
</template>

<style scoped>
.settings {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.header {
  --wails-draggable: drag;
  height: 36px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 4px;
}

.header .icon-button {
  --wails-draggable: no-drag;
}

h1 {
  flex: 1 1 auto;
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}

.scroll {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: var(--track) transparent;
  padding: 2px 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

section {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

h2 {
  margin: 0 0 4px;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--text-muted);
}

.row {
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

label {
  color: var(--text-title);
}

select {
  width: 124px;
  height: 30px;
  padding: 0 8px;
  border: 1px solid var(--field-border);
  border-radius: var(--radius-s);
  background: var(--field-bg);
  color: var(--text);
  font: inherit;
}

select option {
  background: var(--popover);
  color: var(--text);
}

.reset {
  height: 36px;
  border: 1px solid var(--field-border);
  border-radius: var(--radius-m);
  color: var(--text-title);
  font-weight: 500;
}

.reset:hover {
  background: var(--surface-hover);
}

.save-error {
  margin: 0;
  padding: 8px 10px;
  border-radius: var(--radius-m);
  background: var(--field-bg);
  color: var(--danger);
  font-size: 12px;
}
</style>
