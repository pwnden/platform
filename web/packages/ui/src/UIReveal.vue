<script setup lang="ts">
import { DisclosureRoot, DisclosureTrigger, DisclosureContent } from '@sectile/vue/disclosure';
import type { UIRevealProps } from './props';
defineProps<UIRevealProps>();
defineEmits<{ 'update:modelValue': [value: boolean] }>();
</script>

<template>
  <DisclosureRoot class="ui-reveal" :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)">
    <DisclosureTrigger class="ui-disclosure"><span class="ui-disclosure-label"><svg class="ui-disclosure-chevron" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="m9 5 7 7-7 7" /></svg><span>{{ label }}</span></span></DisclosureTrigger>
    <DisclosureContent class="ui-reveal-content"><slot v-if="modelValue" /></DisclosureContent>
  </DisclosureRoot>
</template>

<style scoped>
.ui-reveal { min-width: 0; border-block-start: 1px solid var(--ui-border); }
.ui-reveal > .ui-disclosure { width: 100%; }
.ui-reveal-content { min-width: 0; padding: var(--ui-space-1); }
</style>
