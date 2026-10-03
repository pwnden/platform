<script setup lang="ts">
import { UIStatus } from '@pwnden/ui';
defineProps<{ connected: boolean; sessionRejected?: boolean | undefined }>();
</script>
<template>
  <main class="player-shell">
    <header class="app-header"><h1>pwnden</h1><UIStatus :tone="connected ? 'info' : 'danger'">{{ connected ? '로컬 세션' : '세션 없음' }}</UIStatus></header>
    <div v-if="!connected" class="session-error">
      <h2>{{ sessionRejected ? '서버 연결 정보를 확인해 주세요.' : '서버에 연결해 주세요.' }}</h2>
      <p role="alert">실행 중인 pwnden 서버가 출력한 전체 주소로 접속하세요.</p>
    </div>
    <slot v-else />
  </main>
</template>
<style scoped>
.player-shell { width: 100%; height: 100dvh; display: grid; grid-template-rows: auto minmax(0, 1fr); }
.app-header { min-height: 3.75rem; padding: var(--ui-space-2); display: flex; align-items: center; justify-content: space-between; gap: var(--ui-space-2); border-bottom: 1px solid var(--ui-border); background: var(--ui-surface); }
h1 { font-family: var(--ui-font-mono); font-size: 1.2rem; font-weight: 500; color: var(--ui-accent); }
.session-error { display: grid; align-content: center; justify-items: center; gap: 1rem; padding: var(--ui-space-3); }
.session-error h2 { font-size: 1.25rem; }
@media (max-width: 48rem) { .player-shell { height: auto; min-height: 100dvh; grid-template-rows: auto 1fr; } }
</style>
