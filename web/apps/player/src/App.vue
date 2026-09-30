<script setup lang="ts">
import { ref } from 'vue';
import type { APIClient } from '@pwnden/api';
import type { Problem } from '@pwnden/catalog';
import { ProblemDetail, ProblemList } from '@pwnden/catalog-feature';
import { PlayPanel } from '@pwnden/play-feature';
import { TerminalPanel } from '@pwnden/terminal-feature';
import { UIStatus } from '@pwnden/ui';

defineProps<{ client?: APIClient | undefined }>();
const selected = ref<Problem>();
const busy = ref(false);
const terminalEnabled = ref(false);
function select(problem: Problem) {
  if (!busy.value && selected.value?.slug !== problem.slug) { terminalEnabled.value = false; selected.value = problem; }
}
</script>

<template>
  <main class="player-shell">
    <header class="app-header">
      <h1><span class="brand-prompt" aria-hidden="true">&gt;_</span> pwnden<span class="brand-cursor" aria-hidden="true" /></h1>
      <span class="workspace-label">문제 풀이 작업 공간</span>
      <UIStatus :tone="client ? 'info' : 'danger'">{{ client ? '로컬 세션' : '세션 없음' }}</UIStatus>
    </header>
    <div v-if="!client" class="session-error">
      <h2>세션에 연결할 수 없습니다.</h2>
      <p role="alert">실행 중인 pwnden 서버가 출력한 주소로 다시 접속하세요.</p>
    </div>
    <div v-else class="workspace">
      <aside class="catalog"><ProblemList :catalog="client.catalog" :selected-slug="selected?.slug" :selection-disabled="busy" @select="select" /></aside>
      <div v-if="selected" :key="selected.slug" class="selected-problem">
        <div class="briefing">
          <ProblemDetail :catalog="client.catalog" :slug="selected.slug" :title="selected.title" />
          <PlayPanel :player="client.player" :slug="selected.slug" :kind="selected.kind" @busy="busy = $event" @status="terminalEnabled = $event?.state === 'ready' || $event?.state === 'running'" />
        </div>
        <TerminalPanel class="terminal-pane" :terminals="client.terminals" :slug="selected.slug" :enabled="terminalEnabled" :busy="busy" />
      </div>
      <div v-else class="workspace-empty">
        <div class="empty-content">
          <div class="empty-prompt" aria-hidden="true">pwnden<span>:~$</span><span class="empty-cursor" /></div>
          <h2>어떤 문제부터 풀어볼까?</h2>
          <p>문제 목록에서 문제를 선택하세요.<br />설명과 파일을 확인하고, 풀이 터미널에서 시작할 수 있습니다.</p>
          <div class="empty-workflow" aria-label="문제 풀이 순서"><span>문제 선택</span><span aria-hidden="true">/</span><span>분석 · 실행</span><span aria-hidden="true">/</span><span>플래그 제출</span></div>
        </div>
      </div>
    </div>
    <footer class="app-footer"><span>{{ selected ? selected.slug : '문제 선택 대기' }}</span><span>{{ busy ? '요청 처리 중' : 'pwnden / local workspace' }}</span></footer>
  </main>
</template>

<style scoped>
.player-shell { width: 100%; height: 100dvh; min-height: 38rem; display: grid; grid-template-rows: auto minmax(0, 1fr) auto; }
.app-header { min-height: 4rem; padding: 0.7rem var(--ui-space-3); display: flex; align-items: center; gap: var(--ui-space-4); border-bottom: 1px solid var(--ui-border); background: var(--ui-surface); }
h1 { display: flex; align-items: center; gap: 0.65rem; font-size: 1.3rem; color: var(--ui-accent); }
.brand-prompt { color: var(--ui-muted); font-size: 1rem; }
.brand-cursor { width: 0.5rem; height: 1.25rem; background: var(--ui-accent); opacity: 0.7; box-shadow: var(--ui-glow); }
.workspace-label { flex: 1; color: var(--ui-muted); font-size: 0.85rem; }
.workspace { min-height: 0; display: grid; grid-template-columns: 17rem minmax(0, 1fr); }
.catalog { min-width: 0; min-height: 0; overflow: auto; border-right: 1px solid var(--ui-border); background: var(--ui-surface); }
.selected-problem { min-width: 0; min-height: 0; display: grid; grid-template-columns: minmax(21rem, 0.9fr) minmax(0, 1.2fr); }
.briefing { min-width: 0; min-height: 0; overflow: auto; border-right: 1px solid var(--ui-border); }
.terminal-pane { min-height: 0; }
.workspace-empty { min-width: 0; display: grid; align-items: center; justify-items: center; padding: var(--ui-space-4); }
.empty-content { width: min(100%, 42rem); }
.empty-prompt { margin-bottom: 2.5rem; color: var(--ui-accent); font-size: clamp(1.8rem, 3vw, 3rem); }
.empty-prompt > span:first-child { color: var(--ui-muted); }
.empty-cursor { display: inline-block; width: 0.55em; height: 1em; margin-left: 0.5em; vertical-align: -0.15em; background: var(--ui-accent); box-shadow: var(--ui-glow); animation: cursor-pulse 1.6s ease-in-out infinite; }
.workspace-empty h2 { margin-bottom: 0.8rem; font-size: 1.35rem; }
.workspace-empty p { color: var(--ui-muted); line-height: 1.9; }
.empty-workflow { margin-top: 2.5rem; display: flex; flex-wrap: wrap; gap: 0.8rem; color: var(--ui-muted); font-size: 0.85rem; }
.empty-workflow > span:nth-child(even) { color: var(--ui-border-active); }
.app-footer { min-width: 0; padding: 0.4rem var(--ui-space-3); display: flex; justify-content: space-between; flex-wrap: wrap; gap: 0.5rem; border-top: 1px solid var(--ui-border); color: var(--ui-muted); background: var(--ui-surface); font-size: 0.75rem; overflow-wrap: anywhere; }
.session-error { display: grid; align-content: center; justify-items: center; gap: 1rem; padding: var(--ui-space-3); }
.session-error h2 { font-size: 1.25rem; }
@keyframes cursor-pulse { 0%, 100% { opacity: 0.8; } 50% { opacity: 0.3; } }
@media (max-width: 76rem) { .workspace { grid-template-columns: 15rem minmax(0, 1fr); } .selected-problem { grid-template-columns: minmax(0, 1fr); overflow: auto; } .briefing { overflow: visible; border-right: 0; } .terminal-pane { min-height: 32rem; } }
@media (max-width: 48rem) { .player-shell { height: auto; min-height: 100dvh; grid-template-rows: auto 1fr auto; } .app-header { padding-inline: var(--ui-space-2); gap: var(--ui-space-2); } .workspace-label { display: none; } .app-header :deep(.ui-status) { margin-left: auto; } .workspace { grid-template-columns: minmax(0, 1fr); } .catalog { border-right: 0; border-bottom: 1px solid var(--ui-border); max-height: 18rem; } .selected-problem { overflow: visible; } .workspace-empty { min-height: 25rem; padding: var(--ui-space-3); } .app-footer { padding-inline: var(--ui-space-2); } }
</style>
