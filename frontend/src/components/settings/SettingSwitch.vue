<script setup lang="ts">
import { useId } from 'vue'

defineProps<{ label: string; value: boolean }>()
const emit = defineEmits<{ toggle: [value: boolean] }>()
const id = useId()
</script>

<template>
  <div class="row">
    <span :id="id">{{ label }}</span>
    <button
      class="switch"
      type="button"
      role="switch"
      :aria-checked="value"
      :aria-labelledby="id"
      @click="emit('toggle', !value)"
    >
      <span class="knob" aria-hidden="true"></span>
    </button>
  </div>
</template>

<style scoped>
.row {
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--text-title);
}

.switch {
  position: relative;
  width: 40px;
  height: 22px;
  border-radius: 11px;
  background: var(--track);
  transition: background 0.15s;
}

.switch[aria-checked='true'] {
  background: var(--accent);
}

.knob {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: #fff;
  transition: left 0.15s;
}

.switch[aria-checked='true'] .knob {
  left: 21px;
}
</style>
