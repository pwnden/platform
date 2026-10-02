<script setup lang="ts">
import { computed, onBeforeUnmount, ref, shallowReactive, watch } from 'vue';
import type { BrowserSession, Player, RunStatus } from '@pwnden/play';
import { UIForm, UIIconButton, UISelect, UITextField, UIWebFrame } from '@pwnden/ui';
import type { UIWebFrameHandle, UIWebNavigation } from '@pwnden/ui';
import { webEndpoints } from './web-endpoints';
import type { ProblemWebHandle } from './props';

const props = defineProps<{ player: Player; slug: string; status?: RunStatus | undefined; active: boolean }>();
const endpoints = computed(() => props.status?.state === 'running' ? webEndpoints(props.status.endpoints) : []);
const selected = ref('');
const address = ref('');
const sessions = shallowReactive<Record<string, BrowserSession>>({});
const frames = shallowReactive<Record<string, UIWebFrameHandle>>({});
const states = shallowReactive<Record<string, UIWebNavigation>>({});
const pending = shallowReactive<Record<string, boolean>>({});
const errors = shallowReactive<Record<string, string>>({});
const current = computed(() => endpoints.value.find(endpoint => endpoint.url === selected.value));
const state = computed(() => states[selected.value]);
let alive = true;
onBeforeUnmount(() => { alive = false; });
watch(endpoints, items => {
  if (!items.some(item => item.url === selected.value)) selected.value = items[0]?.url ?? '';
  for (const url of Object.keys(sessions)) if (!items.some(item => item.url === url)) { delete sessions[url]; delete states[url]; delete frames[url]; }
}, { immediate: true });
async function prepare() {
  const endpoint = current.value;
  if (!endpoint || sessions[endpoint.url] || pending[endpoint.url]) return;
  const slug = props.slug;
  pending[endpoint.url] = true;
  delete errors[endpoint.url];
  try {
    const session = await props.player.browser(slug, endpoint.name);
    if (!alive || props.slug !== slug || !endpoints.value.some(item => item.url === endpoint.url)) return;
    if (session.target !== new URL(endpoint.url).origin) throw new Error('endpoint changed');
    sessions[endpoint.url] = session;
  } catch {
    if (alive && props.slug === slug) errors[endpoint.url] = '문제 웹을 열지 못했습니다. 새로고침으로 다시 시도하세요.';
  } finally { if (alive) delete pending[endpoint.url]; }
}
watch([() => props.active, selected, endpoints], () => { if (props.active) void prepare(); }, { immediate: true });
function pagePath(url: URL) { return url.pathname + url.search + url.hash; }
watch(() => state.value?.url ?? current.value?.url ?? '', url => { address.value = url ? pagePath(new URL(url)) : ''; }, { immediate: true });
function navigate() {
  const page = state.value?.url ?? current.value?.url;
  if (!page) return;
  const previous = new URL(page);
  const input = address.value.trim();
  address.value = pagePath(previous);
  if (!input) return;
  try {
    const url = new URL(input, previous);
    if (url.origin !== new URL(current.value?.url ?? '').origin || url.username || url.password) throw new Error('outside target');
    delete errors[selected.value];
    address.value = pagePath(url);
    frames[selected.value]?.navigate(url.href);
  } catch { /* Keep the current page and restore its path when input cannot be used. */ }
}
function reload() {
  delete errors[selected.value];
  if (frames[selected.value]) frames[selected.value]?.reload();
  else void prepare();
}
const handle: ProblemWebHandle = {
  get url() { return state.value?.url ?? current.value?.url ?? ''; },
  get busy() { return !!pending[selected.value] || !!state.value?.busy; },
  get canBack() { return !!state.value?.canBack; },
  get canForward() { return !!state.value?.canForward; },
  get canReload() { return !!current.value; },
  back: () => frames[selected.value]?.back(), forward: () => frames[selected.value]?.forward(), reload,
};
defineExpose(handle);
</script>

<template>
  <section class="web-panel" aria-label="문제 웹">
    <UISelect v-if="endpoints.length > 1" id="web-service" label="웹 서비스" v-model="selected" :options="endpoints.map(endpoint => ({ value: endpoint.url, label: endpoint.name }))" />
    <UIForm class="web-address-bar" :submit="navigate">
      <div class="web-navigation" role="group" aria-label="웹 탐색">
        <UIIconButton label="뒤로 가기" icon="back" :disabled="!handle.canBack || handle.busy" @click="handle.back()" />
        <UIIconButton label="앞으로 가기" icon="forward" :disabled="!handle.canForward || handle.busy" @click="handle.forward()" />
        <UIIconButton label="문제 웹 새로고침" icon="refresh" :busy="handle.busy" :disabled="!handle.canReload" @click="reload" />
      </div>
      <UITextField id="web-address" label="문제 웹 경로" label-hidden size="compact" v-model="address" :disabled="!sessions[selected]" />
    </UIForm>
    <div v-if="current" class="web-viewport" :aria-busy="!!pending[selected]">
      <p v-if="errors[selected] || state?.error" class="web-notice" role="alert">{{ errors[selected] || state?.error }}</p>
      <template v-for="endpoint in endpoints" :key="endpoint.url">
        <UIWebFrame v-if="sessions[endpoint.url]" :ref="value => { if (value) frames[endpoint.url] = value as unknown as UIWebFrameHandle; else delete frames[endpoint.url]; }" :hidden="endpoint.url !== selected" :src="sessions[endpoint.url]!.url" :target="sessions[endpoint.url]!.target" :title="`${endpoint.name} 문제 사이트`" @navigation="states[endpoint.url] = $event" />
      </template>
      <p v-if="pending[selected]" class="web-loading" role="status">문제 웹을 여는 중…</p>
    </div>
    <div v-else class="web-viewport"><p class="web-loading" role="status">{{ status?.state === 'unavailable' ? '문제 웹을 준비하지 못했습니다. 환경 상태를 확인하세요.' : '문제 웹 준비 중…' }}</p></div>
  </section>
</template>

<style scoped>
.web-panel { box-sizing: border-box; height: 100%; min-height: 0; display: flex; flex-direction: column; gap: var(--ui-space-2); padding: var(--ui-space-2); }
.web-address-bar { flex: none; min-width: 0; display: flex; align-items: center; gap: var(--ui-space-1); }
.web-navigation { flex: none; display: flex; align-items: center; gap: var(--ui-space-1); }
.web-address-bar > :deep(.ui-field) { flex: 1; min-width: 0; }
.web-viewport { position: relative; flex: 1; min-width: 0; min-height: 0; border: 1px solid var(--ui-border); border-radius: var(--ui-radius-surface); overflow: hidden; }
.web-viewport :deep(.ui-web-frame) { border-radius: max(0px, calc(var(--ui-radius-surface) - 1px)); }
.web-viewport :deep(.ui-web-frame[hidden]) { display: none; }
p { color: var(--ui-muted); }
.web-loading { position: absolute; inset: 0; display: grid; place-content: center; padding: var(--ui-space-2); background: var(--ui-surface); }
.web-notice { position: absolute; z-index: 1; inset-inline: var(--ui-space-2); top: var(--ui-space-2); padding: var(--ui-space-2); border: 1px solid var(--ui-danger); border-radius: var(--ui-radius-control); background: var(--ui-surface-raised); }
p[role='alert'] { color: var(--ui-danger); }
</style>
