<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { RunStatus } from '@pwnden/play';
import { UIButton, UILink, UIPanel, UISelect, UIWebFrame } from '@pwnden/ui';
import { webEndpoints } from './web-endpoints';

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
</script>

<template>
  <UIPanel title="문제 웹" headingID="web-heading" class="web-panel">
    <template #actions>
      <UIButton variant="ghost" size="compact" class="web-action" :disabled="!current" aria-label="문제 웹 새로고침" title="문제 웹 새로고침" @click="reload">
        <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M20 7v5h-5M4 17v-5h5m10.3-4a8 8 0 0 0-13.9-3M4.7 16a8 8 0 0 0 13.9 3" /></svg>
      </UIButton>
      <UILink v-if="current" :href="current.url" new-tab variant="ghost" size="compact" class="web-action" aria-label="문제 웹을 새 탭에서 열기" title="새 탭에서 열기"><span class="ui-sr-only">새 탭에서 열기</span></UILink>
    </template>
    <UISelect v-if="endpoints.length > 1" id="web-service" label="웹 서비스" v-model="selected" :options="endpoints.map(endpoint => ({ value: endpoint.url, label: endpoint.name }))" />
    <div v-if="current" class="web-viewport">
      <template v-for="endpoint in endpoints" :key="endpoint.url">
        <UIWebFrame v-if="opened.includes(endpoint.url)" :key="`${endpoint.url}:${revisions[endpoint.url] ?? 0}`" :hidden="endpoint.url !== selected" :src="endpoint.url" :title="`${endpoint.name} 문제 사이트`" />
      </template>
    </div>
    <p v-else-if="status?.state === 'unavailable'" role="alert">전원 버튼으로 환경을 종료한 뒤 터미널을 다시 연결하세요.</p>
    <p v-else>터미널을 연결하면 문제 웹이 준비됩니다.</p>
  </UIPanel>
</template>

<style scoped>
.web-panel { height: 100%; }
.web-panel :deep(.ui-panel-body) { display: flex; flex-direction: column; gap: var(--ui-space-2); }
.web-action { width: var(--ui-control-size-compact); padding: 0; }
.web-action svg { width: 1rem; height: 1rem; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
.web-viewport { flex: 1; min-width: 0; min-height: 0; border: 1px solid var(--ui-border); border-radius: var(--ui-radius-surface); overflow: hidden; }
.web-viewport :deep(.ui-web-frame) { border-radius: max(0px, calc(var(--ui-radius-surface) - 1px)); }
.web-viewport :deep(.ui-web-frame[hidden]) { display: none; }
p { color: var(--ui-muted); }
p[role='alert'] { color: var(--ui-danger); }
</style>
