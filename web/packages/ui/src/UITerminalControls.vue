<script setup lang="ts">
import type { UITerminalControlsProps } from './props';
import UIButton from './UIButton.vue';
import UIConnectionStatus from './UIConnectionStatus.vue';
defineProps<UITerminalControlsProps>();
defineEmits<{ toggle: []; stop: [] }>();
const actions = { connected: '연결 해제', connecting: '연결 취소', disconnected: '터미널 다시 연결', error: '터미널 다시 연결' };
</script>

<template>
  <div class="ui-terminal-controls" role="group" aria-label="터미널 연결 제어">
    <UIConnectionStatus :state="state" />
    <UIButton variant="ghost" size="compact" :disabled="disabled || busy" :aria-label="actions[state]" :title="actions[state]" @click="$emit('toggle')">
      <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
        <path v-if="state === 'connected'" d="M10 4H5v16h5m6-5 4-3-4-3m-7 3h11" />
        <path v-else-if="state === 'connecting'" d="m6 6 12 12M6 18 18 6" />
        <path v-else d="M14 4h5v16h-5m-6-5 4-3-4-3m-4 3h8" />
      </svg>
      <span class="connection-action-label">{{ actions[state] }}</span>
    </UIButton>
    <UIButton variant="danger" size="compact" :disabled="busy" aria-label="문제 환경 종료" title="문제 환경 종료" @click="$emit('stop')">
      <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M12 3v9m-5-6a8 8 0 1 0 10 0" /></svg>
    </UIButton>
  </div>
</template>

<style scoped>
.ui-terminal-controls { display: flex; flex: none; align-items: center; gap: var(--ui-space-1); white-space: nowrap; }
.ui-terminal-controls :deep(.ui-button) { gap: var(--ui-space-1); min-width: 2.5rem; }
.ui-terminal-controls svg { flex: none; width: 1.125rem; height: 1.125rem; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
@container (max-width: 20rem) { .connection-action-label { display: none; } }
</style>
