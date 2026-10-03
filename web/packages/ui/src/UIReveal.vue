<script setup lang="ts">
import { DisclosureRoot, DisclosureTrigger, DisclosureContent } from '@sectile/vue/disclosure';
import type { UIRevealProps } from './props';
import UIIcon from './UIIcon.vue';
defineProps<UIRevealProps>();
defineEmits<{ 'update:modelValue': [value: boolean] }>();
</script>

<template>
  <DisclosureRoot class="ui-reveal" :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)">
    <DisclosureTrigger class="ui-disclosure"><span class="ui-disclosure-label"><UIIcon name="chevron-right" class="ui-disclosure-chevron" /><span>{{ label }}</span></span></DisclosureTrigger>
    <DisclosureContent class="ui-reveal-content"><slot v-if="modelValue" /></DisclosureContent>
  </DisclosureRoot>
</template>

<style scoped>
.ui-reveal { min-width: 0; }
.ui-reveal > .ui-disclosure { width: 100%; }
.ui-reveal-content { min-width: 0; padding: var(--ui-space-1); }
</style>
