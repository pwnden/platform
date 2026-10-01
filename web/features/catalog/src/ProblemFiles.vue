<script setup lang="ts">
import { onUnmounted, ref } from 'vue';
import type { Catalog, ProblemFile } from '@pwnden/catalog';
import { UIButton, UIPanel, UIFile, UICode } from '@pwnden/ui';
const props = defineProps<{ catalog: Catalog; slug: string; files: readonly ProblemFile[] }>();
const downloading = ref('');
const downloadFailed = ref(false);
interface Reading { open: boolean; pending: boolean; failed: boolean; content?: string; notice?: string }
const readings = ref<Record<string, Reading>>({});
const previewLimit = 1 << 20;
let active = true;
onUnmounted(() => { active = false; });
function reading(id: string): Reading {
  return readings.value[id] ?? (readings.value[id] = { open: false, pending: false, failed: false });
}
async function reveal(file: ProblemFile, open: boolean) {
  const state = reading(file.id);
  state.open = open;
  if (!open || state.pending || state.content !== undefined || state.notice) return;
  state.pending = true; state.failed = false;
  try {
    if (file.size > previewLimit) {
      state.notice = '큰 자료는 터미널 탭에서 살펴보세요. 자료는 작업 공간에 준비돼 있습니다.';
      return;
    }
    const bytes = await props.catalog.download(props.slug, file.id, previewLimit);
    if (!active) return;
    try {
      const source = new TextDecoder('utf-8', { fatal: true }).decode(bytes);
      if (/[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F]/.test(source)) throw new Error('binary');
      state.content = source;
    } catch { state.notice = '텍스트로 표시할 수 없는 자료입니다. 터미널 탭에서 분석할 수 있습니다.'; }
  } catch { if (active) state.failed = true; }
  finally { if (active) state.pending = false; }
}
async function download(file: ProblemFile) {
  if (downloading.value) return;
  downloading.value = file.id; downloadFailed.value = false;
  try {
    const bytes = await props.catalog.download(props.slug, file.id);
    if (!active) return;
    const url = URL.createObjectURL(new Blob([new Uint8Array(bytes)], { type: 'application/octet-stream' }));
    const link = document.createElement('a');
    link.href = url; link.download = file.name.split('/').at(-1) || 'download';
    document.body.append(link); link.click(); link.remove();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  } catch { if (active) downloadFailed.value = true; }
  finally { if (active) downloading.value = ''; }
}
</script>

<template>
  <UIPanel title="분석 자료" headingID="materials-heading" class="materials-panel">
    <ul class="files">
      <li v-for="file in files" :key="file.id">
        <UIFile :name="file.name" :size="file.size" :model-value="reading(file.id).open" :busy="downloading === file.id" :disabled="Boolean(downloading)" @update:model-value="open => reveal(file, open)" @download="download(file)">
          <p v-if="reading(file.id).pending" role="status">자료를 불러오는 중…</p>
          <UICode v-if="reading(file.id).content !== undefined" :source="reading(file.id).content!" :label="file.name" />
          <p v-if="reading(file.id).notice">{{ reading(file.id).notice }}</p>
          <div v-if="reading(file.id).failed" role="alert">
            <p>자료를 불러오지 못했습니다.</p>
            <UIButton size="compact" @click="reveal(file, true)">자료 다시 불러오기</UIButton>
          </div>
        </UIFile>
      </li>
    </ul>
    <p v-if="downloadFailed" role="alert">파일을 다운로드하지 못했습니다. 서버 연결을 확인하고 다시 시도하세요.</p>
  </UIPanel>
</template>

<style scoped>
.materials-panel { height: 100%; }
.materials-panel :deep(.ui-panel-body) { display: flex; flex-direction: column; gap: var(--ui-space-2); overflow-y: auto; }
.files { --ui-radius-outer: var(--ui-radius-surface); --ui-radius-inset: 1px; list-style: none; padding: 0; margin: 0; border: 1px solid var(--ui-border); border-radius: var(--ui-radius-outer); }
.files > li { --ui-file-radius-start: var(--ui-radius-flush); --ui-file-radius-end: var(--ui-radius-flush); }
.files > li:first-child { --ui-file-radius-start: max(0px, calc(var(--ui-radius-outer) - var(--ui-radius-inset))); }
.files > li:last-child { --ui-file-radius-end: max(0px, calc(var(--ui-radius-outer) - var(--ui-radius-inset))); }
.files li + li { border-top: 1px solid var(--ui-border); }
</style>
