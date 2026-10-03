<script setup lang="ts">
import type { Catalog, Problem, ProblemDetail as ProblemDetails } from '@pwnden/catalog';
import { ProblemDetail, categoryLabel } from '@pwnden/catalog-feature';
import { UIBadge, UIDifficultyBadge } from '@pwnden/ui';
defineProps<{ catalog: Catalog; problem: Problem; initialDetail?: ProblemDetails | undefined; busy: boolean }>();
defineEmits<{ loaded: [detail: ProblemDetails]; select: [problem: Problem] }>();
</script>
<template>
  <div class="briefing">
    <header class="problem-header">
      <h2 id="problem-heading" :title="problem.title">{{ problem.title }}</h2>
      <div class="problem-badges">
        <UIBadge :title="`분야: ${categoryLabel(problem.category)}`"><span class="ui-sr-only">분야: </span>{{ categoryLabel(problem.category) }}</UIBadge>
        <UIDifficultyBadge v-if="problem.difficulty" :level="problem.difficulty" />
      </div>
    </header>
    <div class="briefing-scroll" tabindex="0" role="region" aria-labelledby="problem-heading">
      <ProblemDetail :catalog="catalog" :slug="problem.slug" :initial-detail="initialDetail" :selection-disabled="busy" @loaded="$emit('loaded', $event)" @select="$emit('select', $event)" />
    </div>
  </div>
</template>
<style scoped>
.briefing { height: 100%; min-width: 0; min-height: 0; display: flex; flex-direction: column; }
.problem-header { flex: none; display: flex; align-items: center; justify-content: space-between; min-height: var(--ui-workspace-header-size); gap: var(--ui-space-1); padding: var(--ui-space-2); border-bottom: 1px solid var(--ui-border); background: var(--ui-surface); }
.problem-header h2 { min-width: 0; font-size: 1rem; line-height: 1.4; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.problem-badges { flex: none; display: flex; align-items: center; gap: var(--ui-space-1); }
.briefing-scroll { flex: 1; min-height: 0; min-width: 0; overflow: auto; }
@media (max-width: 48rem) {
  .briefing { height: auto; border-bottom: 1px solid var(--ui-border); }
  .problem-header { position: sticky; top: 0; z-index: 2; padding: var(--ui-space-2); }
  .briefing-scroll { overflow: visible; }
}
</style>
