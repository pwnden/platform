<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import type { Catalog, ProblemDetail, ProblemFile } from '@pwnden/catalog';
import { UIButton, UIPanel, UIMarkdown, UIReveal, UIFile, UICode } from '@pwnden/ui';

const props = defineProps<{ catalog: Catalog; slug: string }>();
const detail = ref<ProblemDetail>();
const pending = ref(false);
const failed = ref(false);
const downloading = ref('');
const downloadFailed = ref(false);
interface Reading { open: boolean; pending: boolean; failed: boolean; content?: string; notice?: string }
const readings = ref<Record<string, Reading>>({});
const previewLimit = 1 << 20;
function reading(id: string): Reading {
  return readings.value[id] ?? (readings.value[id] = { open: false, pending: false, failed: false });
}
async function reveal(id: string, open: boolean, file?: ProblemFile) {
  const state = reading(id);
  state.open = open;
  if (!open || state.pending || state.content !== undefined || state.notice) return;
  state.pending = true;
  state.failed = false;
  try {
    if (file) {
      if (file.size > previewLimit) {
        state.notice = '큰 자료는 오른쪽 터미널에서 살펴보세요. 자료는 작업 공간에 준비돼 있습니다.';
        return;
      }
      const bytes = await props.catalog.download(props.slug, file.id, previewLimit);
      if (!active) return;
      try {
        const source = new TextDecoder('utf-8', { fatal: true }).decode(bytes);
        if (/[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F]/.test(source)) throw new Error('binary');
        state.content = source;
      } catch {
        state.notice = '텍스트로 표시할 수 없는 자료입니다. 오른쪽 터미널에서 분석할 수 있습니다.';
      }
    } else {
      const content = await props.catalog.guidance(props.slug, id);
      if (active) state.content = content;
    }
  } catch {
    if (active) state.failed = true;
  } finally {
    if (active) state.pending = false;
  }
}
let active = true;
onUnmounted(() => { active = false; });

async function load() {
  if (pending.value) return;
  pending.value = true;
  failed.value = false;
  try {
    const result = await props.catalog.detail(props.slug);
    if (active) detail.value = result;
  } catch {
    if (active) failed.value = true;
  } finally {
    if (active) pending.value = false;
  }
}

async function download(file: ProblemFile) {
  if (downloading.value) return;
  downloading.value = file.id;
  downloadFailed.value = false;
  try {
    const bytes = await props.catalog.download(props.slug, file.id);
    if (!active) return;
    const url = URL.createObjectURL(new Blob([new Uint8Array(bytes)], { type: 'application/octet-stream' }));
    const link = document.createElement('a');
    link.href = url;
    link.download = file.name.split('/').at(-1) || 'download';
    document.body.append(link);
    link.click();
    link.remove();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  } catch {
    if (active) downloadFailed.value = true;
  } finally {
    if (active) downloading.value = '';
  }
}
onMounted(load);
</script>

<template>
  <div :aria-busy="pending" class="detail-panel">
    <p v-if="pending" role="status">문제 설명과 파일을 불러오는 중…</p>
    <div v-if="failed" role="alert">
      <p>문제 설명과 파일을 불러오지 못했습니다. 서버 연결을 확인하고 다시 시도하세요.</p>
      <UIButton :disabled="pending" @click="load">상세 다시 불러오기</UIButton>
    </div>
    <template v-else-if="detail">
      <UIPanel title="문제 설명" headingID="detail-heading" :heading-level="3" class="content-section">
        <UIMarkdown v-if="detail.description" :source="detail.description" :heading-offset="2" />
        <p v-else>등록된 문제 설명이 없습니다.</p>
      </UIPanel>
      <UIPanel v-if="detail.files.length" title="분석 자료" headingID="materials-heading" :heading-level="3" class="content-section">
        <ul class="files">
          <li v-for="file in detail.files" :key="file.id">
            <UIFile :name="file.name" :size="file.size" :model-value="reading(file.id).open" :busy="downloading === file.id" :disabled="Boolean(downloading)" @update:model-value="open => reveal(file.id, open, file)" @download="download(file)">
              <p v-if="reading(file.id).pending" role="status">자료를 불러오는 중…</p>
              <UICode v-if="reading(file.id).content !== undefined" :source="reading(file.id).content!" :label="file.name" />
              <p v-if="reading(file.id).notice">{{ reading(file.id).notice }}</p>
              <div v-if="reading(file.id).failed" role="alert">
                <p>자료를 불러오지 못했습니다.</p>
                <UIButton size="compact" @click="reveal(file.id, true, file)">자료 다시 불러오기</UIButton>
              </div>
            </UIFile>
          </li>
        </ul>
        <p v-if="downloadFailed" role="alert">파일을 다운로드하지 못했습니다. 서버 연결을 확인하고 다시 시도하세요.</p>
      </UIPanel>
    </template>
    <slot name="play" />
    <template v-if="detail">
      <UIPanel v-if="detail.hintCount" title="힌트" headingID="hints-heading" :heading-level="3" class="content-section">
        <UIReveal v-for="n in detail.hintCount" :key="n" :label="`힌트 ${n}`" :model-value="reading(`hint-${n}`).open" @update:model-value="open => reveal(`hint-${n}`, open)">
          <p v-if="reading(`hint-${n}`).pending" role="status">힌트를 불러오는 중…</p>
          <UIMarkdown v-if="reading(`hint-${n}`).content" :source="reading(`hint-${n}`).content!" :heading-offset="2" />
          <div v-if="reading(`hint-${n}`).failed" role="alert">
            <p>힌트를 불러오지 못했습니다.</p>
            <UIButton size="compact" @click="reveal(`hint-${n}`, true)">힌트 다시 불러오기</UIButton>
          </div>
        </UIReveal>
      </UIPanel>
      <UIPanel v-if="detail.walkthrough" title="해설" headingID="walkthrough-heading" :heading-level="3" class="content-section">
        <UIReveal label="해설 보기 · 정답 포함" :model-value="reading('walkthrough').open" @update:model-value="open => reveal('walkthrough', open)">
          <p v-if="reading('walkthrough').pending" role="status">해설을 불러오는 중…</p>
          <UIMarkdown v-if="reading('walkthrough').content" :source="reading('walkthrough').content!" :heading-offset="2" />
          <div v-if="reading('walkthrough').failed" role="alert">
            <p>해설을 불러오지 못했습니다.</p>
            <UIButton size="compact" @click="reveal('walkthrough', true)">해설 다시 불러오기</UIButton>
          </div>
        </UIReveal>
      </UIPanel>
    </template>
  </div>
</template>

<style scoped>
.detail-panel { min-width: 0; }
.detail-panel > p, .detail-panel > [role='alert'] { padding: var(--ui-space-3); }
.content-section { --ui-panel-inset: var(--ui-space-3); border-bottom: 1px solid var(--ui-border); }
.content-section :deep(.ui-panel-heading) { background: var(--ui-surface-raised); }
.content-section :deep(.ui-reveal:first-child) { border-top: 0; }
.files { --ui-radius-outer: var(--ui-radius-surface); --ui-radius-inset: 1px; list-style: none; padding: 0; margin: 0; border: 1px solid var(--ui-border); border-radius: var(--ui-radius-outer); }
.files > li { --ui-file-radius-start: var(--ui-radius-flush); --ui-file-radius-end: var(--ui-radius-flush); }
.files > li:first-child { --ui-file-radius-start: max(0px, calc(var(--ui-radius-outer) - var(--ui-radius-inset))); }
.files > li:last-child { --ui-file-radius-end: max(0px, calc(var(--ui-radius-outer) - var(--ui-radius-inset))); }
.files li + li { border-top: 1px solid var(--ui-border); }
@media (max-width: 48rem) { .content-section { --ui-panel-inset: var(--ui-space-2); } }
</style>
