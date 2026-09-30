<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import type { Catalog, ProblemDetail, ProblemFile } from '@pwnden/catalog';
import { UIButton, UIPanel } from '@pwnden/ui';

const props = defineProps<{ catalog: Catalog; slug: string; title: string }>();
const detail = ref<ProblemDetail>();
const pending = ref(false);
const failed = ref(false);
const downloading = ref('');
const downloadFailed = ref(false);
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
  <UIPanel :title="title" headingID="detail-heading" :aria-busy="pending" class="detail-panel">
    <p v-if="pending" role="status">문제 설명과 파일을 불러오는 중…</p>
    <div v-if="failed" role="alert">
      <p>문제 설명과 파일을 불러오지 못했습니다. 서버 연결을 확인하고 다시 시도하세요.</p>
      <UIButton :disabled="pending" @click="load">상세 다시 불러오기</UIButton>
    </div>
    <template v-else-if="detail">
      <h3>문제 설명</h3>
      <pre v-if="detail.description" class="description">{{ detail.description }}</pre>
      <p v-else>등록된 문제 설명이 없습니다.</p>
      <h3>배포 파일</h3>
      <ul v-if="detail.files.length" class="files">
        <li v-for="file in detail.files" :key="file.id">
          <span>{{ file.name }} <small>({{ file.size.toLocaleString('ko-KR') }} 바이트)</small></span>
          <UIButton size="compact" :busy="downloading === file.id" :disabled="Boolean(downloading)" @click="download(file)">
            {{ downloading === file.id ? '다운로드 중…' : '다운로드' }}
          </UIButton>
        </li>
      </ul>
      <p v-else>배포 파일이 없습니다. 문제 실행 후 제공되는 접속 주소를 이용하세요.</p>
      <p v-if="downloadFailed" role="alert">파일을 다운로드하지 못했습니다. 서버 연결을 확인하고 다시 시도하세요.</p>
    </template>
  </UIPanel>
</template>

<style scoped>
.detail-panel { border-bottom: 1px solid var(--ui-border); }
.detail-panel :deep(h2) { font-size: 1.2rem; color: var(--ui-foreground); }
h3 { margin: var(--ui-space-3) 0 var(--ui-space-1); color: var(--ui-muted); font-size: 0.85rem; }
h3:first-child { margin-top: 0; }
.description { margin: 0; white-space: pre-wrap; overflow-wrap: anywhere; font: inherit; max-width: 75ch; line-height: 1.85; }
.files { list-style: none; padding: 0; margin: 0; }
.files li { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: var(--ui-space-1); padding-block: var(--ui-space-1); border-top: 1px solid var(--ui-border); }
.files span { min-width: 0; overflow-wrap: anywhere; }
small { color: var(--ui-muted); }
</style>
