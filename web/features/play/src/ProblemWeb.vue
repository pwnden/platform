<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { RunStatus } from '@pwnden/play';
import { UISelect, UIWebFrame } from '@pwnden/ui';
import { webEndpoints } from './web-endpoints';
import type { ProblemWebHandle } from './props';

const props = defineProps<{ status?: RunStatus | undefined; active: boolean }>();
const endpoints = computed(() => props.status?.state === 'running' ? webEndpoints(props.status.endpoints) : []);
const selected = ref('');
const opened = ref<readonly string[]>([]);
const revisions = ref<Record<string, number>>({});
const current = computed(() => endpoints.value.find(endpoint => endpoint.url === selected.value));
watch(endpoints, items => {
  if (!items.some(item => item.url === selected.value)) selected.value = items[0]?.url ?? '';
  opened.value = opened.value.filter(url => items.some(item => item.url === url));
}, { immediate: true });
watch([() => props.active, selected], () => {
  if (props.active && selected.value && !opened.value.includes(selected.value)) opened.value = [...opened.value, selected.value];
}, { immediate: true });
function reload() {
  if (selected.value) revisions.value[selected.value] = (revisions.value[selected.value] ?? 0) + 1;
}
const handle: ProblemWebHandle = { get url() { return current.value?.url ?? ''; }, reload };
defineExpose(handle);
</script>

<template>
  <section class="web-panel" aria-label="문제 웹">
    <UISelect v-if="endpoints.length > 1" id="web-service" label="웹 서비스" v-model="selected" :options="endpoints.map(endpoint => ({ value: endpoint.url, label: endpoint.name }))" />
    <div v-if="current" class="web-viewport">
      <template v-for="endpoint in endpoints" :key="endpoint.url">
        <UIWebFrame v-if="opened.includes(endpoint.url)" :key="`${endpoint.url}:${revisions[endpoint.url] ?? 0}`" :hidden="endpoint.url !== selected" :src="endpoint.url" :title="`${endpoint.name} 문제 사이트`" />
      </template>
    </div>
    <p v-else-if="status?.state === 'unavailable'" role="alert">실행 환경에 문제가 있습니다. 문제에서 나간 뒤 10분 후 다시 열면 환경을 새로 준비합니다.</p>
    <p v-else>터미널을 연결하면 문제 웹이 준비됩니다.</p>
  </section>
</template>

<style scoped>
.web-panel { box-sizing: border-box; height: 100%; min-height: 0; display: flex; flex-direction: column; gap: var(--ui-space-2); padding: var(--ui-space-2); }
.web-viewport { flex: 1; min-width: 0; min-height: 0; border: 1px solid var(--ui-border); border-radius: var(--ui-radius-surface); overflow: hidden; }
.web-viewport :deep(.ui-web-frame) { border-radius: max(0px, calc(var(--ui-radius-surface) - 1px)); }
.web-viewport :deep(.ui-web-frame[hidden]) { display: none; }
p { color: var(--ui-muted); }
p[role='alert'] { color: var(--ui-danger); }
</style>
