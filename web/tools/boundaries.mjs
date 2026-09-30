import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import ts from 'typescript';
import { parse } from '@vue/compiler-sfc';

// This policy is independent of manifests: adding a dependency cannot grant access.
export const policy = {
  '@pwnden/catalog': { directory: 'domains/catalog', dependencies: [] },
  '@pwnden/play': { directory: 'domains/play', dependencies: [] },
  '@pwnden/terminal': { directory: 'domains/terminal', dependencies: [] },
  '@pwnden/ui': { directory: 'packages/ui', dependencies: ['vue', '@sectile/vue', '@xterm/xterm', '@xterm/addon-fit', 'md4x'] },
  '@pwnden/api': { directory: 'packages/api', dependencies: ['@pwnden/catalog', '@pwnden/play', '@pwnden/terminal'] },
  '@pwnden/catalog-feature': { directory: 'features/catalog', dependencies: ['vue', '@pwnden/catalog', '@pwnden/ui'] },
  '@pwnden/play-feature': { directory: 'features/play', dependencies: ['vue', '@pwnden/play', '@pwnden/ui'] },
  '@pwnden/terminal-feature': { directory: 'features/terminal', dependencies: ['vue', '@pwnden/terminal', '@pwnden/ui'] },
  '@pwnden/player': {
    directory: 'apps/player',
    dependencies: ['vue', '@pwnden/api', '@pwnden/catalog', '@pwnden/catalog-feature', '@pwnden/play-feature', '@pwnden/terminal-feature', '@pwnden/ui'],
    development: ['vite', '@vitejs/plugin-vue'],
  },
};

const exactVersion = /^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/;
const mixedAcronym = /(?:Api|Http|Html|Json|Css|Dom|Tcp|Url|Ui|Sdk|Dto|Ssr|Ci)(?=[A-Z0-9_]|$)/;
const inside = (file, directory) => file === directory || file.startsWith(`${directory}${path.sep}`);

function files(directory) {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap(entry => {
    if (['node_modules', 'dist'].includes(entry.name)) return [];
    const file = path.join(directory, entry.name);
    if (entry.isDirectory()) return files(file);
    return /\.(?:[cm]?[jt]sx?|vue|css)$/.test(entry.name) ? [file] : [];
  });
}

export function checkBoundaries(root) {
  root = fs.realpathSync(root);
  const errors = [];
  const report = (file, message) => errors.push(`${path.relative(root, file)}: ${message}`);
  const workspaceManifestFile = path.join(root, 'package.json');
  const workspaceManifest = JSON.parse(fs.readFileSync(workspaceManifestFile, 'utf8'));
  for (const group of ['dependencies', 'peerDependencies', 'devDependencies', 'optionalDependencies']) {
    for (const [name, version] of Object.entries(workspaceManifest[group] ?? {})) {
      const pinnedVersion = version.startsWith('npm:') ? version.slice(version.lastIndexOf('@') + 1) : version;
      if (!exactVersion.test(pinnedVersion)) report(workspaceManifestFile, `dependency must have an exact version: ${name}`);
      if (name.startsWith('@sectile/')) report(workspaceManifestFile, 'Sectile dependencies belong to the UI package');
    }
  }
  const packages = Object.entries(policy).map(([name, rule]) => {
    const directory = path.join(root, rule.directory);
    const manifest = JSON.parse(fs.readFileSync(path.join(directory, 'package.json'), 'utf8'));
    if (manifest.name !== name) report(directory, `expected package ${name}`);
    return { name, rule, directory: fs.realpathSync(directory), manifest };
  });
  const byName = new Map(packages.map(item => [item.name, item]));
  const owner = file => packages.find(item => inside(file, item.directory));
  const graph = new Map(packages.map(item => [item.name, new Set()]));
  const moduleGraph = new Map();
  function moduleEdge(file, target) {
    file = fs.realpathSync(file);
    const edges = moduleGraph.get(file) ?? new Set();
    edges.add(fs.realpathSync(target));
    moduleGraph.set(file, edges);
  }

  for (const parent of ['apps', 'packages', 'domains', 'features']) {
    for (const entry of fs.readdirSync(path.join(root, parent), { withFileTypes: true })) {
      if (!entry.isDirectory()) continue;
      const directory = path.join(root, parent, entry.name);
      if (!packages.some(item => item.directory === fs.realpathSync(directory))) report(directory, 'package needs an explicit boundary policy');
    }
  }

  function checkSpecifier(item, file, specifier) {
    const dependency = specifier.startsWith('@') ? specifier.split('/').slice(0, 2).join('/') : specifier.split('/')[0];
    if (specifier.startsWith('.') || path.isAbsolute(specifier)) {
      const target = path.resolve(path.dirname(file), specifier);
      const resolved = [target, `${target}.ts`, `${target}.vue`, `${target}.css`, path.join(target, 'index.ts')]
        .find(candidate => fs.existsSync(candidate) && fs.statSync(candidate).isFile());
      if (!resolved || owner(fs.realpathSync(resolved)) !== item || !inside(target, item.directory)) {
        report(file, `relative/absolute import must stay inside its package: ${specifier}`);
      } else moduleEdge(file, resolved);
      return;
    }
    const workspacePackage = byName.get(dependency);
    if (workspacePackage) {
      const subpath = specifier === dependency ? '.' : `.${specifier.slice(dependency.length)}`;
      if (!Object.hasOwn(workspacePackage.manifest.exports ?? {}, subpath)) {
        report(file, `only public package exports are allowed: ${specifier}`);
      } else {
        const target = path.resolve(workspacePackage.directory, workspacePackage.manifest.exports[subpath]);
        if (fs.existsSync(target)) moduleEdge(file, target);
      }
    }
    const developmentFile = !inside(file, path.join(item.directory, 'src'));
    const allowed = [...item.rule.dependencies, ...(developmentFile ? item.rule.development ?? [] : [])];
    const declared = {
      ...item.manifest.dependencies, ...item.manifest.peerDependencies,
      ...(developmentFile ? item.manifest.devDependencies : {}),
    };
    if (!allowed.includes(dependency) || !Object.hasOwn(declared, dependency)) {
      report(file, `dependency is not allowed and declared: ${specifier}`);
    }
    if (workspacePackage) graph.get(item.name).add(dependency);
  }

  function checkScript(item, file, content) {
    const source = ts.createSourceFile(file, content, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
    for (const diagnostic of source.parseDiagnostics) report(file, ts.flattenDiagnosticMessageText(diagnostic.messageText, ' '));
    for (const reference of source.referencedFiles) checkSpecifier(item, file, path.isAbsolute(reference.fileName) ? reference.fileName : `./${reference.fileName}`);
    if (item.rule.directory.startsWith('domains/') && (source.typeReferenceDirectives.length || source.libReferenceDirectives.length)) report(file, 'domain sources use the configured ES libraries without ambient references');
    function visit(node) {
      if (ts.isIdentifier(node) && mixedAcronym.test(node.text)) report(file, `use all-lowercase or all-uppercase acronyms: ${node.text}`);
      if (file.endsWith('vite.config.mjs') && ts.isIdentifier(node) && node.text === 'alias') report(file, 'Vite aliases require an explicit boundary policy');
      if ((ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) && node.moduleSpecifier) {
        checkSpecifier(item, file, node.moduleSpecifier.text);
        if (item.name === '@pwnden/ui' && ts.isExportDeclaration(node) && node.moduleSpecifier.text.startsWith('@sectile/')) {
          report(file, 'UI owns its public API; Sectile exports must stay private');
        }
      }
      if (ts.isImportEqualsDeclaration(node) && ts.isExternalModuleReference(node.moduleReference) && node.moduleReference.expression) {
        checkSpecifier(item, file, node.moduleReference.expression.text);
      }
      if (ts.isImportTypeNode(node) && ts.isLiteralTypeNode(node.argument)) checkSpecifier(item, file, node.argument.literal.text);
      if (ts.isCallExpression(node) && (node.expression.kind === ts.SyntaxKind.ImportKeyword || (ts.isIdentifier(node.expression) && node.expression.text === 'require'))) {
        const argument = node.arguments[0];
        if (!argument || !ts.isStringLiteralLike(argument)) report(file, 'module loading requires a literal public specifier');
        else checkSpecifier(item, file, argument.text);
      }
      if (ts.isPropertyAccessExpression(node) && ts.isMetaProperty(node.expression) && node.name.text === 'glob') {
        report(file, 'glob module loading needs an explicit boundary policy');
      }
      ts.forEachChild(node, visit);
    }
    visit(source);
  }

  function checkStyles(item, file, content) {
    for (const match of content.matchAll(/@(?:import|use|forward)\s+(?:url\(\s*)?["']([^"']+)["']/g)) checkSpecifier(item, file, match[1]);
    if (/@(?:import|use|forward)\s+(?!["']|url\(\s*["'])/.test(content)) report(file, 'style imports require a quoted literal specifier');
  }

  for (const item of packages) {
    const manifestFile = path.join(item.directory, 'package.json');
    for (const group of ['dependencies', 'peerDependencies', 'devDependencies', 'optionalDependencies']) {
      for (const [dependency, version] of Object.entries(item.manifest[group] ?? {})) {
        const allowed = [...item.rule.dependencies, ...(group === 'devDependencies' ? item.rule.development ?? [] : [])];
        if (!allowed.includes(dependency)) report(manifestFile, `manifest dependency is not allowed: ${dependency}`);
        const target = byName.get(dependency);
        if (target ? version !== `workspace:${target.manifest.version}` : !exactVersion.test(version)) report(manifestFile, `dependency must have an exact version: ${dependency}`);
        if (target) graph.get(item.name).add(dependency);
      }
    }
    for (const [key, value] of Object.entries(item.manifest.exports ?? {})) {
      if (typeof value !== 'string' || !value.startsWith('./src/') || key.includes('*') || value.includes('*')) report(manifestFile, 'exports must name explicit source entry points');
      else {
        const target = path.resolve(item.directory, value);
        if (!fs.existsSync(target) || !inside(fs.realpathSync(target), item.directory)) report(manifestFile, 'export must resolve inside its package');
      }
    }
    if (item.manifest.imports) report(manifestFile, 'package aliases require an explicit boundary policy');
    const configFile = path.join(item.directory, 'tsconfig.json');
    const config = ts.readConfigFile(configFile, ts.sys.readFile);
    const parsed = ts.parseJsonConfigFileContent(config.config, ts.sys, item.directory);
    if (parsed.options.paths || parsed.options.baseUrl) report(configFile, 'use public package exports for cross-package resolution');
    if (item.rule.directory.startsWith('domains/') && (parsed.options.lib?.some(lib => lib !== 'lib.es2023.d.ts') || parsed.options.types?.length)) report(configFile, 'domains use ES libraries without ambient framework/browser types');

    for (const file of files(item.directory)) {
      if (owner(fs.realpathSync(file)) !== item) { report(file, 'source symlinks must stay inside their package'); continue; }
      const content = fs.readFileSync(file, 'utf8');
      if (file.endsWith('.vue')) {
        const result = parse(content, { filename: file });
        for (const error of result.errors) report(file, String(error));
        for (const script of [result.descriptor.script, result.descriptor.scriptSetup]) {
          if (script?.src) report(file, 'SFC scripts must be local inline modules');
          if (script) checkScript(item, file, script.content);
        }
        for (const style of result.descriptor.styles) {
          if (style.src) checkSpecifier(item, file, style.src);
          checkStyles(item, file, style.content);
        }
        if (result.descriptor.template?.src) report(file, 'SFC templates must be local inline markup');
      } else if (file.endsWith('.css')) checkStyles(item, file, content);
      else checkScript(item, file, content);
    }
  }

  const visiting = new Set();
  const visited = new Set();
  function visit(name) {
    if (visiting.has(name)) { report(root, `dependency cycle at ${name}`); return; }
    if (visited.has(name)) return;
    visiting.add(name);
    for (const next of graph.get(name) ?? []) visit(next);
    visiting.delete(name);
    visited.add(name);
  }
  for (const name of graph.keys()) visit(name);
  visiting.clear();
  visited.clear();
  function visitModule(file) {
    if (visiting.has(file)) { report(file, 'module dependency cycle'); return; }
    if (visited.has(file)) return;
    visiting.add(file);
    for (const next of moduleGraph.get(file) ?? []) visitModule(next);
    visiting.delete(file);
    visited.add(file);
  }
  for (const file of moduleGraph.keys()) visitModule(file);
  return errors;
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const errors = checkBoundaries(fileURLToPath(new URL('..', import.meta.url)));
  if (errors.length) { console.error(errors.join('\n')); process.exitCode = 1; }
  else console.log('Package boundaries and acronym spelling passed.');
}
