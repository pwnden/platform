<script setup lang="ts">
import { ref } from 'vue';
import type { APIClient } from '@pwnden/api';
import type { Problem } from '@pwnden/catalog';
import { ProblemList } from '@pwnden/catalog-feature';
import { PlayPanel } from '@pwnden/play-feature';

defineProps<{ client?: APIClient | undefined }>();
const selected = ref<Problem>();
const busy = ref(false);
function select(problem: Problem) {
  if (!busy.value) selected.value = problem;
}
</script>

<template>
  <main>
    <h1>pwnden</h1>
    <p v-if="!client" role="alert">세션에 연결할 수 없습니다. 실행 중인 pwnden 서버가 출력한 주소로 다시 접속하세요.</p>
    <div v-else class="workspace">
      <ProblemList :catalog="client.catalog" :selection-disabled="busy" @select="select" />
      <PlayPanel
        v-if="selected" :key="selected.slug" :player="client.player" :slug="selected.slug"
        :title="selected.title" @busy="busy = $event"
      />
      <p v-else>목록에서 풀어볼 문제를 선택하세요.</p>
    </div>
  </main>
</template>

<style scoped>
main { max-width: 72rem; padding: var(--ui-space-4); margin: auto; }
h1 { margin-top: 0; }
.workspace { display: grid; grid-template-columns: minmax(14rem, 1fr) minmax(0, 2fr); gap: var(--ui-space-4); }
@media (max-width: 44rem) { .workspace { grid-template-columns: 1fr; } main { padding: var(--ui-space-2); } }
</style>
