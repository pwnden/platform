<script setup lang="ts">
import type { UILinkProps } from './props';
withDefaults(defineProps<UILinkProps>(), { variant: 'secondary', size: 'default', newTab: false });
</script>

<template>
  <a class="ui-button ui-link" :class="[`ui-button--${variant}`, `ui-button--${size}`]" :href="href" :target="newTab ? '_blank' : undefined" :rel="newTab ? 'noopener noreferrer' : undefined">
    <slot />
    <svg v-if="newTab" viewBox="0 0 24 24" aria-hidden="true" focusable="false"><path d="M14 3h7v7m0-7L10 14M10 3H3v18h18v-7" /></svg>
  </a>
</template>

<style scoped>
.ui-link { gap: var(--ui-space-1); text-decoration: none; }
svg { flex: none; width: 1rem; height: 1rem; fill: none; stroke: currentColor; stroke-width: 1.75; stroke-linecap: round; stroke-linejoin: round; }
</style>
