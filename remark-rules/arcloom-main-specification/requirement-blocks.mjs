import { toString } from "mdast-util-to-string";
import { visit } from "unist-util-visit";

const BLOCK_LABELS = [
  "Preconditions",
  "Input and Acceptance",
  "Behavioral Rules",
  "Invariants",
  "Side Effects",
  "Concurrency and Idempotency",
  "Failure Handling",
  "References",
];

const IGNORED_LABELS = new Set(["GIVEN", "WHEN", "THEN"]);
const BLOCK_INDEX = new Map(BLOCK_LABELS.map((label, index) => [label, index]));

function labelOf(listItem) {
  const paragraph = listItem.children?.find(
    (child) => child.type === "paragraph",
  );
  if (!paragraph || paragraph.children?.[0]?.type !== "strong") return null;
  const strong = paragraph.children[0];
  const label = toString(strong);
  const after = paragraph.children.slice(1);
  const suffix = after.map((child) => toString(child)).join("");
  if (!suffix.startsWith(":")) return null;
  return label;
}

/** Validate Arcloom Requirement block ordering in an mdast tree. */
export function validateRequirementBlocks(tree, file) {
  let current = null;
  for (const node of tree.children ?? []) {
    if (node.type === "heading" && node.depth <= 2) {
      current = null;
      continue;
    }
    if (node.type === "heading" && node.depth === 3) {
      const title = toString(node).trim();
      current = title.startsWith("Requirement:")
        ? { seen: new Set(), lastIndex: -1, hasScenario: false }
        : null;
      continue;
    }
    if (!current) continue;
    if (
      node.type === "heading" &&
      node.depth === 4 &&
      toString(node).trim().startsWith("Scenario:")
    ) {
      current.hasScenario = true;
      continue;
    }
    if (node.type === "blockquote") continue;
    visit(node, "listItem", (item) => {
      const label = labelOf(item);
      if (!label || IGNORED_LABELS.has(label)) return;
      if (!BLOCK_INDEX.has(label)) {
        file.message(`unknown Requirement block "${label}".`, item);
        return;
      }
      if (current.hasScenario) {
        file.message(`Requirement block "${label}" must precede Scenarios.`, item);
        return;
      }
      if (current.seen.has(label)) {
        file.message(`Requirement repeats Requirement block "${label}".`, item);
        return;
      }
      const index = BLOCK_INDEX.get(label);
      if (index < current.lastIndex) {
        file.message(`Requirement block "${label}" is out of order.`, item);
        return;
      }
      current.seen.add(label);
      current.lastIndex = index;
    });
  }
}
