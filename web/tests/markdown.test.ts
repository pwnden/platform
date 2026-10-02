import { expect, it, vi } from 'vitest';
import { createSSRApp, h } from 'vue';
import { renderToString } from '@vue/server-renderer';
import UIMarkdown from '../packages/ui/src/UIMarkdown.vue';
import * as markdown from '../packages/ui/src/markdown';

const render = (source: string) => renderToString(createSSRApp({ render: () => h(UIMarkdown, { source }) }));

it('places brief headings under the panel title with preserved source hierarchy', async () => {
  const html = await renderToString(createSSRApp({ render: () => h(UIMarkdown, { source: '## Objective\n\n### Start here', headingOffset: 1 }) }));
  expect(html).toMatch(/<h3[^>]*>Objective<\/h3>/);
  expect(html).toMatch(/<h4[^>]*>Start here<\/h4>/);
});

it('initializes real WASM and renders headings, nested lists, tables, tasks and escaped code', async () => {
  const html = await render('# 설명\n\n**강조** &amp; `code`\n\n- 바깥\n  - 안쪽\n\n- [x] 완료\n- [ ] 대기\n\n| A | B |\n| :- | -: |\n| x | y |\n\n```html\n<script>alert(1)</script>\n```');
  expect(html).toMatch(/<h4 id="markdown-[^"]+">설명<\/h4>/);
  expect(html).toContain('<strong>강조</strong> &amp; <code>code</code>');
  expect(html).toContain('<ul><li><p>바깥</p><ul><li>안쪽</li></ul>');
  expect(html).toContain('type="checkbox" disabled checked aria-label="완료됨"');
  expect(html).toContain('aria-label="미완료"');
  expect(html).toContain('<table><thead><tr><th class="markdown-align-left">A</th><th class="markdown-align-right">B</th>');
  expect(html).toContain('class="ui-code"');
  expect(html).toContain('ui-syntax--');
  expect(html.replace(/<[^>]*>/g, '')).toContain('&lt;script&gt;alert(1)&lt;/script&gt;');
  expect(html).not.toContain('<script');
  expect(html).toContain('aria-busy="false"');
});

it('highlights fenced code using its language and leaves inline code plain under the existing CSP', async () => {
  const html = await render('```python\n# 설명\ndef solve():\n    return "pwnden"\n```\n\n`return "pwnden"`');
  expect(html).toContain('class="ui-syntax--comment"');
  expect(html).toContain('class="ui-syntax--keyword"');
  expect(html).toContain('class="ui-syntax--string"');
  expect(html).toContain('<code>return &quot;pwnden&quot;</code>');
  expect(html).not.toMatch(/style=|<script|<style/);
});

it('renders md4x message blocks with a sender and formatted dialogue', async () => {
  const html = await render('::message{from="moru17"}\n복구 키 파일이 공개됐어요. **확인해 주세요.**\n\n[안내](https://example.com/help)도 보냈어요.\n::\n\n일반 본문');
  expect(html).toMatch(/<blockquote class="markdown-message"><p class="markdown-message-from"><svg[^>]*ui-icon--message[^>]*aria-hidden="true"[^>]*>.*?<\/svg><span>moru17<\/span><\/p><div class="markdown-message-body">/);
  expect(html).toContain('<strong>확인해 주세요.</strong>');
  expect(html).toContain('href="https://example.com/help" target="_blank" rel="noopener noreferrer"');
  expect(html).toContain('</div></blockquote><p>일반 본문</p>');
  expect(html).not.toContain('::message');
});

it('supports unnamed messages while preserving ordinary blockquotes', async () => {
  const html = await render('::message\n의뢰인의 메시지\n::\n\n> 일반 인용문');
  expect(html).toContain('<blockquote class="markdown-message"><div class="markdown-message-body"><p>의뢰인의 메시지</p></div></blockquote>');
  expect(html).toContain('<blockquote><p>일반 인용문</p></blockquote>');
  expect(html).not.toContain('markdown-message-from');
});

it('renders briefing roles with distinct sections and headings below the panel title', async () => {
  const source = '::objective\n복구 키를 찾자.\n::\n\n::resources{title="전달받은 자료"}\n- `files/recovery.txt`\n::\n\n::knowledge\nBase64 예제\n\n```sh\nprintf hello\n```\n::\n\n::submission\n`pwnden{...}` 전체를 제출하자.\n::';
  const html = await renderToString(createSSRApp({ render: () => h(UIMarkdown, { source, headingOffset: 1 }) }));
  for (const [role, title] of [['objective', '의뢰 목표'], ['resources', '전달받은 자료'], ['knowledge', '시작 전 알아둘 것'], ['submission', '정답 형식']]) {
    expect(html).toContain(`class="markdown-section markdown-section--${role}" aria-label="${title}"`);
    expect(html).toMatch(new RegExp(`<h3 class="markdown-section-title"><svg[^>]*ui-icon--${role}[^>]*aria-hidden="true"[^>]*>.*?</svg><span>${title}</span></h3>`));
  }
  expect(html).toContain('<li><code>files/recovery.txt</code></li>');
  expect(html).toContain('class="ui-code"');
  expect(html).toContain('<code>pwnden{...}</code>');
  expect(html).not.toContain('::objective');
});

it('keeps role labels and attributes safe while preserving ordinary headings and unknown blocks', async () => {
  const html = await render('::resources{title="<img src=x onerror=bad>" onclick="bad" style="display:none"}\n자료 설명\n::\n\n::knowledge{title=""}\n배경 설명\n::\n\n::other-role\n일반 내용\n::\n\n## 일반 제목');
  expect(html).toContain('&lt;img src=x onerror=bad&gt;');
  expect(html).toContain('aria-label="시작 전 알아둘 것"');
  expect(html).toContain('일반 내용');
  expect(html).toMatch(/<h5[^>]*>일반 제목<\/h5>/);
  expect(html).not.toMatch(/<img\b|onclick=|style=|markdown-section--other-role/);
  const malformed = await renderToString(createSSRApp({ render: () => h('div', [markdown.renderMarkdownNode(['objective', { title: { toString: 'bad' }, innerHTML: '<script>bad</script>' }, ['p', {}, '목표']], 'test')]) }));
  expect(malformed).toContain('aria-label="의뢰 목표"');
  expect(malformed).not.toContain('script');
});

it('escapes message senders and retains the existing attribute and component allowlist', async () => {
  const html = await render('::message{from="<img src=x onerror=bad>" onclick="bad" style="display:none"}\n<script>bad</script>\n\n::iframe{src="https://example.com"}\n본문\n::\n::');
  expect(html).toContain('&lt;img src=x onerror=bad&gt;');
  expect(html).toContain('&lt;script&gt;bad&lt;/script&gt;');
  expect(html).toContain('본문');
  expect(html).not.toMatch(/<(?:script|iframe|img)\b|onclick=|style=/);
  const malformed = await renderToString(createSSRApp({ render: () => h('div', [markdown.renderMarkdownNode(['message', { from: { toString: 'bad' }, innerHTML: '<script>bad</script>' }, ['p', {}, '내용']], 'test')]) }));
  expect(malformed).toContain('<blockquote class="markdown-message"><div class="markdown-message-body"><p>내용</p></div></blockquote>');
  expect(malformed).not.toContain('markdown-message-from');
});

it('escapes raw HTML and keeps author components and attributes out of rendered DOM', async () => {
  const html = await render('<script>alert(1)</script>\n\n::iframe{src="https://example.com" onclick="bad"}\n**표시할 내용**\n::\n\n![설명](https://example.com/tracker.png)');
  expect(html).toContain('&lt;script&gt;alert(1)&lt;/script&gt;');
  expect(html).toContain('<strong>표시할 내용</strong>');
  expect(html).toContain('[이미지: 설명]');
  expect(html).not.toMatch(/<(?:script|iframe|img)\b/);
  expect(html).not.toContain('onclick=');
  expect(html).not.toContain('tracker.png');
});

it('confines links to HTTP, HTTPS and document anchors while dropping author attributes', async () => {
  const nodes = [
    ['a', { href: 'https://example.com/path', onclick: 'bad', style: 'display:none' }, '좋은 링크'],
    ...['javascript:alert(1)', 'JaVaScRiPt:alert(1)', 'data:text/html,bad', 'file:///etc/passwd', '//example.com', '../README.md'].map(href => ['a', { href }, '문자열']),
  ];
  const html = await renderToString(createSSRApp({ render: () => h('div', nodes.map(node => markdown.renderMarkdownNode(node as Parameters<typeof markdown.renderMarkdownNode>[0], 'test'))) }));
  expect(html.match(/<a /g)).toHaveLength(1);
  expect(html).toContain('href="https://example.com/path" target="_blank" rel="noopener noreferrer"');
  expect(html).not.toMatch(/onclick|style=|javascript|file:|data:/i);
  const anchors = await render('# Topic\n\n[이동](#topic)');
  const id = anchors.match(/<h4 id="([^"]+)">Topic/)!;
  expect(anchors).toContain(`href="#${id[1]}"`);
});

it('gives repeated headings distinct IDs across component instances', async () => {
  const html = await renderToString(createSSRApp({ render: () => h('div', [h(UIMarkdown, { source: '# Same' }), h(UIMarkdown, { source: '# Same' })]) }));
  const ids = [...html.matchAll(/<h4 id="([^"]+)"/g)].map(match => match[1]);
  expect(ids).toHaveLength(2);
  expect(new Set(ids).size).toBe(2);
});

it('ignores malformed author attributes without evaluating or stringifying them', async () => {
  const html = await renderToString(createSSRApp({ render: () => h('table', [h('tr', [markdown.renderMarkdownNode(['td', { align: { toString: 'bad' }, innerHTML: '<script>bad</script>' }, '내용'], 'test')])]) }));
  expect(html).toContain('<td>내용</td>');
  expect(html).not.toContain('script');
});

it('keeps escaped source and offers retry when initialization or parsing fails', async () => {
  const parse = vi.spyOn(markdown, 'parseMarkdown').mockRejectedValueOnce(new Error('WASM unavailable'));
  try {
    const html = await render('<script>원문</script>');
    expect(html).toContain('role="alert"');
    expect(html).toContain('설명 다시 표시');
    expect(html).toContain('&lt;script&gt;원문&lt;/script&gt;');
    expect(html).not.toContain('<script');
    expect(html).toContain('aria-busy="false"');
  } finally { parse.mockRestore(); }
});
