import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import wails from '@wailsio/runtime/plugins/vite'

// Leaves dist/.mode-<mode> behind, so build:frontend (build/Taskfile.yml) can
// tell a dev bundle from a release one: its sources checksum ignores DEV, and
// a release exe once shipped the dev bundle with its debug handle.
function modeMarker(mode: string): Plugin {
  return {
    name: 'mode-marker',
    apply: 'build',
    generateBundle() {
      this.emitFile({ type: 'asset', fileName: `.mode-${mode}`, source: '' })
    },
  }
}

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  server: {
    host: '127.0.0.1',
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  plugins: [vue(), wails('./bindings'), modeMarker(mode)],
}))
