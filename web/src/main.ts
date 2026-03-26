import './app.css'
import App from './App.svelte'
import { mount } from 'svelte'

const savedTheme = localStorage.getItem('tracelog-theme') || 'dark';
document.documentElement.setAttribute('data-theme', savedTheme);

const app = mount(App, { target: document.getElementById('app')! })

export default app
