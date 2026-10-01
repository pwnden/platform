/// <reference lib="esnext.disposable" />
import type { HighlighterCore, ThemeRegistration } from 'shiki/types';

const languages = {
  python: () => import('@shikijs/langs/python'),
  bash: () => import('@shikijs/langs/bash'),
  javascript: () => import('@shikijs/langs/javascript'),
  typescript: () => import('@shikijs/langs/typescript'),
  json: () => import('@shikijs/langs/json'),
  html: () => import('@shikijs/langs/html'),
  css: () => import('@shikijs/langs/css'),
  c: () => import('@shikijs/langs/c'),
  cpp: () => import('@shikijs/langs/cpp'),
  go: () => import('@shikijs/langs/go'),
  rust: () => import('@shikijs/langs/rust'),
  sql: () => import('@shikijs/langs/sql'),
  yaml: () => import('@shikijs/langs/yaml'),
  toml: () => import('@shikijs/langs/toml'),
  dockerfile: () => import('@shikijs/langs/dockerfile'),
  java: () => import('@shikijs/langs/java'),
  php: () => import('@shikijs/langs/php'),
  powershell: () => import('@shikijs/langs/powershell'),
  asm: () => import('@shikijs/langs/asm'),
  diff: () => import('@shikijs/langs/diff'),
};
type Language = keyof typeof languages;
const aliases: Readonly<Record<string, Language>> = {
  py: 'python', sh: 'bash', shell: 'bash', shellscript: 'bash', console: 'bash', js: 'javascript', mjs: 'javascript', cjs: 'javascript',
  ts: 'typescript', h: 'c', cc: 'cpp', cxx: 'cpp', hpp: 'cpp', rs: 'rust', yml: 'yaml', ps1: 'powershell', assembly: 'asm', s: 'asm', patch: 'diff',
};

export function codeLanguage(language?: string, filename = ''): Language | undefined {
  const name = filename.split(/[\\/]/).at(-1)?.toLowerCase() ?? '';
  const hint = (language === undefined ? name === 'dockerfile' ? name : name.split('.').at(-1) : language.split(/\s/)[0])?.toLowerCase() ?? '';
  if (Object.hasOwn(languages, hint)) return hint as Language;
  return Object.hasOwn(aliases, hint) ? aliases[hint] : undefined;
}

const palette = { text: '#d2dfef', comment: '#8a9db6', keyword: '#c5acff', string: '#8fdbba', number: '#e7c38e', function: '#8cdce6', type: '#7ccaff' };
const colorClasses = new Map(Object.entries(palette).map(([name, color]) => [color, `ui-syntax--${name}`]));
const theme: ThemeRegistration = {
  name: 'pwnden', type: 'dark', colors: { 'editor.background': '#050a12', 'editor.foreground': palette.text },
  tokenColors: [
    { scope: ['comment', 'punctuation.definition.comment'], settings: { foreground: palette.comment } },
    { scope: ['keyword', 'storage', 'constant.language'], settings: { foreground: palette.keyword } },
    { scope: ['string'], settings: { foreground: palette.string } },
    { scope: ['constant.numeric'], settings: { foreground: palette.number } },
    { scope: ['entity.name.function', 'support.function'], settings: { foreground: palette.function } },
    { scope: ['entity.name.type', 'entity.name.tag', 'support.type'], settings: { foreground: palette.type } },
  ],
};

let highlighter: Promise<HighlighterCore> | undefined;
const loaded = new Map<Language, Promise<void>>();
export interface CodeToken { readonly content: string; readonly className: string }
export type CodeLines = readonly (readonly CodeToken[])[];

export async function highlightCode(source: string, language?: string, filename?: string): Promise<CodeLines | undefined> {
  const lang = codeLanguage(language, filename);
  // Bound synchronous tokenization; larger materials remain readable as escaped text.
  if (!lang || source.length > 128 * 1024) return;
  highlighter ??= Promise.all([import('shiki/core'), import('shiki/engine/javascript')]).then(([core, engine]) => core.createHighlighterCore({
    langs: [], themes: [theme], engine: engine.createJavaScriptRegexEngine(),
  })).catch(error => { highlighter = undefined; throw error; });
  const instance = await highlighter;
  if (!loaded.has(lang)) loaded.set(lang, instance.loadLanguage(languages[lang]).catch(error => { loaded.delete(lang); throw error; }));
  await loaded.get(lang);
  return instance.codeToTokens(source, { lang, theme: 'pwnden' }).tokens.map(line => line.map(token => ({
    content: token.content, className: colorClasses.get(token.color?.toLowerCase() ?? '') ?? 'ui-syntax--text',
  })));
}
