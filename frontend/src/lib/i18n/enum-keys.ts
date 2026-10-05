import type {StaticMessageKey} from "./types";
export const enumKeys={
  "content.check": {
    "mathematics": "content.check.mathematics",
    "explanations": "content.check.explanations",
    "relationships": "content.check.relationships",
    "sources": "content.check.sources",
    "illustrations": "content.check.illustrations"
  },
  "content.field": {
    "title": "content.field.title",
    "titleZh": "content.field.titleZh",
    "statement": "content.field.statement",
    "scope": "content.field.scope",
    "system": "content.field.system",
    "proof": "content.field.proof",
    "author": "content.field.author",
    "url": "content.field.url",
    "accessedAt": "content.field.accessedAt",
    "license": "content.field.license",
    "attribution": "content.field.attribution",
    "batchSha256": "content.field.batchSha256",
    "relativePath": "content.field.relativePath",
    "sha256": "content.field.sha256",
    "legacyId": "content.field.legacyId",
    "note": "content.field.note",
    "id": "content.field.id",
    "path": "content.field.path",
    "kind": "content.field.kind"
  },
  "content.enum": {
    "concept": "content.enum.concept",
    "definition": "content.enum.definition",
    "axiom": "content.enum.axiom",
    "theorem": "content.enum.theorem",
    "corollary": "content.enum.corollary",
    "method": "content.enum.method",
    "mathematical-thinking": "content.enum.mathematical-thinking",
    "prerequisite": "content.enum.prerequisite",
    "derivation": "content.enum.derivation",
    "related": "content.enum.related",
    "original": "content.enum.original",
    "external": "content.enum.external",
    "editing": "content.enum.editing",
    "submitted": "content.enum.submitted",
    "pending": "content.enum.pending",
    "approved": "content.enum.approved",
    "returned": "content.enum.returned",
    "draft": "content.enum.draft",
    "active": "content.enum.active",
    "withdrawn": "content.enum.withdrawn",
    "approve": "content.enum.approve",
    "return": "content.enum.return",
    "knowledge": "content.enum.knowledge",
    "unit": "content.enum.unit",
    "path": "content.enum.path",
    "asset": "content.enum.asset"
  },
  "question.check": {
    "mathematics": "question.check.mathematics",
    "explanations": "question.check.explanations",
    "coverage": "question.check.coverage",
    "sources": "question.check.sources",
    "illustrations": "question.check.illustrations",
    "generation": "question.check.generation",
    "objectives": "question.check.objectives"
  },
  "question.enum": {
    "numeric": "question.enum.numeric",
    "single_choice": "question.enum.single_choice",
    "rational": "question.enum.rational",
    "percentage": "question.enum.percentage",
    "rational_arithmetic": "question.enum.rational_arithmetic",
    "rational_comparison": "question.enum.rational_comparison",
    "missing_operand": "question.enum.missing_operand",
    "add": "question.enum.add",
    "subtract": "question.enum.subtract",
    "multiply": "question.enum.multiply",
    "divide": "question.enum.divide",
    "compare": "question.enum.compare",
    "left": "question.enum.left",
    "right": "question.enum.right",
    "nonzero_divisor": "question.enum.nonzero_divisor",
    "nonnegative_result": "question.enum.nonnegative_result",
    "distinct_operands": "question.enum.distinct_operands",
    "negate": "question.enum.negate",
    "plus_one": "question.enum.plus_one",
    "minus_one": "question.enum.minus_one",
    "reciprocal": "question.enum.reciprocal",
    "core": "question.enum.core",
    "supplementary": "question.enum.supplementary",
    "template": "question.enum.template",
    "instance": "question.enum.instance",
    "blueprint": "question.enum.blueprint"
  },
  "learning.state": {
    "unlearned": "learning.state.unlearned",
    "learning": "learning.state.learning",
    "learned": "learning.state.learned",
    "needs-review": "learning.state.needs-review",
    "mastered": "learning.state.mastered"
  },
  "learning.format": {
    "INVALID_SYNTAX": "learning.format.INVALID_SYNTAX",
    "INPUT_TOO_LONG": "learning.format.INPUT_TOO_LONG",
    "ZERO_DENOMINATOR": "learning.format.ZERO_DENOMINATOR",
    "PERCENT_REQUIRED": "learning.format.PERCENT_REQUIRED",
    "RESULT_TOO_LARGE": "learning.format.RESULT_TOO_LARGE",
    "INVALID_MODE": "learning.format.INVALID_MODE"
  },
  "feedback.status": {
    "new": "feedback.status.new",
    "processing": "feedback.status.processing",
    "waiting_details": "feedback.status.waiting_details",
    "resolved": "feedback.status.resolved",
    "closed": "feedback.status.closed"
  },
  "correction.status": {
    "corrected_passed": "correction.status.corrected_passed",
    "corrected_failed": "correction.status.corrected_failed",
    "retake_required": "correction.status.retake_required",
    "review_material": "correction.status.review_material",
    "checked_unaffected": "correction.status.checked_unaffected",
    "awaiting_review": "correction.status.awaiting_review"
  },
  "notification.type": {
    "checking": "notification.type.checking",
    "corrected": "notification.type.corrected",
    "retake": "notification.type.retake",
    "review_material": "notification.type.review_material",
    "path_unavailable": "notification.type.path_unavailable"
  },
  "feedback.category": {
    "math_error": "feedback.category.math_error",
    "explanation": "feedback.category.explanation",
    "illustration": "feedback.category.illustration",
    "reference": "feedback.category.reference",
    "typo": "feedback.category.typo",
    "other": "feedback.category.other",
    "site": "feedback.category.site",
    "answer_error": "feedback.category.answer_error",
    "grading_error": "feedback.category.grading_error",
    "technical_issue": "feedback.category.technical_issue",
    "accessibility": "feedback.category.accessibility",
    "unclear_explanation": "feedback.category.unclear_explanation",
    "suggestion": "feedback.category.suggestion"
  },
  "feedback.basis": {
    "duplicate": "feedback.basis.duplicate",
    "not_reproducible": "feedback.basis.not_reproducible",
    "out_of_scope": "feedback.basis.out_of_scope",
    "suggestion_recorded": "feedback.basis.suggestion_recorded",
    "clarified": "feedback.basis.clarified",
    "withdrawn": "feedback.basis.withdrawn",
    "revision_published": "feedback.basis.revision_published",
    "service_fixed": "feedback.basis.service_fixed"
  },
  "correction.side": {
    "original": "correction.side.original",
    "replacement": "correction.side.replacement"
  },
  "correction.field": {
    "id": "correction.field.id",
    "version": "correction.field.version",
    "sha256": "correction.field.sha256"
  },
  "question.state": {
    "editing": "question.state.editing",
    "submitted": "question.state.submitted",
    "pending": "question.state.pending",
    "approved": "question.state.approved",
    "returned": "question.state.returned",
    "prepared": "question.state.prepared",
    "published": "question.state.published",
    "active": "question.state.active"
  },
  "correction.plan": {
    "draft": "correction.plan.draft",
    "pending": "correction.plan.pending",
    "approved": "correction.plan.approved",
    "rejected": "correction.plan.rejected"
  },
  "correction.jobType": {
    "withdrawal_impact": "correction.jobType.withdrawal_impact",
    "rule_impact": "correction.jobType.rule_impact",
    "approved_plan": "correction.jobType.approved_plan",
    "attempt_terminal": "correction.jobType.attempt_terminal"
  },
  "correction.jobState": {
    "queued": "correction.jobState.queued",
    "running": "correction.jobState.running",
    "succeeded": "correction.jobState.succeeded",
    "retry_wait": "correction.jobState.retry_wait",
    "failed": "correction.jobState.failed"
  },
  "learning.attempt": {
    "active": "learning.attempt.active",
    "submitted": "learning.attempt.submitted",
    "answered": "learning.attempt.answered",
    "revealed": "learning.attempt.revealed",
    "abandoned": "learning.attempt.abandoned",
    "expired": "learning.attempt.expired",
    "started": "learning.attempt.started",
    "completed": "learning.attempt.completed",
    "enrolled": "learning.attempt.enrolled"
  }
} satisfies Record<string,Record<string,StaticMessageKey>>;
