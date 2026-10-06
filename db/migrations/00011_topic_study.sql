-- +goose Up
-- 独立个人主题学习；旧数学、成绩、来源和00001—00009保持原字节。
ALTER TABLE goose_db_version ADD COLUMN IF NOT EXISTS topic_study_enabled boolean NOT NULL DEFAULT false;
UPDATE goose_db_version SET topic_study_enabled=true WHERE version_id=0;

CREATE TABLE study_records (
 owner_user_id uuid NOT NULL REFERENCES auth_users(id), knowledge_id text NOT NULL CHECK(knowledge_id ~ '^[a-z][a-z0-9-]{0,63}$'),
 state text NOT NULL CHECK(state IN ('unlearned','learning','completed','reviewing')),
 sequence bigint NOT NULL CHECK(sequence BETWEEN 0 AND 9007199254740991),
 body jsonb NOT NULL CHECK(octet_length(body::text)<=16384), last_known_ref jsonb NOT NULL, updated_at timestamptz NOT NULL,
 PRIMARY KEY(owner_user_id,knowledge_id),
 CHECK((body->>'knowledgeId'=knowledge_id AND body->>'state'=state AND (body->>'sequence')::bigint=sequence) IS TRUE),
 CHECK((last_known_ref->>'id'=knowledge_id AND (last_known_ref->>'version')::integer>0 AND last_known_ref->>'sha256' ~ '^[a-f0-9]{64}$') IS TRUE)
);
CREATE TABLE study_events (
 id uuid PRIMARY KEY, owner_user_id uuid NOT NULL REFERENCES auth_users(id),knowledge_id text NOT NULL,
 knowledge_version integer NOT NULL CHECK(knowledge_version>0),knowledge_sha256 text NOT NULL CHECK(knowledge_sha256 ~ '^[a-f0-9]{64}$'),
 taxonomy_version_id text CHECK(taxonomy_version_id ~ '^[a-f0-9]{64}$'),
 kind text NOT NULL CHECK(kind IN ('started','completed','review-started','review-finished','note-saved','note-deleted')),
 recorded_at timestamptz NOT NULL,note_revision bigint CHECK(note_revision BETWEEN 1 AND 9007199254740991),review_id uuid,
 source_kind text NOT NULL CHECK(source_kind IN ('native','legacy')),origin_event_id uuid,
 action text NOT NULL,idempotency_key uuid NOT NULL,
 CHECK((source_kind='native' AND origin_event_id IS NULL AND taxonomy_version_id IS NOT NULL) OR (source_kind='legacy' AND origin_event_id IS NOT NULL)),
 CHECK((kind IN ('note-saved','note-deleted'))=(note_revision IS NOT NULL)),
 CHECK((kind IN ('review-started','review-finished'))=(review_id IS NOT NULL)),
 UNIQUE(owner_user_id,action,knowledge_id,idempotency_key),
 FOREIGN KEY(owner_user_id,knowledge_id) REFERENCES study_records(owner_user_id,knowledge_id)
);
CREATE INDEX study_event_history ON study_events(owner_user_id,recorded_at DESC,id DESC);
CREATE INDEX study_event_knowledge_history ON study_events(owner_user_id,knowledge_id,recorded_at DESC,id DESC);
CREATE UNIQUE INDEX study_event_legacy_identity ON study_events(owner_user_id,origin_event_id) WHERE source_kind='legacy';
CREATE TABLE study_notes (
 owner_user_id uuid NOT NULL REFERENCES auth_users(id),knowledge_id text NOT NULL,revision bigint NOT NULL CHECK(revision BETWEEN 1 AND 9007199254740991),
 body text NOT NULL CHECK(octet_length(body)<=65536 AND char_length(body)<=16000),knowledge_ref jsonb NOT NULL,
 deleted boolean NOT NULL DEFAULT false,updated_at timestamptz NOT NULL,
 PRIMARY KEY(owner_user_id,knowledge_id),CHECK(NOT deleted OR body=''),
 CHECK((knowledge_ref->>'id'=knowledge_id AND (knowledge_ref->>'version')::integer>0 AND knowledge_ref->>'sha256' ~ '^[a-f0-9]{64}$') IS TRUE),
 FOREIGN KEY(owner_user_id,knowledge_id) REFERENCES study_records(owner_user_id,knowledge_id)
);
CREATE TABLE study_idempotency (
 owner_user_id uuid NOT NULL REFERENCES auth_users(id), action text NOT NULL,knowledge_id text NOT NULL,key uuid NOT NULL,
 input_sha text NOT NULL CHECK(input_sha ~ '^[a-f0-9]{64}$'),receipt jsonb NOT NULL CHECK(octet_length(receipt::text)<=2097152),
 PRIMARY KEY(owner_user_id,action,knowledge_id,key),
 CHECK(NOT receipt ? 'body'),CHECK(NOT receipt ? 'note'),
 CHECK((receipt->>'actorId'=owner_user_id::text) IS TRUE)
);
CREATE TABLE study_content_changes (
 id uuid PRIMARY KEY, publication_id text NOT NULL REFERENCES publication_snapshots(id),knowledge_id text NOT NULL,
 kind text NOT NULL CHECK(kind IN ('added','updated','withdrawn')),before_ref jsonb,after_ref jsonb,recorded_at timestamptz NOT NULL,
 UNIQUE(publication_id,knowledge_id,kind)
);
CREATE TRIGGER study_event_immutable BEFORE UPDATE OR DELETE ON study_events FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER study_receipt_immutable BEFORE UPDATE OR DELETE ON study_idempotency FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER study_change_immutable BEFORE UPDATE OR DELETE ON study_content_changes FOR EACH ROW EXECUTE FUNCTION reject_content_update();
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION guard_topic_study_marker() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.version_id=0 AND OLD.topic_study_enabled AND (TG_OP='DELETE' OR NOT NEW.topic_study_enabled OR NEW.version_id<>0) THEN RAISE EXCEPTION 'permanent topic study capability'; END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
CREATE OR REPLACE TRIGGER topic_study_marker_guard BEFORE UPDATE OR DELETE ON goose_db_version FOR EACH ROW EXECUTE FUNCTION guard_topic_study_marker();
CREATE FUNCTION guard_study_note() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' OR NEW.owner_user_id<>OLD.owner_user_id OR NEW.knowledge_id<>OLD.knowledge_id OR NEW.revision<>OLD.revision+1 THEN RAISE EXCEPTION 'study note revision conflict'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER study_note_guard BEFORE UPDATE OR DELETE ON study_notes FOR EACH ROW EXECUTE FUNCTION guard_study_note();
-- +goose StatementEnd
UPDATE topic_learning_state SET study_enabled=true WHERE singleton;

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM study_records) OR EXISTS(SELECT 1 FROM study_events) OR EXISTS(SELECT 1 FROM study_notes) OR EXISTS(SELECT 1 FROM study_idempotency) OR EXISTS(SELECT 1 FROM study_content_changes) THEN RAISE EXCEPTION 'nonempty topic study cannot be removed'; END IF;
END $$;
-- +goose StatementEnd
DROP TABLE study_events,study_notes,study_idempotency,study_content_changes,study_records;
DROP FUNCTION guard_study_note();
UPDATE topic_learning_state SET study_enabled=false WHERE singleton;
-- 保留goose version 0的永久标记及保护触发器，不允许降级清除。
