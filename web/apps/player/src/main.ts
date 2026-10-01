import { createApp, h, shallowRef, ref } from 'vue';
import { createAPI } from '@pwnden/api';
import type { APIClient } from '@pwnden/api';
import '@pwnden/ui/theme.css';
import App from './App.vue';
import { takeSessionToken, forgetSessionToken } from './session';

const token = takeSessionToken();
const client = shallowRef<APIClient>();
const sessionRejected = ref(false);
if (token) client.value = createAPI({ token, onUnauthorized() {
  forgetSessionToken(token);
  sessionRejected.value = true;
  client.value = undefined;
} });
createApp({ setup: () => () => h(App, { client: client.value, sessionRejected: sessionRejected.value }) }).mount('#app');
