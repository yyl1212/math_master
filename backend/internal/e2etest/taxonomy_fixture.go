package e2etest

// Explicitly reset only the additional tables owned by this isolated harness.
// Production migration and publication commands never use this list.
var taxonomyResetTableNames = []string{
	"taxonomy_heads", "taxonomy_release_assignments", "taxonomy_releases",
	"taxonomy_review_bindings", "taxonomy_submission_assignments", "taxonomy_assignment_drafts",
	"taxonomy_idempotency", "taxonomy_nodes", "taxonomy_versions", "taxonomy_source_batches",
}
