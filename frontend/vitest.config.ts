import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vitest/config'

// Separate from vite.config.ts so tests don't load the Wails bindings plugin.
export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'happy-dom',
    include: ['src/**/*.test.ts'],
  },
})
