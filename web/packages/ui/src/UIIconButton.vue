<script setup lang="ts">
import UIButton from './UIButton.vue';
import UIIcon from './UIIcon.vue';
import type { UIIconButtonProps } from './props';
withDefaults(defineProps<UIIconButtonProps>(), { busy: false, disabled: false });
</script>

<template>
  <UIButton variant="ghost" size="compact" class="ui-icon-control" :busy="busy" :disabled="disabled" :aria-label="label" :title="label">
    <UIIcon :name="busy ? 'loader' : icon" :class="{ 'ui-icon-progress': busy }" />
  </UIButton>
</template>
