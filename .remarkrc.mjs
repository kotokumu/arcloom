import remarkGfm from "remark-gfm";
import remarkLintNoUndefinedReferences from "remark-lint-no-undefined-references";
import remarkPresetLintRecommended from "remark-preset-lint-recommended";

export default {
  plugins: [
    remarkGfm,
    remarkPresetLintRecommended,
    [remarkLintNoUndefinedReferences, false],
  ],
};
