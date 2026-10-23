import { createApp } from 'vue'
import App from './App.vue'
import './styles.css'

createApp(App).mount('#app')

if (import.meta.env.MODE !== 'production') {
  void import('./debug').then((m) => m.installDebugHandle())
}
