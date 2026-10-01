-- +goose Up
-- P1 immutable bytes and version identities remain unchanged.
ALTER TABLE imported_packages ADD CONSTRAINT imported_packages_identity_sha UNIQUE(id,version,sha256);
CREATE TABLE content_workspaces (
 id uuid PRIMARY KEY CHECK(id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 owner_user_id uuid NOT NULL REFERENCES auth_users(id),
 catalogue_version integer NOT NULL REFERENCES catalogue_versions(version),
 package jsonb NOT NULL CHECK(jsonb_typeof(package)='object' AND octet_length(package::text)<=4194304),
 source_map jsonb NOT NULL DEFAULT '[]' CHECK(jsonb_typeof(source_map)='array' AND octet_length(source_map::text)<=524288),
 author_ids jsonb NOT NULL DEFAULT '[]' CHECK(jsonb_typeof(author_ids)='array'),
 legacy_unattributed boolean NOT NULL DEFAULT false,
 base_submission_id uuid,
 revision bigint NOT NULL CHECK(revision>=1),
 status text NOT NULL DEFAULT 'editing' CHECK(status IN ('editing','submitted')),
 gate jsonb NOT NULL DEFAULT '{}',
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(), updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX content_workspaces_owner ON content_workspaces(owner_user_id,created_at,id);
CREATE TABLE content_workspace_assets (
 workspace_id uuid NOT NULL REFERENCES content_workspaces(id), asset_id text NOT NULL,
 sha256 text NOT NULL CHECK(sha256 ~ '^[0-9a-f]{64}$'), bytes bytea NOT NULL CHECK(octet_length(bytes)<=1048576),
 PRIMARY KEY(workspace_id,asset_id), CHECK(encode(sha256(bytes),'hex')=sha256)
);
CREATE TABLE content_submissions (
 id uuid PRIMARY KEY CHECK(id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 workspace_id uuid NOT NULL REFERENCES content_workspaces(id), owner_user_id uuid NOT NULL REFERENCES auth_users(id), revision bigint NOT NULL CHECK(revision>=1),
 package_id text NOT NULL, package_version integer NOT NULL, package_sha256 text NOT NULL,
 catalogue_version integer NOT NULL, catalogue_sha256 text NOT NULL,
 frozen_body jsonb NOT NULL, frozen_bytes bytea NOT NULL CHECK(octet_length(frozen_bytes)<=4194304),
 frozen_digest text NOT NULL CHECK(frozen_digest ~ '^[0-9a-f]{64}$'), gate jsonb NOT NULL,
 status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','returned')),
 sealed boolean NOT NULL DEFAULT false, created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(workspace_id,revision),
 FOREIGN KEY(package_id,package_version,package_sha256) REFERENCES imported_packages(id,version,sha256),
 FOREIGN KEY(catalogue_version,catalogue_sha256) REFERENCES catalogue_versions(version,sha256),
 CHECK(encode(sha256(frozen_bytes),'hex')=frozen_digest),
 CHECK(convert_from(frozen_bytes,'UTF8')::jsonb=frozen_body),
 CHECK(jsonb_typeof(frozen_body->'authorIds')='array'),
 CHECK(frozen_body->>'catalogueSha256'=catalogue_sha256 AND (frozen_body->>'catalogueVersion')::integer=catalogue_version),
 CHECK(frozen_body->'package'->>'id'=package_id AND (frozen_body->'package'->>'version')::integer=package_version)
);
ALTER TABLE content_workspaces ADD FOREIGN KEY(base_submission_id) REFERENCES content_submissions(id);
CREATE INDEX content_submissions_status ON content_submissions(status,created_at,id);
CREATE TABLE content_submission_authors (
 submission_id uuid NOT NULL REFERENCES content_submissions(id),user_id uuid NOT NULL REFERENCES auth_users(id), PRIMARY KEY(submission_id,user_id)
);
CREATE TABLE content_submission_members (
 submission_id uuid NOT NULL REFERENCES content_submissions(id),package_id text NOT NULL,package_version integer NOT NULL,
 kind text NOT NULL,id text NOT NULL,version integer NOT NULL,sha256 text NOT NULL CHECK(sha256 ~ '^[0-9a-f]{64}$'),
 PRIMARY KEY(submission_id,kind,id),
 FOREIGN KEY(package_id,package_version,kind,id,version) REFERENCES package_members(package_id,package_version,kind,id,version)
);
CREATE TABLE content_review_decisions (
 id uuid PRIMARY KEY CHECK(id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 submission_id uuid NOT NULL UNIQUE REFERENCES content_submissions(id),reviewer_user_id uuid NOT NULL REFERENCES auth_users(id),
 frozen_digest text NOT NULL CHECK(frozen_digest ~ '^[0-9a-f]{64}$'),decision text NOT NULL CHECK(decision IN ('approve','return')),
 checks jsonb NOT NULL,independence_note text NOT NULL DEFAULT '',note text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK(char_length(note)>=10 AND note ~ '[^[:space:]]' AND char_length(note)<=1000 AND octet_length(note)<=3000),
 CHECK(decision='return' OR (char_length(independence_note)>=10 AND independence_note ~ '[^[:space:]]' AND char_length(independence_note)<=1000 AND octet_length(independence_note)<=3000 AND checks @> '{"mathematics":true,"explanations":true,"relationships":true,"sources":true,"illustrations":true}'))
);
CREATE TABLE content_publication_manifests (
 snapshot_id text PRIMARY KEY REFERENCES publication_snapshots(id),base_head text REFERENCES publication_snapshots(id),
 manifest jsonb NOT NULL,manifest_bytes bytea NOT NULL CHECK(octet_length(manifest_bytes)<=4194304),
 sha256 text NOT NULL CHECK(sha256 ~ '^[0-9a-f]{64}$'),diff jsonb NOT NULL,
 creator_user_id uuid NOT NULL REFERENCES auth_users(id),created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK(encode(sha256(manifest_bytes),'hex')=sha256),CHECK(convert_from(manifest_bytes,'UTF8')::jsonb=manifest)
);
CREATE TABLE content_withdrawals (
 id uuid PRIMARY KEY CHECK(id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 kind text NOT NULL CHECK(kind IN ('knowledge','unit','path','asset')),target_id text,target_version integer,sha256 text NOT NULL CHECK(sha256 ~ '^[0-9a-f]{64}$'),
 actor_user_id uuid NOT NULL REFERENCES auth_users(id),reason text NOT NULL,request_id text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK((kind='asset' AND target_id IS NULL AND target_version IS NULL) OR (kind<>'asset' AND target_id IS NOT NULL AND target_version>0)),
 CHECK(char_length(reason)>=10 AND reason ~ '[^[:space:]]' AND char_length(reason)<=1000 AND octet_length(reason)<=3000)
);
CREATE UNIQUE INDEX content_withdrawal_version ON content_withdrawals(kind,target_id,target_version) WHERE kind<>'asset';
CREATE UNIQUE INDEX content_withdrawal_asset ON content_withdrawals(sha256) WHERE kind='asset';
CREATE TABLE content_workflow_events (
 id uuid PRIMARY KEY CHECK(id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 actor_user_id uuid NOT NULL REFERENCES auth_users(id),action text NOT NULL,object_kind text NOT NULL,object_id text NOT NULL,
 before_digest text,after_digest text,before_state text,after_state text,reason text NOT NULL DEFAULT '',request_id text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),CHECK(octet_length(reason)<=3000)
);
CREATE TABLE content_idempotency (
 actor_user_id uuid NOT NULL REFERENCES auth_users(id),route text NOT NULL CHECK(route IN ('createDraft','saveDraft','adoptDraft','submitDraft','reviseSubmission','decideReview','prepareRelease','activateRelease','withdrawVersion')),
 key uuid NOT NULL CHECK(key::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),
 request_sha256 text NOT NULL CHECK(request_sha256 ~ '^[0-9a-f]{64}$'),result_bytes bytea NOT NULL CHECK(octet_length(result_bytes)<=4194304),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(actor_user_id,route,key),CHECK(jsonb_typeof(convert_from(result_bytes,'UTF8')::jsonb) IS NOT NULL)
);
-- +goose StatementBegin
CREATE FUNCTION content_submission_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'immutable submission'; END IF;
 IF (to_jsonb(NEW)-'status'-'sealed') IS DISTINCT FROM (to_jsonb(OLD)-'status'-'sealed') THEN RAISE EXCEPTION 'immutable frozen body'; END IF;
 IF OLD.sealed THEN
  IF NOT NEW.sealed OR OLD.status<>'pending' OR NEW.status NOT IN ('approved','returned') THEN RAISE EXCEPTION 'final submission'; END IF;
 ELSE
  IF NOT NEW.sealed OR NEW.status<>'pending' THEN RAISE EXCEPTION 'invalid seal'; END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION content_submission_child_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE parent_sealed boolean;
BEGIN
 IF TG_OP<>'INSERT' THEN RAISE EXCEPTION 'immutable submission relation'; END IF;
 SELECT sealed INTO parent_sealed FROM content_submissions WHERE id=NEW.submission_id FOR SHARE;
 IF parent_sealed THEN RAISE EXCEPTION 'sealed submission'; END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION content_submission_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE s content_submissions; actual_authors jsonb; expected_authors jsonb; member_count integer; expected_count integer;
BEGIN
 SELECT * INTO s FROM content_submissions WHERE id=NEW.id;
 IF NOT s.sealed THEN RAISE EXCEPTION 'unsealed submission'; END IF;
 SELECT coalesce(jsonb_agg(user_id::text ORDER BY user_id),'[]') INTO actual_authors FROM content_submission_authors WHERE submission_id=s.id;
 SELECT coalesce(jsonb_agg(value ORDER BY value),'[]') INTO expected_authors FROM jsonb_array_elements(s.frozen_body->'authorIds');
 IF actual_authors<>expected_authors OR NOT actual_authors @> jsonb_build_array(s.owner_user_id::text) THEN RAISE EXCEPTION 'frozen author mismatch'; END IF;
 SELECT count(*) INTO expected_count FROM package_members WHERE package_id=s.package_id AND package_version=s.package_version;
 SELECT count(*) INTO member_count FROM content_submission_members WHERE submission_id=s.id AND package_id=s.package_id AND package_version=s.package_version;
 IF member_count<>expected_count OR member_count<>(SELECT count(*) FROM content_submission_members WHERE submission_id=s.id) THEN RAISE EXCEPTION 'frozen member mismatch'; END IF;
 IF EXISTS(SELECT 1 FROM content_submission_members m WHERE m.submission_id=s.id AND m.sha256 IS DISTINCT FROM CASE m.kind
  WHEN 'knowledge' THEN (SELECT sha256 FROM knowledge_versions WHERE id=m.id AND version=m.version)
  WHEN 'unit' THEN (SELECT sha256 FROM unit_versions WHERE id=m.id AND version=m.version)
  WHEN 'path' THEN (SELECT sha256 FROM path_versions WHERE id=m.id AND version=m.version)
  WHEN 'asset' THEN (SELECT asset_sha256 FROM package_members WHERE package_id=m.package_id AND package_version=m.package_version AND kind=m.kind AND id=m.id) END) THEN RAISE EXCEPTION 'frozen member digest mismatch'; END IF;
 IF s.status='pending' AND EXISTS(SELECT 1 FROM content_review_decisions WHERE submission_id=s.id) OR
 s.status<>'pending' AND NOT EXISTS(SELECT 1 FROM content_review_decisions WHERE submission_id=s.id AND frozen_digest=s.frozen_digest AND decision=CASE s.status WHEN 'approved' THEN 'approve' ELSE 'return' END) THEN RAISE EXCEPTION 'review state mismatch'; END IF;
 RETURN NULL;
END $$;
CREATE FUNCTION content_review_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE s content_submissions;
BEGIN
 SELECT * INTO s FROM content_submissions WHERE id=NEW.submission_id FOR SHARE;
 IF NOT s.sealed OR s.status<>'pending' OR s.frozen_digest<>NEW.frozen_digest THEN RAISE EXCEPTION 'invalid review target'; END IF;
 IF NEW.decision='approve' AND EXISTS(SELECT 1 FROM content_submission_authors WHERE submission_id=s.id AND user_id=NEW.reviewer_user_id) THEN RAISE EXCEPTION 'author cannot approve'; END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION content_review_complete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM content_submissions WHERE id=NEW.submission_id AND sealed AND frozen_digest=NEW.frozen_digest AND status=CASE NEW.decision WHEN 'approve' THEN 'approved' ELSE 'returned' END) THEN RAISE EXCEPTION 'review state mismatch'; END IF;
 RETURN NULL;
END $$;
CREATE FUNCTION content_manifest_member_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT 1 FROM content_publication_manifests WHERE snapshot_id=CASE WHEN TG_OP='INSERT' THEN NEW.snapshot_id ELSE OLD.snapshot_id END) OR
 (TG_OP='UPDATE' AND EXISTS(SELECT 1 FROM content_publication_manifests WHERE snapshot_id=NEW.snapshot_id)) THEN RAISE EXCEPTION 'immutable manifest members'; END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;RETURN NEW;
END $$;
CREATE FUNCTION content_manifest_snapshot_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF EXISTS(SELECT 1 FROM content_publication_manifests WHERE snapshot_id=OLD.id) THEN
  IF TG_OP='DELETE' THEN RAISE EXCEPTION 'immutable manifest snapshot'; END IF;
  IF NEW.id<>OLD.id OR NEW.catalogue_version<>OLD.catalogue_version OR OLD.status<>'draft' OR NEW.status<>'published' THEN RAISE EXCEPTION 'immutable manifest snapshot'; END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER content_submission_immutable BEFORE UPDATE OR DELETE ON content_submissions FOR EACH ROW EXECUTE FUNCTION content_submission_guard();
CREATE TRIGGER content_authors_immutable BEFORE INSERT OR UPDATE OR DELETE ON content_submission_authors FOR EACH ROW EXECUTE FUNCTION content_submission_child_guard();
CREATE TRIGGER content_members_immutable BEFORE INSERT OR UPDATE OR DELETE ON content_submission_members FOR EACH ROW EXECUTE FUNCTION content_submission_child_guard();
CREATE CONSTRAINT TRIGGER content_submission_sealed AFTER INSERT OR UPDATE ON content_submissions DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION content_submission_complete();
CREATE CONSTRAINT TRIGGER content_review_final AFTER INSERT ON content_review_decisions DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION content_review_complete();
CREATE TRIGGER content_review_target BEFORE INSERT ON content_review_decisions FOR EACH ROW EXECUTE FUNCTION content_review_guard();
CREATE TRIGGER content_review_immutable BEFORE UPDATE OR DELETE ON content_review_decisions FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER content_manifest_immutable BEFORE UPDATE OR DELETE ON content_publication_manifests FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER content_withdrawal_immutable BEFORE UPDATE OR DELETE ON content_withdrawals FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER content_events_immutable BEFORE UPDATE OR DELETE ON content_workflow_events FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER content_idempotency_immutable BEFORE UPDATE OR DELETE ON content_idempotency FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER content_manifest_members_immutable BEFORE INSERT OR UPDATE OR DELETE ON publication_members FOR EACH ROW EXECUTE FUNCTION content_manifest_member_guard();
CREATE TRIGGER content_manifest_snapshot_immutable BEFORE UPDATE OR DELETE ON publication_snapshots FOR EACH ROW EXECUTE FUNCTION content_manifest_snapshot_guard();
-- +goose Down
DROP TRIGGER content_manifest_members_immutable ON publication_members;
DROP TRIGGER content_manifest_snapshot_immutable ON publication_snapshots;
ALTER TABLE content_workspaces DROP CONSTRAINT content_workspaces_base_submission_id_fkey;
DROP TABLE content_idempotency,content_workflow_events,content_withdrawals,content_publication_manifests,content_review_decisions,content_submission_members,content_submission_authors,content_submissions,content_workspace_assets,content_workspaces;
DROP FUNCTION content_submission_guard(),content_submission_child_guard(),content_submission_complete(),content_review_guard(),content_review_complete(),content_manifest_member_guard(),content_manifest_snapshot_guard();
ALTER TABLE imported_packages DROP CONSTRAINT imported_packages_identity_sha;
