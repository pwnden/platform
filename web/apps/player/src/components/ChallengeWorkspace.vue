<script setup lang="ts">
import { ref } from 'vue';
import type { APIClient } from '@pwnden/api';
import type { Problem, ProblemFile, ProblemTool, ProblemDetail } from '@pwnden/catalog';
import type { RunStatus } from '@pwnden/play';
import { PlayPanel } from '@pwnden/play-feature';
import { UISplit } from '@pwnden/ui';
import ChallengeBriefing from './ChallengeBriefing.vue';
import ChallengeTools from './ChallengeTools.vue';
defineProps<{ client: APIClient; problem: Problem; initialDetail?: ProblemDetail | undefined }>();
const emit = defineEmits<{ select: [problem: Problem]; busy: [value: boolean]; completed: [] }>();
const answer = ref<string>();
const busy = ref(false);
const files = ref<readonly ProblemFile[]>([]);
const allowed = ref<readonly ProblemTool[]>([]);
const runStatus = ref<RunStatus>();
const briefingWidth = ref(50);
function loaded(detail: ProblemDetail) {
  answer.value = detail.answer; files.value = detail.files; allowed.value = detail.tools;
}
function setBusy(value: boolean) { busy.value = value; emit('busy', value); }
</script>
<template>
  <UISplit v-model="briefingWidth" class="selected-problem" label="설명과 작업 영역 너비" :min="30" :max="70">
    <template #before><ChallengeBriefing :catalog="client.catalog" :problem="problem" :initial-detail="initialDetail" :busy="busy" @loaded="loaded" @select="emit('select', $event)" /></template>
    <template #after>
      <div class="tool-workspace">
        <PlayPanel :player="client.player" :workspaces="client.workspaces" :slug="problem.slug" :kind="problem.kind" :answer="answer" @completed="emit('completed')" @busy="setBusy" @status="runStatus = $event">
          <template #default="{ environment, blocked }"><ChallengeTools :client="client" :slug="problem.slug" :files="files" :allowed="allowed" :status="runStatus" :environment="environment" :blocked="blocked" /></template>
        </PlayPanel>
      </div>
    </template>
  </UISplit>
</template>
<style scoped>
.selected-problem { min-width: 0; min-height: 0; }
.tool-workspace { height: 100%; min-height: 0; min-width: 0; display: flex; flex-direction: column; }
@media (max-width: 48rem) { .tool-workspace { height: 36rem; } }
</style>
