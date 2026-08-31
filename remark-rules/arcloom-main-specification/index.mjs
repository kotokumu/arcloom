import { lintRule } from "unified-lint-rule";

import { validateRequirementBlocks } from "./requirement-blocks.mjs";
import { validateRequirementHeadings } from "./requirement-heading.mjs";
import { validateScenarioHeadings } from "./scenario-heading.mjs";

const MAIN_SPECIFICATION_PATH =
  /(?:^|\/)openspec\/specs\/(?!_schema(?:\/|$)).+\/spec\.md$/;

/**
 * Check the Arcloom-specific structure of one main specification.
 *
 * @param {import("mdast").Root} tree
 * @param {import("vfile").VFile} file
 * @returns {void}
 */
function validateArcloomMainSpecification(tree, file) {
  const path = file.path?.replaceAll("\\", "/") ?? "";
  if (!MAIN_SPECIFICATION_PATH.test(path)) return;

  validateRequirementHeadings(tree, file);
  validateRequirementBlocks(tree, file);
  validateScenarioHeadings(tree, file);
}

const remarkLintArcloomMainSpecification = lintRule(
  "remark-lint:arcloom-main-specification",
  validateArcloomMainSpecification,
);

export default remarkLintArcloomMainSpecification;
