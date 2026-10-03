<script setup lang="ts">
import { computed, onUnmounted, provide, ref } from 'vue';
import { RouterView, useRouter } from 'vue-router';
import type { APIClient } from '@pwnden/api';
import AppLayout from './layouts/AppLayout.vue';
import { playerContext } from './context';
const props = defineProps<{ client?: APIClient | undefined; sessionRejected?: boolean }>();
const completedSlug = ref('');
const busy = ref(false);
provide(playerContext, { client: computed(() => props.client), completedSlug, busy });
const router = useRouter();
const removeGuard = router.beforeEach((to, from) => !busy.value || to.path === from.path);
onUnmounted(removeGuard);
</script>
<template>
  <AppLayout :connected="!!client" :session-rejected="sessionRejected">
    <RouterView v-if="client" />
  </AppLayout>
</template>
