import { h, type VNodeChild } from 'vue';
import type { ComarkNode } from 'md4x/standalone';

let parser: Promise<typeof import('md4x/standalone')> | undefined;

export async function parseMarkdown(source: string): Promise<ComarkNode[]> {
  parser ??= import('md4x/standalone').then(async module => {
    await module.init();
    return module;
  }).catch(error => { parser = undefined; throw error; });
  return (await parser).parseAST(source).nodes;
}

const elements = new Set(['p', 'ul', 'ol', 'li', 'blockquote', 'pre', 'code', 'strong', 'em', 'del', 's', 'hr', 'br', 'table', 'thead', 'tbody', 'tfoot', 'tr', 'th', 'td', 'span', 'div', 'mark']);

function linkTarget(value: unknown, prefix: string): string | undefined {
  if (typeof value !== 'string') return;
  try {
    if (value.startsWith('#')) return `#${prefix}-${decodeURIComponent(value.slice(1))}`;
    const url = new URL(value);
    if (url.protocol === 'http:' || url.protocol === 'https:') return url.href;
  } catch { /* Invalid and relative URLs remain text. */ }
}

export function renderMarkdownNode(node: ComarkNode, prefix: string, headingOffset = 3): VNodeChild {
  if (typeof node === 'string') return node;
  const [tag, attributes, ...content] = node;
  const children = content.map(child => renderMarkdownNode(child, prefix, headingOffset));
  if (tag === 'a') {
    const href = linkTarget(attributes.href, prefix);
    if (!href) return h('span', children);
    return h('a', { href, ...(href.startsWith('#') ? {} : { target: '_blank', rel: 'noopener noreferrer' }) }, children);
  }
  if (tag === 'img') return h('span', { class: 'markdown-image' }, typeof attributes.alt === 'string' ? `[이미지: ${attributes.alt}]` : '[이미지]');
  if (tag && /^h[1-6]$/.test(tag)) {
    const id = typeof attributes.id === 'string' ? `${prefix}-${attributes.id}` : undefined;
    return h(`h${Math.min(Number(tag[1]) + headingOffset, 6)}`, { id }, children);
  }
  // Unknown components and raw HTML contribute escaped text/allowed children only.
  if (!tag || !elements.has(tag)) return children;
  const props: Record<string, unknown> = {};
  if (tag === 'ol' && typeof attributes.start === 'number' && Number.isSafeInteger(attributes.start)) props.start = attributes.start;
  if ((tag === 'th' || tag === 'td') && typeof attributes.align === 'string' && ['left', 'center', 'right'].includes(attributes.align)) props.class = `markdown-align-${attributes.align}`;
  if (tag === 'li' && attributes.task === true) {
    props.class = 'markdown-task';
    children.unshift(h('input', { type: 'checkbox', disabled: true, checked: attributes.checked === true, 'aria-label': attributes.checked === true ? '완료됨' : '미완료' }));
  }
  const element = h(tag, props, children);
  return tag === 'table' ? h('div', { class: 'markdown-table', tabindex: 0, role: 'region', 'aria-label': '설명 표' }, [element]) : element;
}
