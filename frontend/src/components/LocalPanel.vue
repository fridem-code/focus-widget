<script setup lang="ts">
defineProps<{ title: string; note: string; playing: boolean }>()

// Bar heights of the level meter, as fractions of its height.
const bars = [0.45, 0.8, 0.6, 1, 0.7, 0.5, 0.85, 0.4, 0.65]
</script>

<!-- Stands where the YouTube player would be while music plays in the page:
     generated noise (T17) or the user's files (T22). There is no video, so it
     says what plays and that it needs no network. -->
<template>
  <div class="noise" :class="{ playing }">
    <div class="meter" aria-hidden="true">
      <span v-for="(h, i) in bars" :key="i" :style="{ '--h': h }"></span>
    </div>
    <div class="title">{{ title }}</div>
    <div class="note">{{ note }}</div>
  </div>
</template>

<style scoped>
.noise {
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
  gap: 8px;
}

.meter {
  height: 44px;
  display: flex;
  align-items: center;
  gap: 5px;
  margin-bottom: 4px;
}

.meter span {
  width: 5px;
  height: calc(var(--h) * 100%);
  border-radius: 3px;
  background: var(--accent);
  opacity: 0.45;
  transform: scaleY(0.35);
  transition:
    transform 0.4s,
    opacity 0.4s;
}

/* Still on purpose: a looping animation made the glass window recompose every
   frame, about 2.5 % CPU against 0.1 % for the noise itself (T17). */
.playing .meter span {
  opacity: 0.9;
  transform: scaleY(1);
}

.title {
  font-size: 14px;
  font-weight: 600;
  color: #e6e8ee;
}
</style>
