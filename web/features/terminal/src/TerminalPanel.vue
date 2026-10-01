<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref, shallowRef, watch } from 'vue';
import type { TerminalConnection, TerminalSession, TerminalSize, Terminals, RetainedEnvironment } from '@pwnden/terminal';
import { UIButton, UITerminal, UIPanel, UITerminalControls } from '@pwnden/ui';
import type { UITerminalHandle } from '@pwnden/ui';

const props = defineProps<{ terminals: Terminals; slug: string; paused?: boolean }>();
const emit = defineEmits<{ ready: []; stopped: [] }>();
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
let manuallyDisconnected = false;
const visible = () => typeof document === 'undefined' || document.visibilityState !== 'hidden';

function disconnect() {
  generation++;
  const owned = connection;
  connection = undefined; session.value = undefined; state.value = 'closed';
  owned?.close();
}
async function connect() {
  if (!active || state.value !== 'closed' || props.paused || manuallyDisconnected || !visible()) return;
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
          workspace_full: '최대 10개 환경이 유지 중입니다. 환경 하나를 종료한 뒤 다시 연결하세요.',
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
    if (active && current === generation) screen.value?.focus();
  } catch {
    if (active && current === generation) {
      state.value = 'closed'; failed.value = true;
      if (!message.value) message.value = '터미널에 연결하지 못했습니다. 다시 연결하세요.';
    }
  }
}
function toggleConnection() {
  if (state.value !== 'closed') {
    manuallyDisconnected = true;
    disconnect(); message.value = ''; failed.value = false; retained.value = [];
  } else {
    manuallyDisconnected = false;
    void connect();
  }
}
async function loadRetained() {
  const current = generation;
  try { const items = await props.terminals.list(); if (active && current === generation) retained.value = items; }
  catch { if (active && current === generation) message.value = '유지 환경 목록을 불러오지 못했습니다. 다시 연결하세요.'; }
}
async function endEnvironment(slug = props.slug) {
  if (ending.value) return;
  ending.value = true;
  try {
    await props.terminals.stop(slug);
    if (!active) return;
    if (slug === props.slug) { manuallyDisconnected = true; disconnect(); failed.value = false; message.value = '문제 환경을 종료했습니다.'; emit('stopped'); }
    else { await loadRetained(); await connect(); }
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
  else { manuallyDisconnected = false; void connect(); }
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
  <UIPanel title="풀이 터미널" headingID="terminal-heading" class="terminal-panel">
    <template #actions>
      <UITerminalControls :state="state === 'ready' ? 'connected' : state === 'connecting' ? 'connecting' : failed ? 'error' : 'disconnected'" :disabled="paused" :busy="ending" @toggle="toggleConnection" @stop="endEnvironment()" />
    </template>
    <p v-if="message" :role="failed ? 'alert' : 'status'">{{ message }}</p>
    <ul v-if="retained.length" class="retained">
      <li v-for="environment in retained" :key="environment.slug">
        <span>{{ environment.title }} · {{ environment.connected ? '사용 중' : '자동 정리 대기' }}</span>
        <UIButton variant="danger" size="compact" :disabled="ending" @click="endEnvironment(environment.slug)">종료</UIButton>
      </li>
    </ul>
    <div class="terminal-screen">
      <UITerminal ref="screen" label="문제 풀이 셸" class="terminal-renderer" :enabled="state === 'ready'" @input="input" @resize="resize" />
    </div>
    <p class="terminal-footnote">연결을 해제하거나 문제에서 나가면 10분 후 환경이 정리됩니다.</p>
  </UIPanel>
</template>

<style scoped>
.terminal-panel { height: 100%; background: var(--ui-terminal-background); container-type: inline-size; }
.terminal-panel :deep(.ui-panel-body) { display: flex; flex-direction: column; gap: var(--ui-space-2); padding: var(--ui-space-2); overflow-y: auto; }
.terminal-screen { position: relative; flex: 1; min-height: 6rem; min-width: 0; }
.terminal-screen :deep(.ui-terminal) { position: absolute; inset: 0; min-height: 0; }
.terminal-footnote { color: var(--ui-muted); font-size: 0.75rem; padding-inline: var(--ui-space-1); }
.retained { padding: 0; margin: 0; list-style: none; display: grid; gap: var(--ui-space-1); }
.retained li { display: flex; align-items: center; justify-content: space-between; gap: var(--ui-space-1); }
</style>
