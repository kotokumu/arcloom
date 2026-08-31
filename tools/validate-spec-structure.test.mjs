import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const command = path.join(repositoryRoot, 'tools', 'validate-spec-structure.mjs');

const structure = {
  version: 2,
  requirement_slug_pattern: '^[a-z0-9]+(?:-[a-z0-9]+)*$',
  blocks: [
    { label: '前提条件' },
    { label: '入力と受理' },
    { label: '振る舞いの規則' },
    { label: '不変条件' },
    { label: '副作用' },
    { label: '排他・冪等' },
    { label: '失敗の扱い' },
    { label: '参照' },
  ],
  scenario: { allowed_tags: ['happy', 'error', 'boundary', 'permission', 'concurrency', 'idempotency', 'compatibility'] },
};

function workspace(t, spec) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'arcloom-spec-structure-'));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  fs.mkdirSync(path.join(root, 'openspec', 'specs', '_schema'), { recursive: true });
  fs.mkdirSync(path.join(root, 'openspec', 'specs', 'fixture'), { recursive: true });
  fs.writeFileSync(
    path.join(root, 'openspec', 'specs', '_schema', 'requirement-structure.json'),
    `${JSON.stringify(structure, null, 2)}\n`,
  );
  fs.writeFileSync(path.join(root, 'openspec', 'specs', 'fixture', 'spec.md'), spec);
  return root;
}

function run(root) {
  return spawnSync(process.execPath, [command, root], { encoding: 'utf8' });
}

function spec(body) {
  return `# Fixture Specification\n\n## Purpose\n\nA sufficiently complete fixture purpose for structural validation.\n\n## Requirements\n\n${body}`;
}

function requirement(extra = '') {
  return `### Requirement: observe-result\n\nThe fixture MUST expose one result.\n\n` +
    `- **入力と受理**: The request is valid.\n` +
    `- **振る舞いの規則**: The result is observable.\n` +
    `- **不変条件**: The request identity remains unchanged.\n${extra}\n` +
    `#### Scenario: Result is observed [happy]\n\n` +
    `- **GIVEN** a valid request\n- **WHEN** it is submitted\n- **THEN** the result is observable\n`;
}

test('accepts canonical blocks and an allowed Scenario tag', (t) => {
  const result = run(workspace(t, spec(requirement())));
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /Validated 1 main spec/);
});

for (const [name, extra, message] of [
  ['unknown block', '- **状態と遷移**: Active becomes complete.\n', /unknown Requirement block "状態と遷移"/],
  ['repeated block', '- **振る舞いの規則**: The result is repeated.\n', /repeats Requirement block "振る舞いの規則"/],
  ['out-of-order block', '- **前提条件**: The fixture exists.\n', /Requirement block "前提条件" is out of order/],
]) {
  test(`rejects ${name}`, (t) => {
    const result = run(workspace(t, spec(requirement(extra))));
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, message);
    assert.match(result.stderr, /openspec\/specs\/fixture\/spec\.md:\d+/);
  });
}

test('rejects an unknown Scenario tag', (t) => {
  const input = requirement().replace('[happy]', '[isolation]');
  const result = run(workspace(t, spec(input)));
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /unknown Scenario tag "isolation"/);
});

test('rejects a missing Scenario tag', (t) => {
  const input = requirement().replace(' [happy]', '');
  const result = run(workspace(t, spec(input)));
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /must end with one allowed tag/);
});

test('ignores structural-looking Markdown in fences and HTML comments', (t) => {
  const ignored = `\n\`\`\`markdown\n- **Unknown fenced block**: ignored\n#### Scenario: Ignored [unknown]\n\`\`\`\n` +
    `<!--\n- **Unknown commented block**: ignored\n#### Scenario: Ignored [unknown]\n-->\n`;
  const result = run(workspace(t, spec(`${ignored}\n${requirement()}`)));
  assert.equal(result.status, 0, result.stderr);
});
