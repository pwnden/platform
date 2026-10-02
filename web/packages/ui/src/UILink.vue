<script setup lang="ts">
import type { UILinkProps } from './props';
import UIIcon from './UIIcon.vue';
withDefaults(defineProps<UILinkProps>(), { variant: 'secondary', size: 'default', newTab: false, iconOnly: false });
</script>

<template>
  <a class="ui-button ui-link" :class="[`ui-button--${variant}`, `ui-button--${size}`, { 'ui-icon-control': iconOnly }]" :href="href" :target="newTab ? '_blank' : undefined" :rel="newTab ? 'noopener noreferrer' : undefined">
    <slot />
    <UIIcon v-if="newTab" name="external" />
  </a>
</template>

<style scoped>
.ui-link { gap: var(--ui-space-1); text-decoration: none; }
svg { flex: none; width: 1rem; height: 1rem; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
</style>
