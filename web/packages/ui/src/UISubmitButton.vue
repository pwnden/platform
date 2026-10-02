<script setup lang="ts">
import { FormSubmit } from '@sectile/vue/form';
import UIButton from './UIButton.vue';
import type { UISubmitButtonProps } from './props';
withDefaults(defineProps<UISubmitButtonProps>(), { variant: 'primary', size: 'default' });
</script>

<template>
  <FormSubmit as-child>
    <UIButton type="submit" :variant="variant" :size="size" :disabled="disabled" :busy="busy"><slot /></UIButton>
  </FormSubmit>
</template>
