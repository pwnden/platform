<script setup lang="ts">
import { computed, ref } from 'vue';
import type { APIClient } from '@pwnden/api';
import type { Problem, ProblemFile } from '@pwnden/catalog';
import type { RunStatus } from '@pwnden/play';
import { ProblemDetail, ProblemFiles, ProblemList, categoryLabel } from '@pwnden/catalog-feature';
import { PlayPanel, ProblemWeb, webEndpoints } from '@pwnden/play-feature';
import type { PlayPanelHandle } from '@pwnden/play-feature';
import { TerminalPanel } from '@pwnden/terminal-feature';
import { UIBadge, UISplit, UIStatus, UITabs } from '@pwnden/ui';

defineProps<{ client?: APIClient | undefined; sessionRejected?: boolean }>();
const selected = ref<Problem>();
const busy = ref(false);
const play = ref<PlayPanelHandle>();
const files = ref<readonly ProblemFile[]>([]);
const runStatus = ref<RunStatus>();
const webAvailable = ref(false);
const tool = ref('terminal');
const tools = computed(() => [
  { value: 'terminal', label: '터미널' },
  ...(files.value.length ? [{ value: 'files', label: '파일' }] : []),
  ...(webAvailable.value ? [{ value: 'web', label: '웹' }] : []),
]);
function observe(status: RunStatus | undefined) {
  runStatus.value = status;
  if (status && webEndpoints(status.endpoints).length) webAvailable.value = true;
}
const catalogWidth = ref(20);
const briefingWidth = ref(50);
function select(problem: Problem) {
  if (!busy.value && selected.value?.slug !== problem.slug) {
    files.value = []; runStatus.value = undefined; webAvailable.value = false; tool.value = 'terminal';
    selected.value = problem;
  }
}
</script>

<template>
  <main class="player-shell">
    <header class="app-header">
      <h1>pwnden</h1>
      <UIStatus :tone="client ? 'info' : 'danger'">{{ client ? '로컬 세션' : '세션 없음' }}</UIStatus>
    </header>
    <div v-if="!client" class="session-error">
      <h2>{{ sessionRejected ? '서버 연결 정보를 확인해 주세요.' : '서버에 연결해 주세요.' }}</h2>
      <p role="alert">실행 중인 pwnden 서버가 출력한 전체 주소로 접속하세요.</p>
    </div>
    <UISplit v-else v-model="catalogWidth" class="workspace" label="문제 목록 너비" :min="14" :max="36">
      <template #before><aside class="catalog"><ProblemList :catalog="client.catalog" :selected-slug="selected?.slug" :selection-disabled="busy" @select="select" /></aside></template>
      <template #after>
        <UISplit v-if="selected" :key="selected.slug" v-model="briefingWidth" class="selected-problem" label="설명과 작업 영역 너비" :min="30" :max="70">
          <template #before><div class="briefing">
            <header class="problem-header">
              <h2 id="problem-heading" :title="selected.title">{{ selected.title }}</h2>
              <UIBadge :title="`분야: ${categoryLabel(selected.category)}`"><span class="ui-sr-only">분야: </span>{{ categoryLabel(selected.category) }}</UIBadge>
            </header>
            <div class="briefing-scroll" tabindex="0" role="region" aria-labelledby="problem-heading">
              <ProblemDetail :catalog="client.catalog" :slug="selected.slug" @loaded="files = $event.files">
                <template #play>
                  <PlayPanel ref="play" :player="client.player" :slug="selected.slug" :kind="selected.kind" @busy="busy = $event" @status="observe" />
                </template>
              </ProblemDetail>
            </div>
          </div></template>
          <template #after>
            <UITabs v-model="tool" :items="tools" label="풀이 도구" class="tool-pane">
              <template #terminal><TerminalPanel :terminals="client.terminals" :slug="selected.slug" :foreground="tool === 'terminal'" @ready="play?.refresh()" @stopped="play?.refresh()" /></template>
              <template #files><ProblemFiles :catalog="client.catalog" :slug="selected.slug" :files="files" /></template>
              <template #web><ProblemWeb :status="runStatus" :active="tool === 'web'" /></template>
            </UITabs>
          </template>
        </UISplit>
        <div v-else class="workspace-empty">
        <div class="empty-content">
          <h2>풀어볼 문제를 선택하세요.</h2>
          <p>문제 이름을 검색하거나 분야별로 찾아볼 수 있습니다.</p>
        </div>
        </div>
      </template>
    </UISplit>
  </main>
</template>

<style scoped>
.player-shell { width: 100%; height: 100dvh; display: grid; grid-template-rows: auto minmax(0, 1fr); }
.app-header { min-height: 3.75rem; padding: var(--ui-space-2); display: flex; align-items: center; justify-content: space-between; gap: var(--ui-space-2); border-bottom: 1px solid var(--ui-border); background: var(--ui-surface); }
h1 { font-family: var(--ui-font-mono); font-size: 1.2rem; font-weight: 500; color: var(--ui-accent); }
.workspace { min-height: 0; }
.catalog { height: 100%; min-width: 0; min-height: 0; overflow: hidden; background: var(--ui-surface); }
.selected-problem { min-width: 0; min-height: 0; }
.briefing { height: 100%; min-width: 0; min-height: 0; display: flex; flex-direction: column; }
.problem-header { flex: none; display: flex; align-items: center; justify-content: space-between; min-height: var(--ui-workspace-header-size); gap: var(--ui-space-1); padding: var(--ui-space-2); border-bottom: 1px solid var(--ui-border); background: var(--ui-surface); }
.problem-header h2 { min-width: 0; font-size: 1rem; line-height: 1.4; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.briefing-scroll { flex: 1; min-height: 0; min-width: 0; overflow: auto; }
.tool-pane { min-height: 0; }
.workspace-empty { height: 100%; min-width: 0; display: grid; align-items: center; justify-items: center; padding: var(--ui-space-4); }
.empty-content { width: min(100%, 42rem); display: grid; gap: var(--ui-space-2); }
.workspace-empty h2 { font-size: 1.35rem; }
.workspace-empty p { color: var(--ui-muted); line-height: 1.9; }
.session-error { display: grid; align-content: center; justify-items: center; gap: 1rem; padding: var(--ui-space-3); }
.session-error h2 { font-size: 1.25rem; }
@media (max-width: 48rem) {
  .player-shell { height: auto; min-height: 100dvh; grid-template-rows: auto 1fr; }
  .catalog { height: 30rem; border-bottom: 1px solid var(--ui-border); }
  .briefing { height: auto; border-bottom: 1px solid var(--ui-border); }
  .problem-header { position: sticky; top: 0; z-index: 2; padding: var(--ui-space-2); }
  .briefing-scroll { overflow: visible; }
  .tool-pane { height: 36rem; }
  .workspace-empty { min-height: 25rem; padding: var(--ui-space-3); }
}
</style>
