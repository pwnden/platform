<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import type { Catalog, Problem } from '@pwnden/catalog';
import { UIButton } from '@pwnden/ui';

const props = defineProps<{ catalog: Catalog; selectionDisabled?: boolean }>();
const emit = defineEmits<{ select: [problem: Problem] }>();
const problems = ref<readonly Problem[]>([]);
const pending = ref(false);
const failed = ref(false);
let active = true;
onUnmounted(() => { active = false; });

async function load() {
  if (pending.value) return;
  pending.value = true;
  failed.value = false;
  try {
    const items = await props.catalog.list();
    if (active) problems.value = items;
  } catch {
    if (active) failed.value = true;
  } finally {
    if (active) pending.value = false;
  }
}
onMounted(load);
</script>

<template>
  <section aria-labelledby="catalog-heading" :aria-busy="pending">
    <h2 id="catalog-heading">문제 목록</h2>
    <UIButton :busy="pending" @click="load">{{ pending ? '불러오는 중…' : '목록 새로고침' }}</UIButton>
    <p v-if="failed" role="alert">목록을 불러오지 못했습니다. 서버 연결을 확인하고 다시 시도하세요.</p>
    <p v-else-if="!pending && problems.length === 0">등록된 문제가 없습니다.</p>
    <ul v-if="problems.length" class="problem-list">
      <li v-for="problem in problems" :key="problem.slug">
        <UIButton :disabled="selectionDisabled" @click="emit('select', problem)">{{ problem.title }}</UIButton>
        <span>{{ problem.category }} · {{ problem.kind === 'file' ? '파일 문제' : '서비스 문제' }}</span>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.problem-list { padding: 0; list-style: none; }
.problem-list li { display: grid; justify-items: start; gap: 0.25rem; padding-block: var(--ui-space-2); }
.problem-list span { color: var(--ui-muted); overflow-wrap: anywhere; }
</style>
