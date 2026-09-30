<script setup lang="ts">
import { TextField } from '@sectile/vue/text';
import type { UITextFieldProps } from './props';

defineOptions({ inheritAttrs: false });
defineProps<UITextFieldProps>();
const emit = defineEmits<{ 'update:modelValue': [value: string] }>();
</script>

<template>
  <div class="ui-field">
    <label :for="id">{{ label }}</label>
    <TextField
      v-bind="$attrs"
      :id="id"
      class="ui-input"
      :model-value="modelValue"
      :disabled="disabled"
      :required="required"
      autocomplete="off"
      :spellcheck="false"
      @update:model-value="emit('update:modelValue', String($event))"
    />
  </div>
</template>
