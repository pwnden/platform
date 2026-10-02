<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import type { Player, RunStatus, Workspaces, WorkspaceConnection, WorkspaceInfo } from '@pwnden/play';
import { UIButton, UITextField, UIIconButton } from '@pwnden/ui';

const props = defineProps<{ player: Player; workspaces: Workspaces; slug: string; kind: RunStatus['kind'] }>();
const emit = defineEmits<{ busy: [value: boolean]; status: [value: RunStatus | undefined] }>();
const pending = ref(false);
const preparing = ref(false);
const status = ref<RunStatus>();
const flag = ref('');
const result = ref<'idle' | 'accepted' | 'rejected' | 'error'>('idle');
const message = ref('');
const environmentError = ref('');
const retained = ref<readonly WorkspaceInfo[]>([]);
const ending = ref(false);
let active = true;
let connection: WorkspaceConnection | undefined;
let generation = 0;
const visible = () => typeof document === 'undefined' || document.visibilityState !== 'hidden';
const accepted = computed(() => result.value === 'accepted');
const ready = computed(() => !preparing.value && !!status.value && (status.value.state === 'running' || status.value.state === 'ready'));
const feedback = computed(() => pending.value ? '확인 중…' : message.value || environmentError.value || (preparing.value ? '풀이 환경 준비 중…' : ''));
function disconnect() { generation++; connection?.close(); connection = undefined; }
async function prepare() {
  if (!active || !visible() || connection) return;
  const current = ++generation;
  preparing.value = true; environmentError.value = ''; retained.value = [];
  try {
    connection = props.workspaces.connect(props.slug, code => {
      if (!active || current !== generation) return;
      status.value = undefined; emit('status', undefined);
      const reasons: Record<string, string> = {
        workspace_full: '환경 10개가 유지 중입니다. 환경 하나를 정리한 뒤 다시 준비하세요.',
        unauthorized: '서버가 출력한 전체 주소로 다시 접속하세요.',
        network_error: '서버 연결을 확인한 뒤 다시 연결하세요.',
        not_running: '풀이 환경이 종료되었습니다. 다시 연결하세요.',
      };
      environmentError.value = reasons[code] ?? '환경 준비에 실패했습니다. 다시 연결하세요.';
      if (code === 'workspace_full') void loadRetained(current);
    });
    const observed = await connection.ready;
    if (active && current === generation) { status.value = observed; emit('status', observed); }
  } catch { if (active && current === generation && !environmentError.value) environmentError.value = '환경을 준비하지 못했습니다. 다시 연결하세요.'; }
  finally { if (active && current === generation) preparing.value = false; }
}
async function loadRetained(current: number) {
  try { const items = await props.workspaces.list(); if (active && current === generation) retained.value = items; }
  catch { if (active && current === generation) environmentError.value = '유지 환경 목록을 불러오지 못했습니다. 다시 연결하세요.'; }
}
async function endEnvironment(slug: string) {
  if (ending.value) return;
  ending.value = true;
  try { await props.workspaces.stop(slug); if (active) { disconnect(); await prepare(); } }
  catch { if (active) environmentError.value = '환경을 정리하지 못했습니다. 다시 시도하세요.'; }
  finally { if (active) ending.value = false; }
}
async function retry() {
  if (preparing.value || pending.value) return;
  disconnect(); await prepare();
}
async function submit() {
  if (pending.value || accepted.value || !ready.value || !flag.value.trim()) return;
  pending.value = true; emit('busy', true); message.value = '';
  try {
    const answer = await props.player.submit(props.slug, flag.value);
    if (active) { result.value = answer.accepted ? 'accepted' : 'rejected'; message.value = answer.accepted ? '해결 완료' : '플래그를 다시 확인하세요.'; }
  } catch { if (active) { result.value = 'error'; message.value = '제출하지 못했습니다. 다시 시도하세요.'; } }
  finally { if (active) { pending.value = false; emit('busy', false); } }
}
function visibility() { if (visible()) void prepare(); else disconnect(); }
onMounted(() => { if (typeof document !== 'undefined') document.addEventListener('visibilitychange', visibility); void prepare(); });
onUnmounted(() => { active = false; if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', visibility); disconnect(); });
</script>

<template>
  <section class="submission-bar" :class="{ 'submission-bar--accepted': accepted }" aria-label="플래그 제출" :aria-busy="pending || preparing">
    <div class="submission-caption">
      <label for="flag">플래그</label>
      <p id="submission-feedback" class="submission-feedback" :class="{ 'submission-feedback--error': result === 'rejected' || result === 'error' || !!environmentError }" role="status" aria-live="polite" :title="feedback">{{ feedback }}</p>
      <div class="environment-retry"><UIIconButton v-if="environmentError" label="풀이 환경 다시 연결" icon="refresh" :disabled="pending" @click="retry" /></div>
    </div>
    <form @submit.prevent="submit">
      <UITextField id="flag" v-model="flag" label="플래그 제출" label-hidden placeholder="찾은 플래그" :readonly="accepted" :tone="accepted ? 'success' : result === 'rejected' ? 'danger' : 'default'" :aria-invalid="result === 'rejected' || undefined" aria-describedby="submission-feedback" :disabled="pending" required />
      <UIButton class="submit-button" variant="primary" type="submit" :busy="pending" :disabled="accepted || !ready || !flag.trim()">
        <span>{{ accepted ? '완료' : '제출' }}</span>
        <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false" :class="{ 'submit-progress': pending }"><path v-if="pending" d="M20 7v5h-5M4 17v-5h5m10.3-4a8 8 0 0 0-13.9-3M4.7 16a8 8 0 0 0 13.9 3" /><path v-else-if="accepted" d="m5 12 4 4L19 6" /><path v-else d="m5 12 14 0m-6-6 6 6-6 6" /></svg>
      </UIButton>
    </form>
    <div v-if="retained.length" class="environment-recovery" role="region" aria-label="유지 환경 정리">
      <ul><li v-for="environment in retained" :key="environment.slug"><span>{{ environment.title }}</span><UIButton variant="danger" size="compact" :disabled="ending" @click="endEnvironment(environment.slug)">정리</UIButton></li></ul>
    </div>
  </section>
</template>

<style scoped>
.submission-bar { position: relative; flex: none; min-width: 0; padding: var(--ui-space-2); border-top: 1px solid var(--ui-border); background: var(--ui-surface); display: grid; gap: var(--ui-space-1); container-type: inline-size; transition: border-color var(--ui-state-duration) var(--ui-state-easing), background-color var(--ui-state-duration) var(--ui-state-easing); }
.submission-bar--accepted { border-color: var(--ui-success); }
.submission-caption { display: flex; align-items: center; gap: var(--ui-space-1); height: var(--ui-control-size-compact); min-width: 0; }
.submission-caption label { flex: none; font-size: 0.85rem; color: var(--ui-muted); }
.environment-retry { flex: none; width: var(--ui-control-size-compact); height: var(--ui-control-size-compact); }
.submission-feedback { flex: 1; min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; color: var(--ui-muted); font-size: 0.85rem; }
.submission-feedback--error { color: var(--ui-danger); }
.submission-bar--accepted .submission-feedback { color: var(--ui-success); }
form { display: grid; grid-template-columns: minmax(0, 1fr) 6rem; align-items: center; gap: var(--ui-space-1); min-width: 0; }
.submit-button { width: 100%; }
@container (max-width: 20rem) { form { grid-template-columns: minmax(0, 1fr); } }
.submit-button svg { flex: none; width: 1rem; height: 1rem; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
.submission-bar--accepted .submit-button { color: var(--ui-success); border-color: var(--ui-success); opacity: 1; }
.submit-progress { animation: submit-progress 1s linear infinite; }
@keyframes submit-progress { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .submit-progress { animation: none; } }
.environment-recovery { position: absolute; bottom: 100%; inset-inline: var(--ui-space-2); z-index: 3; max-height: 16rem; overflow: auto; padding: var(--ui-space-2); border: 1px solid var(--ui-border-active); border-radius: var(--ui-radius-surface); background: var(--ui-surface-raised); }
.environment-recovery ul { padding: 0; margin: 0; list-style: none; display: grid; gap: var(--ui-space-1); }
.environment-recovery li { display: flex; align-items: center; justify-content: space-between; gap: var(--ui-space-1); }
</style>
