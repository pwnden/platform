<script setup lang="ts">
import { UIButton, UIIcon } from '@pwnden/ui';
import type { EnvironmentState } from './props';
defineProps<{ state: EnvironmentState }>();
</script>

<template>
  <div v-if="state.preparing || state.error" class="environment-overlay" :aria-busy="state.preparing && !state.error">
    <div class="environment-content">
      <p class="environment-feedback" :class="{ 'environment-feedback--error': state.error }" role="status" aria-live="polite">
        <UIIcon :name="state.error ? 'error' : 'loader'" :class="{ 'environment-progress': !state.error }" />
        <span>{{ state.error || '풀이 환경 준비 중…' }}</span>
      </p>
      <UIButton v-if="state.error" aria-label="풀이 환경 다시 연결" size="compact" :disabled="state.preparing || state.ending" @click="state.retry()"><UIIcon name="refresh" />다시 연결</UIButton>
      <div v-if="state.retained.length" class="environment-recovery" role="region" aria-label="유지 환경 정리">
        <ul><li v-for="environment in state.retained" :key="environment.slug"><span>{{ environment.title }}</span><UIButton variant="danger" size="compact" :disabled="state.ending" @click="state.stop(environment.slug)">정리</UIButton></li></ul>
      </div>
    </div>
  </div>
</template>

<style scoped>
.environment-overlay { position: absolute; inset: 0; z-index: 3; display: grid; place-items: center; overflow: auto; padding: var(--ui-space-3); background: var(--ui-surface); }
.environment-content { width: 100%; max-width: 28rem; min-width: 0; display: grid; justify-items: center; gap: var(--ui-space-2); }
.environment-feedback { margin: 0; display: flex; align-items: center; gap: var(--ui-space-1); color: var(--ui-muted); line-height: 1.6; }
.environment-feedback svg { flex: none; }
.environment-feedback--error { color: var(--ui-danger); }
.environment-recovery { width: 100%; min-width: 0; max-height: 16rem; overflow: auto; padding: var(--ui-space-2); border: 1px solid var(--ui-border); border-radius: var(--ui-radius-surface); background: var(--ui-surface-raised); }
.environment-recovery ul { padding: 0; margin: 0; list-style: none; display: grid; gap: var(--ui-space-1); }
.environment-recovery li { display: flex; align-items: center; justify-content: space-between; gap: var(--ui-space-1); }
.environment-recovery li span { min-width: 0; overflow-wrap: anywhere; }
.environment-progress { animation: environment-progress 1s linear infinite; }
@keyframes environment-progress { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .environment-progress { animation: none; } }
</style>
