import { onMounted, onUnmounted, type Ref } from 'vue'

/**
 * Closes a popover on a pointer press outside `inside`, on Escape, and when
 * the window loses focus (a click into the YouTube iframe never reaches our
 * document, but it does blur the window). `onEscape` lets the caller return
 * focus to the trigger.
 */
export function useDismiss(
  open: Ref<boolean>,
  inside: Readonly<Ref<HTMLElement | null | undefined>>[],
  onEscape?: () => void,
) {
  const onPointer = (e: PointerEvent) => {
    if (!open.value) return
    if (inside.some((r) => r.value?.contains(e.target as Node))) return
    open.value = false
  }
  const onKey = (e: KeyboardEvent) => {
    if (open.value && e.key === 'Escape') {
      open.value = false
      onEscape?.()
    }
  }
  const onBlur = () => {
    open.value = false
  }
  onMounted(() => {
    document.addEventListener('pointerdown', onPointer, true)
    document.addEventListener('keydown', onKey)
    window.addEventListener('blur', onBlur)
  })
  onUnmounted(() => {
    document.removeEventListener('pointerdown', onPointer, true)
    document.removeEventListener('keydown', onKey)
    window.removeEventListener('blur', onBlur)
  })
}
