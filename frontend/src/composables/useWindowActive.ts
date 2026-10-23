import { onMounted, onUnmounted, ref } from 'vue'

/**
 * Whether the widget window is the active one, for the dimmed "not in focus"
 * look (SPEC §3.4). Focus moving into the YouTube iframe blurs our window
 * object but keeps the document focused, so document.hasFocus() decides.
 */
export function useWindowActive() {
  const active = ref(document.hasFocus())
  const update = () => setTimeout(() => (active.value = document.hasFocus()))
  onMounted(() => {
    window.addEventListener('focus', update)
    window.addEventListener('blur', update)
  })
  onUnmounted(() => {
    window.removeEventListener('focus', update)
    window.removeEventListener('blur', update)
  })
  return active
}
