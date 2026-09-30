import { builtinEnvironments } from 'vitest/runtime';

// Real client-compiled Vue components on a custom renderer, with no DOM library.
export default { ...builtinEnvironments.node, name: 'player', viteEnvironment: 'client' };
