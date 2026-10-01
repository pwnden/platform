<script setup lang="ts">
import type { UIWebFrameProps } from './props';
defineProps<UIWebFrameProps>();
</script>

<template>
  <iframe class="ui-web-frame" :src="src" :title="title" referrerpolicy="no-referrer" sandbox="allow-scripts allow-same-origin allow-forms allow-modals allow-popups allow-downloads" />
</template>

<style scoped>
.ui-web-frame { display: block; width: 100%; height: 100%; min-height: 0; border: 0; border-radius: var(--ui-radius-flush); background: white; color-scheme: normal; }
</style>
