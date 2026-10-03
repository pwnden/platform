<script setup lang="ts">
import { ref } from 'vue';
import type { APIClient } from '@pwnden/api';
import type { Problem } from '@pwnden/catalog';
import AppLayout from './layouts/AppLayout.vue';
import ChallengeLayout from './layouts/ChallengeLayout.vue';
import ChallengeWorkspace from './components/ChallengeWorkspace.vue';
import ChallengeEmpty from './components/ChallengeEmpty.vue';
defineProps<{ client?: APIClient | undefined; sessionRejected?: boolean }>();
const selected = ref<Problem>();
const completedSlug = ref('');
const busy = ref(false);
function select(problem: Problem) {
  if (!busy.value && selected.value?.slug !== problem.slug) selected.value = problem;
}
</script>
<template>
  <AppLayout :connected="!!client" :session-rejected="sessionRejected">
    <ChallengeLayout v-if="client" :client="client" :selected-slug="selected?.slug" :completed-slug="completedSlug" :disabled="busy" @select="select">
      <ChallengeWorkspace v-if="selected" :key="selected.slug" :client="client" :problem="selected" @select="select" @busy="busy = $event" @completed="completedSlug = selected.slug" />
      <ChallengeEmpty v-else />
    </ChallengeLayout>
  </AppLayout>
</template>
