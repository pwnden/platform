<script setup lang="ts">
import { ref } from 'vue';
import type { APIClient } from '@pwnden/api';
import type { Problem } from '@pwnden/catalog';
import { ProblemList } from '@pwnden/catalog-feature';
import { UISplit } from '@pwnden/ui';
defineProps<{ client: APIClient; selectedSlug?: string | undefined; completedSlug: string; disabled: boolean }>();
defineEmits<{ select: [problem: Problem] }>();
const catalogWidth = ref(20);
</script>
<template>
  <UISplit v-model="catalogWidth" class="workspace" label="문제 목록 너비" :min="14" :max="36">
    <template #before><aside class="catalog"><ProblemList :catalog="client.catalog" :selected-slug="selectedSlug" :completed-slug="completedSlug" :selection-disabled="disabled" @select="$emit('select', $event)" /></aside></template>
    <template #after><slot /></template>
  </UISplit>
</template>
<style scoped>
.workspace { min-height: 0; }
.catalog { height: 100%; min-width: 0; min-height: 0; overflow: hidden; background: var(--ui-surface); }
@media (max-width: 48rem) { .catalog { height: 30rem; border-bottom: 1px solid var(--ui-border); } }
</style>
