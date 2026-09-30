<script setup lang="ts">
import type { UIButtonProps } from './props';

withDefaults(defineProps<UIButtonProps>(), { type: 'button', disabled: false, busy: false, variant: 'secondary', size: 'default' });
</script>

<template>
  <button class="ui-button" :class="[`ui-button--${variant}`, `ui-button--${size}`]" :type="type" :disabled="disabled || busy" :aria-busy="busy || undefined">
    <slot />
  </button>
</template>
