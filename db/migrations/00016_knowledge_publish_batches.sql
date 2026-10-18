-- +goose Up
CREATE TABLE managed_knowledge_publish_jobs (
 job_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),owner_user_id uuid NOT NULL REFERENCES auth_users(id),
 mode text NOT NULL CHECK(mode IN ('selected','all')),key text NOT NULL CHECK(octet_length(key) BETWEEN 1 AND 128),
 input_sha256 text NOT NULL CHECK(input_sha256 ~ '^[0-9a-f]{64}$'),
 sequence bigint NOT NULL DEFAULT 1 CHECK(sequence BETWEEN 1 AND 9007199254740991),
 total bigint NOT NULL CHECK(total BETWEEN 0 AND 9007199254740991),pending bigint NOT NULL CHECK(pending>=0),published bigint NOT NULL DEFAULT 0 CHECK(published>=0),skipped bigint NOT NULL DEFAULT 0 CHECK(skipped>=0),failed bigint NOT NULL DEFAULT 0 CHECK(failed>=0),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(job_id,owner_user_id),UNIQUE(owner_user_id,key),CHECK(total=pending+published+skipped+failed)
);
CREATE INDEX managed_publish_owner ON managed_knowledge_publish_jobs(owner_user_id,created_at DESC,job_id);
CREATE TABLE managed_knowledge_publish_items (
 job_id uuid NOT NULL REFERENCES managed_knowledge_publish_jobs(job_id),knowledge_id text NOT NULL CHECK(knowledge_id ~ '^k-[0-9a-f]{56}$'),
 expected_token uuid NOT NULL,ordinal bigint NOT NULL CHECK(ordinal>=0),
 external_id text NOT NULL CHECK(octet_length(external_id)<=512),title text NOT NULL CHECK(char_length(title)<=128),title_zh text NOT NULL CHECK(char_length(title_zh)<=128),
 state text NOT NULL DEFAULT 'pending' CHECK(state IN ('pending','published','skipped','failed')),
 error_code text NOT NULL DEFAULT '' CHECK(octet_length(error_code)<=64),error_path text NOT NULL DEFAULT '' CHECK(octet_length(error_path)<=512),
 PRIMARY KEY(job_id,knowledge_id),UNIQUE(job_id,ordinal),CHECK(state='failed' OR (error_code='' AND error_path=''))
);
CREATE INDEX managed_publish_pending ON managed_knowledge_publish_items(job_id,ordinal) WHERE state='pending';
CREATE TABLE managed_knowledge_publish_receipts (
 owner_user_id uuid NOT NULL,action text NOT NULL CHECK(action IN ('create','step')),key text NOT NULL CHECK(octet_length(key) BETWEEN 1 AND 128),job_id uuid NOT NULL,
 input_sha256 text NOT NULL CHECK(input_sha256 ~ '^[0-9a-f]{64}$'),receipt jsonb NOT NULL CHECK(jsonb_typeof(receipt)='object' AND octet_length(receipt::text)<=1048576),
 PRIMARY KEY(owner_user_id,action,key),FOREIGN KEY(job_id,owner_user_id) REFERENCES managed_knowledge_publish_jobs(job_id,owner_user_id)
);
CREATE TRIGGER managed_publish_receipt_immutable BEFORE UPDATE OR DELETE ON managed_knowledge_publish_receipts FOR EACH ROW EXECUTE FUNCTION reject_content_update();
-- +goose StatementBegin
CREATE FUNCTION managed_publish_item_guard() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF TG_OP='DELETE' OR OLD.state<>'pending' OR NEW.state='pending' OR (to_jsonb(NEW)-ARRAY['state','error_code','error_path']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['state','error_code','error_path']) THEN RAISE EXCEPTION 'publish snapshot or terminal item is immutable';END IF;RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER managed_publish_item_guard BEFORE UPDATE OR DELETE ON managed_knowledge_publish_items FOR EACH ROW EXECUTE FUNCTION managed_publish_item_guard();
-- +goose StatementBegin
CREATE FUNCTION managed_publish_job_guard() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF TG_OP='DELETE' OR (to_jsonb(NEW)-ARRAY['sequence','pending','published','skipped','failed']) IS DISTINCT FROM (to_jsonb(OLD)-ARRAY['sequence','pending','published','skipped','failed']) OR NEW.sequence<>OLD.sequence+1 OR OLD.pending=0 OR NEW.pending>=OLD.pending OR NEW.published<OLD.published OR NEW.skipped<OLD.skipped OR NEW.failed<OLD.failed THEN RAISE EXCEPTION 'invalid publish progress';END IF;RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER managed_publish_job_guard BEFORE UPDATE OR DELETE ON managed_knowledge_publish_jobs FOR EACH ROW EXECUTE FUNCTION managed_publish_job_guard();
-- +goose StatementBegin
CREATE FUNCTION managed_publish_counts_guard() RETURNS trigger LANGUAGE plpgsql AS $$ DECLARE j managed_knowledge_publish_jobs; n bigint;p bigint;d bigint;s bigint;f bigint;BEGIN
 SELECT * INTO j FROM managed_knowledge_publish_jobs WHERE job_id=NEW.job_id;
 SELECT count(*),count(*) FILTER(WHERE state='pending'),count(*) FILTER(WHERE state='published'),count(*) FILTER(WHERE state='skipped'),count(*) FILTER(WHERE state='failed') INTO n,p,d,s,f FROM managed_knowledge_publish_items WHERE job_id=NEW.job_id;
 IF (j.total,j.pending,j.published,j.skipped,j.failed) IS DISTINCT FROM (n,p,d,s,f) THEN RAISE EXCEPTION 'publish counts mismatch';END IF;RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER managed_publish_job_counts AFTER INSERT OR UPDATE ON managed_knowledge_publish_jobs DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION managed_publish_counts_guard();
CREATE CONSTRAINT TRIGGER managed_publish_item_counts AFTER INSERT OR UPDATE ON managed_knowledge_publish_items DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION managed_publish_counts_guard();
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM managed_knowledge_publish_jobs) OR EXISTS(SELECT 1 FROM managed_knowledge_publish_items) OR EXISTS(SELECT 1 FROM managed_knowledge_publish_receipts) THEN RAISE EXCEPTION 'publish migration cannot be rolled back after use';END IF;
END $$;
-- +goose StatementEnd
DROP TABLE managed_knowledge_publish_receipts,managed_knowledge_publish_items,managed_knowledge_publish_jobs;
DROP FUNCTION managed_publish_counts_guard(),managed_publish_job_guard(),managed_publish_item_guard();
