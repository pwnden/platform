<script setup lang="ts">
import type { UICodeProps } from './props';
defineProps<UICodeProps>();
</script>

<template>
  <pre class="ui-code" tabindex="0" :aria-label="label"><code>{{ source }}</code></pre>
</template>

<style scoped>
.ui-code { max-height: 32rem; overflow: auto; margin: 0; padding: var(--ui-space-2); background: var(--ui-background); border: 1px solid var(--ui-border); color: var(--ui-foreground); font-family: var(--ui-font-mono); font-size: 0.82rem; line-height: 1.65; white-space: pre; tab-size: 4; }
.ui-code:focus-visible { outline: 2px solid var(--ui-accent); outline-offset: 2px; }
</style>
