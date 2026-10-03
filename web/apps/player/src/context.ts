import { inject } from 'vue';
import type { InjectionKey, Ref, ComputedRef } from 'vue';
import type { APIClient } from '@pwnden/api';

export interface PlayerContext {
  client: ComputedRef<APIClient | undefined>;
  busy: Ref<boolean>;
  completedSlug: Ref<string>;
}
export const playerContext: InjectionKey<PlayerContext> = Symbol('player');
export function usePlayer() {
  const context = inject(playerContext);
  if (!context) throw new Error('Player context is unavailable.');
  return context;
}
