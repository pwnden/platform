<script setup lang="ts">
import { FormRoot } from '@sectile/vue/form';
import type { UIFormProps } from './props';
const props = defineProps<UIFormProps>();
function submit() { return props.submit(); }
</script>

<template>
  <FormRoot :on-submit="submit"><slot /></FormRoot>
</template>
