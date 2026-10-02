<script setup lang="ts">
import {
  ArrowLeft, ArrowRight, BookOpen, Check, ChevronDown, ChevronRight, Download,
  ExternalLink, Files, FolderOpen, Globe, Link, Link2Off, LoaderCircle,
  MessageSquare, Plug, Power, Send, SquareTerminal, Target, TriangleAlert, Unplug,
  RefreshCw,
} from '@lucide/vue';
import type { UIIconName, UIIconProps } from './props';

defineOptions({ inheritAttrs: false });
withDefaults(defineProps<UIIconProps>(), { size: 16 });
const icons = {
  back: ArrowLeft, forward: ArrowRight, refresh: RefreshCw, download: Download,
  external: ExternalLink, check: Check, 'chevron-down': ChevronDown,
  'chevron-right': ChevronRight, loader: LoaderCircle, link: Link,
  'link-off': Link2Off, power: Power, plug: Plug, unplug: Unplug,
  error: TriangleAlert, objective: Target, resources: FolderOpen,
  knowledge: BookOpen, submission: Send, message: MessageSquare,
  terminal: SquareTerminal, files: Files, web: Globe,
} satisfies Record<UIIconName, unknown>;
</script>

<template>
  <component :is="icons[name]" v-bind="$attrs" :class="['ui-icon', `ui-icon--${name}`]" :size="size" :stroke-width="1.75" aria-hidden="true" focusable="false" />
</template>

<style scoped>
.ui-icon { display: block; flex: none; pointer-events: none; }
</style>
