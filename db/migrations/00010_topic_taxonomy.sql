-- +goose Up
CREATE TABLE taxonomy_source_batches (
 snapshot_id text PRIMARY KEY CHECK(snapshot_id ~ '^[a-f0-9]{64}$'),
 batch integer NOT NULL CHECK(batch>0), body jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE taxonomy_versions (
 id text PRIMARY KEY CHECK(id ~ '^[a-f0-9]{64}$'),
 snapshot_id text NOT NULL REFERENCES taxonomy_source_batches(snapshot_id),
 taxonomy_sha text NOT NULL CHECK(taxonomy_sha ~ '^[a-f0-9]{64}$'),
 raw_classification_sha text NOT NULL CHECK(raw_classification_sha ~ '^[a-f0-9]{64}$'),
 attribution text NOT NULL, license text NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE taxonomy_nodes (
 taxonomy_version_id text NOT NULL REFERENCES taxonomy_versions(id),
 id text NOT NULL, code text NOT NULL, parent_id text, kind text NOT NULL CHECK(kind IN ('primary','auxiliary','other')),
 level integer NOT NULL CHECK(level BETWEEN 1 AND 3), body jsonb NOT NULL,
 PRIMARY KEY(taxonomy_version_id,id), UNIQUE(taxonomy_version_id,code),
 FOREIGN KEY(taxonomy_version_id,parent_id) REFERENCES taxonomy_nodes(taxonomy_version_id,id) DEFERRABLE INITIALLY DEFERRED,
 CHECK((level=1 AND parent_id IS NULL AND kind='primary') OR (level>1 AND parent_id IS NOT NULL))
);
CREATE INDEX taxonomy_node_parent ON taxonomy_nodes(taxonomy_version_id,parent_id,code);
CREATE TABLE taxonomy_assignment_drafts (
 workspace_id uuid NOT NULL REFERENCES content_workspaces(id), knowledge_id text NOT NULL,
 draft_revision bigint NOT NULL CHECK(draft_revision>0), assignment_revision bigint NOT NULL CHECK(assignment_revision>0),
 taxonomy_version_id text NOT NULL REFERENCES taxonomy_versions(id),
 knowledge_sha text NOT NULL, body jsonb NOT NULL, digest text NOT NULL,
 updated_at timestamptz NOT NULL DEFAULT clock_timestamp(), PRIMARY KEY(workspace_id,knowledge_id)
);
CREATE TABLE taxonomy_submission_assignments (
 submission_id uuid PRIMARY KEY REFERENCES content_submissions(id),
 taxonomy_version_id text NOT NULL REFERENCES taxonomy_versions(id),
 body jsonb NOT NULL, digest text NOT NULL CHECK(digest ~ '^[a-f0-9]{64}$'), created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE taxonomy_review_bindings (
 decision_id uuid PRIMARY KEY REFERENCES content_review_decisions(id),
 submission_id uuid NOT NULL REFERENCES taxonomy_submission_assignments(submission_id), digest text NOT NULL
);
CREATE TABLE taxonomy_releases (
 id uuid PRIMARY KEY, taxonomy_version_id text NOT NULL REFERENCES taxonomy_versions(id),
 knowledge_publication_id text REFERENCES publication_snapshots(id),
 base_knowledge_head text, base_taxonomy_head uuid, status text NOT NULL CHECK(status IN ('draft','published')),
 manifest_sha text NOT NULL CHECK(manifest_sha ~ '^[a-f0-9]{64}$'), assignments_sha text NOT NULL CHECK(assignments_sha ~ '^[a-f0-9]{64}$'),
 body jsonb NOT NULL, creator_user_id uuid REFERENCES auth_users(id), created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE taxonomy_release_assignments (
 release_id uuid NOT NULL REFERENCES taxonomy_releases(id), taxonomy_version_id text NOT NULL,
 knowledge_id text NOT NULL, knowledge_version integer NOT NULL, knowledge_sha text NOT NULL,
 topic_id text NOT NULL, evidence_submission_id uuid REFERENCES taxonomy_submission_assignments(submission_id),
 PRIMARY KEY(release_id,knowledge_id,knowledge_version,topic_id),
 FOREIGN KEY(taxonomy_version_id,topic_id) REFERENCES taxonomy_nodes(taxonomy_version_id,id),
 FOREIGN KEY(knowledge_id,knowledge_version) REFERENCES knowledge_versions(id,version)
);
CREATE INDEX taxonomy_knowledge_topics ON taxonomy_release_assignments(release_id,topic_id,knowledge_id);
CREATE TABLE taxonomy_heads (
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton), release_id uuid NOT NULL REFERENCES taxonomy_releases(id)
);
CREATE TABLE taxonomy_idempotency (
 actor_user_id uuid NOT NULL REFERENCES auth_users(id), action text NOT NULL,target text NOT NULL,key uuid NOT NULL,
 input_sha text NOT NULL, receipt jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(actor_user_id,action,target,key)
);
CREATE TABLE topic_learning_state (
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),
 experience_mode text NOT NULL DEFAULT 'legacy' CHECK(experience_mode IN ('legacy','topics')),
 study_enabled boolean NOT NULL DEFAULT false, retired_at timestamptz
);
INSERT INTO topic_learning_state(singleton) VALUES(true);
CREATE TRIGGER taxonomy_batch_immutable BEFORE UPDATE OR DELETE ON taxonomy_source_batches FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER taxonomy_version_immutable BEFORE UPDATE OR DELETE ON taxonomy_versions FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER taxonomy_node_immutable BEFORE UPDATE OR DELETE ON taxonomy_nodes FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER taxonomy_submission_immutable BEFORE UPDATE OR DELETE ON taxonomy_submission_assignments FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER taxonomy_review_immutable BEFORE UPDATE OR DELETE ON taxonomy_review_bindings FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER taxonomy_receipt_immutable BEFORE UPDATE OR DELETE ON taxonomy_idempotency FOR EACH ROW EXECUTE FUNCTION reject_content_update();
-- +goose StatementBegin
CREATE FUNCTION guard_taxonomy_release() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'immutable taxonomy release'; END IF;
 IF OLD.status<>'draft' OR NEW.status<>'published' OR (to_jsonb(NEW)-'status')<>(to_jsonb(OLD)-'status') THEN RAISE EXCEPTION 'immutable taxonomy release'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER taxonomy_release_guard BEFORE UPDATE OR DELETE ON taxonomy_releases FOR EACH ROW EXECUTE FUNCTION guard_taxonomy_release();
CREATE FUNCTION guard_taxonomy_members() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP<>'INSERT' OR NOT EXISTS(SELECT 1 FROM taxonomy_releases WHERE id=NEW.release_id AND status='draft') THEN RAISE EXCEPTION 'immutable taxonomy membership'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER taxonomy_member_guard BEFORE INSERT OR UPDATE OR DELETE ON taxonomy_release_assignments FOR EACH ROW EXECUTE FUNCTION guard_taxonomy_members();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM taxonomy_source_batches) OR EXISTS(SELECT 1 FROM taxonomy_releases) OR EXISTS(SELECT 1 FROM taxonomy_assignment_drafts) OR EXISTS(SELECT 1 FROM taxonomy_submission_assignments) OR EXISTS(SELECT 1 FROM taxonomy_idempotency) OR EXISTS(SELECT 1 FROM topic_learning_state WHERE experience_mode<>'legacy' OR study_enabled) THEN RAISE EXCEPTION 'nonempty topic taxonomy cannot be removed'; END IF;
END $$;
-- +goose StatementEnd
DROP TABLE taxonomy_heads,taxonomy_release_assignments,taxonomy_review_bindings,taxonomy_submission_assignments,taxonomy_assignment_drafts,taxonomy_idempotency,taxonomy_releases,taxonomy_nodes,taxonomy_versions,taxonomy_source_batches,topic_learning_state;
DROP FUNCTION guard_taxonomy_members(),guard_taxonomy_release();
