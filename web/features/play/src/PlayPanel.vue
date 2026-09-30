<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue';
import type { Player, RunStatus } from '@pwnden/play';
import { UIButton, UITextField } from '@pwnden/ui';

const props = defineProps<{ player: Player; slug: string; kind: RunStatus['kind'] }>();
const emit = defineEmits<{ busy: [value: boolean]; status: [value: RunStatus | undefined] }>();
const pending = ref(false);
const status = ref<RunStatus>();
const flag = ref('');
const message = ref('');
const failed = ref(false);
let active = true;
onUnmounted(() => { active = false; });

async function perform(operation: 'run' | 'stop' | 'submit' | 'status') {
  if (pending.value) return;
  pending.value = true;
  emit('busy', true);
  message.value = '';
  failed.value = false;
  try {
    if (operation === 'run') {
      await props.player.run(props.slug);
      if (active) {
        message.value = '문제가 준비되었습니다.';
      }
    } else if (operation === 'stop') {
      await props.player.stop(props.slug);
      if (active) message.value = '문제를 중지했습니다.';
    } else if (operation === 'submit') {
      const result = await props.player.submit(props.slug, flag.value);
      if (active) { flag.value = ''; message.value = result.accepted ? '정답입니다.' : '정답이 아닙니다. 다시 시도하세요.'; }
    }
    const result = await props.player.status(props.slug);
    if (active) status.value = result;
  } catch {
    // Refresh after a failed mutation: an already-running conflict must recover
    // the existing endpoints, and a lost response must not invent local state.
    if (operation !== 'status') {
      try {
        const result = await props.player.status(props.slug);
        if (active) status.value = result;
      } catch { if (active) status.value = undefined; }
    } else if (active) status.value = undefined;
    if (active) { failed.value = true; message.value = '요청을 완료하지 못했습니다. 서버 연결을 확인하고 실행 상태를 새로고침하세요.'; }
  } finally {
    if (active) { pending.value = false; emit('busy', false); emit('status', status.value); }
  }
}
onMounted(() => perform('status'));
</script>

<template>
  <section aria-labelledby="play-heading" :aria-busy="pending">
    <h2 id="play-heading">실행과 정답 확인</h2>
    <p v-if="!status && !pending">실행 상태를 확인하지 못했습니다.</p>
    <p v-else-if="status?.state === 'ready'">파일 문제입니다. 서비스 실행 없이 배포 파일을 분석하고 정답을 제출할 수 있습니다.</p>
    <p v-else-if="status?.state === 'stopped'">중지됨</p>
    <p v-else-if="status?.state === 'running'">실행 중</p>
    <p v-else-if="status?.state === 'unavailable'" role="alert">서비스가 정상 실행 중이지 않습니다. 문제를 중지해 정리한 뒤 다시 실행하세요.</p>
    <div class="actions">
      <UIButton v-if="kind === 'service'" :disabled="pending || status?.state !== 'stopped'" @click="perform('run')">문제 실행</UIButton>
      <UIButton v-if="kind === 'service'" :disabled="pending || status?.state === 'stopped'" @click="perform('stop')">문제 중지</UIButton>
      <UIButton :disabled="pending" @click="perform('status')">실행 상태 새로고침</UIButton>
    </div>
    <p v-if="pending" role="status">요청을 처리하는 중…</p>
    <p v-if="message" :role="failed ? 'alert' : 'status'">{{ message }}</p>
    <ul v-if="status?.endpoints.length" class="endpoints">
      <li v-for="endpoint in status.endpoints" :key="endpoint.name">
        {{ endpoint.name }}:
        <a v-if="endpoint.url.startsWith('http://')" :href="endpoint.url" target="_blank" rel="noopener noreferrer">{{ endpoint.url }} (새 탭)</a>
        <code v-else>{{ endpoint.url }}</code>
      </li>
    </ul>
    <form @submit.prevent="perform('submit')">
      <UITextField id="flag" v-model="flag" label="플래그" :disabled="pending" required />
      <UIButton type="submit" :disabled="pending || !flag.trim() || !status || (status.state !== 'ready' && status.state !== 'running')">정답 확인</UIButton>
    </form>
  </section>
</template>

<style scoped>
.actions { display: flex; flex-wrap: wrap; gap: var(--ui-space-1); }
form { display: grid; justify-items: start; gap: var(--ui-space-2); margin-top: var(--ui-space-4); }
.ui-field { width: 100%; }
.endpoints { padding-left: var(--ui-space-3); overflow-wrap: anywhere; }
</style>
