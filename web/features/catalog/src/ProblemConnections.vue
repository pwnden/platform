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
  if (!readings.value[id]) readings.value[id] = { open: false, pending: false, failed: false };
  return readings.value[id]!;
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
async function revealGoals(open: boolean) {
  goalsOpen.value = open;
  if (open) await Promise.all((props.problem.learning?.teaches ?? []).map(concept => reveal(concept.id, true)));
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
      <div v-if="pending || failed" class="connection-status" aria-live="polite">
        <p v-if="pending" role="status">연결된 문제를 불러오는 중…</p>
        <div v-else-if="failed" role="alert">
          <p>문제 연결을 불러오지 못했습니다.</p>
          <UIButton size="compact" @click="load">연결 다시 불러오기</UIButton>
        </div>
      </div>
      <div v-if="!pending && !failed" class="problem-links" aria-label="선수 지식에 따른 문제 연결">
        <section class="path-group" aria-labelledby="before-heading">
          <h4 id="before-heading">먼저 풀어볼 문제</h4>
          <div class="connection-list">
            <UIButton v-for="item in links.before" :key="item.slug" variant="row" class="connection-button" :disabled="disabled" @click="emit('select', item)">{{ item.title }}<span v-if="item.solvedAt" class="completed"> · 완료</span></UIButton>
            <p v-if="!links.before.length" class="no-connections">연결된 문제 없음</p>
          </div>
        </section>
        <section class="path-group" aria-labelledby="after-heading">
          <h4 id="after-heading">이어서 풀어볼 문제</h4>
          <div class="connection-list">
            <UIButton v-for="item in links.after" :key="item.slug" variant="row" class="connection-button" :disabled="disabled" @click="emit('select', item)">{{ item.title }}<span v-if="item.solvedAt" class="completed"> · 완료</span></UIButton>
            <p v-if="!links.after.length" class="no-connections">연결된 문제 없음</p>
          </div>
        </section>
        <section v-if="links.related.length" class="path-group" aria-labelledby="related-heading">
          <h4 id="related-heading">함께 풀어볼 문제</h4>
          <div class="connection-list">
            <UIButton v-for="item in links.related" :key="item.slug" variant="row" class="connection-button" :disabled="disabled" @click="emit('select', item)">{{ item.title }}<span v-if="item.solvedAt" class="completed"> · 완료</span></UIButton>
          </div>
        </section>
      </div>
      <UIReveal v-if="problem.learning?.teaches.length" :model-value="goalsOpen" label="학습 목표 보기 · 풀이 원리 포함" @update:model-value="revealGoals">
        <div class="goal-notes">
          <div v-for="concept in problem.learning.teaches" :key="concept.id" class="goal-note">
            <p v-if="reading(concept.id).pending" role="status">{{ concept.title }} 설명을 불러오는 중…</p>
            <UIMarkdown v-if="reading(concept.id).content" :source="reading(concept.id).content!" :heading-offset="3" />
            <div v-if="reading(concept.id).failed" role="alert"><p>{{ concept.title }} 설명을 불러오지 못했습니다.</p><UIButton size="compact" @click="reveal(concept.id, true)">설명 다시 불러오기</UIButton></div>
          </div>
        </div>
      </UIReveal>
    </div>
  </UIPanel>
</template>

<style scoped>
.connections-panel { container-type: inline-size; }
.learning-body { display: grid; gap: var(--ui-space-3); }
.knowledge, .problem-links, .connection-list, .goal-notes { display: grid; align-content: start; gap: var(--ui-space-1); min-width: 0; }
h4 { margin: 0; font-size: 0.875rem; font-weight: 600; color: var(--ui-muted); }
.problem-links { gap: var(--ui-space-2); }
.path-group { display: grid; grid-template-columns: 9rem minmax(0, 1fr); gap: var(--ui-space-1); align-items: start; min-width: 0; }
.path-group h4 { display: flex; align-items: center; min-height: var(--ui-control-size); }
.connection-list { gap: 0; }
.connection-button { white-space: normal; overflow-wrap: anywhere; text-align: left; }
.no-connections { display: flex; align-items: center; min-height: var(--ui-control-size); font-size: 0.875rem; color: var(--ui-muted); }
.completed { color: var(--ui-success); }
.connection-status { font-size: 0.875rem; }
.goal-notes { gap: var(--ui-space-3); }
@container (max-width: 28rem) {
  .path-group { grid-template-columns: minmax(0, 1fr); gap: 0; }
  .path-group h4 { min-height: 0; }
}
</style>
