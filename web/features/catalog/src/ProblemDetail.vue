<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import type { Catalog, Problem, ProblemDetail } from '@pwnden/catalog';
import { UIButton, UIPanel, UIMarkdown, UIReveal } from '@pwnden/ui';
import ProblemConnections from './ProblemConnections.vue';

const props = defineProps<{ catalog: Catalog; slug: string; selectionDisabled?: boolean; initialDetail?: ProblemDetail | undefined }>();
const emit = defineEmits<{ loaded: [detail: ProblemDetail]; select: [problem: Problem] }>();
const detail = ref<ProblemDetail | undefined>(props.initialDetail);
const pending = ref(false);
const failed = ref(false);
interface Reading { open: boolean; pending: boolean; failed: boolean; content?: string }
const readings = ref<Record<string, Reading>>({});
function reading(id: string): Reading {
  return readings.value[id] ?? (readings.value[id] = { open: false, pending: false, failed: false });
}
async function reveal(id: string, open: boolean) {
  const state = reading(id);
  state.open = open;
  if (!open || state.pending || state.content !== undefined) return;
  state.pending = true;
  state.failed = false;
  try {
    const content = await props.catalog.guidance(props.slug, id);
    if (active) state.content = content;
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
    if (active) { detail.value = result; emit('loaded', result); }
  } catch {
    if (active) failed.value = true;
  } finally {
    if (active) pending.value = false;
  }
}

onMounted(() => { if (detail.value) emit('loaded', detail.value); else void load(); });
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
    </template>
    <slot name="play" />
    <template v-if="detail">
      <ProblemConnections v-if="detail.learning" :catalog="catalog" :problem="detail" :disabled="selectionDisabled" class="content-section" @select="emit('select', $event)" />
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
@media (max-width: 48rem) { .content-section { --ui-panel-inset: var(--ui-space-2); } }
</style>
