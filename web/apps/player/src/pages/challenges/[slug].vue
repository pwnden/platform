<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { Problem, ProblemDetail } from '@pwnden/catalog';
import { APIError } from '@pwnden/api';
import { UIButton } from '@pwnden/ui';
import ChallengeWorkspace from '../../components/ChallengeWorkspace.vue';
import { usePlayer } from '../../context';
const { client, busy, completedSlug } = usePlayer();
const route = useRoute('/challenges/[slug]');
const router = useRouter();
const slug = String(route.params.slug);
const detail = ref<ProblemDetail>();
const pending = ref(false);
const failed = ref(false);
const missing = ref(false);
let active = true;
onUnmounted(() => { active = false; busy.value = false; });
async function load() {
  if (pending.value || !client.value) return;
  pending.value = true; failed.value = false; missing.value = false;
  try {
    const result = await client.value.catalog.detail(slug);
    if (active) detail.value = result;
  } catch (error) {
    if (active) { failed.value = true; missing.value = error instanceof APIError && error.code === 'not_found'; }
  } finally { if (active) pending.value = false; }
}
function select(problem: Problem) {
  if (!busy.value && slug !== problem.slug) void router.push({ path: `/challenges/${problem.slug}`, query: { ...route.query, tool: undefined } });
}
onMounted(load);
</script>
<template>
  <ChallengeWorkspace v-if="client && detail" :client="client" :problem="detail" :initial-detail="detail" @select="select" @busy="busy = $event" @completed="completedSlug = slug" />
  <div v-else class="page-notice">
    <p v-if="pending" role="status">문제 설명을 불러오는 중…</p>
    <template v-else-if="failed">
      <h2>{{ missing ? '문제를 찾을 수 없습니다.' : '문제를 불러오지 못했습니다.' }}</h2>
      <p role="alert">{{ missing ? '문제 목록에서 다른 문제를 선택하세요.' : '서버 연결을 확인하고 다시 시도하세요.' }}</p>
      <UIButton v-if="!missing" @click="load">문제 다시 불러오기</UIButton>
    </template>
  </div>
</template>
<style scoped>
.page-notice { display: grid; align-content: center; justify-items: start; gap: var(--ui-space-2); padding: var(--ui-space-3); min-width: 0; }
h2 { font-size: 1.25rem; }
</style>
