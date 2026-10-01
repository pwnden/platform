<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import type { Catalog, Problem } from '@pwnden/catalog';
import { UIButton, UITextField, UISelect, UIPanel, UIIconButton } from '@pwnden/ui';
import { categoryLabel, filterProblems } from './browse';

const props = defineProps<{ catalog: Catalog; selectionDisabled?: boolean; selectedSlug?: string | undefined }>();
const emit = defineEmits<{ select: [problem: Problem] }>();
const problems = ref<readonly Problem[]>([]);
const pending = ref(false);
const failed = ref(false);
const query = ref('');
const category = ref('');
const page = ref(1);
const pageSize = 20;
const results = ref<HTMLElement>();
const filtered = computed(() => filterProblems(problems.value, query.value, category.value));
const pageCount = computed(() => Math.max(1, Math.ceil(filtered.value.length / pageSize)));
const currentPage = computed(() => Math.min(page.value, pageCount.value));
const offset = computed(() => (currentPage.value - 1) * pageSize);
const groups = computed(() => {
  const result = new Map<string, Problem[]>();
  for (const problem of filtered.value.slice(offset.value, offset.value + pageSize)) {
    const group = result.get(problem.category) ?? [];
    group.push(problem);
    result.set(problem.category, group);
  }
  return [...result].map(([category, problems]) => ({ category, problems }));
});
const options = computed(() => {
  const counts = new Map<string, number>();
  for (const problem of problems.value) counts.set(problem.category, (counts.get(problem.category) ?? 0) + 1);
  return [{ value: '', label: '전체 분야' }, ...[...counts].sort(([a], [b]) => categoryLabel(a).localeCompare(categoryLabel(b), 'ko'))
    .map(([value, count]) => ({ value, label: `${categoryLabel(value)} (${count})` }))];
});
watch([query, category], () => { page.value = 1; });
watch([query, category, currentPage], () => { if (results.value) results.value.scrollTop = 0; }, { flush: 'post' });
let active = true;
onUnmounted(() => { active = false; });

async function load() {
  if (pending.value) return;
  pending.value = true;
  failed.value = false;
  try {
    const items = await props.catalog.list();
    if (active) {
      problems.value = items;
      if (category.value && !items.some(problem => problem.category === category.value)) category.value = '';
    }
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
    <template #actions><UIIconButton label="문제 목록 새로고침" icon="refresh" :busy="pending" @click="load" /></template>
    <div class="catalog-filters">
      <UITextField id="problem-search" v-model="query" label="검색" type="search" placeholder="문제 이름 또는 키워드" />
      <UISelect id="problem-category" v-model="category" label="분야" :options="options" />
    </div>
    <div class="catalog-viewport">
      <p v-if="failed" class="catalog-notice" role="alert">목록을 불러오지 못했습니다. 새로고침으로 다시 시도하세요.</p>
    <div ref="results" class="catalog-results" tabindex="0" role="region" aria-label="문제 검색 결과">
      <p v-if="!pending && problems.length === 0">등록된 문제가 없습니다.</p>
      <div v-else-if="!pending && !filtered.length" class="no-results">
        <p role="status">검색 결과가 없습니다.</p>
        <UIButton variant="ghost" size="compact" @click="query = ''; category = ''">검색 조건 초기화</UIButton>
      </div>
      <section v-for="group in groups" :key="group.category" class="problem-group" :aria-label="categoryLabel(group.category)">
        <h3>{{ categoryLabel(group.category) }}</h3>
        <ul class="problem-list">
          <li v-for="problem in group.problems" :key="problem.slug">
            <UIButton variant="row" :aria-pressed="selectedSlug === problem.slug" :disabled="selectionDisabled" @click="emit('select', problem)">
              <span class="problem-title" :title="problem.title">{{ problem.title }}</span>
            </UIButton>
          </li>
        </ul>
      </section>
    </div>
    </div>
    <nav class="catalog-pagination" aria-label="문제 목록 페이지">
      <p role="status" aria-live="polite">{{ filtered.length ? `${offset + 1}–${Math.min(offset + pageSize, filtered.length)} / ${filtered.length}개` : '0개' }}</p>
      <div v-if="pageCount > 1" class="page-actions">
        <UIButton variant="ghost" size="compact" :disabled="currentPage === 1 || pending" @click="page = currentPage - 1">이전</UIButton>
        <UIButton variant="ghost" size="compact" :disabled="currentPage === pageCount || pending" @click="page = currentPage + 1">다음</UIButton>
      </div>
    </nav>
  </UIPanel>
</template>

<style scoped>
.catalog-panel { height: 100%; }
.catalog-panel :deep(.ui-panel-body) { padding: 0; display: flex; flex-direction: column; overflow: hidden; }
.catalog-filters { padding: var(--ui-space-2); display: grid; gap: var(--ui-space-2); border-bottom: 1px solid var(--ui-border); flex: none; }
.catalog-viewport { position: relative; flex: 1; min-height: 0; }
.catalog-results { height: 100%; min-height: 0; overflow-y: auto; padding: var(--ui-space-2); }
.catalog-notice { position: absolute; inset: 0; z-index: 1; padding: var(--ui-space-2); background: var(--ui-surface); overflow: auto; }
.problem-group + .problem-group { margin-top: var(--ui-space-3); }
.problem-group h3 { margin: 0; padding: var(--ui-inset-control); border: 1px solid transparent; color: var(--ui-muted); font-size: 0.85rem; font-weight: 600; }
.problem-list { padding: 0; margin: 0; list-style: none; display: grid; gap: 0.25rem; }
.problem-title { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 500; }
.catalog-pagination { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; min-height: var(--ui-workspace-header-size); gap: var(--ui-space-1); padding: var(--ui-space-2); border-top: 1px solid var(--ui-border); flex: none; color: var(--ui-muted); font-size: 0.85rem; }
.catalog-pagination p { min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.page-actions { display: flex; gap: 0.25rem; }
.catalog-results > p, .no-results { padding: 0; }
.no-results { display: grid; gap: var(--ui-space-1); }
@media (width > 48rem) and (height <= 32rem) {
  .catalog-panel :deep(.ui-panel-body) { display: block; overflow-y: auto; }
  .catalog-results { overflow: visible; }
}
</style>
