---
description: Review the changes and docs, update/expand/fix various markdown, docs, and embeded files.
agent: build

---

TASK: Make sure our docs are in a happy place.

STEPS:
1. Review changes and file contents to gain understanding for the user query, base your exploration effort on the user's query.
2. Update, expand, and/or fix the various file kinds based on the user query.
3. Provide a summary of your changes back to the user.

FILE KINDS:
- user facing: README.md docs/**.md
- agent facing: AGENTS.md **/AGENTS.md (excluding any embeds named the same)
- designs: **/.design/**.md
- embedded: **/embeds/**.*
- working: PLAN.md IDEAS.MD SCRATCH.md TASK.md (IMPORTANT: Do NOT read, evaluate, or edit these files)

USER QUERY:

$ARGUMENTS
