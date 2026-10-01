import assert from 'node:assert/strict';
import { test } from 'node:test';
import { checkSpacing } from './spacing.mjs';

test('accepts equal insets and treats nested calculations as one value', () => {
  for (const value of ['0', 'var(--ui-space-2)', '8px 8px', '1rem 1rem 1rem 1rem', 'calc((var(--height) - 1.4rem) / 2 - 1px)']) {
    assert.deepEqual(checkSpacing(`.control { padding: ${value}; }`), []);
  }
});

test('rejects unequal insets and directional overrides in responsive rules', () => {
  for (const value of ['8px 16px', '0 8px 16px', '1px 1px 1px 2px']) {
    assert.equal(checkSpacing(`.control { padding: ${value} !important; }`).length, 1);
  }
  for (const property of ['padding-left', 'padding-inline', 'padding-block-start']) {
    assert.equal(checkSpacing(`@media (max-width: 48rem) { .control { ${property}: 0; } }`).length, 1);
  }
});

test('checks all Vue style blocks while keeping document content and comments outside the policy', () => {
  const source = '<template><p>padding: 1px 2px;</p></template><style>/* padding-left: 4px; */ .a { padding: 8px; }</style><style scoped>.b { padding: 1px 2px; }</style>';
  assert.equal(checkSpacing(source, 'Component.vue').length, 1);
  assert.deepEqual(checkSpacing('.item { content: "padding: 1px 2px;"; margin-top: 8px; gap: 4px; }'), []);
  assert.equal(checkSpacing('.broken { padding:').length, 1);
});
