import { createRouter, createWebHistory } from 'vue-router';
import type { RouterHistory } from 'vue-router';
import { routes } from 'vue-router/auto-routes';

export function createPlayerRouter(history: RouterHistory = createWebHistory()) {
  return createRouter({ history, routes });
}
