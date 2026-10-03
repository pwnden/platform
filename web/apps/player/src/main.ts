import { createApp, h, shallowRef, ref } from 'vue';
import { createAPI } from '@pwnden/api';
import type { APIClient } from '@pwnden/api';
import '@pwnden/ui/theme.css';
import App from './App.vue';
import { takeSessionToken, forgetSessionToken } from './session';
import { createPlayerRouter } from './router';
import { handleHotUpdate } from 'vue-router/auto-routes';

const token = takeSessionToken();
const client = shallowRef<APIClient>();
const sessionRejected = ref(false);
if (token) client.value = createAPI({ token, onUnauthorized() {
  forgetSessionToken(token);
  sessionRejected.value = true;
  client.value = undefined;
} });
const router = createPlayerRouter();
if (import.meta.hot) handleHotUpdate(router);
createApp({ setup: () => () => h(App, { client: client.value, sessionRejected: sessionRejected.value }) }).use(router).mount('#app');
