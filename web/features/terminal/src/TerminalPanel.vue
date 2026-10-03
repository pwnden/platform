<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref, shallowRef, watch } from 'vue';
import type { TerminalConnection, TerminalSession, TerminalSize, Terminals, RetainedEnvironment } from '@pwnden/terminal';
import { UIButton, UITerminal } from '@pwnden/ui';
import type { UITerminalHandle } from '@pwnden/ui';
import type { TerminalPanelHandle } from './props';

const props = defineProps<{ terminals: Terminals; slug: string; paused?: boolean; foreground?: boolean }>();
const emit = defineEmits<{ ready: [] }>();
const screen = ref<UITerminalHandle>();
const state = ref<'closed' | 'connecting' | 'ready'>('closed');
const session = shallowRef<TerminalSession>();
const message = ref('');
const failed = ref(false);
const retained = ref<readonly RetainedEnvironment[]>([]);
const ending = ref(false);
let size: TerminalSize = { cols: 80, rows: 24 };
let connection: TerminalConnection | undefined;
let generation = 0;
let active = true;
const visible = () => typeof document === 'undefined' || document.visibilityState !== 'hidden';

function disconnect() {
  generation++;
  const owned = connection;
  connection = undefined; session.value = undefined; state.value = 'closed';
  owned?.close();
}
async function connect() {
  if (!active || state.value !== 'closed' || props.paused || !visible()) return;
  disconnect();
  const current = generation;
  state.value = 'connecting'; message.value = ''; failed.value = false; retained.value = [];
  screen.value?.clear();
  try {
    connection = props.terminals.connect(props.slug, size, event => {
      if (!active || current !== generation) return;
      if (event.type === 'output') screen.value?.write(event.data, event.acknowledge);
      else if (event.type === 'exit') {
        session.value = undefined; state.value = 'closed';
        message.value = '셸이 종료되었습니다. 종료 코드: ' + event.code;
      } else if (event.type === 'error') {
        failed.value = true;
        const reasons: Record<string, string> = {
          terminal_busy: '다른 탭에서 이 문제의 터미널을 사용 중입니다.',
          workspace_full: '최대 10개 환경이 사용 중이거나 정리할 수 없는 상태입니다. 여유가 생긴 뒤 다시 연결하세요.',
          resource_limit: '실행 자원 한도가 사용 중이며 자동으로 정리할 대기 환경이 없습니다. 여유가 생긴 뒤 다시 연결하세요.',
          unauthorized: '서버가 출력한 전체 주소로 다시 접속하세요.',
          invalid_request: '페이지를 새로고침한 뒤 다시 연결하세요.',
          deadline_exceeded: '환경 준비 시간이 끝났습니다. 다시 연결하세요.',
          network_error: '서버 연결이 끊겼습니다. 다시 연결하세요.',
          output_backpressure: '출력 처리가 지연되었습니다. 다시 연결하세요.',
          execution_failed: '풀이 환경을 준비하지 못했습니다. Docker 상태를 확인하고 다시 연결하세요.',
          cleanup_failed: '환경 정리에 실패했습니다. 실행 상태를 확인하세요.',
          not_running: '문제 실행 상태를 확인한 뒤 다시 연결하세요.',
        };
        message.value = reasons[event.code] ?? '터미널에 연결하지 못했습니다. 다시 연결하세요.';
        session.value = undefined; state.value = 'closed';
        if (event.code === 'workspace_full') void loadRetained();
      } else {
        session.value = undefined; state.value = 'closed';
        if (!message.value) message.value = '터미널 연결이 종료되었습니다.';
      }
    });
    const result = await connection.ready;
    if (!active || current !== generation) { result.close(); return; }
    session.value = result; state.value = 'ready'; result.resize(size);
    message.value = '';
    emit('ready');
    await nextTick();
    if (active && current === generation && props.foreground !== false) screen.value?.focus();
  } catch {
    if (active && current === generation) {
      state.value = 'closed'; failed.value = true;
      if (!message.value) message.value = '터미널에 연결하지 못했습니다. 다시 연결하세요.';
    }
  }
}
async function refresh() {
  if (state.value === 'connecting' || ending.value || props.paused) return;
  disconnect();
  await connect();
}
const handle: TerminalPanelHandle = { get busy() { return state.value === 'connecting' || ending.value || !!props.paused; }, refresh };
defineExpose(handle);
async function loadRetained() {
  const current = generation;
  try { const items = await props.terminals.list(); if (active && current === generation) retained.value = items; }
  catch { if (active && current === generation) message.value = '유지 환경 목록을 불러오지 못했습니다. 다시 연결하세요.'; }
}
async function endEnvironment(slug: string) {
  if (ending.value) return;
  ending.value = true;
  try {
    await props.terminals.stop(slug);
    if (!active) return;
    await loadRetained(); await connect();
  } catch { if (active) { failed.value = true; message.value = '환경을 종료하지 못했습니다. 다시 시도하세요.'; } }
  finally { if (active) ending.value = false; }
}
function visibility() {
  if (!visible()) disconnect();
  else if (!props.paused) void connect();
}
function resize(next: TerminalSize) { size = next; session.value?.resize(next); }
function input(data: Uint8Array) {
  for (let offset = 0; offset < data.byteLength; offset += 16 * 1024) session.value?.input(data.slice(offset, offset + 16 * 1024));
}
watch(() => props.paused, paused => {
  if (paused) disconnect();
  else void connect();
});
onMounted(() => {
  if (typeof document !== 'undefined') document.addEventListener('visibilitychange', visibility);
  void connect();
});
onUnmounted(() => {
  active = false;
  if (typeof document !== 'undefined') document.removeEventListener('visibilitychange', visibility);
  disconnect();
});
</script>

<template>
  <section class="terminal-panel" aria-label="문제 풀이 터미널" :aria-busy="state === 'connecting'">
    <div class="terminal-screen">
    <ul v-if="retained.length" class="retained">
      <li v-for="environment in retained" :key="environment.slug">
        <span>{{ environment.title }} · {{ environment.connected ? '사용 중' : '자동 정리 대기' }}</span>
        <UIButton variant="danger" size="compact" :disabled="ending" @click="endEnvironment(environment.slug)">종료</UIButton>
      </li>
    </ul>
      <p v-if="message" class="terminal-notice" :role="failed ? 'alert' : 'status'">{{ message }}</p>
      <UITerminal ref="screen" label="문제 풀이 셸" class="terminal-renderer" :enabled="state === 'ready'" @input="input" @resize="resize" />
      <p v-if="state === 'connecting'" class="terminal-progress" role="status">풀이 환경에 연결하는 중…</p>
    </div>
  </section>
</template>

<style scoped>
.terminal-panel { box-sizing: border-box; height: 100%; min-height: 0; padding: var(--ui-space-2); background: var(--ui-terminal-background); display: flex; flex-direction: column; gap: var(--ui-space-2); overflow-y: auto; }
.terminal-screen { position: relative; flex: 1; min-height: 6rem; min-width: 0; }
.terminal-screen :deep(.ui-terminal) { position: absolute; inset: 0; min-height: 0; padding: 0; }
.terminal-progress { position: absolute; inset: 0; display: grid; place-content: center; pointer-events: none; color: var(--ui-muted); }
.terminal-notice { position: absolute; top: 0; inset-inline: 0; z-index: 1; padding: var(--ui-space-2); background: var(--ui-surface-raised); color: var(--ui-muted); }
.retained { position: absolute; inset: 0; z-index: 2; padding: var(--ui-space-2); margin: 0; list-style: none; display: grid; align-content: start; gap: var(--ui-space-1); background: var(--ui-surface); overflow: auto; }
.retained li { display: flex; align-items: center; justify-content: space-between; gap: var(--ui-space-1); }
</style>
