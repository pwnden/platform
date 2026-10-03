<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { connections } from '@pwnden/catalog';
import type { Catalog, Problem } from '@pwnden/catalog';
import { UIButton, UIPanel, UIReveal, UIMarkdown } from '@pwnden/ui';

const props = defineProps<{ catalog: Catalog; problem: Problem; disabled?: boolean }>();
const emit = defineEmits<{ select: [problem: Problem] }>();
const catalog = ref<readonly Problem[]>([]);
const pending = ref(false), failed = ref(false);
const goalsOpen = ref(false);
const links = computed(() => connections(props.problem, catalog.value));
const readings = ref<Record<string, { open: boolean; pending: boolean; failed: boolean; content?: string }>>({});
let active = true;
onUnmounted(() => { active = false; });
function reading(id: string) {
  return readings.value[id] ?? (readings.value[id] = { open: false, pending: false, failed: false });
}
async function reveal(id: string, open: boolean) {
  const state = reading(id);
  state.open = open;
  if (!open || state.pending || state.content !== undefined) return;
  state.pending = true; state.failed = false;
  try {
    const result = await props.catalog.guidance(props.problem.slug, `concept-${id}`);
    if (active) state.content = result;
  } catch { if (active) state.failed = true; }
  finally { if (active) state.pending = false; }
}
async function load() {
  if (pending.value) return;
  pending.value = true; failed.value = false;
  try {
    const result = await props.catalog.list();
    if (active) catalog.value = result;
  } catch { if (active) failed.value = true; }
  finally { if (active) pending.value = false; }
}
onMounted(load);
</script>

<template>
  <UIPanel title="학습 연결" headingID="connections-heading" :heading-level="3" class="connections-panel">
    <div class="learning-body">
      <div v-if="problem.learning?.requires.length" class="knowledge">
        <h4>선수 지식</h4>
        <UIReveal v-for="concept in problem.learning.requires" :key="concept.id" :label="concept.title" :model-value="reading(concept.id).open" @update:model-value="open => reveal(concept.id, open)">
          <p v-if="reading(concept.id).pending" role="status">설명을 불러오는 중…</p>
          <UIMarkdown v-if="reading(concept.id).content" :source="reading(concept.id).content!" :heading-offset="3" />
          <div v-if="reading(concept.id).failed" role="alert">
            <p>선수 지식 설명을 불러오지 못했습니다.</p>
            <UIButton size="compact" @click="reveal(concept.id, true)">설명 다시 불러오기</UIButton>
          </div>
        </UIReveal>
      </div>
      <div class="connection-status" aria-live="polite">
        <p v-if="pending" role="status">연결된 문제를 불러오는 중…</p>
        <div v-else-if="failed" role="alert">
          <p>문제 연결을 불러오지 못했습니다.</p>
          <UIButton size="compact" @click="load">연결 다시 불러오기</UIButton>
        </div>
      </div>
      <div v-if="!pending && !failed" class="learning-path" aria-label="선수 지식에 따른 문제 연결">
        <section class="path-group" aria-labelledby="before-heading">
          <h4 id="before-heading">먼저 풀어볼 문제</h4>
          <UIButton v-for="item in links.before" :key="item.slug" variant="row" class="connection-button" :disabled="disabled" @click="emit('select', item)">{{ item.title }}<span v-if="item.solvedAt" class="completed"> · 완료</span></UIButton>
          <p v-if="!links.before.length" class="no-connections">연결된 문제 없음</p>
        </section>
        <svg class="path-arrow" viewBox="0 0 24 24" aria-hidden="true"><path d="M3 12h17m-6-6 6 6-6 6" fill="none" stroke="currentColor" stroke-width="1.5" /></svg>
        <section class="path-group current" aria-labelledby="current-heading"><h4 id="current-heading">현재 문제</h4><p>{{ problem.title }}</p></section>
        <svg class="path-arrow" viewBox="0 0 24 24" aria-hidden="true"><path d="M3 12h17m-6-6 6 6-6 6" fill="none" stroke="currentColor" stroke-width="1.5" /></svg>
        <section class="path-group" aria-labelledby="after-heading">
          <h4 id="after-heading">이어서 풀어볼 문제</h4>
          <UIButton v-for="item in links.after" :key="item.slug" variant="row" class="connection-button" :disabled="disabled" @click="emit('select', item)">{{ item.title }}<span v-if="item.solvedAt" class="completed"> · 완료</span></UIButton>
          <p v-if="!links.after.length" class="no-connections">연결된 문제 없음</p>
        </section>
      </div>
      <section v-if="links.related.length" class="path-group" aria-labelledby="related-heading">
        <h4 id="related-heading">함께 풀어볼 문제</h4>
        <UIButton v-for="item in links.related" :key="item.slug" variant="row" class="connection-button" :disabled="disabled" @click="emit('select', item)">{{ item.title }}<span v-if="item.solvedAt" class="completed"> · 완료</span></UIButton>
      </section>
      <UIReveal v-if="problem.learning?.teaches.length" v-model="goalsOpen" label="학습 목표 보기 · 풀이 원리 포함">
        <UIReveal v-for="concept in problem.learning.teaches" :key="concept.id" :label="concept.title" :model-value="reading(concept.id).open" @update:model-value="open => reveal(concept.id, open)">
          <p v-if="reading(concept.id).pending" role="status">설명을 불러오는 중…</p>
          <UIMarkdown v-if="reading(concept.id).content" :source="reading(concept.id).content!" :heading-offset="3" />
          <div v-if="reading(concept.id).failed" role="alert"><p>학습 목표 설명을 불러오지 못했습니다.</p><UIButton size="compact" @click="reveal(concept.id, true)">설명 다시 불러오기</UIButton></div>
        </UIReveal>
      </UIReveal>
    </div>
  </UIPanel>
</template>

<style scoped>
.connections-panel { container-type: inline-size; }
.learning-body { display: grid; gap: var(--ui-space-3); }
.knowledge, .path-group { display: grid; align-content: start; gap: var(--ui-space-1); min-width: 0; }
h4 { font-size: 0.875rem; font-weight: 600; color: var(--ui-muted); }
.learning-path { display: grid; grid-template-columns: minmax(0, 1fr) 1.5rem minmax(0, 1fr) 1.5rem minmax(0, 1fr); gap: var(--ui-space-1); align-items: start; }
.path-arrow { width: 1.5rem; height: 1.5rem; color: var(--ui-accent); }
.current p { color: var(--ui-accent); overflow-wrap: anywhere; }
.connection-button { white-space: normal; overflow-wrap: anywhere; text-align: left; }
.no-connections { font-size: 0.8rem; color: var(--ui-muted); }
.completed { color: var(--ui-success); }
.connection-status { min-height: var(--ui-control-size-compact, 1.75rem); font-size: 0.8rem; }
@container (max-width: 32rem) {
  .learning-path { grid-template-columns: minmax(0, 1fr); }
  .path-arrow { transform: rotate(90deg); }
}
</style>
