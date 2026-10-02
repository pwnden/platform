<script setup lang="ts">
import { computed } from 'vue';
import { DisclosureRoot, DisclosureTrigger, DisclosureContent } from '@sectile/vue/disclosure';
import type { UIFileProps } from './props';
import UIButton from './UIButton.vue';
import UIIcon from './UIIcon.vue';

const props = defineProps<UIFileProps>();
defineEmits<{ 'update:modelValue': [value: boolean]; download: [] }>();
const filename = computed(() => props.name.split('/').at(-1) || props.name);
const directory = computed(() => props.name.slice(0, props.name.length - filename.value.length));
const sizeLabel = computed(() => {
  if (props.size < 1024) return `${props.size.toLocaleString('ko-KR')} B`;
  const unit = props.size < 1024 ** 2 ? 'KiB' : props.size < 1024 ** 3 ? 'MiB' : 'GiB';
  const divisor = unit === 'KiB' ? 1024 : unit === 'MiB' ? 1024 ** 2 : 1024 ** 3;
  return `${(props.size / divisor).toLocaleString('ko-KR', { maximumFractionDigits: 1 })} ${unit}`;
});
</script>

<template>
  <DisclosureRoot class="ui-file" :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)">
    <div class="ui-file-header">
      <DisclosureTrigger class="ui-disclosure ui-file-toggle" :aria-label="`${name} ${modelValue ? '미리보기 닫기' : '미리보기 열기'}`" :title="name">
        <span class="ui-disclosure-label ui-file-identity">
          <UIIcon name="chevron-right" class="ui-disclosure-chevron" />
          <span v-if="directory" class="ui-file-directory">{{ directory }}</span>
          <span class="ui-file-name">{{ filename }}</span>
        </span>
        <span class="ui-file-size" :title="`${size.toLocaleString('ko-KR')} 바이트`">{{ sizeLabel }}</span>
      </DisclosureTrigger>
      <UIButton variant="ghost" size="compact" class="ui-file-download" :busy="busy" :disabled="disabled" :aria-label="`${name} 다운로드`" :title="`${name} 다운로드`" @click="$emit('download')">
        <UIIcon :name="busy ? 'loader' : 'download'" :class="{ 'ui-file-progress': busy }" />
        <span>다운로드</span>
      </UIButton>
    </div>
    <DisclosureContent class="ui-file-preview"><slot v-if="modelValue" /></DisclosureContent>
  </DisclosureRoot>
</template>

<style scoped>
.ui-file { min-width: 0; container-type: inline-size; }
.ui-file-header { border-start-start-radius: var(--ui-file-radius-start, var(--ui-radius-surface)); border-start-end-radius: var(--ui-file-radius-start, var(--ui-radius-surface)); }
.ui-file-header:has(+ .ui-file-preview[hidden]), .ui-file-preview { border-end-start-radius: var(--ui-file-radius-end, var(--ui-radius-surface)); border-end-end-radius: var(--ui-file-radius-end, var(--ui-radius-surface)); }
.ui-file-header { display: flex; align-items: center; gap: var(--ui-space-1); padding: var(--ui-space-1); background: var(--ui-surface-raised); }
.ui-file-toggle { flex: 1; min-width: 0; display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; gap: var(--ui-space-1); height: var(--ui-control-size-compact); min-height: var(--ui-control-size-compact); padding: var(--ui-inset-compact); font-size: 0.85rem; line-height: 1.4; }
.ui-file-identity { min-width: 0; max-width: 100%; }
.ui-file-directory { flex-shrink: 2; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--ui-muted); }
.ui-file-name { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-weight: 500; }
.ui-file-size { color: var(--ui-muted); white-space: nowrap; }
.ui-file-download { gap: var(--ui-space-1); min-width: 2.5rem; }
.ui-file-download svg { flex: none; width: 1rem; height: 1rem; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
.ui-file-progress { animation: file-progress 1s linear infinite; }
@keyframes file-progress { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .ui-file-progress { animation: none; } }
.ui-file-preview { min-width: 0; border-top: 1px solid var(--ui-border); background: var(--ui-background); }
.ui-file-preview :deep(.ui-code) { border: 0; border-radius: var(--ui-radius-flush); }
.ui-file-preview :deep(.ui-code:last-child) { border-end-start-radius: var(--ui-file-radius-end, var(--ui-radius-surface)); border-end-end-radius: var(--ui-file-radius-end, var(--ui-radius-surface)); }
.ui-file-preview :deep(p), .ui-file-preview :deep([role='alert']) { padding: var(--ui-space-2); }
.ui-file-preview :deep([role='alert'] p) { padding: 0; margin-bottom: var(--ui-space-1); }
@container (max-width: 20rem) { .ui-file-download { width: var(--ui-control-size-compact); padding: 0; } .ui-file-download span { display: none; } }
</style>
