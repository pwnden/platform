<script setup lang="ts">
import type { UITerminalControlsProps } from './props';
import UIButton from './UIButton.vue';
import UISwitch from './UISwitch.vue';
defineProps<UITerminalControlsProps>();
defineEmits<{ toggle: []; stop: [] }>();
const actions = { connected: '연결 해제', connecting: '연결 취소', disconnected: '터미널 다시 연결', error: '터미널 다시 연결' };
const labels = { connected: '연결됨', connecting: '준비·연결 중', disconnected: '연결 해제됨', error: '연결 오류' };
</script>

<template>
  <div class="ui-terminal-controls" role="group" aria-label="터미널 연결 제어">
    <UISwitch label="터미널 연결" :model-value="state === 'connected' || state === 'connecting'" :disabled="disabled || busy" :busy="state === 'connecting'" :class="{ 'ui-switch--error': state === 'error' }" :title="actions[state]" @update:model-value="$emit('toggle')">
      <span role="status">{{ labels[state] }}</span>
    </UISwitch>
    <UIButton variant="danger" size="compact" :busy="busy" aria-label="문제 환경 종료" title="풀이 환경과 컨테이너를 즉시 종료" @click="$emit('stop')">
      <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M12 3v9m-5-6a8 8 0 1 0 10 0" /></svg>
      <span>환경 종료</span>
    </UIButton>
  </div>
</template>

<style scoped>
.ui-terminal-controls { display: flex; flex: 0 1 auto; min-width: 0; max-width: 100%; align-items: center; flex-wrap: wrap; gap: var(--ui-space-1); }
.ui-terminal-controls :deep(.ui-button) { gap: var(--ui-space-1); min-width: 2.5rem; }
.ui-terminal-controls svg { flex: none; width: 1.125rem; height: 1.125rem; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
</style>
