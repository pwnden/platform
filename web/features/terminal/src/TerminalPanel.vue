<script setup lang="ts">
import { onUnmounted, ref, shallowRef, watch } from 'vue';
import type { TerminalConnection, TerminalSession, TerminalSize, Terminals } from '@pwnden/terminal';
import { UIButton, UITerminal } from '@pwnden/ui';
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
  <section aria-labelledby="terminal-heading">
    <h2 id="terminal-heading">풀이 터미널</h2>
    <p>문제의 풀이 환경에서 명령을 실행합니다. 연결은 최대 15분 동안 유지되며, 종료하면 셸과 임시 컨테이너가 정리됩니다.</p>
    <p v-if="!enabled">문제 실행 상태가 준비되면 연결할 수 있습니다.</p>
    <div class="actions">
      <UIButton :disabled="!enabled || busy || state !== 'closed'" @click="connect">터미널 연결</UIButton>
      <UIButton :disabled="state === 'closed'" @click="disconnect">연결 종료</UIButton>
    </div>
    <p v-if="state === 'connecting'" role="status">풀이 환경에 연결하는 중…</p>
    <p v-else-if="state === 'ready'" role="status">연결됨</p>
    <p v-if="message" :role="failed ? 'alert' : 'status'">{{ message }}</p>
    <UITerminal ref="screen" label="문제 풀이 셸" :enabled="state === 'ready'" @input="input" @resize="resize" />
  </section>
</template>

<style scoped>
.actions { display: flex; flex-wrap: wrap; gap: var(--ui-space-1); margin-bottom: var(--ui-space-2); }
</style>
