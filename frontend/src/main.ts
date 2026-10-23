import '@fontsource-variable/inter'
import { createApp } from 'vue'
import App from './App.vue'
import './styles.css'

const dev = import.meta.env.MODE !== 'production'

if (dev && new URLSearchParams(location.search).has('catalog-check')) {
  // Dev-only page that plays every catalog source (scripts/catalog-check).
  void import('./dev/catalogCheck').then((m) => m.mountCatalogCheck())
} else {
  createApp(App).mount('#app')
  if (dev) void import('./debug').then((m) => m.installDebugHandle())
}
