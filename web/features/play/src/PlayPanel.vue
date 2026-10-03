<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import type { Player, RunStatus, Workspaces, WorkspaceConnection, WorkspaceInfo } from '@pwnden/play';
import { UIForm, UISubmitButton, UITextField, UIIcon } from '@pwnden/ui';
import EnvironmentStatus from './EnvironmentStatus.vue';
import type { EnvironmentState } from './props';

const props = defineProps<{ player: Player; workspaces: Workspaces; slug: string; kind: RunStatus['kind']; answer?: string | undefined }>();
const emit = defineEmits<{ busy: [value: boolean]; status: [value: RunStatus | undefined]; completed: [] }>();
const pending = ref(false);
const preparing = ref(false);
const status = ref<RunStatus>();
const flag = ref('');
const result = ref<'idle' | 'accepted' | 'rejected' | 'error'>('idle');
const message = ref('');
watch(() => props.answer, answer => {
  if (answer !== undefined && result.value !== 'accepted') { flag.value = answer; result.value = 'accepted'; message.value = '해결 완료'; }
}, { immediate: true });
const environmentError = ref('');
const retained = ref<readonly WorkspaceInfo[]>([]);
const ending = ref(false);
let active = true;
let connection: WorkspaceConnection | undefined;
let generation = 0;
const visible = () => typeof document === 'undefined' || document.visibilityState !== 'hidden';
const accepted = computed(() => result.value === 'accepted');
const ready = computed(() => !preparing.value && !environmentError.value && !!status.value && (status.value.state === 'running' || status.value.state === 'ready'));
const feedback = computed(() => pending.value ? '확인 중…' : message.value);
const blocked = computed(() => preparing.value || !!environmentError.value);
const environment = computed<EnvironmentState>(() => ({ preparing: preparing.value, error: environmentError.value, retained: retained.value, ending: ending.value, retry, stop: endEnvironment }));
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
        workspace_full: '환경 10개가 사용 중이거나 정리할 수 없는 상태입니다. 여유가 생긴 뒤 다시 연결하세요.',
        resource_limit: '실행 자원 한도가 사용 중이며 자동으로 정리할 대기 환경이 없습니다. 여유가 생긴 뒤 다시 연결하세요.',
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
    if (active) { result.value = answer.accepted ? 'accepted' : 'rejected'; message.value = answer.accepted ? '해결 완료' : '플래그를 다시 확인하세요.'; if (answer.accepted) emit('completed'); }
  } catch { if (active) { result.value = 'error'; message.value = '제출하지 못했습니다. 다시 시도하세요.'; } }
  finally { if (active) { pending.value = false; emit('busy', false); } }
}
function visibility() { if (visible()) void prepare(); else disconnect(); }
onMounted(() => { if (typeof document !== 'undefined') document.addEventListener('visibilitychange', visibility); void prepare(); });
onUnmounted(() => { active = false; if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', visibility); disconnect(); });
</script>

<template>
  <div class="play-workspace">
    <div class="play-tools"><slot :environment="environment" :blocked="blocked"><EnvironmentStatus :state="environment" /></slot></div>
    <section class="submission-bar" :class="{ 'submission-bar--accepted': accepted }" aria-label="플래그 제출" :aria-busy="pending">
      <div class="submission-caption">
        <label for="flag">정답 제출</label>
        <p id="submission-feedback" class="submission-feedback" :class="{ 'submission-feedback--error': result === 'rejected' || result === 'error' }" role="status" aria-live="polite" :title="feedback">{{ feedback }}</p>
      </div>
      <UIForm :submit="submit">
        <UITextField id="flag" v-model="flag" label="정답 플래그" label-hidden placeholder="문제에서 찾아낸 플래그를 입력하세요" :readonly="accepted" :tone="accepted ? 'success' : result === 'rejected' ? 'danger' : 'default'" :aria-invalid="result === 'rejected' || undefined" aria-describedby="submission-feedback" :disabled="pending" required />
        <UISubmitButton class="submit-button" :busy="pending" :disabled="accepted || !ready || !flag.trim()">
          <span>{{ accepted ? '완료' : '제출' }}</span>
          <UIIcon :name="pending ? 'loader' : accepted ? 'check' : 'forward'" :class="{ 'submit-progress': pending }" />
        </UISubmitButton>
      </UIForm>
    </section>
  </div>
</template>

<style scoped>
.play-workspace { height: 100%; min-height: 0; min-width: 0; display: flex; flex-direction: column; }
.play-tools { position: relative; flex: 1; min-width: 0; min-height: 0; }
.submission-bar { position: relative; flex: none; min-width: 0; padding: var(--ui-space-2); border-top: 1px solid var(--ui-border); background: var(--ui-surface); display: grid; gap: var(--ui-space-1); container-type: inline-size; transition: border-color var(--ui-state-duration) var(--ui-state-easing), background-color var(--ui-state-duration) var(--ui-state-easing); }
.submission-bar--accepted { border-color: var(--ui-success); }
.submission-caption { display: flex; align-items: center; gap: var(--ui-space-1); height: var(--ui-control-size-compact); min-width: 0; }
.submission-caption label { flex: none; font-size: 1rem; font-weight: 600; line-height: 1.4; color: var(--ui-foreground); white-space: nowrap; }
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
</style>
