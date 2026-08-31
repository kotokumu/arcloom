import assert from "node:assert/strict";
import test from "node:test";
import { unified } from "unified";
import remarkParse from "remark-parse";
import { VFile } from "vfile";
import { validateScenarioHeadings } from "./scenario-heading.mjs";

async function messages(markdown) {
  const file = new VFile({ path: "spec.md", value: markdown });
  const processor = unified().use(remarkParse);
  const tree = processor.parse(file);
  validateScenarioHeadings(tree, file);
  return file.messages;
}

const validTags = [
  "happy",
  "error",
  "boundary",
  "permission",
  "concurrency",
  "idempotency",
  "compatibility",
];

test("accepts every allowed Scenario tag", async () => {
  const document = [
    "## Requirements",
    "### Requirement: example",
    ...validTags.map((tag) => `#### Scenario: Example [${tag}]`),
  ].join("\n");

  assert.deepEqual(await messages(document), []);
});

test("reports a missing tag at the Scenario heading", async () => {
  const result = await messages([
    "## Requirements",
    "### Requirement: example",
    "#### Scenario: Missing tag",
  ].join("\n"));

  assert.equal(result.length, 1);
  assert.equal(result[0].reason, "Scenario heading must end with one allowed tag.");
  assert.equal(result[0].line, 3);
  assert.equal(result[0].column, 1);
});

test("reports an unknown tag at the Scenario heading", async () => {
  const result = await messages([
    "## Requirements",
    "### Requirement: example",
    "#### Scenario: Unknown [eventual]",
  ].join("\n"));

  assert.equal(result.length, 1);
  assert.match(result[0].reason, /Unknown Scenario tag: eventual/);
  assert.equal(result[0].line, 3);
});

test("compares the terminal tag exactly without trimming", async () => {
  const result = await messages([
    "## Requirements",
    "### Requirement: example",
    "#### Scenario: Spaced tag [ happy ]",
  ].join("\n"));

  assert.equal(result.length, 1);
  assert.match(result[0].reason, /Unknown Scenario tag:  happy /);
});

test("allows earlier bracket groups when the terminal tag is allowed", async () => {
  assert.deepEqual(
    await messages([
      "## Requirements",
      "### Requirement: example",
      "#### Scenario: Brackets in title [example] [happy]",
    ].join("\n")),
    [],
  );
});

test("reports an empty terminal tag as missing", async () => {
  const result = await messages([
    "## Requirements",
    "### Requirement: example",
    "#### Scenario: Empty tag []",
  ].join("\n"));

  assert.equal(result.length, 1);
  assert.equal(result[0].reason, "Scenario heading must end with one allowed tag.");
});

test("ignores Scenario headings outside a Requirement section", async () => {
  assert.deepEqual(
    await messages(["## Other", "#### Scenario: Outside"].join("\n")),
    [],
  );
});

test("resets Requirement scope at the next depth-2 heading", async () => {
  assert.deepEqual(
    await messages([
      "## Requirements",
      "### Requirement: first",
      "#### Scenario: Valid [happy]",
      "## Other",
      "#### Scenario: Ignored",
    ].join("\n")),
    [],
  );
});

test("resets Requirement scope at every unrelated top-level depth-3 heading", async () => {
  assert.deepEqual(
    await messages([
      "## Requirements",
      "### Requirement: first",
      "#### Scenario: Valid [happy]",
      "### Notes",
      "#### Scenario: Ignored",
    ].join("\n")),
    [],
  );
});

test("ignores code fences and comments", async () => {
  assert.deepEqual(
    await messages([
      "## Requirements",
      "### Requirement: example",
      "```md",
      "#### Scenario: In a fence",
      "```",
      "<!-- #### Scenario: In a comment -->",
      "#### Scenario: Valid [happy]",
    ].join("\n")),
    [],
  );
});

test("ignores unrelated depth-4 headings", async () => {
  assert.deepEqual(
    await messages([
      "## Requirements",
      "### Requirement: example",
      "#### Notes without Scenario prefix",
      "#### Scenario details without the required prefix",
    ].join("\n")),
    [],
  );
});
