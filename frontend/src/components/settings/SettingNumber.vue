<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'
import { parseNumber, type NumberSpec } from '../../lib/settingsForm'

const props = defineProps<{ label: string; value: number; spec: NumberSpec; unit: string }>()
const emit = defineEmits<{ commit: [value: number] }>()

const id = useId()
const text = ref(String(props.value))
const error = ref<string | null>(null)
const focused = ref(false)

// Follow changes made elsewhere (tray, another field) unless the user is typing here.
watch(
  () => props.value,
  (v) => {
    if (!focused.value) {
      text.value = String(v)
      error.value = null
    }
  },
)

const invalid = computed(() => error.value !== null)

function onInput() {
  const parsed = parseNumber(text.value, props.spec)
  error.value = parsed.ok ? null : parsed.error
}

// Saved on change (Enter or leaving the field), never while invalid.
function onChange() {
  const parsed = parseNumber(text.value, props.spec)
  if (!parsed.ok) {
    error.value = parsed.error
    return
  }
  error.value = null
  if (parsed.value !== props.value) emit('commit', parsed.value)
}
</script>

<template>
  <div class="field">
    <div class="row">
      <label :for="id">{{ label }}</label>
      <div class="input">
        <!-- type="text": a number input would hand Vue a number (or "" for junk)
             and hide what the user actually typed from the validation. -->
        <input
          :id="id"
          v-model="text"
          type="text"
          inputmode="numeric"
          :aria-invalid="invalid"
          :aria-describedby="invalid ? `${id}-error` : undefined"
          @input="onInput"
          @change="onChange"
          @focus="focused = true"
          @blur="focused = false"
        />
        <span class="unit">{{ unit }}</span>
      </div>
    </div>
    <div v-if="invalid" :id="`${id}-error`" class="error" role="alert">{{ error }}</div>
  </div>
</template>

<style scoped>
.row {
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

label {
  color: var(--text-title);
}

.input {
  display: flex;
  align-items: center;
  gap: 6px;
}

input {
  width: 54px;
  height: 30px;
  border: 1px solid var(--field-border);
  border-radius: var(--radius-s);
  background: var(--field-bg);
  color: var(--text);
  font: inherit;
  text-align: center;
  font-variant-numeric: tabular-nums;
  user-select: text;
}

input[aria-invalid='true'] {
  border-color: var(--danger);
}

input:focus-visible {
  outline: 2px solid var(--accent-text);
  outline-offset: 1px;
}

.unit {
  width: 52px;
  font-size: 12px;
  color: var(--text-muted);
}

.error {
  margin: -2px 0 4px;
  text-align: right;
  font-size: 11px;
  color: var(--danger);
}
</style>
