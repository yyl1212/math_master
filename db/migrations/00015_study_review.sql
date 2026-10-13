-- +goose Up
-- This extension never changes the original learning facts or note-sharing schema.
-- +goose StatementBegin
CREATE FUNCTION study_review_blank(body text) RETURNS boolean LANGUAGE sql IMMUTABLE STRICT AS $$
 SELECT btrim(body,E' \t\n\r\v\f'||chr(133)||chr(160)||chr(5760)||chr(8192)||chr(8193)||chr(8194)||chr(8195)||chr(8196)||chr(8197)||chr(8198)||chr(8199)||chr(8200)||chr(8201)||chr(8202)||chr(8232)||chr(8233)||chr(8239)||chr(8287)||chr(12288))=''
$$;
CREATE FUNCTION study_review_material_bytes(sections jsonb) RETURNS bigint LANGUAGE plpgsql IMMUTABLE STRICT AS $$
DECLARE s jsonb; item jsonb; total bigint:=0;
BEGIN
 IF jsonb_typeof(sections)<>'array' THEN RETURN -1; END IF;
 FOR s IN SELECT value FROM jsonb_array_elements(sections) LOOP
  IF jsonb_typeof(s)<>'object' OR (SELECT count(*) FROM jsonb_object_keys(s))<>2 OR jsonb_typeof(s->'kind') IS DISTINCT FROM 'string' OR s->>'kind' NOT IN ('statement','conditions','scope','system','equations','proof','proof-scope') OR jsonb_typeof(s->'items') IS DISTINCT FROM 'array' OR jsonb_array_length(s->'items')=0 THEN RETURN -1; END IF;
  FOR item IN SELECT value FROM jsonb_array_elements(s->'items') LOOP
   IF jsonb_typeof(item)<>'string' THEN RETURN -1; END IF;
   total:=total+octet_length(item#>>'{}');
  END LOOP;
 END LOOP;
 RETURN total;
END $$;
-- +goose StatementEnd
CREATE TABLE study_review_sessions (
 session_id uuid PRIMARY KEY CHECK(substr(session_id::text,15,1)='4'),
 owner_user_id uuid NOT NULL,knowledge_id text NOT NULL,
 review_id uuid NOT NULL CHECK(substr(review_id::text,15,1)='4'),
 knowledge_ref jsonb NOT NULL CHECK(managed_ref_valid(knowledge_ref,knowledge_id)),
 mode text NOT NULL CHECK(mode IN ('read','recall')),
 target text NOT NULL CHECK(target IN ('statement','proof') AND (mode='recall' OR target='statement')),
 status text NOT NULL DEFAULT 'working' CHECK(status IN ('working','compared','completed','abandoned')),
 sequence bigint NOT NULL DEFAULT 1 CHECK(sequence BETWEEN 1 AND 9007199254740991),
 draft_revision bigint NOT NULL DEFAULT 0 CHECK(draft_revision BETWEEN 0 AND 9007199254740991),
 answer text CHECK(char_length(answer)<=16000 AND octet_length(answer)<=65536),
 unanswered boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 compared_at timestamptz,graded_at timestamptz,abandoned_at timestamptz,draft_updated_at timestamptz,
 grade jsonb,
 UNIQUE(session_id,owner_user_id,knowledge_id),
 UNIQUE(session_id,owner_user_id,knowledge_id,review_id,knowledge_ref,mode,target),
 FOREIGN KEY(owner_user_id,knowledge_id) REFERENCES managed_study_records(owner_user_id,knowledge_id),
 CHECK((mode='read' AND answer IS NULL AND NOT unanswered) OR (mode='recall' AND answer IS NOT NULL)),
 CHECK((status='working' AND compared_at IS NULL AND graded_at IS NULL AND abandoned_at IS NULL AND grade IS NULL AND NOT unanswered)
 OR (status='compared' AND compared_at IS NOT NULL AND graded_at IS NULL AND abandoned_at IS NULL AND grade IS NULL)
 OR (status='completed' AND compared_at IS NOT NULL AND graded_at IS NOT NULL AND abandoned_at IS NULL AND grade IS NOT NULL AND jsonb_typeof(grade)='object' AND grade->>'rating' IN ('forgotten','uncertain','familiar'))
 OR (status='abandoned' AND abandoned_at IS NOT NULL AND graded_at IS NULL AND grade IS NULL)),
 CHECK(mode='read' OR compared_at IS NULL OR unanswered=study_review_blank(answer)),
 CHECK(NOT unanswered OR grade IS NULL OR grade->>'rating'='forgotten')
);
CREATE UNIQUE INDEX study_review_one_active ON study_review_sessions(owner_user_id,knowledge_id) WHERE status IN ('working','compared');
CREATE INDEX study_review_history_order ON study_review_sessions(owner_user_id,graded_at DESC,session_id DESC) WHERE status='completed';
-- +goose StatementBegin
CREATE FUNCTION study_review_session_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'review sessions cannot be deleted'; END IF;
 IF OLD.status IN ('completed','abandoned') OR ROW(NEW.session_id,NEW.owner_user_id,NEW.knowledge_id,NEW.review_id,NEW.knowledge_ref,NEW.mode,NEW.target,NEW.created_at) IS DISTINCT FROM ROW(OLD.session_id,OLD.owner_user_id,OLD.knowledge_id,OLD.review_id,OLD.knowledge_ref,OLD.mode,OLD.target,OLD.created_at) OR NEW.sequence<>OLD.sequence+1 THEN RAISE EXCEPTION 'review immutable fields or sequence changed'; END IF;
 IF OLD.status='compared' AND ROW(NEW.answer,NEW.unanswered,NEW.compared_at,NEW.draft_revision,NEW.draft_updated_at) IS DISTINCT FROM ROW(OLD.answer,OLD.unanswered,OLD.compared_at,OLD.draft_revision,OLD.draft_updated_at) THEN RAISE EXCEPTION 'review answer frozen'; END IF;
 IF OLD.status='working' AND NEW.status IN ('working','compared') THEN
  IF NEW.draft_revision<>OLD.draft_revision+1 OR (NEW.status='working' AND NEW.draft_updated_at IS NULL) THEN RAISE EXCEPTION 'review draft revision invalid'; END IF;
 ELSIF NEW.status='abandoned' OR (OLD.status='compared' AND NEW.status='completed') THEN
  IF NEW.draft_revision<>OLD.draft_revision OR ROW(NEW.answer,NEW.unanswered,NEW.draft_updated_at) IS DISTINCT FROM ROW(OLD.answer,OLD.unanswered,OLD.draft_updated_at) THEN RAISE EXCEPTION 'review completion changed answer'; END IF;
 ELSE RAISE EXCEPTION 'review state transition invalid'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER study_review_session_guard BEFORE UPDATE OR DELETE ON study_review_sessions FOR EACH ROW EXECUTE FUNCTION study_review_session_guard();
CREATE TABLE study_review_snapshots (
 session_id uuid PRIMARY KEY,owner_user_id uuid NOT NULL,knowledge_id text NOT NULL,
 title text NOT NULL CHECK(char_length(title)<=1024 AND octet_length(title)<=4096),
 title_zh text NOT NULL CHECK(char_length(title_zh)<=1024 AND octet_length(title_zh)<=4096),
 knowledge_type text NOT NULL CHECK(knowledge_type IN ('concept','definition','axiom','theorem','corollary','method','mathematical-thinking','other')),
 prompt jsonb NOT NULL,expected jsonb NOT NULL,
 sha256 text NOT NULL CHECK(sha256 ~ '^[0-9a-f]{64}$'),
 FOREIGN KEY(session_id,owner_user_id,knowledge_id) REFERENCES study_review_sessions(session_id,owner_user_id,knowledge_id),
 CHECK(study_review_material_bytes(prompt)>=0 AND study_review_material_bytes(expected)>0 AND study_review_material_bytes(prompt)+study_review_material_bytes(expected)<=524288)
);
CREATE TRIGGER study_review_snapshot_immutable BEFORE UPDATE OR DELETE ON study_review_snapshots FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TABLE study_review_schedules (
 owner_user_id uuid NOT NULL,knowledge_id text NOT NULL,
 stage integer NOT NULL CHECK(stage BETWEEN 0 AND 5),due_at timestamptz NOT NULL,
 last_ref jsonb NOT NULL CHECK(managed_ref_valid(last_ref,knowledge_id)),
 last_rating text NOT NULL CHECK(last_rating IN ('forgotten','uncertain','familiar')),
 sequence bigint NOT NULL CHECK(sequence BETWEEN 1 AND 9007199254740991),graded_at timestamptz NOT NULL,
 source_session_id uuid NOT NULL,
 PRIMARY KEY(owner_user_id,knowledge_id),
 FOREIGN KEY(owner_user_id,knowledge_id) REFERENCES managed_study_records(owner_user_id,knowledge_id),
 FOREIGN KEY(source_session_id,owner_user_id,knowledge_id) REFERENCES study_review_sessions(session_id,owner_user_id,knowledge_id)
);
CREATE INDEX study_review_due ON study_review_schedules(owner_user_id,due_at,knowledge_id);
-- +goose StatementBegin
CREATE FUNCTION study_review_schedule_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE s study_review_sessions;
BEGIN
 SELECT * INTO s FROM study_review_sessions WHERE session_id=NEW.source_session_id AND owner_user_id=NEW.owner_user_id AND knowledge_id=NEW.knowledge_id;
 IF NOT FOUND OR s.status<>'completed' OR ROW(s.knowledge_ref,s.grade->>'rating',(s.grade->'schedule'->>'stage')::integer,(s.grade->'schedule'->>'sequence')::bigint,(s.grade->'schedule'->>'dueAt')::timestamptz,(s.grade->>'gradedAt')::timestamptz) IS DISTINCT FROM ROW(NEW.last_ref,NEW.last_rating,NEW.stage,NEW.sequence,NEW.due_at,NEW.graded_at) THEN RAISE EXCEPTION 'review schedule must match completed self-rating'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE CONSTRAINT TRIGGER study_review_schedule_guard AFTER INSERT OR UPDATE ON study_review_schedules DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION study_review_schedule_guard();
CREATE TABLE study_review_events (
 event_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 session_id uuid NOT NULL,owner_user_id uuid NOT NULL,knowledge_id text NOT NULL,review_id uuid NOT NULL,
 sequence bigint NOT NULL CHECK(sequence BETWEEN 1 AND 9007199254740991),
 action text NOT NULL CHECK(action IN ('opened','draft-saved','compared','graded','abandoned')),
 knowledge_ref jsonb NOT NULL CHECK(managed_ref_valid(knowledge_ref,knowledge_id)),
 mode text NOT NULL CHECK(mode IN ('read','recall')),target text NOT NULL CHECK(target IN ('statement','proof')),
 grade jsonb,
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(session_id,owner_user_id,knowledge_id,review_id,knowledge_ref,mode,target) REFERENCES study_review_sessions(session_id,owner_user_id,knowledge_id,review_id,knowledge_ref,mode,target),
 UNIQUE(session_id,sequence),
 CHECK((action='graded' AND grade IS NOT NULL AND jsonb_typeof(grade)='object' AND grade->>'rating' IN ('forgotten','uncertain','familiar')) OR (action<>'graded' AND grade IS NULL))
);
CREATE UNIQUE INDEX study_review_one_cycle_grade ON study_review_events(owner_user_id,knowledge_id,review_id) WHERE action='graded';
CREATE TRIGGER study_review_event_immutable BEFORE UPDATE OR DELETE ON study_review_events FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TABLE study_review_receipts (
 actor_user_id uuid NOT NULL REFERENCES auth_users(id),
 action text NOT NULL CHECK(action IN ('start','draft','compare','grade','abandon')),
 key text NOT NULL CHECK(octet_length(key) BETWEEN 1 AND 128),resource text NOT NULL,
 input_sha256 text NOT NULL CHECK(input_sha256 ~ '^[0-9a-f]{64}$'),
 session_id uuid NOT NULL,knowledge_id text NOT NULL,
 receipt jsonb NOT NULL CHECK(jsonb_typeof(receipt)='object' AND NOT receipt ?| ARRAY['body','answer','prompt','reference']),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(actor_user_id,action,key),
 FOREIGN KEY(session_id,actor_user_id,knowledge_id) REFERENCES study_review_sessions(session_id,owner_user_id,knowledge_id)
);
CREATE TRIGGER study_review_receipt_immutable BEFORE UPDATE OR DELETE ON study_review_receipts FOR EACH ROW EXECUTE FUNCTION reject_content_update();

-- +goose Down
-- Isolated empty-database tests only; never apply Down on production.
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM study_review_sessions) OR EXISTS(SELECT 1 FROM study_review_snapshots) OR EXISTS(SELECT 1 FROM study_review_schedules) OR EXISTS(SELECT 1 FROM study_review_events) OR EXISTS(SELECT 1 FROM study_review_receipts) THEN RAISE EXCEPTION 'study review migration cannot be rolled back after use'; END IF;
END $$;
-- +goose StatementEnd
DROP TABLE study_review_receipts,study_review_events,study_review_schedules,study_review_snapshots,study_review_sessions;
DROP FUNCTION study_review_schedule_guard(),study_review_session_guard(),study_review_material_bytes(jsonb),study_review_blank(text);
