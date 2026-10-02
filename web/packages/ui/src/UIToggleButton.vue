<script setup lang="ts">
import { ToggleButton } from '@sectile/vue/toggle-button';
import UIButton from './UIButton.vue';
import type { UIToggleButtonProps } from './props';
withDefaults(defineProps<UIToggleButtonProps>(), { variant: 'secondary', size: 'default' });
defineEmits<{ 'update:modelValue': [value: boolean] }>();
</script>

<template>
  <ToggleButton as-child :model-value="modelValue" :disabled="disabled || busy" @update:model-value="$emit('update:modelValue', $event)">
    <UIButton :variant="variant" :size="size" :disabled="disabled" :busy="busy"><slot /></UIButton>
  </ToggleButton>
</template>
