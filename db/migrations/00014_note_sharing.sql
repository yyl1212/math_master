-- +goose Up
CREATE TABLE note_share_submissions (
 submission_id uuid PRIMARY KEY,
 owner_user_id uuid NOT NULL,
 knowledge_id text NOT NULL,
 note_revision bigint NOT NULL CHECK(note_revision BETWEEN 1 AND 9007199254740991),
 note_knowledge_ref jsonb NOT NULL CHECK(managed_ref_valid(note_knowledge_ref,knowledge_id)),
 title text NOT NULL CHECK(char_length(title) BETWEEN 1 AND 120),
 body text NOT NULL CHECK(char_length(body) BETWEEN 1 AND 16000 AND octet_length(body)<=65536),
 body_sha256 text NOT NULL CHECK(body_sha256 ~ '^[0-9a-f]{64}$'),
 knowledge_title text NOT NULL,knowledge_title_zh text NOT NULL,
 topic_keys jsonb NOT NULL CHECK(jsonb_typeof(topic_keys)='array'),
 submitted_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(submission_id,owner_user_id,knowledge_id),
 FOREIGN KEY(owner_user_id,knowledge_id) REFERENCES managed_study_notes(owner_user_id,knowledge_id)
);
CREATE TRIGGER note_share_submission_immutable BEFORE UPDATE OR DELETE ON note_share_submissions FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE INDEX note_share_submission_owner ON note_share_submissions(owner_user_id,knowledge_id,submitted_at DESC,submission_id DESC);
CREATE INDEX note_share_submission_topics ON note_share_submissions USING gin(topic_keys);
CREATE TABLE note_share_heads (
 owner_user_id uuid NOT NULL,knowledge_id text NOT NULL,
 status text NOT NULL DEFAULT 'unshared' CONSTRAINT note_share_head_status CHECK(status IN ('unshared','pending','shared','returned','withdrawn','hidden')),
 sequence bigint NOT NULL DEFAULT 0 CHECK(sequence BETWEEN 0 AND 9007199254740991),
 current_submission_id uuid,approved_at timestamptz,
 PRIMARY KEY(owner_user_id,knowledge_id),
 FOREIGN KEY(owner_user_id,knowledge_id) REFERENCES managed_study_notes(owner_user_id,knowledge_id),
 FOREIGN KEY(current_submission_id,owner_user_id,knowledge_id) REFERENCES note_share_submissions(submission_id,owner_user_id,knowledge_id),
 CHECK((status='unshared')=(current_submission_id IS NULL)),
 CHECK((status='shared')=(approved_at IS NOT NULL))
);
CREATE INDEX note_share_public_order ON note_share_heads(status,approved_at DESC,current_submission_id DESC);
CREATE TABLE note_share_events (
 event_id uuid PRIMARY KEY,owner_user_id uuid NOT NULL,knowledge_id text NOT NULL,
 submission_id uuid NOT NULL,actor_user_id uuid NOT NULL REFERENCES auth_users(id),
 action text NOT NULL CHECK(action IN ('submit','withdraw','approve','return','hide')),
 reason text NOT NULL DEFAULT '' CHECK(char_length(reason)<=1000),
 sequence bigint NOT NULL CHECK(sequence BETWEEN 1 AND 9007199254740991),
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(owner_user_id,knowledge_id) REFERENCES note_share_heads(owner_user_id,knowledge_id),
 FOREIGN KEY(submission_id,owner_user_id,knowledge_id) REFERENCES note_share_submissions(submission_id,owner_user_id,knowledge_id),
 UNIQUE(owner_user_id,knowledge_id,sequence),
 CHECK((action IN ('return','hide'))=(char_length(reason)>=10))
);
CREATE INDEX note_share_event_submission ON note_share_events(submission_id,sequence DESC);
CREATE TRIGGER note_share_event_immutable BEFORE UPDATE OR DELETE ON note_share_events FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TABLE note_share_receipts (
 actor_user_id uuid NOT NULL REFERENCES auth_users(id),action text NOT NULL CHECK(action IN ('submit','withdraw','approve','return','hide')),
 key text NOT NULL CHECK(octet_length(key) BETWEEN 1 AND 128),resource text NOT NULL,
 input_sha256 text NOT NULL CHECK(input_sha256 ~ '^[0-9a-f]{64}$'),receipt jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(actor_user_id,action,key)
);
CREATE TRIGGER note_share_receipt_immutable BEFORE UPDATE OR DELETE ON note_share_receipts FOR EACH ROW EXECUTE FUNCTION reject_content_update();

-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM note_share_heads) OR EXISTS(SELECT 1 FROM note_share_submissions) OR EXISTS(SELECT 1 FROM note_share_events) OR EXISTS(SELECT 1 FROM note_share_receipts) THEN
 RAISE EXCEPTION 'note sharing migration cannot be rolled back after use';
 END IF;
END $$;
-- +goose StatementEnd
DROP TABLE note_share_receipts,note_share_events,note_share_heads,note_share_submissions;
