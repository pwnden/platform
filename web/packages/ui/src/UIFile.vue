<script setup lang="ts">
import { computed, useId } from 'vue';
import type { UIFileProps } from './props';
import UIButton from './UIButton.vue';

const props = defineProps<UIFileProps>();
defineEmits<{ 'update:modelValue': [value: boolean]; download: [] }>();
const previewID = `file-preview-${useId()}`;
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
  <div class="ui-file">
    <div class="ui-file-header">
      <button type="button" class="ui-disclosure ui-file-toggle" :aria-expanded="modelValue" :aria-controls="previewID" :aria-label="`${name} ${modelValue ? '미리보기 닫기' : '미리보기 열기'}`" :title="name" @click="$emit('update:modelValue', !modelValue)">
        <span class="ui-disclosure-label ui-file-identity">
          <svg class="ui-disclosure-chevron" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="m9 5 7 7-7 7" /></svg>
          <span v-if="directory" class="ui-file-directory">{{ directory }}</span>
          <span class="ui-file-name">{{ filename }}</span>
        </span>
        <span class="ui-file-size" :title="`${size.toLocaleString('ko-KR')} 바이트`">{{ sizeLabel }}</span>
      </button>
      <UIButton variant="ghost" size="compact" class="ui-file-download" :busy="busy" :disabled="disabled" :aria-label="`${name} 다운로드`" :title="`${name} 다운로드`" @click="$emit('download')">
        <svg viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M12 3v12m-5-5 5 5 5-5M5 17v4h14v-4" /></svg>
        <span>{{ busy ? '다운로드 중…' : '다운로드' }}</span>
      </UIButton>
    </div>
    <div :id="previewID" class="ui-file-preview" :hidden="!modelValue"><slot v-if="modelValue" /></div>
  </div>
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
.ui-file-download svg { width: 1rem; height: 1rem; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
.ui-file-preview { min-width: 0; border-top: 1px solid var(--ui-border); background: var(--ui-background); }
.ui-file-preview :deep(.ui-code) { border: 0; border-radius: var(--ui-radius-flush); }
.ui-file-preview :deep(.ui-code:last-child) { border-end-start-radius: var(--ui-file-radius-end, var(--ui-radius-surface)); border-end-end-radius: var(--ui-file-radius-end, var(--ui-radius-surface)); }
.ui-file-preview :deep(p), .ui-file-preview :deep([role='alert']) { padding: var(--ui-space-2); }
.ui-file-preview :deep([role='alert'] p) { padding: 0; margin-bottom: var(--ui-space-1); }
@container (max-width: 20rem) { .ui-file-download { width: var(--ui-control-size-compact); padding: 0; } .ui-file-download span { display: none; } }
</style>
