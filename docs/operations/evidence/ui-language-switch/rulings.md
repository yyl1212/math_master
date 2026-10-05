# 执行裁决记录

以下为 Native 执行期间的原始裁决记录，交付时按影响与代价说明。

- Ruling: Use a private byte-equivalent approved plan copy with bilingual Task headings for the native skill extractor — original Chinese headings are not recognised by task-brief — cost if wrong: brief regeneration only.

- Ruling: Docs-only PR35 CI does not block implementing its human-approved text after PR34 has merged; keep PR35 conditional merge and rebase implementation afterwards — no runtime prerequisite differs — cost if wrong: clean docs rebase.

- Task 2: Ruling: Testing Library role queries do not support Playwright exact:true — remove invalid option, preserve literal accessible name — typecheck is required before final task completion.

- Task 3: Ruling: Pure presentational components that need translated accessibility attributes are hydrated client boundaries; data fetch/auth pages remain server-only — preserves SSR and private configuration boundary — cost if wrong: client display bundle size. Explicit UI text only is source-migrated; raw data expressions are never message keys.

- Task 5: Ruling: File-processing errors now carry explicit internal codes while retaining their original Error.message — prevents translating arbitrary exception prose and preserves existing import/export callers — cost if wrong: a previously unclassified local error uses the safe generic message. API validators are unchanged.

- Task 7: Ruling: More than 1000 discriminated message members triggered TypeScript TS2590 — use an opaque compact UiMessage DTO constructed only by the key/parameter-typed uiMessage factory; serialized shape and required parameter checks remain unchanged — cost if wrong: code narrowing by individual message key now needs the typed factory instead of a giant union. Final compatibility review required.

- Task 8: Ruling: The Go correction createPlan action does not naturally require fresh password verification — inject one schema-valid REAUTHENTICATION_REQUIRED response in the isolated browser test to exercise the existing optional dialog, then perform real password verification and real same-key createPlan retry — keeps actual authorization unchanged — cost if wrong: a future backend reauthentication policy needs an additional real gate fixture.
