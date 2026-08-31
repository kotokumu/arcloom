import assert from 'node:assert/strict';
import test from 'node:test';
import { unified } from 'unified';
import remarkParse from 'remark-parse';
import { VFile } from 'vfile';
import { validateRequirementHeadings } from './requirement-heading.mjs';

function check(markdown) {
  const file = new VFile({ path: 'fixture.md', value: markdown });
  const tree = unified().use(remarkParse).parse(file);
  validateRequirementHeadings(tree, file);
  return file.messages;
}

test('accepts a lowercase kebab-case Requirement slug', () => {
  assert.deepEqual(check('### Requirement: valid-requirement\n'), []);
});

test('rejects uppercase, spaces, and empty Requirement slugs', () => {
  const messages = check([
    '### Requirement: Uppercase\n',
    '### Requirement: has spaces\n',
    '### Requirement:\n',
  ].join('\n'));

  assert.equal(messages.length, 3);
  assert.deepEqual(messages.map((message) => message.line), [1, 3, 5]);
});

test('ignores Requirement-looking text in code fences and HTML comments', () => {
  assert.deepEqual(check([
    '```markdown',
    '### Requirement: Uppercase',
    '```',
    '',
    '<!--',
    '### Requirement: has spaces',
    '-->',
  ].join('\n')), []);
});

test('ignores unrelated headings', () => {
  assert.deepEqual(check([
    '# Requirement: Uppercase',
    '## Requirement: has spaces',
    '### Other: Uppercase',
    '#### Requirement: has spaces',
  ].join('\n')), []);
});

test('ignores Requirement headings inside blockquotes', () => {
  assert.deepEqual(check('> ### Requirement: Invalid Slug\n'), []);
});
