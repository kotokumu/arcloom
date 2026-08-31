import { toString } from "mdast-util-to-string";

/**
 * Validate the slug on level-three Requirement headings in an mdast tree.
 *
 * @param {import("mdast").Root} tree Parsed Markdown tree.
 * @param {import("vfile").VFile} file Source file receiving diagnostics.
 * @returns {void}
 */
export function validateRequirementHeadings(tree, file) {
  for (const node of tree.children ?? []) {
    if (node.type === "heading" && node.depth === 3) {
      const heading = toString(node).trim();
      if (!heading.startsWith("Requirement:")) continue;

      const slug = heading.slice("Requirement:".length).trim();
      if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(slug)) {
        file.message(
          `Requirement heading must use a non-empty lowercase kebab-case slug; received "${slug}".`,
          node,
        );
      }
    }
  }
}
