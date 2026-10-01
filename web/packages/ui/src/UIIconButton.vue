<script setup lang="ts">
import UIButton from './UIButton.vue';
import type { UIIconButtonProps } from './props';
withDefaults(defineProps<UIIconButtonProps>(), { busy: false, disabled: false });
</script>

<template>
  <UIButton variant="ghost" size="compact" class="ui-icon-control" :busy="busy" :disabled="disabled" :aria-label="label" :title="label">
    <svg :class="{ 'ui-icon-progress': busy }" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path v-if="busy || icon === 'refresh'" d="M20 7v5h-5M4 17v-5h5m10.3-4a8 8 0 0 0-13.9-3M4.7 16a8 8 0 0 0 13.9 3" />
      <path v-else d="M12 3v12m-5-5 5 5 5-5M4 16v5h16v-5" />
    </svg>
  </UIButton>
</template>
