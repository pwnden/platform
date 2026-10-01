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
    <UISwitch label="터미널 연결" compact :model-value="state === 'connected' || state === 'connecting'" :disabled="disabled || busy" :busy="state === 'connecting'" :class="{ 'ui-switch--error': state === 'error' }" :title="`${labels[state]} · ${actions[state]}`" @update:model-value="$emit('toggle')">
      <template #default><span role="status">{{ labels[state] }}</span></template>
      <template #thumb="{ checked, busy: preparing }">
        <svg v-if="preparing" class="ui-terminal-progress" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M20 12a8 8 0 1 1-8-8" /></svg>
        <svg v-else-if="checked" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" /></svg>
        <svg v-else viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="m16 9 2.5-2.5a3.54 3.54 0 0 0-5-5L11 4M8 15l-2.5 2.5a3.54 3.54 0 0 0 5 5L13 20M7 7l2 2m6 6 2 2M3 10h3m12 4h3M10 3v3m4 12v3" /></svg>
      </template>
    </UISwitch>
    <UIButton variant="danger" size="compact" :busy="busy" aria-label="문제 환경 종료" title="풀이 환경과 컨테이너를 즉시 종료" @click="$emit('stop')">
      <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M12 3v9m-5-6a8 8 0 1 0 10 0" /></svg>
    </UIButton>
  </div>
</template>

<style scoped>
.ui-terminal-controls { display: flex; flex: 0 1 auto; min-width: 0; max-width: 100%; align-items: center; flex-wrap: wrap; gap: var(--ui-space-1); }
.ui-terminal-controls :deep(.ui-button) { width: var(--ui-control-size-compact); padding: 0; }
.ui-terminal-controls svg { flex: none; width: 1.125rem; height: 1.125rem; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
.ui-terminal-controls :deep(.ui-switch-thumb) svg { width: 1rem; height: 1rem; }
.ui-terminal-progress { animation: terminal-progress 1s linear infinite; }
@keyframes terminal-progress { to { transform: rotate(360deg); } }
</style>
