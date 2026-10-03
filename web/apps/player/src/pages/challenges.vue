<script setup lang="ts">
import { computed } from 'vue';
import { RouterView, useRoute, useRouter } from 'vue-router';
import type { Problem } from '@pwnden/catalog';
import ChallengeLayout from '../layouts/ChallengeLayout.vue';
import { usePlayer } from '../context';
const { client, busy, completedSlug } = usePlayer();
const route = useRoute();
const router = useRouter();
const selectedSlug = computed(() => 'slug' in route.params && typeof route.params.slug === 'string' ? route.params.slug : undefined);
function select(problem: Problem) {
  if (!busy.value && selectedSlug.value !== problem.slug) void router.push({ path: `/challenges/${problem.slug}`, query: { ...route.query, tool: undefined } });
}
</script>
<template>
  <ChallengeLayout v-if="client" :client="client" :selected-slug="selectedSlug" :completed-slug="completedSlug" :disabled="busy" @select="select">
    <RouterView v-slot="{ Component }"><component :is="Component" :key="selectedSlug ?? 'index'" /></RouterView>
  </ChallengeLayout>
</template>
