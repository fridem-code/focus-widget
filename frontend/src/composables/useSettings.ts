import { Events } from '@wailsio/runtime'
import { shallowRef } from 'vue'
import { SettingsChanged } from '../../bindings/events'
import { Theme, type Settings } from '../../bindings/focuswidget/internal/config/models'
import { SettingsService } from '../../bindings/focuswidget/services'
import type { SettingsPatch } from '../../bindings/focuswidget/services/models'

/**
 * The current settings, kept in step with settings:changed (which also covers
 * changes made by the tray, the window and the music row).
 */
export function createSettingsStore(initial: Settings) {
  const settings = shallowRef(initial)
  const off = Events.On(SettingsChanged, (ev) => {
    settings.value = ev.data
  })
  return {
    settings,
    /**
     * Saves the changed fields; resolves to an error message if Go rejects
     * them. The new settings arrive as settings:changed, which Go sends in the
     * order the changes were made; replies to overlapping calls may not be.
     */
    async patch(p: SettingsPatch): Promise<string | null> {
      try {
        await SettingsService.Patch(p)
        return null
      } catch (err) {
        return err instanceof Error ? err.message : String(err)
      }
    },
    stop: off,
  }
}

export type SettingsStore = ReturnType<typeof createSettingsStore>

/** Pins the colour scheme (SPEC §3.8): "system" follows prefers-color-scheme. */
export function applyTheme(theme: Theme, root: HTMLElement = document.documentElement) {
  if (theme === Theme.ThemeDark || theme === Theme.ThemeLight) root.dataset.theme = theme
  else delete root.dataset.theme
}

/** Whether the page shows the dark theme: the pinned one, or the system's. */
export function themeIsDark(theme: Theme, systemDark: boolean): boolean {
  if (theme === Theme.ThemeDark) return true
  if (theme === Theme.ThemeLight) return false
  return systemDark
}

/**
 * Asks Go for the window glass tinted for the page's theme (T16) and marks the
 * page: only data-glass="on" makes --glass translucent, so a window left
 * without glass (transparency effects off in Windows) stays opaque.
 */
export async function syncGlass(
  dark: boolean,
  setGlass: (dark: boolean) => Promise<boolean>,
  root: HTMLElement = document.documentElement,
) {
  const on = await setGlass(dark).catch(() => false)
  if (on) root.dataset.glass = 'on'
  else delete root.dataset.glass
}
