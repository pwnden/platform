<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import type { Catalog, Problem } from '@pwnden/catalog';
import { UIButton, UIPanel, UIMarkdown } from '@pwnden/ui';

const props = defineProps<{ catalog: Catalog; problem: Problem }>();
const readings = ref<Record<string, { pending: boolean; failed: boolean; content?: string }>>({});
let active = true;
onUnmounted(() => { active = false; });
function reading(name: string) {
  if (!readings.value[name]) readings.value[name] = { pending: false, failed: false };
  return readings.value[name]!;
}
async function load(name: string) {
  const state = reading(name);
  if (state.pending || state.content !== undefined) return;
  state.pending = true; state.failed = false;
  try {
    const result = await props.catalog.guidance(props.problem.slug, `cli-${name}`);
    if (active) state.content = result;
  } catch { if (active) state.failed = true; }
  finally { if (active) state.pending = false; }
}
onMounted(() => { for (const name of props.problem.cli ?? []) void load(name); });
</script>

<template>
  <UIPanel v-if="problem.cli?.length" title="참고 자료" headingID="references-heading" :heading-level="3" class="references-panel">
    <p class="references-intro">이 문제의 터미널에서 쓸 수 있는 명령입니다. 어떤 명령으로 어떻게 조사할지는 직접 정합니다.</p>
    <div class="reference-list">
      <section v-for="name in problem.cli" :key="name" class="reference-item" :aria-labelledby="`cli-${name}-heading`">
        <p v-if="reading(name).pending" role="status">{{ name }} 설명을 불러오는 중…</p>
        <UIMarkdown v-if="reading(name).content" :source="reading(name).content!" :heading-offset="3" />
        <div v-else-if="reading(name).failed" role="alert">
          <p>{{ name }} 설명을 불러오지 못했습니다.</p>
          <UIButton size="compact" @click="load(name)">다시 불러오기</UIButton>
        </div>
      </section>
    </div>
  </UIPanel>
</template>

<style scoped>
.references-panel { --ui-panel-inset: var(--ui-space-3); }
.references-intro { margin: 0 0 var(--ui-space-2); color: var(--ui-muted); font-size: 0.8125rem; }
.reference-list { display: grid; gap: var(--ui-space-2); }
.reference-item { margin-top: var(--ui-space-1); border-top: 1px solid var(--ui-border); }
.reference-item:first-child { margin-top: 0; border-top: none; }
</style>
