import './style.css'
import { mount } from 'svelte'
import App from './App.svelte'
import { initTheme } from './lib/theme.js'

// Initialize light/dark theme tracking
initTheme()

const target = document.getElementById('app') || document.body
const app = mount(App, { target })

export default app
