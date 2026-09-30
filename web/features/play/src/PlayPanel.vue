<script setup lang="ts">
import { onUnmounted, ref } from 'vue';
import type { Player, Run } from '@pwnden/play';
import { UIButton, UITextField } from '@pwnden/ui';

const props = defineProps<{ player: Player; slug: string; title: string }>();
const emit = defineEmits<{ busy: [value: boolean] }>();
const pending = ref(false);
const run = ref<Run>();
const flag = ref('');
const message = ref('');
const failed = ref(false);
let active = true;
onUnmounted(() => { active = false; });

async function perform(operation: 'run' | 'stop' | 'submit') {
  if (pending.value) return;
  pending.value = true;
  emit('busy', true);
  message.value = '';
  failed.value = false;
  try {
    if (operation === 'run') {
      const result = await props.player.run(props.slug);
      if (active) {
        run.value = result;
        message.value = '문제가 준비되었습니다.';
      }
    } else if (operation === 'stop') {
      await props.player.stop(props.slug);
      if (active) { run.value = undefined; message.value = '문제를 중지했습니다.'; }
    } else {
      const result = await props.player.submit(props.slug, flag.value);
      if (active) { flag.value = ''; message.value = result.accepted ? '정답입니다.' : '정답이 아닙니다. 다시 시도하세요.'; }
    }
  } catch {
    if (active) { failed.value = true; message.value = '요청을 완료하지 못했습니다. 서버 연결과 문제의 실행 상태를 확인하세요.'; }
  } finally {
    if (active) { pending.value = false; emit('busy', false); }
  }
}
</script>

<template>
  <section aria-labelledby="play-heading" :aria-busy="pending">
    <h2 id="play-heading">{{ title }}</h2>
    <div class="actions">
      <UIButton :disabled="pending" @click="perform('run')">문제 실행</UIButton>
      <UIButton :disabled="pending" @click="perform('stop')">문제 중지</UIButton>
    </div>
    <p v-if="pending" role="status">요청을 처리하는 중…</p>
    <p v-if="message" :role="failed ? 'alert' : 'status'">{{ message }}</p>
    <ul v-if="run?.endpoints.length" class="endpoints">
      <li v-for="endpoint in run.endpoints" :key="endpoint.name">
        {{ endpoint.name }}:
        <a v-if="endpoint.url.startsWith('http://')" :href="endpoint.url" target="_blank" rel="noopener noreferrer">{{ endpoint.url }} (새 탭)</a>
        <code v-else>{{ endpoint.url }}</code>
      </li>
    </ul>
    <form @submit.prevent="perform('submit')">
      <UITextField id="flag" v-model="flag" label="플래그" :disabled="pending" required />
      <UIButton type="submit" :disabled="pending || !flag.trim()">정답 확인</UIButton>
    </form>
  </section>
</template>

<style scoped>
.actions { display: flex; flex-wrap: wrap; gap: var(--ui-space-1); }
form { display: grid; justify-items: start; gap: var(--ui-space-2); margin-top: var(--ui-space-4); }
.ui-field { width: 100%; }
.endpoints { padding-left: var(--ui-space-3); overflow-wrap: anywhere; }
</style>
