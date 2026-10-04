<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { connections } from '@pwnden/catalog';
import type { Catalog, Problem } from '@pwnden/catalog';
import { UIButton, UIPanel, UIReveal, UIMarkdown, UIIcon, UIDifficultyBadge } from '@pwnden/ui';
import { categoryLabel } from './browse';

const props = defineProps<{ catalog: Catalog; problem: Problem; disabled?: boolean }>();
const emit = defineEmits<{ select: [problem: Problem] }>();
const catalog = ref<readonly Problem[]>([]);
const pending = ref(false), failed = ref(false);
const goalsOpen = ref(false);
const links = computed(() => connections(props.problem, catalog.value));
const groups = computed(() => [
  { id: 'before', title: '먼저 풀어볼 문제', description: '이 문제에 필요한 기초를 연습할 수 있습니다.', items: links.value.before },
  { id: 'after', title: '이어서 풀어볼 문제', description: '이 문제에서 배운 지식을 다음 문제에 활용해 보세요.', items: links.value.after },
  { id: 'related', title: '함께 풀어볼 문제', description: '같거나 연관된 지식을 다른 문제에서 연습해 보세요.', items: links.value.related },
].filter(group => group.items.length));
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
      <UIReveal v-if="problem.learning?.teaches.length" :model-value="goalsOpen" label="학습 목표 보기 · 풀이 원리 포함" @update:model-value="revealGoals">
        <div class="goal-notes">
          <div v-for="concept in problem.learning.teaches" :key="concept.id" class="goal-note">
            <p v-if="reading(concept.id).pending" role="status">{{ concept.title }} 설명을 불러오는 중…</p>
            <UIMarkdown v-if="reading(concept.id).content" :source="reading(concept.id).content!" :heading-offset="3" />
            <div v-if="reading(concept.id).failed" role="alert"><p>{{ concept.title }} 설명을 불러오지 못했습니다.</p><UIButton size="compact" @click="reveal(concept.id, true)">설명 다시 불러오기</UIButton></div>
          </div>
        </div>
      </UIReveal>
      <div v-if="pending || failed" class="connection-status" aria-live="polite">
        <p v-if="pending" role="status">연결된 문제를 불러오는 중…</p>
        <div v-else-if="failed" role="alert">
          <p>문제 연결을 불러오지 못했습니다.</p>
          <UIButton size="compact" @click="load">연결 다시 불러오기</UIButton>
        </div>
      </div>
      <div v-if="!pending && !failed && groups.length" class="problem-links" aria-label="선수 지식에 따른 문제 연결">
        <section v-for="group in groups" :key="group.id" class="path-group" :aria-labelledby="`${group.id}-heading`">
          <div class="path-heading">
            <h4 :id="`${group.id}-heading`">{{ group.title }}</h4>
            <span class="path-count">{{ group.items.length }}개</span>
          </div>
          <p class="path-description">{{ group.description }}</p>
          <ul class="connection-list">
            <li v-for="item in group.items" :key="item.slug">
              <UIButton variant="row" class="connection-button" :disabled="disabled" @click="emit('select', item)">
                <span class="connection-content">
                  <span class="connection-title">{{ item.title }}</span>
                  <span class="connection-meta">
                    <span>{{ categoryLabel(item.category) }}</span>
                    <span>{{ item.kind === 'file' ? '파일 분석' : '서비스 실습' }}</span>
                    <UIDifficultyBadge v-if="item.difficulty" :level="item.difficulty" />
                    <span v-else>난이도 미지정</span>
                    <span class="connection-progress" :class="{ completed: item.solvedAt }"><UIIcon v-if="item.solvedAt" name="check" :size="14" />{{ item.solvedAt ? '해결 완료' : '미해결' }}</span>
                  </span>
                </span>
                <UIIcon name="chevron-right" class="connection-arrow" />
              </UIButton>
            </li>
          </ul>
        </section>
      </div>
    </div>
  </UIPanel>
</template>

<style scoped>
.learning-body { display: grid; gap: var(--ui-space-2); }
.knowledge, .problem-links, .connection-list, .goal-notes { display: grid; align-content: start; gap: var(--ui-space-1); min-width: 0; }
h4 { margin: 0; font-size: 0.875rem; font-weight: 600; color: var(--ui-muted); }
.problem-links { gap: var(--ui-space-3); }
.path-group { display: grid; gap: var(--ui-space-1); align-content: start; min-width: 0; }
.path-heading { display: flex; align-items: baseline; gap: var(--ui-space-1); }
.path-heading h4 { color: var(--ui-foreground); }
.path-count, .path-description { color: var(--ui-muted); font-size: 0.8125rem; }
.path-description { margin: 0; }
.connection-list { padding: 0; margin: 0; list-style: none; gap: 0; border-top: 1px solid var(--ui-border); }
.connection-list li { min-width: 0; border-bottom: 1px solid var(--ui-border); }
.connection-button { width: 100%; max-width: 100%; height: auto; padding: var(--ui-space-1); white-space: normal; text-align: left; }
.connection-content { flex: 1; min-width: 0; display: grid; gap: var(--ui-space-1); }
.connection-title { color: var(--ui-foreground); font-weight: 500; overflow-wrap: anywhere; }
.connection-meta { display: flex; flex-wrap: wrap; align-items: center; gap: var(--ui-space-1); color: var(--ui-muted); font-size: 0.8125rem; }
.connection-progress { display: inline-flex; align-items: center; gap: var(--ui-space-1); }
.connection-arrow { color: var(--ui-muted); }
.completed { color: var(--ui-success); }
.connection-status { font-size: 0.875rem; }
.goal-notes { gap: var(--ui-space-3); }
</style>
