import assert from "node:assert/strict";
import test from "node:test";

import remarkParse from "remark-parse";
import { unified } from "unified";
import { VFile } from "vfile";

import remarkLintArcloomMainSpecification from "./index.mjs";

async function lint(value, path) {
  const file = new VFile({ path, value });
  const processor = unified()
    .use(remarkParse)
    .use(remarkLintArcloomMainSpecification);
  const tree = processor.parse(file);

  await processor.run(tree, file);

  return file.messages;
}

test("reports all Arcloom structure violations through one remark rule", async () => {
  const messages = await lint(
    [
      "### Requirement: INVALID SLUG",
      "",
      "- **Unknown**: value",
      "",
      "#### Scenario: example [unknown]",
    ].join("\n"),
    "/workspace/openspec/specs/example/spec.md",
  );

  assert.equal(messages.length, 3);
  assert.deepEqual(messages.map(({ source }) => source), [
    "remark-lint",
    "remark-lint",
    "remark-lint",
  ]);
  assert.deepEqual(messages.map(({ ruleId }) => ruleId), [
    "arcloom-main-specification",
    "arcloom-main-specification",
    "arcloom-main-specification",
  ]);
});

test("ignores files outside the main specification directory", async () => {
  const messages = await lint(
    [
      "### Requirement: INVALID SLUG",
      "",
      "- **Unknown**: value",
      "",
      "#### Scenario: example [unknown]",
    ].join("\n"),
    "/workspace/openspec/changes/example/specs/example/spec.md",
  );

  assert.deepEqual(messages, []);
});

test("ignores parsed Markdown without a file path", async () => {
  const file = new VFile({ value: "### Requirement: INVALID SLUG" });
  const processor = unified()
    .use(remarkParse)
    .use(remarkLintArcloomMainSpecification);
  const tree = processor.parse(file);

  await processor.run(tree, file);

  assert.deepEqual(file.messages, []);
});
