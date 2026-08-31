import test from 'node:test';
import assert from 'node:assert/strict';
import { unified } from 'unified';
import remarkParse from 'remark-parse';
import { VFile } from 'vfile';
import { validateRequirementBlocks } from './requirement-blocks.mjs';

function messages(markdown) {
  const tree = unified().use(remarkParse).parse(markdown);
  const file = new VFile({ path: 'spec.md', value: markdown });
  validateRequirementBlocks(tree, file);
  return file.messages.map((message) => message.reason);
}

test('accepts ordered blocks with omitted optional blocks', () => {
  assert.deepEqual(messages(`### Requirement: example\n\n- **Preconditions**: ready\n- **Behavioral Rules**: does it\n- **References**: [link](https://example.com)\n\n#### Scenario: works\n`), []);
});

test('reports unknown labels', () => {
  assert.match(messages('### Requirement: x\n\n- **Not A Block**: value')[0], /unknown Requirement block/);
});

test('reports duplicate labels', () => {
  assert.match(messages('### Requirement: x\n\n- **Invariants**: one\n- **Invariants**: two')[0], /repeats Requirement block/);
});

test('reports out-of-order labels', () => {
  assert.match(messages('### Requirement: x\n\n- **References**: one\n- **Invariants**: two')[0], /out of order/);
});

test('reports blocks after the first Scenario heading', () => {
  assert.match(messages('### Requirement: x\n\n#### Scenario: works\n\n- **References**: later')[0], /must precede Scenarios/);
});

test('resets state at the next Requirement', () => {
  assert.deepEqual(messages('### Requirement: one\n\n- **Invariants**: one\n\n### Requirement: two\n\n- **Invariants**: two'), []);
});

test('ignores code fences and HTML comments', () => {
  assert.deepEqual(messages('### Requirement: x\n\n```md\n- **Unknown**: code\n```\n\n<!-- - **Unknown**: comment -->\n'), []);
});

test('ignores GIVEN, WHEN, and THEN labels', () => {
  assert.deepEqual(messages('### Requirement: x\n\n- **GIVEN**: setup\n- **WHEN**: action\n- **THEN**: result\n'), []);
});

test('ignores Requirement-looking blocks inside top-level blockquotes', () => {
  assert.deepEqual(messages('### Requirement: x\n\n> - **Unknown**: quoted\n'), []);
});

test('requires the colon to immediately follow the strong label', () => {
  assert.deepEqual(messages('### Requirement: x\n\n- **References** : spaced\n'), []);
});

test('requires labels to match exactly, without trimming strong-label whitespace', () => {
  assert.match(messages('### Requirement: x\n\n- **`References `**: spaced')[0], /unknown Requirement block/);
});
