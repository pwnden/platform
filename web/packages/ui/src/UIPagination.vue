<script setup lang="ts">
import { PaginationRoot, PaginationPrevious, PaginationNext } from '@sectile/vue/pagination';
import type { UIPaginationProps } from './props';
defineProps<UIPaginationProps>();
defineEmits<{ 'update:modelValue': [value: number] }>();
function controlLabel(control: string) {
  const labels: Record<string, string> = { 'first-page': '처음', 'previous-page': '이전', 'next-page': '다음', 'last-page': '마지막' };
  return labels[control] ?? control;
}
</script>

<template>
  <PaginationRoot v-slot="{ page, pageCount }" class="ui-pagination" :total="total" :model-value="modelValue" :items-per-page="pageSize" :disabled="disabled" :label="label" :get-control-label="controlLabel" @update:model-value="$emit('update:modelValue', $event)">
    <p role="status" aria-live="polite">{{ total ? `${(page - 1) * pageSize + 1}–${Math.min(page * pageSize, total)} / ${total}개` : '0개' }}</p>
    <div v-if="pageCount > 1" class="ui-pagination-actions">
      <PaginationPrevious class="ui-button ui-button--ghost ui-button--compact" :disabled="disabled || page === 1">이전</PaginationPrevious>
      <PaginationNext class="ui-button ui-button--ghost ui-button--compact" :disabled="disabled || page === pageCount">다음</PaginationNext>
    </div>
  </PaginationRoot>
</template>

<style scoped>
.ui-pagination { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; min-height: var(--ui-workspace-header-size); gap: var(--ui-space-1); padding: var(--ui-space-2); border-top: 1px solid var(--ui-border); flex: none; color: var(--ui-muted); font-size: 0.85rem; }
.ui-pagination p { min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.ui-pagination-actions { display: flex; gap: calc(var(--ui-space-1) / 2); }
</style>
