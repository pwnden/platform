<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue';
import type { Catalog, ProblemFile } from '@pwnden/catalog';
import { UIButton, UISelect, UICode } from '@pwnden/ui';
import type { ProblemFilesHandle } from './props';
const props = withDefaults(defineProps<{ catalog: Catalog; slug: string; files: readonly ProblemFile[]; foreground?: boolean }>(), { foreground: true });
const selected = ref('');
const current = computed(() => props.files.find(file => file.id === selected.value));
const downloading = ref('');
const downloadFailed = ref(false);
interface Reading { pending: boolean; failed: boolean; content?: string; notice?: string }
const readings = ref<Record<string, Reading>>({});
const previewLimit = 1 << 20;
let active = true;
onUnmounted(() => { active = false; });
function reading(id: string): Reading {
  readings.value[id] ??= { pending: false, failed: false };
  return readings.value[id]!;
}
async function reveal(file: ProblemFile) {
  const state = reading(file.id);
  if (state.pending || state.content !== undefined || state.notice) return;
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
async function download() {
  const file = current.value;
  if (!file || downloading.value) return;
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
watch(() => props.files, files => {
  if (!files.some(file => file.id === selected.value)) selected.value = files[0]?.id ?? '';
}, { immediate: true });
watch([() => props.foreground, current], () => {
  if (props.foreground && current.value) void reveal(current.value);
}, { immediate: true });
const handle: ProblemFilesHandle = {
  get busy() { return !!downloading.value; },
  get filename() { return current.value?.name ?? ''; },
  download,
};
defineExpose(handle);
</script>

<template>
  <section class="materials-panel" aria-label="분석 자료">
    <UISelect v-if="files.length > 1" id="material-file" label="파일" v-model="selected" :options="files.map(file => ({ value: file.id, label: file.name }))" />
    <p v-else-if="current" class="filename" :title="current.name">{{ current.name }}</p>
    <template v-if="current">
      <p v-if="reading(current.id).pending" role="status">자료를 불러오는 중…</p>
      <UICode v-if="reading(current.id).content !== undefined" class="material-source" :source="reading(current.id).content!" :label="current.name" />
      <p v-if="reading(current.id).notice">{{ reading(current.id).notice }}</p>
      <div v-if="reading(current.id).failed" role="alert">
        <p>자료를 불러오지 못했습니다.</p>
        <UIButton size="compact" @click="reveal(current)">자료 다시 불러오기</UIButton>
      </div>
    </template>
    <p v-if="downloadFailed" role="alert">파일을 다운로드하지 못했습니다. 서버 연결을 확인하고 다시 시도하세요.</p>
  </section>
</template>

<style scoped>
.materials-panel { box-sizing: border-box; height: 100%; min-height: 0; min-width: 0; display: flex; flex-direction: column; gap: var(--ui-space-2); padding: var(--ui-space-2); overflow-y: auto; }
.filename { flex: none; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--ui-muted); font-family: var(--ui-font-mono); font-size: 0.85rem; }
.material-source { flex: 1; min-height: 0; max-height: none; }
</style>
