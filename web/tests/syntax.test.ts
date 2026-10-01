import { expect, it, vi } from 'vitest';
import { createSSRApp, h } from 'vue';
import { renderToString } from '@vue/server-renderer';
import { codeLanguage, highlightCode } from '../packages/ui/src/syntax';
import * as syntax from '../packages/ui/src/syntax';
import UICode from '../packages/ui/src/UICode.vue';

it('selects code languages from fenced hints and material filenames using an allowlist', () => {
  expect(codeLanguage(undefined, 'files/checker.py')).toBe('python');
  expect(codeLanguage(undefined, 'src\\LOCK.CPP')).toBe('cpp');
  expect(codeLanguage(undefined, 'Dockerfile')).toBe('dockerfile');
  expect(codeLanguage('sh', 'checker.py')).toBe('bash');
  expect(codeLanguage('python title=checker.py')).toBe('python');
  for (const language of ['text', 'unknown', 'constructor', '__proto__', 'toString', '<script>']) expect(codeLanguage(language)).toBeUndefined();
});

it('uses real Shiki for Python, Bash and source files while preserving indentation and terminal newlines', async () => {
  const source = '# 주석\ndef solve():\n    return "pwnden{test}"\n\n';
  const lines = await highlightCode(source, undefined, 'checker.py');
  expect(lines!.map(line => line.map(token => token.content).join('')).join('\n')).toBe(source);
  expect(lines!.flat().map(token => token.className)).toEqual(expect.arrayContaining(['ui-syntax--comment', 'ui-syntax--keyword', 'ui-syntax--string']));
  const shell = await highlightCode('echo "hello"\n', 'bash');
  expect(shell!.flat().some(token => token.className === 'ui-syntax--string')).toBe(true);
  const html = await renderToString(createSSRApp({ render: () => h(UICode, { source, label: 'checker.py' }) }));
  expect(html).toContain('class="ui-syntax--keyword"');
  expect(html).toContain('tabindex="0" aria-label="checker.py"');
  expect(html).not.toMatch(/style=|<style|<script/);
});

it('loads every declared grammar and keeps generated token classes inside the fixed palette', async () => {
  const snippets = {
    javascript: 'const flag = "pwnden";', typescript: 'const flag: string = "pwnden";', json: '{"flag": 1}', html: '<script>let x = 1</script>', css: 'body { color: red; }',
    c: 'int main() { return 0; }', cpp: '#include <iostream>\nint main() { return 0; }', go: 'package main\nfunc main() {}', rust: 'fn main() {}', sql: 'SELECT * FROM notes;',
    yaml: 'flag: true', toml: 'flag = true', dockerfile: 'FROM debian:trixie', java: 'class Main {}', php: '<?php echo "hi";', powershell: 'Write-Host "hi"', asm: 'mov eax, 1', diff: '+ added\n- removed',
  };
  for (const [language, source] of Object.entries(snippets)) {
    const lines = await highlightCode(source, language);
    expect(lines, language).toBeDefined();
    expect(lines!.map(line => line.map(token => token.content).join('')).join('\n')).toBe(source);
    for (const token of lines!.flat()) expect(token.className).toMatch(/^ui-syntax--(?:text|comment|keyword|string|number|function|type)$/);
  }
});

it('keeps unknown and large sources available as escaped selectable text without tokenizing', async () => {
  const source = '<script>alert(1)</script>\n  raw';
  expect(await highlightCode(source, 'unknown')).toBeUndefined();
  expect(await highlightCode('x'.repeat(128 * 1024 + 1), 'python')).toBeUndefined();
  const html = await renderToString(createSSRApp({ render: () => h(UICode, { source, label: 'unknown.txt' }) }));
  expect(html).toContain('&lt;script&gt;alert(1)&lt;/script&gt;\n  raw');
  expect(html).not.toContain('<script');
});

it('keeps raw source readable when grammar loading fails', async () => {
  const highlight = vi.spyOn(syntax, 'highlightCode').mockRejectedValueOnce(new Error('offline chunk'));
  try {
    const html = await renderToString(createSSRApp({ render: () => h(UICode, { source: '<script>raw</script>', label: 'checker.py' }) }));
    expect(html).toContain('&lt;script&gt;raw&lt;/script&gt;');
    expect(html).not.toContain('<script');
  } finally { highlight.mockRestore(); }
});
