<!-- Stands where the YouTube iframe will be until T6 creates it on the first
     Start (SPEC §3.4); nothing of ours is ever drawn over the real iframe. -->
<template>
  <div class="placeholder">
    <svg
      aria-hidden="true"
      width="40"
      height="40"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="1.5"
      stroke-linejoin="round"
    >
      <rect x="2.5" y="5" width="19" height="14" rx="3" />
      <path d="M10 9.5l5 2.5-5 2.5z" fill="currentColor" />
    </svg>
    Видео запустится вместе с фокусом
  </div>
</template>

<style scoped>
.placeholder {
  width: 356px;
  height: 204px;
  flex-shrink: 0;
  border-radius: var(--radius-l);
  background: var(--player-bg);
  color: var(--player-placeholder);
  font-size: 12px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
}
</style>
