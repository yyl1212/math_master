-- +goose Up
-- 有限迁入与显式切换；安装不改变体验模式，旧迁移与数学/答案/成绩不改写。
ALTER TABLE goose_db_version ADD COLUMN IF NOT EXISTS topic_cutover_enabled boolean NOT NULL DEFAULT false;
UPDATE goose_db_version SET topic_cutover_enabled=true WHERE version_id=0;
ALTER TABLE topic_learning_state ADD COLUMN IF NOT EXISTS retirement_enabled boolean NOT NULL DEFAULT false;
UPDATE topic_learning_state SET retirement_enabled=true WHERE singleton;
CREATE TABLE study_migration_batches (
 id uuid PRIMARY KEY,sequence bigint GENERATED ALWAYS AS IDENTITY UNIQUE,started_at timestamptz NOT NULL,completed_at timestamptz NOT NULL,
 input_cursor jsonb,output_cursor jsonb,processed integer NOT NULL CHECK(processed BETWEEN 0 AND 50),
 created_events integer NOT NULL CHECK(created_events>=0),linked_events integer NOT NULL CHECK(linked_events>=0),conflicts integer NOT NULL CHECK(conflicts>=0),done boolean NOT NULL,
 CHECK(created_events+linked_events=processed),CHECK(completed_at>=started_at),
 CHECK(input_cursor IS NULL OR octet_length(input_cursor::text)<=512),CHECK(output_cursor IS NULL OR octet_length(output_cursor::text)<=512)
);
CREATE TABLE topic_cutovers (
 singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton),id uuid NOT NULL UNIQUE,
 expected_pair jsonb NOT NULL CHECK(octet_length(expected_pair::text)<=8192),code_sha text NOT NULL CHECK(code_sha ~ '^[a-f0-9]{40}$'),
 migration_batch_id uuid NOT NULL REFERENCES study_migration_batches(id),reason text NOT NULL CHECK(char_length(reason) BETWEEN 10 AND 1000),
 backup_record text NOT NULL CHECK(char_length(backup_record) BETWEEN 1 AND 256),recorded_at timestamptz NOT NULL
);
CREATE TABLE study_legacy_event_links (
 owner_user_id uuid NOT NULL REFERENCES auth_users(id),legacy_event_id uuid NOT NULL,study_event_id uuid NOT NULL UNIQUE REFERENCES study_events(id),knowledge_id text NOT NULL,
 source_sha text NOT NULL CHECK(source_sha ~ '^[a-f0-9]{64}$'),knowledge_publication_id text NOT NULL REFERENCES publication_snapshots(id),
 projected_record boolean NOT NULL,batch_id uuid NOT NULL REFERENCES study_migration_batches(id) DEFERRABLE INITIALLY DEFERRED,
 recorded_at timestamptz NOT NULL,linked_at timestamptz NOT NULL,
 PRIMARY KEY(owner_user_id,legacy_event_id),FOREIGN KEY(legacy_event_id,owner_user_id) REFERENCES learning_events(id,owner_user_id),
 FOREIGN KEY(owner_user_id,knowledge_id) REFERENCES study_records(owner_user_id,knowledge_id)
);
CREATE INDEX study_legacy_record_origin ON study_legacy_event_links(owner_user_id,knowledge_id) WHERE projected_record;
CREATE INDEX study_legacy_source_cursor ON learning_events(recorded_at,id) WHERE kind IN ('started','completed');
CREATE TRIGGER study_migration_batch_immutable BEFORE UPDATE OR DELETE ON study_migration_batches FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER topic_cutover_immutable BEFORE UPDATE OR DELETE ON topic_cutovers FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER study_legacy_link_immutable BEFORE UPDATE OR DELETE ON study_legacy_event_links FOR EACH ROW EXECUTE FUNCTION reject_content_update();
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION guard_topic_cutover_marker() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF OLD.version_id=0 AND OLD.topic_cutover_enabled AND (TG_OP='DELETE' OR NOT NEW.topic_cutover_enabled OR NEW.version_id<>0) THEN RAISE EXCEPTION 'permanent topic cutover capability'; END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;RETURN NEW;
END $$;
CREATE OR REPLACE TRIGGER topic_cutover_marker_guard BEFORE UPDATE OR DELETE ON goose_db_version FOR EACH ROW EXECUTE FUNCTION guard_topic_cutover_marker();
CREATE FUNCTION guard_study_legacy_link() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM learning_events e JOIN study_events s ON s.id=NEW.study_event_id WHERE e.id=NEW.legacy_event_id AND e.owner_user_id=NEW.owner_user_id AND e.knowledge_id=NEW.knowledge_id AND e.seal_sha256=NEW.source_sha AND e.knowledge_publication_id=NEW.knowledge_publication_id AND e.recorded_at=NEW.recorded_at AND e.kind IN ('started','completed') AND s.source_kind='legacy' AND s.origin_event_id=e.id AND s.owner_user_id=e.owner_user_id AND s.knowledge_id=e.knowledge_id AND s.knowledge_version=e.knowledge_version AND s.knowledge_sha256=e.knowledge_sha256 AND s.kind=e.kind AND s.recorded_at=e.recorded_at AND s.taxonomy_head IS NULL AND s.taxonomy_version_id IS NULL) THEN RAISE EXCEPTION 'legacy study source mismatch'; END IF;
 RETURN NEW;
END $$;
CREATE CONSTRAINT TRIGGER study_legacy_link_guard AFTER INSERT ON study_legacy_event_links DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION guard_study_legacy_link();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM topic_cutovers) OR EXISTS(SELECT 1 FROM study_migration_batches) OR EXISTS(SELECT 1 FROM study_legacy_event_links) OR EXISTS(SELECT 1 FROM topic_learning_state WHERE experience_mode='topics') THEN RAISE EXCEPTION 'nonempty topic cutover cannot be removed'; END IF;
END $$;
-- +goose StatementEnd
DROP TABLE topic_cutovers,study_legacy_event_links,study_migration_batches;
DROP INDEX study_legacy_source_cursor;
DROP FUNCTION guard_study_legacy_link();
UPDATE topic_learning_state SET retirement_enabled=false WHERE singleton;
-- 永久标记保留。真实环境只恢复兼容二进制，不通过Down清理历史。
