<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import type { Player, RunStatus } from '@pwnden/play';
import { UIButton, UILink, UITextField, UIPanel, UIStatus } from '@pwnden/ui';
import type { PlayPanelHandle } from './props';

const props = defineProps<{ player: Player; slug: string; kind: RunStatus['kind'] }>();
const emit = defineEmits<{ busy: [value: boolean]; status: [value: RunStatus | undefined]; stopped: []; started: [] }>();
const pending = ref(false);
const status = ref<RunStatus>();
const flag = ref('');
const message = ref('');
const failed = ref(false);
let active = true;
onUnmounted(() => { active = false; });

async function perform(operation: 'run' | 'stop' | 'submit' | 'status') {
  if (pending.value) return;
  pending.value = true;
  emit('busy', true);
  message.value = '';
  failed.value = false;
  try {
    if (operation === 'run') {
      await props.player.run(props.slug);
      if (active) {
        message.value = '문제가 준비되었습니다.';
        emit('started');
      }
    } else if (operation === 'stop') {
      await props.player.stop(props.slug);
      if (active) { message.value = '문제를 중지했습니다.'; emit('stopped'); }
    } else if (operation === 'submit') {
      const result = await props.player.submit(props.slug, flag.value);
      if (active) { flag.value = ''; message.value = result.accepted ? '정답입니다.' : '정답이 아닙니다. 다시 시도하세요.'; }
    }
    const result = await props.player.status(props.slug);
    if (active) status.value = result;
  } catch {
    // Refresh after a failed mutation: an already-running conflict must recover
    // the existing endpoints, and a lost response must not invent local state.
    if (operation !== 'status') {
      try {
        const result = await props.player.status(props.slug);
        if (active) status.value = result;
      } catch { if (active) status.value = undefined; }
    } else if (active) status.value = undefined;
    if (active) { failed.value = true; message.value = '요청을 완료하지 못했습니다. 서버 연결을 확인하고 실행 상태를 새로고침하세요.'; }
  } finally {
    if (active) { pending.value = false; emit('busy', false); emit('status', status.value); }
  }
  return status.value;
}
onMounted(() => perform('status'));
const handle: PlayPanelHandle = {
  async refresh() { await perform('status'); },
};
defineExpose(handle);
</script>

<template>
  <UIPanel title="실행 및 제출" headingID="play-heading" :heading-level="3" :aria-busy="pending" class="play-panel">
    <template #actions>
      <UIStatus :tone="status?.state === 'running' || status?.state === 'ready' ? 'info' : status?.state === 'unavailable' ? 'danger' : 'muted'">{{ status?.state === 'running' ? '실행 중' : status?.state === 'ready' ? '준비됨' : status?.state === 'stopped' ? '중지됨' : status?.state === 'unavailable' ? '확인 필요' : '상태 확인' }}</UIStatus>
      <UIButton variant="ghost" size="compact" class="refresh" :busy="pending" aria-label="실행 상태 새로고침" title="실행 상태 새로고침" @click="perform('status')">
        <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M20 7v5h-5M4 17v-5h5m10.3-4a8 8 0 0 0-13.9-3M4.7 16a8 8 0 0 0 13.9 3" /></svg>
      </UIButton>
    </template>
    <p v-if="!status && !pending">실행 상태를 확인하지 못했습니다.</p>
    <p v-else-if="status?.state === 'stopped'">문제를 실행하면 접속할 수 있습니다.</p>
    <p v-else-if="status?.state === 'unavailable'" role="alert">서비스가 정상 실행 중이지 않습니다. 문제를 중지해 정리한 뒤 다시 실행하세요.</p>
    <div v-if="kind === 'service'" class="actions">
      <template v-for="endpoint in status?.endpoints" :key="endpoint.name">
        <UILink v-if="endpoint.url.startsWith('http://')" :href="endpoint.url" new-tab variant="primary" :aria-label="`${status?.endpoints.length === 1 ? '문제 열기' : endpoint.name + ' 열기'} (새 탭)`" :title="endpoint.url">{{ status?.endpoints.length === 1 ? '문제 열기' : endpoint.name }}</UILink>
        <code v-else class="endpoint-address">{{ endpoint.url }}</code>
      </template>
      <UIButton :variant="status?.state === 'stopped' ? 'secondary' : 'danger'" :disabled="pending" @click="perform(status?.state === 'stopped' ? 'run' : 'stop')">{{ status?.state === 'stopped' ? '문제 실행' : '문제 중지' }}</UIButton>
    </div>
    <p v-if="pending" role="status">요청을 처리하는 중…</p>
    <p v-if="message" :role="failed ? 'alert' : 'status'">{{ message }}</p>
    <form @submit.prevent="perform('submit')">
      <UITextField id="flag" v-model="flag" label="플래그 제출" placeholder="찾은 플래그를 입력하세요" :disabled="pending" required />
      <UIButton variant="primary" type="submit" :disabled="pending || !flag.trim() || !status || (status.state !== 'ready' && status.state !== 'running')">정답 확인</UIButton>
    </form>
  </UIPanel>
</template>

<style scoped>
.actions { display: flex; flex-wrap: wrap; align-items: center; gap: var(--ui-space-1); }
.play-panel { --ui-panel-inset: var(--ui-space-2); container-type: inline-size; border-bottom: 1px solid var(--ui-border); }
.play-panel :deep(.ui-panel-heading) { background: var(--ui-surface-raised); }
.play-panel :deep(.ui-panel-body) { display: flex; flex-direction: column; gap: var(--ui-space-2); }
p { color: var(--ui-muted); font-size: 0.9rem; }
p[role='alert'] { color: var(--ui-danger); }
form { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: end; gap: var(--ui-space-1); }
form:not(:first-child)::before { content: ''; grid-column: 1 / -1; border-top: 1px solid var(--ui-border); }
.ui-field { width: 100%; }
.endpoint-address { min-width: 0; overflow-wrap: anywhere; white-space: normal; }
.refresh { width: var(--ui-control-size-compact); padding: 0; }
.refresh svg { width: 1rem; height: 1rem; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
@container (max-width: 24rem) { form { grid-template-columns: minmax(0, 1fr); } form > .ui-button { width: 100%; } }
</style>
