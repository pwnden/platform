<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { APIClient } from '@pwnden/api';
import type { Problem, ProblemFile, ProblemTool, ProblemDetail as ProblemDetails } from '@pwnden/catalog';
import type { RunStatus } from '@pwnden/play';
import { ProblemDetail, ProblemFiles, categoryLabel } from '@pwnden/catalog-feature';
import type { ProblemFilesHandle } from '@pwnden/catalog-feature';
import { EnvironmentStatus, PlayPanel, ProblemWeb } from '@pwnden/play-feature';
import type { ProblemWebHandle } from '@pwnden/play-feature';
import { TerminalPanel } from '@pwnden/terminal-feature';
import type { TerminalPanelHandle } from '@pwnden/terminal-feature';
import { UIBadge, UIDifficultyBadge, UIIconButton, UILink, UISplit, UITabs } from '@pwnden/ui';

const props = defineProps<{ client: APIClient; problem: Problem }>();
const emit = defineEmits<{ select: [problem: Problem]; busy: [value: boolean]; completed: [] }>();
const selected = computed(() => props.problem);
const answer = ref<string>();
const busy = ref(false);
watch(busy, value => emit('busy', value));
const terminal = ref<TerminalPanelHandle>();
const materials = ref<ProblemFilesHandle>();
const web = ref<ProblemWebHandle>();
const files = ref<readonly ProblemFile[]>([]);
const runStatus = ref<RunStatus>();
const allowed = ref<readonly ProblemTool[]>([]);
const visited = ref<readonly string[]>([]);
const tool = ref('');
const labels: Record<ProblemTool, string> = { terminal: '터미널', files: '파일', web: '웹' };
const tools = computed(() => allowed.value.map(value => ({ value, label: labels[value], icon: value })));
watch(tool, value => { if (value && !visited.value.includes(value)) visited.value = [...visited.value, value]; });
function loaded(detail: ProblemDetails) {
  answer.value = detail.answer;
  files.value = detail.files; allowed.value = detail.tools;
  if (!detail.tools.includes(tool.value as ProblemTool)) tool.value = detail.tools[0] ?? '';
}
function observe(status: RunStatus | undefined) {
  runStatus.value = status;
}
const briefingWidth = ref(50);
</script>
<template>
        <UISplit v-model="briefingWidth" class="selected-problem" label="설명과 작업 영역 너비" :min="30" :max="70">
          <template #before><div class="briefing">
            <header class="problem-header">
              <h2 id="problem-heading" :title="selected.title">{{ selected.title }}</h2>
              <div class="problem-badges">
                <UIBadge :title="`분야: ${categoryLabel(selected.category)}`"><span class="ui-sr-only">분야: </span>{{ categoryLabel(selected.category) }}</UIBadge>
                <UIDifficultyBadge v-if="selected.difficulty" :level="selected.difficulty" />
              </div>
            </header>
            <div class="briefing-scroll" tabindex="0" role="region" aria-labelledby="problem-heading">
              <ProblemDetail :catalog="client.catalog" :slug="selected.slug" :selection-disabled="busy" @loaded="loaded" @select="emit('select', $event)" />
            </div>
          </div></template>
          <template #after>
            <div class="tool-workspace">
              <PlayPanel :player="client.player" :workspaces="client.workspaces" :slug="selected.slug" :kind="selected.kind" :answer="answer" @completed="emit('completed')" @busy="busy = $event" @status="observe">
                <template #default="{ environment, blocked }">
                  <UITabs v-model="tool" :items="tools" :blocked="blocked" label="풀이 도구" class="tool-pane">
                    <template #overlay><EnvironmentStatus :state="environment" /></template>
                    <template #actions>
                      <template v-if="tool === 'terminal'">
                        <code v-for="endpoint in runStatus?.endpoints.filter(endpoint => endpoint.url.startsWith('tcp://'))" :key="endpoint.name" class="tcp-endpoint" :title="endpoint.url">{{ endpoint.url }}</code>
                        <UIIconButton label="터미널 새로고침" icon="refresh" :busy="!!terminal?.busy" :disabled="!terminal" @click="terminal?.refresh()" />
                      </template>
                      <UIIconButton v-else-if="tool === 'files'" :label="materials?.filename ? `${materials.filename} 다운로드` : '파일 다운로드'" icon="download" :busy="!!materials?.busy" :disabled="!materials?.filename" @click="materials?.download()" />
                      <template v-else-if="tool === 'web'">
                        <UILink v-if="web?.url" :href="web.url" new-tab icon-only variant="ghost" size="compact" aria-label="문제 웹을 새 탭에서 열기" title="새 탭에서 열기"><span class="ui-sr-only">새 탭에서 열기</span></UILink>
                      </template>
                    </template>
                    <template #terminal><TerminalPanel v-if="visited.includes('terminal')" ref="terminal" :terminals="client.terminals" :slug="selected.slug" :paused="!runStatus || (runStatus.state !== 'ready' && runStatus.state !== 'running')" :foreground="tool === 'terminal'" /></template>
                    <template #files><ProblemFiles ref="materials" :catalog="client.catalog" :slug="selected.slug" :files="files" :foreground="tool === 'files'" /></template>
                    <template #web><ProblemWeb ref="web" :player="client.player" :slug="selected.slug" :status="runStatus" :active="tool === 'web'" /></template>
                  </UITabs>
                </template>
              </PlayPanel>
            </div>
          </template>
        </UISplit>
</template>
<style scoped>
.selected-problem { min-width: 0; min-height: 0; }
.briefing { height: 100%; min-width: 0; min-height: 0; display: flex; flex-direction: column; }
.problem-header { flex: none; display: flex; align-items: center; justify-content: space-between; min-height: var(--ui-workspace-header-size); gap: var(--ui-space-1); padding: var(--ui-space-2); border-bottom: 1px solid var(--ui-border); background: var(--ui-surface); }
.problem-header h2 { min-width: 0; font-size: 1rem; line-height: 1.4; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.problem-badges { flex: none; display: flex; align-items: center; gap: var(--ui-space-1); }
.briefing-scroll { flex: 1; min-height: 0; min-width: 0; overflow: auto; }
.tool-workspace { height: 100%; min-height: 0; min-width: 0; display: flex; flex-direction: column; }
.tool-pane { flex: 1; min-height: 0; }
.tcp-endpoint { max-width: 14rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 0.8rem; color: var(--ui-muted); }
@media (max-width: 48rem) {
  .briefing { height: auto; border-bottom: 1px solid var(--ui-border); }
  .problem-header { position: sticky; top: 0; z-index: 2; padding: var(--ui-space-2); }
  .briefing-scroll { overflow: visible; }
  .tool-workspace { height: 36rem; }
}
</style>
