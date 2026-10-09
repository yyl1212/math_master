-- +goose Up
ALTER TABLE goose_db_version ADD COLUMN managed_knowledge_enabled boolean NOT NULL DEFAULT false;
CREATE TABLE knowledge_admin_state (
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),
 content_mode text NOT NULL DEFAULT 'legacy' CHECK(content_mode IN ('legacy','managed')),
 enabled_once boolean NOT NULL DEFAULT false,
 activated_at timestamptz,
 CHECK((content_mode='managed')=enabled_once),CHECK(enabled_once=(activated_at IS NOT NULL))
);
INSERT INTO knowledge_admin_state(singleton) VALUES(true);
CREATE TABLE managed_knowledge (
 internal_id text PRIMARY KEY CHECK(internal_id ~ '^k-[0-9a-f]{56}$'),
 external_id text NOT NULL UNIQUE CHECK(octet_length(external_id) BETWEEN 1 AND 512),
 point jsonb NOT NULL CHECK(jsonb_typeof(point)='object' AND point->>'id'=external_id),
 public_sources jsonb NOT NULL CHECK(jsonb_typeof(public_sources)='array'),
 content_sha256 text NOT NULL CHECK(content_sha256 ~ '^[0-9a-f]{64}$'),
 published boolean NOT NULL DEFAULT false,deleted_at timestamptz,
 edit_token uuid NOT NULL DEFAULT gen_random_uuid(),
 created_by uuid NOT NULL REFERENCES auth_users(id),updated_by uuid NOT NULL REFERENCES auth_users(id),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(internal_id,external_id),CHECK(deleted_at IS NULL OR NOT published)
);
CREATE TABLE managed_knowledge_topics (
 internal_id text NOT NULL,external_id text NOT NULL,
 topic_key text NOT NULL CHECK(topic_key ~ '^[0-9]{2}[A-Z][0-9]{2}$' OR topic_key='project:other'),
 active boolean NOT NULL DEFAULT true,manual_override boolean NOT NULL DEFAULT false,
 updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(external_id,topic_key),FOREIGN KEY(internal_id,external_id) REFERENCES managed_knowledge(internal_id,external_id)
);
CREATE INDEX managed_topic_active ON managed_knowledge_topics(topic_key,internal_id) WHERE active;
CREATE INDEX managed_public_list ON managed_knowledge(updated_at DESC,internal_id) WHERE published AND deleted_at IS NULL;
CREATE TABLE managed_knowledge_sources (
 internal_id text NOT NULL REFERENCES managed_knowledge(internal_id),source_id text NOT NULL,
 source_core_sha text NOT NULL CHECK(source_core_sha ~ '^[0-9a-f]{64}$'),
 source_file_sha text NOT NULL CHECK(source_file_sha ~ '^[0-9a-f]{64}$'),
 source_profile jsonb NOT NULL,source_point jsonb NOT NULL,
 original_bytes_verified boolean NOT NULL DEFAULT false,
 received_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(internal_id,source_id,source_core_sha)
);
CREATE TABLE managed_knowledge_imports (
 import_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),owner_user_id uuid NOT NULL REFERENCES auth_users(id),
 action text NOT NULL CHECK(action IN ('preview','apply','create','update','publish','unpublish','delete','restore')),
 key text NOT NULL CHECK(octet_length(key) BETWEEN 1 AND 128),resource text NOT NULL,
 input_sha256 text NOT NULL CHECK(input_sha256 ~ '^[0-9a-f]{64}$'),
 source_body jsonb,preview jsonb,preview_token uuid,
 state text NOT NULL CHECK(state IN ('preview','applied')),receipt jsonb,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(owner_user_id,action,key),CHECK((state='applied')=(receipt IS NOT NULL))
);
CREATE TABLE managed_knowledge_events (
 event_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 actor_user_id uuid NOT NULL REFERENCES auth_users(id),knowledge_id text REFERENCES managed_knowledge(internal_id),
 action text NOT NULL CHECK(action IN ('import','link','create','update','publish','unpublish','delete','restore','activate','clean-old')),
 old_sha256 text CHECK(old_sha256 ~ '^[0-9a-f]{64}$'),new_sha256 text CHECK(new_sha256 ~ '^[0-9a-f]{64}$'),
 old_topics jsonb NOT NULL DEFAULT '[]',new_topics jsonb NOT NULL DEFAULT '[]',
 evidence jsonb NOT NULL DEFAULT '{}',recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX managed_events_knowledge ON managed_knowledge_events(knowledge_id,recorded_at DESC,event_id);
CREATE TRIGGER managed_knowledge_event_immutable BEFORE UPDATE OR DELETE ON managed_knowledge_events FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TABLE managed_study_records (
 owner_user_id uuid NOT NULL REFERENCES auth_users(id),knowledge_id text NOT NULL REFERENCES managed_knowledge(internal_id),
 state text NOT NULL CHECK(state IN ('unlearned','learning','completed','reviewing')),sequence bigint NOT NULL DEFAULT 0 CHECK(sequence BETWEEN 0 AND 9007199254740991),
 first_started_at timestamptz,first_completed_at timestamptz,last_completed_at timestamptz,last_read_at timestamptz,last_reviewed_at timestamptz,
 completed_ref jsonb,last_review_ref jsonb,last_review_id uuid,active_review_id uuid,
 PRIMARY KEY(owner_user_id,knowledge_id)
);
CREATE TABLE managed_study_notes (
 owner_user_id uuid NOT NULL,knowledge_id text NOT NULL,
 body text NOT NULL CHECK(octet_length(body)<=65536 AND char_length(body)<=16000),
 revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 9007199254740991),knowledge_ref jsonb NOT NULL,
 deleted boolean NOT NULL DEFAULT false,updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(owner_user_id,knowledge_id),FOREIGN KEY(owner_user_id,knowledge_id) REFERENCES managed_study_records(owner_user_id,knowledge_id),
 CHECK(NOT deleted OR body='')
);
CREATE TABLE managed_study_events (
 event_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),owner_user_id uuid NOT NULL,knowledge_id text NOT NULL,
 knowledge_ref jsonb NOT NULL,topic_keys jsonb NOT NULL CHECK(jsonb_typeof(topic_keys)='array'),
 kind text NOT NULL CHECK(kind IN ('started','completed','review-started','review-finished','note-saved','note-deleted')),
 note_revision bigint,review_id uuid,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(owner_user_id,knowledge_id) REFERENCES managed_study_records(owner_user_id,knowledge_id),
 CHECK((kind IN ('note-saved','note-deleted'))=(note_revision IS NOT NULL)),
 CHECK((kind IN ('review-started','review-finished'))=(review_id IS NOT NULL))
);
CREATE INDEX managed_study_history ON managed_study_events(owner_user_id,recorded_at DESC,event_id DESC);
CREATE TRIGGER managed_study_event_immutable BEFORE UPDATE OR DELETE ON managed_study_events FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TABLE managed_study_idempotency (
 owner_user_id uuid NOT NULL REFERENCES auth_users(id),action text NOT NULL,knowledge_id text NOT NULL REFERENCES managed_knowledge(internal_id),
 key text NOT NULL CHECK(octet_length(key) BETWEEN 1 AND 128),input_sha256 text NOT NULL CHECK(input_sha256 ~ '^[0-9a-f]{64}$'),
 receipt jsonb NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(owner_user_id,action,knowledge_id,key)
);
CREATE TRIGGER managed_study_receipt_immutable BEFORE UPDATE OR DELETE ON managed_study_idempotency FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TABLE managed_study_content_changes (
 event_id uuid PRIMARY KEY REFERENCES managed_knowledge_events(event_id),knowledge_id text NOT NULL REFERENCES managed_knowledge(internal_id),
 kind text NOT NULL CHECK(kind IN ('update','unpublish','delete','restore','publish','link')),
 current_ref jsonb,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TRIGGER managed_study_change_immutable BEFORE UPDATE OR DELETE ON managed_study_content_changes FOR EACH ROW EXECUTE FUNCTION reject_content_update();
-- +goose StatementBegin
CREATE FUNCTION managed_ref_valid(r jsonb,k text) RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
 SELECT coalesce(feedback_shape(r,ARRAY['id','contentSha256','sourceKind']) AND jsonb_typeof(r->'id')='string' AND r->>'id'=k AND r->>'sourceKind'='managed' AND jsonb_typeof(r->'contentSha256')='string' AND r->>'contentSha256' ~ '^[0-9a-f]{64}$',false)
$$;
ALTER TABLE managed_study_records ADD CHECK(completed_ref IS NULL OR managed_ref_valid(completed_ref,knowledge_id));
ALTER TABLE managed_study_records ADD CHECK(last_review_ref IS NULL OR managed_ref_valid(last_review_ref,knowledge_id));
ALTER TABLE managed_study_notes ADD CHECK(managed_ref_valid(knowledge_ref,knowledge_id));
ALTER TABLE managed_study_events ADD CHECK(managed_ref_valid(knowledge_ref,knowledge_id));
CREATE FUNCTION guard_managed_knowledge_state() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' OR OLD.enabled_once AND (NOT NEW.enabled_once OR NEW.content_mode<>'managed') THEN RAISE EXCEPTION 'permanent managed knowledge mode'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER managed_knowledge_state_guard BEFORE UPDATE OR DELETE ON knowledge_admin_state FOR EACH ROW EXECUTE FUNCTION guard_managed_knowledge_state();
CREATE FUNCTION guard_managed_knowledge_marker() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.version_id=0 AND OLD.managed_knowledge_enabled AND (TG_OP='DELETE' OR NOT NEW.managed_knowledge_enabled OR NEW.version_id<>0) THEN RAISE EXCEPTION 'permanent managed knowledge capability'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER managed_knowledge_marker_guard BEFORE UPDATE OR DELETE ON goose_db_version FOR EACH ROW EXECUTE FUNCTION guard_managed_knowledge_marker();
CREATE FUNCTION guard_managed_study_note() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'managed note must be tombstoned'; END IF;
 IF TG_OP='UPDATE' AND ((NEW.owner_user_id,NEW.knowledge_id) IS DISTINCT FROM (OLD.owner_user_id,OLD.knowledge_id) OR NEW.revision<>OLD.revision+1 OR NEW.updated_at<OLD.updated_at) THEN RAISE EXCEPTION 'invalid managed note update'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER managed_study_note_guard BEFORE UPDATE OR DELETE ON managed_study_notes FOR EACH ROW EXECUTE FUNCTION guard_managed_study_note();
ALTER FUNCTION feedback_target_shape(jsonb,jsonb) RENAME TO feedback_target_shape_legacy;
CREATE FUNCTION feedback_target_shape(t jsonb,s jsonb) RETURNS boolean LANGUAGE plpgsql IMMUTABLE AS $$
BEGIN
 IF t->>'kind'<>'managed-knowledge' THEN RETURN feedback_target_shape_legacy(t,s); END IF;
 RETURN (feedback_shape(t,ARRAY['kind','identity','area','part','managedRef']) AND t->'identity'='null' AND t->'area'='null' AND t->'part'='null'
 AND managed_ref_valid(t->'managedRef',t#>>'{managedRef,id}') AND t#>>'{managedRef,id}' ~ '^k-[0-9a-f]{56}$'
 AND feedback_shape(s,ARRAY['kind','publicationId','attemptId','position']) AND s->>'kind'='managed' AND s->'publicationId'='null' AND s->'attemptId'='null' AND s->'position'='null') IS TRUE;
END $$;
ALTER FUNCTION feedback_target_proof(jsonb,jsonb,uuid,boolean) RENAME TO feedback_target_proof_legacy;
CREATE FUNCTION feedback_target_proof(t jsonb,s jsonb,owner uuid,creating boolean) RETURNS boolean LANGUAGE plpgsql STABLE AS $$
BEGIN
 IF t->>'kind'<>'managed-knowledge' THEN RETURN feedback_target_proof_legacy(t,s,owner,creating); END IF;
 IF NOT feedback_target_shape(t,s) OR NOT EXISTS(SELECT 1 FROM auth_users WHERE id=owner) THEN RETURN false; END IF;
 IF creating THEN
 RETURN EXISTS(SELECT 1 FROM managed_knowledge k,knowledge_admin_state st WHERE st.singleton AND st.content_mode='managed' AND k.internal_id=t#>>'{managedRef,id}' AND k.published AND k.deleted_at IS NULL AND k.content_sha256=t#>>'{managedRef,contentSha256}');
 END IF;
 RETURN EXISTS(SELECT 1 FROM managed_knowledge WHERE internal_id=t#>>'{managedRef,id}');
END $$;
DO $$ DECLARE c record; BEGIN
 FOR c IN SELECT conname FROM pg_constraint WHERE conrelid='feedback_tickets'::regclass AND contype='c' AND pg_get_constraintdef(oid) LIKE '%feedback_target_shape_legacy(%' LOOP
 EXECUTE format('ALTER TABLE feedback_tickets DROP CONSTRAINT %I',c.conname);
 END LOOP;
END $$;
ALTER TABLE feedback_tickets ADD CONSTRAINT feedback_managed_target_shape CHECK(feedback_target_shape(target,source));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM knowledge_admin_state WHERE enabled_once) OR EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=0 AND managed_knowledge_enabled) OR EXISTS(SELECT 1 FROM managed_knowledge) THEN RAISE EXCEPTION 'managed knowledge migration cannot be rolled back after use'; END IF;
END $$;
ALTER TABLE feedback_tickets DROP CONSTRAINT feedback_managed_target_shape;
DROP FUNCTION feedback_target_proof(jsonb,jsonb,uuid,boolean);
ALTER FUNCTION feedback_target_proof_legacy(jsonb,jsonb,uuid,boolean) RENAME TO feedback_target_proof;
DROP FUNCTION feedback_target_shape(jsonb,jsonb);
ALTER FUNCTION feedback_target_shape_legacy(jsonb,jsonb) RENAME TO feedback_target_shape;
ALTER TABLE feedback_tickets ADD CONSTRAINT feedback_original_target_shape CHECK(feedback_target_shape(target,source));
DROP TABLE managed_study_content_changes,managed_study_idempotency,managed_study_events,managed_study_notes,managed_study_records;
DROP TABLE managed_knowledge_events,managed_knowledge_imports,managed_knowledge_sources,managed_knowledge_topics,managed_knowledge;
DROP TABLE knowledge_admin_state;
DROP TRIGGER managed_knowledge_marker_guard ON goose_db_version;
DROP FUNCTION guard_managed_knowledge_marker(),guard_managed_knowledge_state(),guard_managed_study_note(),managed_ref_valid(jsonb,text);
ALTER TABLE goose_db_version DROP COLUMN managed_knowledge_enabled;
-- +goose StatementEnd
