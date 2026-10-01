<script setup lang="ts">
import { computed } from 'vue';
import type { UISelectProps } from './props';
const props = defineProps<UISelectProps>();
const emit = defineEmits<{ 'update:modelValue': [value: string] }>();
const selection = computed({ get: () => props.modelValue, set: value => emit('update:modelValue', value) });
</script>

<template>
  <div class="ui-field">
    <label :for="id">{{ label }}</label>
    <select :id="id" v-model="selection" class="ui-input" :disabled="disabled">
      <option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option>
    </select>
  </div>
</template>
