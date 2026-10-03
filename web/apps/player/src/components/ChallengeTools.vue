<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { APIClient } from '@pwnden/api';
import type { ProblemFile, ProblemTool } from '@pwnden/catalog';
import type { RunStatus } from '@pwnden/play';
import { ProblemFiles } from '@pwnden/catalog-feature';
import type { ProblemFilesHandle } from '@pwnden/catalog-feature';
import { EnvironmentStatus, ProblemWeb } from '@pwnden/play-feature';
import type { EnvironmentState, ProblemWebHandle } from '@pwnden/play-feature';
import { TerminalPanel } from '@pwnden/terminal-feature';
import type { TerminalPanelHandle } from '@pwnden/terminal-feature';
import { UIIconButton, UILink, UITabs } from '@pwnden/ui';
const props = defineProps<{ client: APIClient; slug: string; files: readonly ProblemFile[]; allowed: readonly ProblemTool[]; status?: RunStatus | undefined; environment: EnvironmentState; blocked: boolean }>();
const terminal = ref<TerminalPanelHandle>();
const materials = ref<ProblemFilesHandle>();
const web = ref<ProblemWebHandle>();
const visited = ref<readonly string[]>([]);
const tool = ref('');
const labels: Record<ProblemTool, string> = { terminal: '터미널', files: '파일', web: '웹' };
const tools = computed(() => props.allowed.map(value => ({ value, label: labels[value], icon: value })));
const route = useRoute();
const router = useRouter();
watch([() => route.query.tool, () => props.allowed], ([value, allowed]) => {
  tool.value = typeof value === 'string' && allowed.includes(value as ProblemTool) ? value : allowed[0] ?? '';
}, { immediate: true });
watch(tool, value => {
  if (value && !visited.value.includes(value)) visited.value = [...visited.value, value];
  if (!value || route.query.tool === value || (!route.query.tool && value === props.allowed[0])) return;
  void router.replace({ query: { ...route.query, tool: value === props.allowed[0] ? undefined : value } });
}, { immediate: true });
</script>
<template>
  <UITabs v-model="tool" :items="tools" :blocked="blocked" label="풀이 도구" class="tool-pane">
    <template #overlay><EnvironmentStatus :state="environment" /></template>
    <template #actions>
      <template v-if="tool === 'terminal'">
        <code v-for="endpoint in status?.endpoints.filter(endpoint => endpoint.url.startsWith('tcp://'))" :key="endpoint.name" class="tcp-endpoint" :title="endpoint.url">{{ endpoint.url }}</code>
        <UIIconButton label="터미널 새로고침" icon="refresh" :busy="!!terminal?.busy" :disabled="!terminal" @click="terminal?.refresh()" />
      </template>
      <UIIconButton v-else-if="tool === 'files'" :label="materials?.filename ? `${materials.filename} 다운로드` : '파일 다운로드'" icon="download" :busy="!!materials?.busy" :disabled="!materials?.filename" @click="materials?.download()" />
      <template v-else-if="tool === 'web'">
        <UILink v-if="web?.url" :href="web.url" new-tab icon-only variant="ghost" size="compact" aria-label="문제 웹을 새 탭에서 열기" title="새 탭에서 열기"><span class="ui-sr-only">새 탭에서 열기</span></UILink>
      </template>
    </template>
    <template #terminal><TerminalPanel v-if="visited.includes('terminal')" ref="terminal" :terminals="client.terminals" :slug="slug" :paused="!status || (status.state !== 'ready' && status.state !== 'running')" :foreground="tool === 'terminal'" /></template>
    <template #files><ProblemFiles ref="materials" :catalog="client.catalog" :slug="slug" :files="files" :foreground="tool === 'files'" /></template>
    <template #web><ProblemWeb ref="web" :player="client.player" :slug="slug" :status="status" :active="tool === 'web'" /></template>
  </UITabs>
</template>
<style scoped>
.tool-pane { flex: 1; min-height: 0; }
.tcp-endpoint { max-width: 14rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 0.8rem; color: var(--ui-muted); }
</style>
