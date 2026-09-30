import { createApp } from 'vue';
import { createAPI } from '@pwnden/api';
import '@pwnden/ui/theme.css';
import App from './App.vue';
import { takeSessionToken } from './session';

const token = takeSessionToken();
createApp(App, { client: token ? createAPI({ token }) : undefined }).mount('#app');
