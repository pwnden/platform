<script setup lang="ts">
import { RouterLink } from 'vue-router';
</script>
<template><div class="page-notice"><h2>페이지를 찾을 수 없습니다.</h2><RouterLink to="/challenges">문제 목록으로 이동</RouterLink></div></template>
<style scoped>
.page-notice { display: grid; align-content: center; justify-items: start; gap: var(--ui-space-2); padding: var(--ui-space-3); }
h2 { font-size: 1.25rem; }
</style>
