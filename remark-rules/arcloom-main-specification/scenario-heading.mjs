import { toString } from "mdast-util-to-string";

/** Tags used to classify a main-specification Scenario heading. */
const ALLOWED_SCENARIO_TAGS = new Set([
  "happy",
  "error",
  "boundary",
  "permission",
  "concurrency",
  "idempotency",
  "compatibility",
]);

/**
 * Validate Scenario headings in a parsed main specification.
 *
 * The validator deliberately receives mdast and a VFile.  Consequently,
 * fenced code and comments are already represented as non-heading nodes and
 * cannot be mistaken for specification headings.
 *
 * @param {{children?: Array<{type?: string, depth?: number}>}} tree
 * @param {{message: (reason: string, node: object) => unknown}} file
 */
export function validateScenarioHeadings(tree, file) {
  let inRequirement = false;

  for (const node of tree.children ?? []) {
    if (node.type !== "heading") continue;

    const heading = toString(node);

    if (node.depth === 2) {
      inRequirement = false;
      continue;
    }

    if (node.depth === 3) {
      inRequirement = heading.startsWith("Requirement:");
      continue;
    }

    if (node.depth !== 4 || !inRequirement || !heading.startsWith("Scenario:")) {
      continue;
    }

    const terminalTag = heading.match(/\[([^\]]*)\]\s*$/)?.[1];

    if (terminalTag === undefined || terminalTag === "") {
      file.message("Scenario heading must end with one allowed tag.", node);
      continue;
    }

    if (!ALLOWED_SCENARIO_TAGS.has(terminalTag)) {
      file.message(`Unknown Scenario tag: ${terminalTag}`, node);
    }
  }
}
