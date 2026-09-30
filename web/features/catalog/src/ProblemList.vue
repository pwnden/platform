<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import type { Catalog, Problem } from '@pwnden/catalog';
import { UIButton, UIPanel, UIStatus } from '@pwnden/ui';

const props = defineProps<{ catalog: Catalog; selectionDisabled?: boolean; selectedSlug?: string | undefined }>();
const emit = defineEmits<{ select: [problem: Problem] }>();
const problems = ref<readonly Problem[]>([]);
const pending = ref(false);
const failed = ref(false);
let active = true;
onUnmounted(() => { active = false; });

async function load() {
  if (pending.value) return;
  pending.value = true;
  failed.value = false;
  try {
    const items = await props.catalog.list();
    if (active) problems.value = items;
  } catch {
    if (active) failed.value = true;
  } finally {
    if (active) pending.value = false;
  }
}
onMounted(load);
</script>

<template>
  <UIPanel title="문제 목록" headingID="catalog-heading" :aria-busy="pending" class="catalog-panel">
    <template #actions><UIStatus>{{ problems.length }}개</UIStatus></template>
    <UIButton variant="ghost" size="compact" :busy="pending" @click="load">{{ pending ? '불러오는 중…' : '목록 새로고침' }}</UIButton>
    <p v-if="failed" role="alert">목록을 불러오지 못했습니다. 서버 연결을 확인하고 다시 시도하세요.</p>
    <p v-else-if="!pending && problems.length === 0">등록된 문제가 없습니다.</p>
    <ul v-if="problems.length" class="problem-list">
      <li v-for="problem in problems" :key="problem.slug">
        <UIButton variant="row" :aria-pressed="selectedSlug === problem.slug" :disabled="selectionDisabled" @click="emit('select', problem)">
          <span class="problem-entry"><span class="problem-title" :title="problem.title">{{ problem.title }}</span><span class="problem-slug" :title="problem.slug">{{ problem.slug }}</span><span class="problem-meta">{{ problem.category }} · {{ problem.kind === 'file' ? '파일 문제' : '서비스 문제' }}</span></span>
        </UIButton>
      </li>
    </ul>
  </UIPanel>
</template>

<style scoped>
.catalog-panel { height: 100%; }
.catalog-panel :deep(.ui-panel-body) { padding: var(--ui-space-1); }
.problem-list { padding: 0; margin: var(--ui-space-1) 0; list-style: none; display: grid; gap: 0.35rem; }
.problem-entry { display: grid; gap: 0.4rem; padding-block: 0.6rem; min-width: 0; width: 100%; }
.problem-title, .problem-slug, .problem-meta { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.problem-title { font-weight: 500; }
.problem-slug { color: var(--ui-muted); font-size: 0.8rem; }
.problem-meta { color: var(--ui-muted); font-size: 0.8rem; }
[role='alert'], p { margin: var(--ui-space-2); }
</style>
