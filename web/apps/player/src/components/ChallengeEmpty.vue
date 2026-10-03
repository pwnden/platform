<template>
  <div class="workspace-empty"><div class="empty-content">
    <h2>풀어볼 문제를 선택하세요.</h2>
    <p>문제 이름을 검색하거나 분야별로 찾아볼 수 있습니다.</p>
  </div></div>
</template>
<style scoped>
.workspace-empty { height: 100%; min-width: 0; display: grid; align-items: center; justify-items: center; padding: var(--ui-space-4); }
.empty-content { width: min(100%, 42rem); display: grid; gap: var(--ui-space-2); }
h2 { font-size: 1.35rem; }
p { color: var(--ui-muted); line-height: 1.9; }
@media (max-width: 48rem) { .workspace-empty { min-height: 25rem; padding: var(--ui-space-3); } }
</style>
