<script setup lang="ts">
import { onUnmounted, ref, shallowRef, watch } from 'vue';
import type { TerminalConnection, TerminalSession, TerminalSize, Terminals } from '@pwnden/terminal';
import { UIButton, UITerminal, UIPanel, UIStatus } from '@pwnden/ui';
import type { UITerminalHandle } from '@pwnden/ui';

const props = defineProps<{ terminals: Terminals; slug: string; enabled: boolean; busy: boolean }>();
const screen = ref<UITerminalHandle>();
const state = ref<'closed' | 'connecting' | 'ready'>('closed');
const session = shallowRef<TerminalSession>();
const message = ref('');
const failed = ref(false);
let size: TerminalSize = { cols: 80, rows: 24 };
let connection: TerminalConnection | undefined;
let generation = 0;
let active = true;

function disconnect() {
  generation++;
  const owned = connection;
  connection = undefined;
  session.value = undefined;
  state.value = 'closed';
  owned?.close();
}
async function connect() {
  if (state.value !== 'closed' || !props.enabled || props.busy) return;
  disconnect();
  const current = generation;
  state.value = 'connecting'; message.value = ''; failed.value = false;
  screen.value?.clear();
  try {
    connection = props.terminals.connect(props.slug, size, event => {
      if (!active || current !== generation) return;
      if (event.type === 'output') screen.value?.write(event.data, event.acknowledge);
      else if (event.type === 'exit') message.value = `셸이 종료되었습니다. 종료 코드: ${event.code}`;
      else if (event.type === 'error') {
        failed.value = true;
        message.value = event.code === 'terminal_busy' ? '이 문제의 터미널이 다른 탭에 열려 있습니다. 그 연결을 종료한 뒤 다시 연결하세요.' :
          '터미널 연결을 완료하지 못했습니다. 문제 실행 상태를 확인한 뒤 다시 연결하세요.';
        session.value = undefined; state.value = 'closed';
      } else { session.value = undefined; state.value = 'closed'; if (!message.value) message.value = '터미널 연결이 종료되었습니다.'; }
    });
    const result = await connection.ready;
    if (!active || current !== generation) { result.close(); return; }
    session.value = result; state.value = 'ready'; result.resize(size); screen.value?.focus();
  } catch {
    if (active && current === generation) {
      state.value = 'closed'; failed.value = true;
      if (!message.value) message.value = '터미널에 연결하지 못했습니다. 다시 연결하세요.';
    }
  }
}
function resize(next: TerminalSize) { size = next; session.value?.resize(next); }
function input(data: Uint8Array) {
  // Paste input is split into protocol-sized chunks without changing bytes.
  for (let offset = 0; offset < data.byteLength; offset += 16 * 1024) session.value?.input(data.slice(offset, offset + 16 * 1024));
}
watch(() => props.enabled, enabled => { if (!enabled) disconnect(); });
onUnmounted(() => { active = false; disconnect(); });
</script>

<template>
  <UIPanel title="풀이 터미널" headingID="terminal-heading" class="terminal-panel">
    <template #actions><UIStatus role="status" :tone="state === 'ready' ? 'info' : 'muted'">{{ state === 'ready' ? '연결됨' : state === 'connecting' ? '연결 중' : '연결 대기' }}</UIStatus></template>
    <div class="actions">
      <UIButton variant="primary" size="compact" :disabled="!enabled || busy || state !== 'closed'" @click="connect">터미널 연결</UIButton>
      <UIButton variant="ghost" size="compact" :disabled="state === 'closed'" @click="disconnect">연결 종료</UIButton>
      <span class="session-limit">최대 15분</span>
    </div>
    <p v-if="state === 'connecting'" role="status">풀이 환경에 연결하는 중…</p>
    <p v-if="message" :role="failed ? 'alert' : 'status'">{{ message }}</p>
    <div class="terminal-screen">
      <UITerminal ref="screen" label="문제 풀이 셸" :enabled="state === 'ready'" @input="input" @resize="resize" />
      <div v-if="state === 'closed' && !message" class="terminal-empty">
        <span class="shell-prompt" aria-hidden="true">&gt;_</span>
        <p>{{ enabled ? '분석할 준비가 되었다면 터미널을 연결하세요.' : '문제 실행 상태가 준비되면 연결할 수 있습니다.' }}</p>
        <p class="terminal-help">문제의 격리된 풀이 환경에서 명령을 실행합니다.</p>
      </div>
    </div>
    <p class="terminal-footnote">연결을 종료하면 셸과 임시 컨테이너가 정리됩니다.</p>
  </UIPanel>
</template>

<style scoped>
.terminal-panel { height: 100%; background: var(--ui-terminal-background); }
.terminal-panel :deep(.ui-panel-body) { display: flex; flex-direction: column; gap: var(--ui-space-2); padding: var(--ui-space-2); }
.actions { display: flex; align-items: center; flex-wrap: wrap; gap: var(--ui-space-1); }
.session-limit { margin-left: auto; font-size: 0.75rem; color: var(--ui-muted); }
.terminal-screen { position: relative; flex: 1; min-height: 18rem; min-width: 0; }
.terminal-screen :deep(.ui-terminal) { position: absolute; inset: 0; min-height: 0; }
.terminal-empty { position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: var(--ui-space-2); padding: var(--ui-space-2); text-align: center; pointer-events: none; }
.shell-prompt { font-size: 2rem; color: var(--ui-border-active); }
.terminal-empty p { max-width: 34ch; color: var(--ui-muted); font-size: 0.9rem; }
.terminal-help { font-size: 0.8rem !important; }
.terminal-footnote { color: var(--ui-muted); font-size: 0.75rem; padding-inline: var(--ui-space-1); }
</style>
