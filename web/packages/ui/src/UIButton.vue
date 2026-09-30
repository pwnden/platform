<script setup lang="ts">
import type { UIButtonProps } from './props';

withDefaults(defineProps<UIButtonProps>(), { type: 'button', disabled: false, busy: false });
</script>

<template>
  <button class="ui-button" :type="type" :disabled="disabled || busy" :aria-busy="busy || undefined">
    <slot />
  </button>
</template>
