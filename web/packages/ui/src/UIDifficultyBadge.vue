<script setup lang="ts">
import { computed } from 'vue';
import UIBadge from './UIBadge.vue';
import { difficultyLevels } from './difficulty';
import type { UIDifficultyBadgeProps } from './props';
const props = defineProps<UIDifficultyBadgeProps>();
const difficulty = computed(() => difficultyLevels[props.level - 1]!);
</script>

<template>
  <UIBadge :tone="difficulty.tone" :title="`난이도: ${difficulty.label}`"><span class="ui-sr-only">난이도: </span>{{ difficulty.label }}</UIBadge>
</template>
