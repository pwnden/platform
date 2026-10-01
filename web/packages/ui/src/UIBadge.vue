<template>
  <span class="ui-badge"><slot /></span>
</template>

<style scoped>
.ui-badge { display: inline-flex; flex: none; align-items: center; padding: 0.25rem; border: 1px solid var(--ui-border-active); border-radius: var(--ui-radius-control); background: var(--ui-accent-surface); color: var(--ui-accent); font-size: 0.8rem; line-height: 1.4; white-space: nowrap; }
</style>
