import remarkGfm from "remark-gfm";
import remarkLintNoUndefinedReferences from "remark-lint-no-undefined-references";
import remarkPresetLintRecommended from "remark-preset-lint-recommended";
import remarkLintArcloomMainSpecification from "./remark-rules/arcloom-main-specification/index.mjs";

export default {
  plugins: [
    remarkGfm,
    remarkPresetLintRecommended,
    [remarkLintNoUndefinedReferences, false],
    remarkLintArcloomMainSpecification,
  ],
};
