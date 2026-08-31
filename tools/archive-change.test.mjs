import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const archiveCommand = path.join(repositoryRoot, 'tools', 'archive-change.mjs');
const templateConfig = path.join(repositoryRoot, 'openspec', 'config.yaml');
const templateSchema = path.join(repositoryRoot, 'openspec', 'schemas', 'quality-spec');

function run(command, args, cwd) {
  return spawnSync(command, args, { cwd, encoding: 'utf8' });
}

function requireSuccess(result, description) {
  assert.equal(
    result.status,
    0,
    `${description} failed:\n${[result.stdout, result.stderr].filter(Boolean).join('\n')}`,
  );
}

function createWorkspace(t) {
  const workspace = fs.mkdtempSync(path.join(os.tmpdir(), 'arcloom-openspec-'));
  t.after(() => fs.rmSync(workspace, { recursive: true, force: true }));

  const openspecRoot = path.join(workspace, 'openspec');
  fs.mkdirSync(path.join(openspecRoot, 'schemas'), { recursive: true });
  fs.mkdirSync(path.join(openspecRoot, 'specs'), { recursive: true });
  fs.copyFileSync(templateConfig, path.join(openspecRoot, 'config.yaml'));
  fs.cpSync(templateSchema, path.join(openspecRoot, 'schemas', 'quality-spec'), { recursive: true });
  return workspace;
}

function writeFile(workspace, relativePath, content) {
  const target = path.join(workspace, relativePath);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(target, content, 'utf8');
}

function newChange(workspace, name) {
  const result = run('openspec', ['new', 'change', name, '--json'], workspace);
  requireSuccess(result, `create change ${name}`);
  const parsed = JSON.parse(result.stdout);
  assert.equal(parsed.change.schema, 'quality-spec');
}

function modelWith(replacements) {
  return `# Specification Analysis\n\n` +
    `## 4. Main Spec Conceptual Model Replacements\n\n${replacements}\n\n` +
    `## 6. Unresolved Decisions\n\nNone.\n`;
}

function writePlanningArtifacts(workspace, name, {
  replacements = 'None.',
  deltas = new Map(),
  skipSpecs = false,
  tasksComplete = true,
} = {}) {
  newChange(workspace, name);
  const changeRoot = `openspec/changes/${name}`;
  if (skipSpecs) {
    const metadata = path.join(workspace, changeRoot, '.openspec.yaml');
    fs.appendFileSync(metadata, 'skip_specs: true\n', 'utf8');
  }

  writeFile(workspace, `${changeRoot}/proposal.md`, '## Why\n\nValidate publication behavior.\n');
  writeFile(workspace, `${changeRoot}/model.md`, modelWith(replacements));
  writeFile(workspace, `${changeRoot}/design.md`, '## Context\n\nDisposable publication fixture.\n');
  writeFile(
    workspace,
    `${changeRoot}/tasks.md`,
    `## Verification\n\n- [${tasksComplete ? 'x' : ' '}] 1.1 [change] Verify publication behavior\n`,
  );

  for (const [capability, delta] of deltas) {
    writeFile(workspace, `${changeRoot}/specs/${capability}/spec.md`, delta);
  }
}

function scenario(name, outcome) {
  return `#### Scenario: ${name} [happy]\n\n` +
    `- **GIVEN** a consumer with a valid request\n` +
    `- **WHEN** the consumer submits the request\n` +
    `- **THEN** ${outcome}\n`;
}

function requirement(slug, guarantee, scenarioName = 'Observe result') {
  return `### Requirement: ${slug}\n\n` +
    `The product MUST ${guarantee}.\n\n` +
    scenario(scenarioName, guarantee);
}

function mainSpec(capability, { conceptualModel = null, requirements }) {
  const purpose = `The ${capability} capability provides a stable observable contract for publication testing and verification.`;
  return `# ${capability} Specification\n\n## Purpose\n\n${purpose}\n\n` +
    (conceptualModel === null ? '' : `## Conceptual Model\n\n${conceptualModel}\n\n`) +
    `## Requirements\n\n${requirements.join('\n')}`;
}

function publish(workspace, name, env = process.env) {
  return spawnSync(process.execPath, [archiveCommand, name], {
    cwd: workspace,
    encoding: 'utf8',
    env,
  });
}

function archivedChange(workspace, name) {
  const archiveRoot = path.join(workspace, 'openspec', 'changes', 'archive');
  if (!fs.existsSync(archiveRoot)) return null;
  const entry = fs.readdirSync(archiveRoot).find((candidate) => candidate.endsWith(`-${name}`) || candidate === name);
  return entry ? path.join(archiveRoot, entry) : null;
}

test('publishes a nested new capability without losing conceptual or requirement references', (t) => {
  const workspace = createWorkspace(t);
  const name = 'new-capability';
  const capability = 'platform/search';
  const replacements = `### \`${capability}\`\n\n` +
    '```markdown\n' +
    'Search term is the non-empty matching criterion supplied by a consumer.\n\n' +
    '### References\n\n- `[policy]` docs/search-policy.md\n' +
    '```';
  const delta = `## Purpose\n\n` +
    `This capability lets consumers submit a search term and observe all entries satisfying the defined matching rule.\n\n` +
    `## ADDED Requirements\n\n` +
    requirement('return-matching-items', 'return every entry matching the accepted search term') +
    `\n- **References**: [policy] docs/search-policy.md\n`;

  writePlanningArtifacts(workspace, name, {
    replacements,
    deltas: new Map([[capability, delta]]),
  });

  const result = publish(workspace, name);
  requireSuccess(result, 'publish nested new capability');

  const published = fs.readFileSync(path.join(workspace, 'openspec', 'specs', 'platform', 'search', 'spec.md'), 'utf8');
  assert.match(published, /## Purpose/);
  assert.match(published, /## Conceptual Model/);
  assert.match(published, /### References/);
  assert.match(published, /docs\/search-policy\.md/);
  assert.match(published, /### Requirement: return-matching-items/);
  assert.ok(archivedChange(workspace, name));
  assert.equal(fs.existsSync(path.join(workspace, 'openspec', 'changes', name)), false);

  requireSuccess(run('openspec', ['validate', '--specs', '--strict'], workspace), 'strict main-spec validation');
});

test('rejects an unknown delta section after OpenSpec alternate comment termination', (t) => {
  const workspace = createWorkspace(t);
  const name = 'unknown-section';
  const capability = 'platform/search';
  const delta = `## Purpose\n\n` +
    `This capability provides a sufficiently concrete purpose for validating rejected publication input.\n\n` +
    `## ADDED Requirements\n\n` +
    requirement('return-items', 'return the observable fixture result') +
    `\n<!-- OpenSpec also closes comments with --!>\n` +
    `## References\n\n- This section would be discarded by OpenSpec.\n`;

  writePlanningArtifacts(workspace, name, {
    replacements: 'None.',
    deltas: new Map([[capability, delta]]),
  });

  const result = publish(workspace, name);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /contains unsupported section "## References"/);
  assert.equal(fs.existsSync(path.join(workspace, 'openspec', 'specs', 'platform', 'search', 'spec.md')), false);
  assert.equal(fs.existsSync(path.join(workspace, 'openspec', 'changes', name)), true);
  assert.equal(archivedChange(workspace, name), null);
});

test('ignores level-two examples inside a fenced block with an info string', (t) => {
  const workspace = createWorkspace(t);
  const name = 'fenced-example';
  const capability = 'fixture/example';
  const delta = `## Purpose\n\n` +
    `This capability provides a concrete observable contract while documenting a fenced Markdown example.\n\n` +
    `## ADDED Requirements\n\n` +
    `### Requirement: show-example\n\n` +
    `The product MUST return an observable example.\n\n` +
    '```markdown title="example"\n## References\n```\n\n' +
    scenario('Show example', 'the observable example is returned');

  writePlanningArtifacts(workspace, name, {
    replacements: 'None.',
    deltas: new Map([[capability, delta]]),
  });

  const result = publish(workspace, name);
  requireSuccess(result, 'publish a delta containing a fenced heading example');
  assert.ok(archivedChange(workspace, name));
});

test('replaces an existing conceptual model while preserving unrelated main-spec content', (t) => {
  const workspace = createWorkspace(t);
  const name = 'replace-existing';
  const capability = 'catalog/search';
  const original = mainSpec(capability, {
    conceptualModel: 'Search term is the original matching criterion.\n\n### References\n\n- `[policy]` docs/original-policy.md',
    requirements: [
      requirement('keep-existing', 'keep an unrelated observable result'),
      requirement('return-items', 'return the original matching result'),
    ],
  });
  writeFile(workspace, `openspec/specs/${capability}/spec.md`, original);

  const replacements = `### \`${capability}\`\n\n` +
    '```markdown\n' +
    'Search term is the accepted matching criterion.\n\n' +
    '### References\n\n- `[policy]` docs/current-policy.md\n' +
    '```';
  const delta = `## MODIFIED Requirements\n\n` +
    requirement('return-items', 'return the current matching result');
  writePlanningArtifacts(workspace, name, {
    replacements,
    deltas: new Map([[capability, delta]]),
  });

  const result = publish(workspace, name);
  requireSuccess(result, 'publish existing capability replacement');

  const published = fs.readFileSync(path.join(workspace, 'openspec', 'specs', 'catalog', 'search', 'spec.md'), 'utf8');
  assert.match(published, /provides a stable observable contract for publication testing/);
  assert.match(published, /Search term is the accepted matching criterion/);
  assert.match(published, /docs\/current-policy\.md/);
  assert.doesNotMatch(published, /docs\/original-policy\.md/);
  assert.match(published, /### Requirement: keep-existing/);
  assert.match(published, /keep an unrelated observable result/);
  assert.match(published, /return the current matching result/);
  assert.ok(archivedChange(workspace, name));
});

test('restores every staged main spec when a later conceptual-model operation fails', (t) => {
  const workspace = createWorkspace(t);
  const name = 'rollback-staged-specs';
  const alpha = 'fixture/alpha';
  const beta = 'fixture/beta';
  const alphaOriginal = mainSpec(alpha, {
    conceptualModel: 'Alpha concept is unchanged before publication.',
    requirements: [requirement('observe-alpha', 'return the alpha result')],
  });
  const betaOriginal = mainSpec(beta, {
    requirements: [requirement('observe-beta', 'return the beta result')],
  });
  writeFile(workspace, `openspec/specs/${alpha}/spec.md`, alphaOriginal);
  writeFile(workspace, `openspec/specs/${beta}/spec.md`, betaOriginal);

  const replacements = `### \`${alpha}\`\n\n` +
    '```markdown\nAlpha concept would be replaced.\n```\n\n' +
    `### \`${beta}\`\n\nREMOVE`;
  writePlanningArtifacts(workspace, name, {
    replacements,
    deltas: new Map([
      [alpha, `## MODIFIED Requirements\n\n${requirement('observe-alpha', 'return the revised alpha result')}`],
      [beta, `## MODIFIED Requirements\n\n${requirement('observe-beta', 'return the revised beta result')}`],
    ]),
  });

  const result = publish(workspace, name);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /fixture\/beta has no Conceptual Model section to remove/);
  assert.equal(fs.readFileSync(path.join(workspace, `openspec/specs/${alpha}/spec.md`), 'utf8'), alphaOriginal);
  assert.equal(fs.readFileSync(path.join(workspace, `openspec/specs/${beta}/spec.md`), 'utf8'), betaOriginal);
  assert.equal(fs.existsSync(path.join(workspace, 'openspec', 'changes', name)), true);
  assert.equal(archivedChange(workspace, name), null);
});

test('restores specs and the active change when post-archive strict validation fails', (t) => {
  const workspace = createWorkspace(t);
  const name = 'invalid-published-spec';
  const capability = 'fixture/invalid-after-archive';
  const delta = `## Purpose\n\nToo short.\n\n` +
    `## ADDED Requirements\n\n` +
    requirement('observe-result', 'return the observable result');
  writePlanningArtifacts(workspace, name, {
    deltas: new Map([[capability, delta]]),
  });

  const result = publish(workspace, name);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Purpose section is too brief/);
  assert.equal(fs.existsSync(path.join(workspace, `openspec/specs/${capability}/spec.md`)), false);
  assert.equal(fs.existsSync(path.join(workspace, 'openspec', 'changes', name)), true);
  assert.equal(archivedChange(workspace, name), null);
});

test('does not overwrite a concurrent main-spec update during rollback', (t) => {
  const workspace = createWorkspace(t);
  const name = 'concurrent-rollback';
  const capability = 'fixture/concurrent';
  const target = path.join(workspace, `openspec/specs/${capability}/spec.md`);
  const original = mainSpec(capability, {
    requirements: [requirement('observe-result', 'return the original result')],
  });
  const concurrent = mainSpec(capability, {
    requirements: [requirement('observe-result', 'return the concurrently edited result')],
  });
  writeFile(workspace, `openspec/specs/${capability}/spec.md`, original);
  writePlanningArtifacts(workspace, name, {
    deltas: new Map([[
      capability,
      `## MODIFIED Requirements\n\n${requirement('observe-result', 'return the published result')}`,
    ]]),
  });

  const realOpenSpec = run('which', ['openspec'], repositoryRoot);
  requireSuccess(realOpenSpec, 'locate openspec');
  const binDirectory = path.join(workspace, 'fixture-bin');
  const counterFile = path.join(workspace, 'validate-count');
  fs.mkdirSync(binDirectory);
  const wrapper = path.join(binDirectory, 'openspec');
  fs.writeFileSync(wrapper, `#!/usr/bin/env node
const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const args = process.argv.slice(2);
const counter = ${JSON.stringify(counterFile)};
if (args.join(' ') === 'validate --specs --strict --json') {
  const count = fs.existsSync(counter) ? Number(fs.readFileSync(counter, 'utf8')) + 1 : 1;
  fs.writeFileSync(counter, String(count));
  if (count === 2) {
    fs.writeFileSync(${JSON.stringify(target)}, ${JSON.stringify(concurrent)}, 'utf8');
    process.stderr.write('injected post-archive validation failure\\n');
    process.exit(42);
  }
}
const result = spawnSync(${JSON.stringify(realOpenSpec.stdout.trim())}, args, { stdio: 'inherit' });
process.exit(result.status ?? 1);
`, 'utf8');
  fs.chmodSync(wrapper, 0o755);

  const result = publish(workspace, name, {
    ...process.env,
    PATH: `${binDirectory}${path.delimiter}${process.env.PATH}`,
  });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /changed concurrently\. No files were restored/);
  assert.equal(fs.readFileSync(target, 'utf8'), concurrent);
  assert.equal(fs.existsSync(path.join(workspace, 'openspec', 'changes', name)), false);
  assert.ok(archivedChange(workspace, name));
});

test('archives a skip_specs change without modifying main specs', (t) => {
  const workspace = createWorkspace(t);
  const name = 'pure-refactor';
  const capability = 'fixture/existing';
  const original = mainSpec(capability, {
    requirements: [requirement('observe-existing', 'return the existing result')],
  });
  writeFile(workspace, `openspec/specs/${capability}/spec.md`, original);
  writePlanningArtifacts(workspace, name, { skipSpecs: true });

  const result = publish(workspace, name);
  requireSuccess(result, 'archive skip_specs change');
  assert.equal(fs.readFileSync(path.join(workspace, `openspec/specs/${capability}/spec.md`), 'utf8'), original);
  assert.ok(archivedChange(workspace, name));
  assert.equal(fs.existsSync(path.join(workspace, 'openspec', 'changes', name)), false);
});

test('rejects incomplete tasks without modifying main specs or moving the change', (t) => {
  const workspace = createWorkspace(t);
  const name = 'incomplete-task';
  const capability = 'fixture/existing';
  const original = mainSpec(capability, {
    requirements: [requirement('observe-existing', 'return the existing result')],
  });
  writeFile(workspace, `openspec/specs/${capability}/spec.md`, original);
  writePlanningArtifacts(workspace, name, { skipSpecs: true, tasksComplete: false });

  const result = publish(workspace, name);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /has 1 incomplete task/);
  assert.equal(fs.readFileSync(path.join(workspace, `openspec/specs/${capability}/spec.md`), 'utf8'), original);
  assert.equal(fs.existsSync(path.join(workspace, 'openspec', 'changes', name)), true);
  assert.equal(archivedChange(workspace, name), null);
});

test('refuses publication while another archive process owns the lock', (t) => {
  const workspace = createWorkspace(t);
  const name = 'locked-publication';
  writePlanningArtifacts(workspace, name, { skipSpecs: true });
  const lockFile = path.join(workspace, 'openspec', '.archive-change.lock');
  fs.mkdirSync(lockFile);
  fs.writeFileSync(path.join(lockFile, 'owner.json'), '{"pid":123,"change":"other-change"}\n', 'utf8');

  const result = publish(workspace, name);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Another publication is active/);
  assert.equal(fs.existsSync(lockFile), true);
  assert.equal(fs.existsSync(path.join(workspace, 'openspec', 'changes', name)), true);
  assert.equal(archivedChange(workspace, name), null);
});

test('generated agent skills preserve the quality-spec publication boundary', () => {
  for (const agentRoot of ['.agents', '.claude']) {
    const archiveSkill = fs.readFileSync(
      path.join(repositoryRoot, agentRoot, 'skills', 'openspec-archive-change', 'SKILL.md'),
      'utf8',
    );
    const syncSkill = fs.readFileSync(
      path.join(repositoryRoot, agentRoot, 'skills', 'openspec-sync-specs', 'SKILL.md'),
      'utf8',
    );

    assert.match(archiveSkill, /Bash\(node tools\/archive-change\.mjs:\*\)/);
    assert.match(archiveSkill, /If `schemaName` is `quality-spec`, use the repository publisher/);
    assert.match(archiveSkill, /acquire the shared publication lock/);
    assert.match(syncSkill, /If `schemaName` is `quality-spec`, stop without writing main specs/);
    assert.match(syncSkill, /acquire the shared lock/);
  }
});

test('spec instructions inject the WHAT and HOW separation contract', (t) => {
  const workspace = createWorkspace(t);
  const name = 'instruction-contract';
  newChange(workspace, name);
  writeFile(workspace, `openspec/changes/${name}/proposal.md`, '## Why\n\nVerify instruction generation.\n');
  writeFile(workspace, `openspec/changes/${name}/model.md`, modelWith('None.'));

  const result = run('openspec', ['instructions', 'specs', '--change', name, '--json'], workspace);
  requireSuccess(result, 'generate spec instructions');
  const instructions = JSON.parse(result.stdout).instruction;
  assert.match(instructions, /defines WHAT the system guarantees/);
  assert.match(instructions, /does not describe HOW the\s+implementation works/);
  assert.match(instructions, /Physical database schemas, API payload shapes, UI component structures/);
  assert.match(instructions, /The permanent conceptual model belongs in the owning main spec/);
});
