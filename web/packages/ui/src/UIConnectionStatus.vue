<script setup lang="ts">
import type { UIConnectionStatusProps } from './props';
defineProps<UIConnectionStatusProps>();
const labels = { connected: '연결됨', connecting: '준비·연결 중', disconnected: '연결 해제됨', error: '연결 오류' };
</script>

<template>
  <span class="ui-connection-status" :class="`ui-connection-status--${state}`" role="status" :title="labels[state]">
    <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path d="M9 3v4M15 3v4M12 17v4" />
      <template v-if="state === 'connected' || state === 'connecting'">
        <path d="M7 7h10v5a5 5 0 0 1-10 0Z" />
        <path class="connection-current" d="M10 11h4" />
      </template>
      <template v-else>
        <path d="M7 7h10v3M7 12a5 5 0 0 0 10 0" />
        <path v-if="state === 'error'" d="m9 10 6 4m0-4-6 4" />
        <path v-else d="m5 11 14-2" />
      </template>
    </svg>
    <span class="connection-label">{{ labels[state] }}</span>
  </span>
</template>

<style scoped>
.ui-connection-status { display: inline-flex; align-items: center; justify-content: center; flex: none; width: 2rem; min-height: 2.5rem; color: var(--ui-muted); }
.ui-connection-status svg { width: 1.5rem; height: 1.5rem; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
.ui-connection-status--connected { color: var(--ui-accent); }
.ui-connection-status--connecting { color: var(--ui-warning); }
.ui-connection-status--error { color: var(--ui-danger); }
.ui-connection-status--connecting .connection-current { animation: connection-current 1s ease-in-out infinite alternate; }
.connection-label { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
@keyframes connection-current { from { opacity: 0.25; } to { opacity: 1; } }
@media (prefers-reduced-motion: reduce) { .connection-current { animation: none !important; } }
</style>
