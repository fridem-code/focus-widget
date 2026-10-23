<script setup lang="ts" generic="T extends string">
import { useId } from 'vue'

defineProps<{ label: string; value: T; options: { value: T; label: string }[] }>()
const emit = defineEmits<{ select: [value: T] }>()
const id = useId()
</script>

<template>
  <div class="field">
    <div :id="id" class="label">{{ label }}</div>
    <div class="segmented" role="radiogroup" :aria-labelledby="id">
      <button
        v-for="o in options"
        :key="o.value"
        type="button"
        role="radio"
        :aria-checked="o.value === value"
        @click="o.value !== value && emit('select', o.value)"
      >
        {{ o.label }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 4px 0;
}

.label {
  color: var(--text-title);
}

.segmented {
  display: flex;
  gap: 2px;
  padding: 3px;
  border-radius: var(--radius-m);
  background: var(--field-bg);
}

button {
  flex: 1 1 0;
  height: 28px;
  border-radius: var(--radius-s);
  font-size: 12px;
  font-weight: 500;
  color: var(--icon);
}

button[aria-checked='true'] {
  background: var(--segment-active);
  color: var(--text);
  font-weight: 600;
}
</style>
