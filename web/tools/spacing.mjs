import { readdirSync, readFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { compileStyle, parse } from '@vue/compiler-sfc';

function values(value) {
  const result = [];
  let depth = 0, token = '';
  for (const character of value.trim()) {
    if (character === '(') depth++;
    if (character === ')') depth--;
    if (/\s/.test(character) && depth === 0) {
      if (token) result.push(token);
      token = '';
    } else token += character;
  }
  if (token) result.push(token);
  return result;
}

export function checkSpacing(source, filename = 'style.css') {
  const styles = filename.endsWith('.vue')
    ? parse(source, { filename }).descriptor.styles.map(style => style.content)
    : [source];
  const findings = [];
  for (const style of styles) {
    const result = compileStyle({ source: style, filename, id: 'spacing' });
    if (result.errors.length) {
      findings.push(`${filename}: cannot parse component style`);
      continue;
    }
    result.rawResult.root.walkDecls(declaration => {
      const property = declaration.prop.toLowerCase();
      if (!/^padding(?:-|$)/.test(property)) return;
      const value = declaration.value;
      const tokens = values(value);
      if (property !== 'padding' || !tokens.length || tokens.length > 4 || tokens.some(token => token !== tokens[0])) {
        findings.push(`${filename}: ${property}: ${value.trim()} — use one inset for all four sides`);
      }
    });
  }
  return findings;
}

function files(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
    const path = join(directory, entry.name);
    if (['node_modules', 'dist'].includes(entry.name) || entry.name.startsWith('.')) return [];
    return entry.isDirectory() ? files(path) : /\.(vue|css)$/.test(entry.name) ? [path] : [];
  });
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
  const sources = ['apps', 'features', 'packages', 'domains'].flatMap(path => files(join(root, path)));
  const findings = sources.flatMap(path => checkSpacing(readFileSync(path, 'utf8'), path));
  if (findings.length) {
    process.stderr.write(`${findings.join('\n')}\n`);
    process.exitCode = 1;
  } else process.stdout.write('Symmetric component insets passed.\n');
}
