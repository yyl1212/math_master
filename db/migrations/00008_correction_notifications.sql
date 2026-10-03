-- +goose Up
-- 永久启用标记不随空表回退删除。
ALTER TABLE goose_db_version ADD COLUMN IF NOT EXISTS correction_enabled boolean NOT NULL DEFAULT false;
UPDATE goose_db_version SET correction_enabled=true WHERE version_id=0;
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION correction_marker_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF OLD.version_id=0 AND OLD.correction_enabled AND (TG_OP='DELETE' OR NEW.version_id<>0 OR NOT NEW.correction_enabled) THEN RAISE EXCEPTION 'permanent correction enablement'; END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF; RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS correction_marker_immutable ON goose_db_version;
CREATE TRIGGER correction_marker_immutable BEFORE UPDATE OR DELETE ON goose_db_version FOR EACH ROW EXECUTE FUNCTION correction_marker_guard();
-- +goose StatementEnd

CREATE TABLE correction_cases (
 id uuid PRIMARY KEY CHECK(feedback_uuid(id::text)),kind text NOT NULL CHECK(kind IN ('withdrawal','grading_rule')),
 withdrawal_space text,withdrawal_id uuid,content_withdrawal_id uuid REFERENCES content_withdrawals(id),question_withdrawal_id uuid REFERENCES question_withdrawals(id),
 rule_version integer,scope_kind text,knowledge_id text,knowledge_version integer,knowledge_sha256 text,cutoff timestamptz,
 creator_user_id uuid REFERENCES auth_users(id),sequence bigint NOT NULL DEFAULT 1 CHECK(sequence=1),sealed boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),last_backfill_at timestamptz,
 FOREIGN KEY(knowledge_id,knowledge_version,knowledge_sha256) REFERENCES knowledge_versions(id,version,sha256),
 CHECK(((kind='withdrawal' AND cutoff IS NULL AND rule_version IS NULL AND scope_kind IS NULL AND knowledge_id IS NULL AND knowledge_version IS NULL AND knowledge_sha256 IS NULL AND withdrawal_id IS NOT NULL AND ((withdrawal_space='content' AND content_withdrawal_id=withdrawal_id AND question_withdrawal_id IS NULL) OR (withdrawal_space='question' AND question_withdrawal_id=withdrawal_id AND content_withdrawal_id IS NULL))) OR (kind='grading_rule' AND withdrawal_space IS NULL AND withdrawal_id IS NULL AND content_withdrawal_id IS NULL AND question_withdrawal_id IS NULL AND creator_user_id IS NOT NULL AND cutoff IS NOT NULL AND rule_version>0 AND ((scope_kind='all' AND knowledge_id IS NULL AND knowledge_version IS NULL AND knowledge_sha256 IS NULL) OR (scope_kind='knowledge' AND knowledge_id IS NOT NULL AND knowledge_version IS NOT NULL AND knowledge_sha256 IS NOT NULL)))) IS TRUE),
 UNIQUE(withdrawal_space,withdrawal_id)
);
CREATE INDEX correction_case_rotation ON correction_cases(last_backfill_at NULLS FIRST,created_at,id);
CREATE INDEX correction_case_rule_scope ON correction_cases(rule_version,scope_kind,knowledge_id,knowledge_version,knowledge_sha256,cutoff);
CREATE TABLE correction_plans (
 id uuid NOT NULL CHECK(feedback_uuid(id::text)),version integer NOT NULL CHECK(version>0),case_id uuid NOT NULL REFERENCES correction_cases(id),
 parent_id uuid,parent_version integer,creator_user_id uuid NOT NULL REFERENCES auth_users(id),status text NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','pending','approved','rejected')),
 sequence bigint NOT NULL DEFAULT 1 CHECK(sequence BETWEEN 1 AND 9007199254740991),algorithm_version integer NOT NULL DEFAULT 1 CHECK(algorithm_version=1),body jsonb NOT NULL,
 frozen_body jsonb,frozen_bytes bytea,frozen_digest text,sealed boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(id,version),
 FOREIGN KEY(parent_id,parent_version) REFERENCES correction_plans(id,version),
 CHECK((parent_id IS NULL)=(parent_version IS NULL)),CHECK(parent_id IS NULL OR parent_id=id AND parent_version<version),
 CHECK((sealed=(status<>'draft') AND ((NOT sealed AND frozen_body IS NULL AND frozen_bytes IS NULL AND frozen_digest IS NULL) OR (sealed AND frozen_body IS NOT NULL AND frozen_bytes IS NOT NULL AND frozen_digest IS NOT NULL AND frozen_digest=encode(sha256(frozen_bytes),'hex') AND convert_from(frozen_bytes,'UTF8')::jsonb=frozen_body AND frozen_body->>'purpose'='correction-plan-v1'))) IS TRUE),
 CHECK((feedback_shape(body,ARRAY['expectedSequence','parent','algorithmVersion','mappings','reason']) AND body->>'algorithmVersion'='1' AND jsonb_typeof(body->'mappings')='array' AND jsonb_array_length(body->'mappings')<=50 AND feedback_text(body->>'reason',1,4000)) IS TRUE)
);
CREATE INDEX correction_plan_case_page ON correction_plans(case_id,created_at DESC,id DESC,version DESC);
CREATE TABLE correction_jobs (
 id uuid PRIMARY KEY CHECK(feedback_uuid(id::text)),source_key text NOT NULL UNIQUE CHECK(octet_length(source_key) BETWEEN 1 AND 512),case_id uuid NOT NULL REFERENCES correction_cases(id),plan_id uuid,plan_version integer,
 type text NOT NULL CHECK(type IN ('withdrawal_impact','rule_impact','approved_plan','attempt_terminal')),evidence_kind text,evidence_id uuid,owner_user_id uuid REFERENCES auth_users(id),
 state text NOT NULL DEFAULT 'queued' CHECK(state IN ('queued','running','succeeded','retry_wait','failed')),sequence bigint NOT NULL DEFAULT 1 CHECK(sequence BETWEEN 1 AND 9007199254740991),epoch integer NOT NULL DEFAULT 1 CHECK(epoch>0),attempt integer NOT NULL DEFAULT 0 CHECK(attempt BETWEEN 0 AND 8),
 lease_token bigint NOT NULL DEFAULT 0 CHECK(lease_token>=0),lease_until timestamptz,next_run_at timestamptz DEFAULT clock_timestamp(),cursor_kind text,cursor_id uuid,processed_count bigint NOT NULL DEFAULT 0 CHECK(processed_count BETWEEN 0 AND 9007199254740991),error_class text,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),FOREIGN KEY(plan_id,plan_version) REFERENCES correction_plans(id,version),CHECK((plan_id IS NULL)=(plan_version IS NULL)),
 CHECK((evidence_kind IS NULL)=(evidence_id IS NULL)),CHECK(evidence_kind IS NULL OR evidence_kind IN ('learning-event','practice','assessment','enrollment')),CHECK((cursor_kind IS NULL)=(cursor_id IS NULL)),
 CHECK(cursor_kind IS NULL OR cursor_kind IN ('learning-event','practice','assessment','enrollment')),CHECK(error_class IS NULL OR error_class IN ('database','deadline','source','configuration','lease','internal')),
 CHECK((state='running')=(lease_until IS NOT NULL)),CHECK((state IN ('queued','retry_wait'))=(next_run_at IS NOT NULL)),CHECK((type='attempt_terminal')=(evidence_id IS NOT NULL)),CHECK(type<>'approved_plan' OR plan_id IS NOT NULL)
);
CREATE INDEX correction_job_ready ON correction_jobs(state,next_run_at,lease_until,id);
CREATE INDEX correction_job_case_page ON correction_jobs(case_id,created_at DESC,id DESC);
CREATE TABLE correction_results (
 id uuid PRIMARY KEY CHECK(feedback_uuid(id::text)),owner_user_id uuid NOT NULL REFERENCES auth_users(id),case_id uuid NOT NULL REFERENCES correction_cases(id),plan_id uuid,plan_version integer,parent_result_id uuid REFERENCES correction_results(id),
 evidence_kind text NOT NULL CHECK(evidence_kind IN ('learning-event','practice','assessment','enrollment')),evidence_id uuid NOT NULL,knowledge_id text,knowledge_version integer,knowledge_sha256 text,
 status text NOT NULL CHECK(status IN ('corrected_passed','corrected_failed','retake_required','review_material','checked_unaffected','awaiting_review')),
 reason text NOT NULL CHECK(reason IN ('answer_corrected','rule_regraded','source_invalid','intent_changed','coverage_changed','knowledge_changed','insufficient_items','no_approved_basis','conflicting_basis','path_withdrawn','unaffected')),
 score integer,passed boolean,correctness jsonb NOT NULL DEFAULT '[]',handled_case_ids jsonb NOT NULL CHECK(jsonb_typeof(handled_case_ids)='array'),
 basis jsonb NOT NULL,basis_bytes bytea NOT NULL,basis_digest text NOT NULL,source_key text NOT NULL REFERENCES correction_jobs(source_key),sealed boolean NOT NULL DEFAULT false,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(plan_id,plan_version) REFERENCES correction_plans(id,version),FOREIGN KEY(knowledge_id,knowledge_version,knowledge_sha256) REFERENCES knowledge_versions(id,version,sha256),
 CHECK((plan_id IS NULL)=(plan_version IS NULL)),CHECK((knowledge_id IS NULL)=(knowledge_version IS NULL)),CHECK((knowledge_id IS NULL)=(knowledge_sha256 IS NULL)),
 CHECK((basis_digest=encode(sha256(basis_bytes),'hex') AND convert_from(basis_bytes,'UTF8')::jsonb=basis AND basis->>'purpose'='correction-result-v1') IS TRUE),
 CHECK((jsonb_typeof(correctness)='array' AND ((evidence_kind='assessment' AND status IN ('corrected_passed','corrected_failed') AND score BETWEEN 0 AND 5 AND passed=(score>=4) AND passed=(status='corrected_passed') AND jsonb_array_length(correctness)=5) OR (NOT(evidence_kind='assessment' AND status IN ('corrected_passed','corrected_failed')) AND score IS NULL AND passed IS NULL))) IS TRUE),
 UNIQUE(source_key,evidence_kind,evidence_id,basis_digest),UNIQUE(id,owner_user_id)
);
CREATE INDEX correction_result_owner_page ON correction_results(owner_user_id,evidence_kind,evidence_id,created_at DESC,id DESC);
CREATE TABLE correction_dependencies (
 result_id uuid NOT NULL REFERENCES correction_results(id),role text NOT NULL CHECK(role IN ('audit','effective')),kind text NOT NULL CHECK(kind IN ('knowledge','unit','asset','template','instance','blueprint')),id text NOT NULL,version integer,sha256 text NOT NULL CHECK(feedback_identity(jsonb_build_object('id',id,'version',coalesce(version,1),'sha256',sha256),kind='instance')),
 positions integer[] NOT NULL DEFAULT '{}',knowledge_id text GENERATED ALWAYS AS(CASE WHEN kind='knowledge' THEN id END) STORED,unit_id text GENERATED ALWAYS AS(CASE WHEN kind='unit' THEN id END) STORED,template_id text GENERATED ALWAYS AS(CASE WHEN kind='template' THEN id END) STORED,instance_id text GENERATED ALWAYS AS(CASE WHEN kind='instance' THEN id END) STORED,blueprint_id text GENERATED ALWAYS AS(CASE WHEN kind='blueprint' THEN id END) STORED,asset_sha256 text GENERATED ALWAYS AS(CASE WHEN kind='asset' THEN sha256 END) STORED,
 UNIQUE NULLS NOT DISTINCT(result_id,role,kind,id,version,sha256),CHECK((kind='asset')=(version IS NULL)),CHECK(positions <@ ARRAY[1,2,3,4,5]),
 FOREIGN KEY(asset_sha256) REFERENCES assets(sha256),FOREIGN KEY(knowledge_id,version,sha256) REFERENCES knowledge_versions(id,version,sha256),FOREIGN KEY(unit_id,version,sha256) REFERENCES unit_versions(id,version,sha256),FOREIGN KEY(template_id,version,sha256) REFERENCES question_templates(id,version,sha256),FOREIGN KEY(instance_id,version,sha256) REFERENCES question_instances(id,version,sha256),FOREIGN KEY(blueprint_id,version,sha256) REFERENCES question_blueprints(id,version,sha256)
);
CREATE INDEX correction_dependency_reverse ON correction_dependencies(kind,id,version,sha256,result_id) WHERE role='effective';
CREATE INDEX correction_dependency_asset ON correction_dependencies(sha256,result_id) WHERE role='effective' AND kind='asset';
CREATE TABLE correction_events (
 id uuid PRIMARY KEY CHECK(feedback_uuid(id::text)),subject_kind text NOT NULL CHECK(subject_kind IN ('case','plan','job','result','exposure')),subject_id uuid NOT NULL,subject_version integer,
 case_id uuid NOT NULL REFERENCES correction_cases(id),plan_id uuid,plan_version integer,job_id uuid REFERENCES correction_jobs(id),result_id uuid REFERENCES correction_results(id),owner_user_id uuid REFERENCES auth_users(id),
 kind text NOT NULL CHECK(kind IN ('case_registered','plan_created','plan_updated','plan_submitted','plan_approved','plan_rejected','job_created','job_claimed','job_renewed','job_continued','job_succeeded','job_attempt_failed','job_retry','result_sealed','qualification_granted','detail_exposed')),
 sequence bigint NOT NULL CHECK(sequence BETWEEN 1 AND 9007199254740991),actor_user_id uuid REFERENCES auth_users(id),body jsonb NOT NULL,body_bytes bytea NOT NULL,body_digest text NOT NULL,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(plan_id,plan_version) REFERENCES correction_plans(id,version),CHECK((plan_id IS NULL)=(plan_version IS NULL)),CHECK((subject_kind='plan')=(subject_version IS NOT NULL)),
 CHECK((body_digest=encode(sha256(body_bytes),'hex') AND convert_from(body_bytes,'UTF8')::jsonb=body AND body->>'purpose' IN ('correction-case-v1','correction-plan-v1','correction-result-v1','correction-command-v1','notification-command-v1')) IS TRUE),
 UNIQUE NULLS NOT DISTINCT(subject_kind,subject_id,subject_version,sequence)
);
CREATE UNIQUE INDEX correction_qualification_once ON correction_events(owner_user_id,result_id,(body#>>'{body,knowledge,id}'),(body#>>'{body,knowledge,version}'),(body#>>'{body,knowledge,sha256}'),(body#>>'{body,evidenceAttemptId}'),(body#>>'{body,completedEventId}')) NULLS NOT DISTINCT WHERE kind='qualification_granted';
CREATE INDEX correction_exposure_scope ON correction_events(owner_user_id,recorded_at,case_id) WHERE kind='detail_exposed';
CREATE TABLE correction_idempotency (
 owner_user_id uuid NOT NULL REFERENCES auth_users(id),action text NOT NULL CHECK(action IN ('createCase','createPlan','updatePlan','submitPlan','decidePlan','retryJob','notification-read')),resource text NOT NULL CHECK(octet_length(resource) BETWEEN 1 AND 512),key uuid NOT NULL CHECK(feedback_uuid(key::text)),digest text NOT NULL CHECK(digest ~ '^[0-9a-f]{64}$'),status integer NOT NULL CHECK(status IN (200,201)),receipt bytea NOT NULL CHECK(octet_length(receipt)<=2097152),created_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(owner_user_id,action,resource,key),CHECK((convert_from(receipt,'UTF8')::jsonb->>'status')::integer=status)
);
CREATE TABLE correction_rate_limits (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),owner_user_id uuid NOT NULL REFERENCES auth_users(id),scope text NOT NULL CHECK(scope IN ('create','process','retry','notification-read')),consumed_at timestamptz NOT NULL DEFAULT clock_timestamp(),command_key text NOT NULL CHECK(octet_length(command_key) BETWEEN 1 AND 1024),UNIQUE(owner_user_id,scope,command_key)
);
CREATE INDEX correction_rate_window ON correction_rate_limits(owner_user_id,scope,consumed_at);
CREATE TABLE notifications (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid() CHECK(feedback_uuid(id::text)),owner_user_id uuid NOT NULL REFERENCES auth_users(id),dedup_key text NOT NULL UNIQUE CHECK(octet_length(dedup_key) BETWEEN 1 AND 512),type text NOT NULL CHECK(type IN ('checking','corrected','retake','review_material','path_unavailable')),
 evidence_kind text NOT NULL CHECK(evidence_kind IN ('learning-event','practice','assessment','enrollment')),evidence_id uuid NOT NULL,case_id uuid NOT NULL REFERENCES correction_cases(id),result_id uuid,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(result_id,owner_user_id) REFERENCES correction_results(id,owner_user_id),UNIQUE(id,owner_user_id)
);
CREATE INDEX notification_owner_page ON notifications(owner_user_id,created_at DESC,id DESC);
CREATE TABLE notification_reads (
 owner_user_id uuid NOT NULL,notification_id uuid NOT NULL,read_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(owner_user_id,notification_id),FOREIGN KEY(notification_id,owner_user_id) REFERENCES notifications(id,owner_user_id)
);
CREATE INDEX correction_original_asset ON learning_evidence_dependencies(sha256,evidence_kind,evidence_id,owner_user_id) WHERE kind='asset';
CREATE INDEX correction_assessment_scope ON assessment_attempts(rule_version,created_at,id,owner_user_id);
CREATE INDEX correction_practice_scope ON practice_attempts(((seal#>>'{body,ruleVersion}')::integer),created_at,id,owner_user_id);

-- +goose StatementBegin
CREATE FUNCTION correction_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'immutable correction fact'; END $$;
CREATE FUNCTION correction_case_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='INSERT' THEN
  IF NEW.kind='grading_rule' THEN
   IF NOT EXISTS(SELECT 1 FROM auth_user_roles WHERE user_id=NEW.creator_user_id AND role='admin') THEN RAISE EXCEPTION 'administrator required'; END IF;
   NEW.cutoff:=clock_timestamp();NEW.created_at:=NEW.cutoff;
  END IF;RETURN NEW;
 END IF;
 IF TG_OP='DELETE' OR (to_jsonb(NEW)-'last_backfill_at'-'sealed') IS DISTINCT FROM (to_jsonb(OLD)-'last_backfill_at'-'sealed') OR OLD.sealed AND NOT NEW.sealed THEN RAISE EXCEPTION 'immutable correction case source'; END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION correction_case_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE c correction_cases;
BEGIN
 SELECT * INTO c FROM correction_cases WHERE id=NEW.id;
 IF NOT c.sealed OR NOT EXISTS(SELECT 1 FROM correction_events WHERE subject_kind='case' AND subject_id=c.id AND subject_version IS NULL AND sequence=1 AND kind='case_registered' AND case_id=c.id AND actor_user_id IS NOT DISTINCT FROM c.creator_user_id) THEN RAISE EXCEPTION 'registered case event required'; END IF;RETURN NULL;
END $$;
CREATE FUNCTION correction_plan_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE p correction_plans;
BEGIN
 IF TG_OP='INSERT' THEN
  IF NEW.parent_id IS NOT NULL THEN
   SELECT * INTO p FROM correction_plans WHERE id=NEW.parent_id AND version=NEW.parent_version;
   IF p.status='draft' OR p.case_id<>NEW.case_id OR p.creator_user_id<>NEW.creator_user_id OR NEW.version<>(SELECT coalesce(max(version),0)+1 FROM correction_plans WHERE id=NEW.id) THEN RAISE EXCEPTION 'invalid plan lineage'; END IF;
  ELSIF NEW.version<>1 THEN RAISE EXCEPTION 'initial version required'; END IF;
  IF NEW.status<>'draft' OR NEW.sequence<>1 THEN RAISE EXCEPTION 'new draft required'; END IF;RETURN NEW;
 END IF;
 IF TG_OP='DELETE' OR OLD.status IN ('approved','rejected') THEN RAISE EXCEPTION 'immutable correction decision'; END IF;
 IF NEW.sequence<>OLD.sequence+1 OR (to_jsonb(NEW)-'status'-'sequence'-'updated_at'-'body'-'sealed'-'frozen_body'-'frozen_bytes'-'frozen_digest') IS DISTINCT FROM (to_jsonb(OLD)-'status'-'sequence'-'updated_at'-'body'-'sealed'-'frozen_body'-'frozen_bytes'-'frozen_digest') THEN RAISE EXCEPTION 'invalid correction plan advance'; END IF;
 IF OLD.status='pending' AND (NEW.status NOT IN ('approved','rejected') OR (to_jsonb(NEW)-'status'-'sequence'-'updated_at') IS DISTINCT FROM (to_jsonb(OLD)-'status'-'sequence'-'updated_at')) THEN RAISE EXCEPTION 'frozen correction plan'; END IF;
 IF OLD.status='draft' AND NEW.status NOT IN ('draft','pending') THEN RAISE EXCEPTION 'submit before deciding'; END IF;RETURN NEW;
END $$;
CREATE FUNCTION correction_instance_proof(i jsonb,pub uuid,approval jsonb) RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT EXISTS(SELECT 1 FROM question_instances q JOIN question_publication_members m ON m.kind='instance' AND m.id=q.id AND m.version=q.version AND m.sha256=q.sha256 JOIN question_publications p ON p.id=m.publication_id AND p.sealed AND p.status='published' JOIN question_submissions s ON s.id=m.submission_id AND s.sealed AND s.status='approved' JOIN question_review_decisions r ON r.id=m.review_id AND r.submission_id=s.id AND r.decision='approve' AND r.frozen_digest=s.frozen_digest WHERE q.id=i#>>'{identity,id}' AND q.version=(i#>>'{identity,version}')::integer AND q.sha256=i#>>'{identity,sha256}' AND q.sealed AND q.body->'body'=jsonb_set(i,'{identity}',(i->'identity')-'sha256') AND m.publication_id=pub AND m.evidence=approval)
$$;
CREATE FUNCTION correction_plan_proof(frozen jsonb,cid uuid,input jsonb) RETURNS boolean LANGUAGE plpgsql STABLE AS $$
DECLARE b jsonb:=frozen#>'{body,proof}';m jsonb;inm jsonb;idx integer:=0;approval text;
BEGIN
 IF frozen->>'purpose' IS DISTINCT FROM 'correction-plan-v1' OR frozen#>'{body,input}' IS DISTINCT FROM input OR b->>'caseId' IS DISTINCT FROM cid::text OR b->>'algorithmVersion' IS DISTINCT FROM '1' OR jsonb_typeof(b->'mappings') IS DISTINCT FROM 'array' OR jsonb_typeof(b->'authors') IS DISTINCT FROM 'array' OR jsonb_array_length(b->'mappings')<>jsonb_array_length(input->'mappings') THEN RETURN false; END IF;
 FOR m IN SELECT value FROM jsonb_array_elements(b->'mappings') LOOP
  inm:=input->'mappings'->idx;idx:=idx+1;
  IF inm->'original' IS DISTINCT FROM m#>'{original,identity}' OR inm#>'{replacement,identity}' IS DISTINCT FROM m#>'{replacement,identity}' OR inm->>'originalPublicationId' IS DISTINCT FROM m->>'originalPublicationId' OR inm#>>'{replacement,publicationId}' IS DISTINCT FROM m->>'replacementPublicationId' OR NOT correction_instance_proof(m->'original',(m->>'originalPublicationId')::uuid,m->'originalApproval') OR NOT correction_instance_proof(m->'replacement',(m->>'replacementPublicationId')::uuid,m->'replacementApproval') THEN RETURN false; END IF;
 END LOOP;
 FOR approval IN SELECT jsonb_array_elements_text(b->'contentApprovalIds') LOOP IF NOT EXISTS(SELECT 1 FROM content_review_decisions WHERE id=approval::uuid AND decision='approve') THEN RETURN false; END IF;END LOOP;
 FOR approval IN SELECT jsonb_array_elements_text(b->'questionApprovalIds') LOOP IF NOT EXISTS(SELECT 1 FROM question_review_decisions WHERE id=approval::uuid AND decision='approve') THEN RETURN false; END IF;END LOOP;
 RETURN true;
END $$;
CREATE FUNCTION correction_plan_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE p correction_plans;e correction_events;author uuid;
BEGIN
 SELECT * INTO p FROM correction_plans WHERE id=NEW.id AND version=NEW.version;
 SELECT * INTO e FROM correction_events WHERE subject_kind='plan' AND subject_id=p.id AND subject_version=p.version AND sequence=p.sequence AND case_id=p.case_id;
 IF NOT FOUND OR (p.status='draft' AND e.kind NOT IN ('plan_created','plan_updated')) OR (p.status='pending' AND e.kind<>'plan_submitted') OR (p.status='approved' AND e.kind<>'plan_approved') OR (p.status='rejected' AND e.kind<>'plan_rejected') THEN RAISE EXCEPTION 'matching plan event required'; END IF;
 IF p.sealed AND NOT correction_plan_proof(p.frozen_body,p.case_id,p.body) THEN RAISE EXCEPTION 'accurate frozen plan proof required'; END IF;
 IF p.status IN ('approved','rejected') THEN
  IF e.actor_user_id IS NULL OR e.actor_user_id=p.creator_user_id OR NOT EXISTS(SELECT 1 FROM auth_users u JOIN auth_user_roles r ON r.user_id=u.id WHERE u.id=e.actor_user_id AND NOT u.must_change_password AND r.role IN ('reviewer','admin')) OR NOT feedback_text(e.body#>>'{body,reason}',1,4000) THEN RAISE EXCEPTION 'independent current reviewer required'; END IF;
  IF EXISTS(SELECT 1 FROM jsonb_array_elements_text(p.frozen_body#>'{body,proof,authors}') a WHERE a::uuid=e.actor_user_id) THEN RAISE EXCEPTION 'source author cannot review'; END IF;
  IF EXISTS(SELECT 1 FROM correction_events a WHERE a.subject_kind='plan' AND a.subject_id=p.id AND a.subject_version=p.version AND a.kind IN ('plan_created','plan_updated') AND a.actor_user_id=e.actor_user_id) THEN RAISE EXCEPTION 'draft editor cannot review'; END IF;
  IF EXISTS(SELECT 1 FROM jsonb_array_elements_text(p.frozen_body#>'{body,proof,contentApprovalIds}') a JOIN content_review_decisions r ON r.id=a::uuid JOIN content_submission_authors s ON s.submission_id=r.submission_id WHERE s.user_id=e.actor_user_id) OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(p.frozen_body#>'{body,proof,questionApprovalIds}') a JOIN question_review_decisions r ON r.id=a::uuid JOIN question_submission_authors s ON s.submission_id=r.submission_id WHERE s.user_id=e.actor_user_id) THEN RAISE EXCEPTION 'actual mathematical author cannot review'; END IF;
  IF EXISTS(SELECT 1 FROM jsonb_array_elements(p.frozen_body#>'{body,proof,mappings}') m CROSS JOIN LATERAL (VALUES(m#>>'{originalApproval,submissionId}'),(m#>>'{replacementApproval,submissionId}')) sid(v) JOIN question_submission_authors s ON s.submission_id=sid.v::uuid WHERE s.user_id=e.actor_user_id) THEN RAISE EXCEPTION 'instance author cannot review'; END IF;
 END IF;RETURN NULL;
END $$;
CREATE FUNCTION correction_job_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'persistent correction job'; END IF;
 IF (to_jsonb(NEW)-'state'-'sequence'-'epoch'-'attempt'-'lease_token'-'lease_until'-'next_run_at'-'cursor_kind'-'cursor_id'-'processed_count'-'error_class') IS DISTINCT FROM (to_jsonb(OLD)-'state'-'sequence'-'epoch'-'attempt'-'lease_token'-'lease_until'-'next_run_at'-'cursor_kind'-'cursor_id'-'processed_count'-'error_class') OR NEW.sequence<>OLD.sequence+1 OR NEW.epoch NOT IN (OLD.epoch,OLD.epoch+1) OR NEW.lease_token<OLD.lease_token OR NEW.processed_count<OLD.processed_count THEN RAISE EXCEPTION 'invalid job advance'; END IF;
 IF NEW.epoch=OLD.epoch AND NEW.attempt NOT IN (OLD.attempt,OLD.attempt+1) OR NEW.epoch=OLD.epoch+1 AND NEW.attempt<>0 THEN RAISE EXCEPTION 'invalid retry attempt'; END IF;RETURN NEW;
END $$;
CREATE FUNCTION correction_job_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE j correction_jobs;
BEGIN
 SELECT * INTO j FROM correction_jobs WHERE id=NEW.id;
 IF NOT EXISTS(SELECT 1 FROM correction_events WHERE subject_kind='job' AND subject_id=j.id AND sequence=j.sequence AND case_id=j.case_id) THEN RAISE EXCEPTION 'job event required'; END IF;
 IF j.plan_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM correction_plans WHERE id=j.plan_id AND version=j.plan_version AND case_id=j.case_id AND status='approved') THEN RAISE EXCEPTION 'approved job source required'; END IF;
 IF j.evidence_id IS NOT NULL AND (j.owner_user_id IS NULL OR NOT learning_evidence_owner(j.evidence_kind,j.evidence_id,j.owner_user_id)) THEN RAISE EXCEPTION 'owned terminal job evidence required'; END IF;RETURN NULL;
END $$;
CREATE FUNCTION correction_result_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' OR OLD.sealed OR NOT NEW.sealed OR (to_jsonb(NEW)-'sealed') IS DISTINCT FROM (to_jsonb(OLD)-'sealed') THEN RAISE EXCEPTION 'immutable correction result'; END IF;RETURN NEW;
END $$;
CREATE FUNCTION correction_equivalent_json(a jsonb,b jsonb) RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
 SELECT coalesce((a#>'{body,knowledge}')=(b#>'{body,knowledge}') AND a->'parameters'=b->'parameters' AND ((a->'body')-'correctChoiceId'-'correctNumeric'-'explanation')=((b->'body')-'correctChoiceId'-'correctNumeric'-'explanation'),false)
$$;
CREATE FUNCTION correction_replacement_authorized(b jsonb,pos integer) RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT (b#>ARRAY['originalItems',pos::text,'identity'])=(b#>ARRAY['effectiveItems',pos::text,'identity']) OR EXISTS(
  SELECT 1 FROM jsonb_array_elements(b->'planRefs') ref JOIN correction_plans p ON p.id=(ref->>'id')::uuid AND p.version=(ref->>'version')::integer AND p.status='approved' AND p.algorithm_version=1
  CROSS JOIN LATERAL jsonb_array_elements(p.frozen_body#>'{body,proof,mappings}') m
  WHERE m#>'{replacement,identity}'=b#>ARRAY['effectiveItems',pos::text,'identity'] AND (
   m#>'{original,identity}'=b#>ARRAY['originalItems',pos::text,'identity'] OR EXISTS(
    SELECT 1 FROM jsonb_array_elements_text(b->'parentResultIds') pid JOIN correction_results r ON r.id=pid::uuid AND r.sealed
    WHERE r.basis#>ARRAY['body','effectiveItems',pos::text,'identity']=m#>'{original,identity}'
   ))
 )
$$;
CREATE FUNCTION correction_result_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE r correction_results;b jsonb;original_seal jsonb;answers jsonb;item jsonb;effective jsonb;idx integer:=0;other correction_results;handled_id text;dep jsonb;expected_role text;
BEGIN
 SELECT * INTO r FROM correction_results WHERE id=NEW.id;b:=r.basis->'body';
 IF NOT EXISTS(SELECT 1 FROM correction_events WHERE subject_kind='result' AND subject_id=r.id AND case_id=r.case_id AND kind='result_sealed' AND body#>>'{body,digest}'=r.basis_digest) THEN RAISE EXCEPTION 'result seal audit required'; END IF;
 IF NOT r.sealed OR NOT learning_evidence_owner(r.evidence_kind,r.evidence_id,r.owner_user_id) OR NOT EXISTS(SELECT 1 FROM correction_jobs WHERE source_key=r.source_key AND case_id=r.case_id AND plan_id IS NOT DISTINCT FROM r.plan_id AND plan_version IS NOT DISTINCT FROM r.plan_version) THEN RAISE EXCEPTION 'owned sealed correction source required'; END IF;
 IF r.parent_result_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM correction_results WHERE id=r.parent_result_id AND id<>r.id AND sealed AND owner_user_id=r.owner_user_id AND evidence_kind=r.evidence_kind AND evidence_id=r.evidence_id) THEN RAISE EXCEPTION 'owned parent result required'; END IF;
 FOR handled_id IN SELECT jsonb_array_elements_text(r.handled_case_ids) LOOP IF NOT EXISTS(SELECT 1 FROM correction_cases WHERE id=handled_id::uuid AND sealed) THEN RAISE EXCEPTION 'real handled case required'; END IF;END LOOP;
 IF b->'handledCaseIds' IS DISTINCT FROM r.handled_case_ids THEN RAISE EXCEPTION 'handled basis mismatch'; END IF;
 FOR item IN SELECT value FROM jsonb_array_elements(coalesce(b->'parentResultIds','[]')) LOOP
  SELECT * INTO other FROM correction_results WHERE id=(item#>>'{}')::uuid AND sealed;
  IF NOT FOUND OR other.id=r.id OR (other.owner_user_id,other.evidence_kind,other.evidence_id) IS DISTINCT FROM (r.owner_user_id,r.evidence_kind,r.evidence_id) OR other.basis#>'{body,parentResultIds}' @> jsonb_build_array(r.id::text) THEN RAISE EXCEPTION 'invalid parent closure'; END IF;
 END LOOP;
 IF r.evidence_kind='assessment' THEN SELECT seal->'body' INTO original_seal FROM assessment_attempts WHERE id=r.evidence_id AND state='submitted';SELECT coalesce(jsonb_agg(answer ORDER BY position),'[]') INTO answers FROM assessment_answers WHERE attempt_id=r.evidence_id;
 ELSIF r.evidence_kind='practice' THEN SELECT seal->'body',jsonb_build_array(answer) INTO original_seal,answers FROM practice_attempts WHERE id=r.evidence_id AND state='answered';END IF;
 IF r.evidence_kind IN ('assessment','practice') AND (original_seal IS NULL OR b->'originalSeal' IS DISTINCT FROM original_seal OR b->'originalAnswers' IS DISTINCT FROM answers) THEN RAISE EXCEPTION 'original answers and seal required'; END IF;
 IF r.evidence_kind IN ('assessment','practice') AND (r.knowledge_id IS NULL OR original_seal->'knowledge' IS DISTINCT FROM jsonb_build_object('id',r.knowledge_id,'version',r.knowledge_version,'sha256',r.knowledge_sha256)) THEN RAISE EXCEPTION 'original knowledge identity required'; END IF;
 IF r.status IN ('corrected_passed','corrected_failed') THEN
  IF r.plan_id IS NULL OR NOT EXISTS(SELECT 1 FROM correction_plans WHERE id=r.plan_id AND version=r.plan_version AND case_id=r.case_id AND status='approved' AND algorithm_version=1) OR jsonb_array_length(b->'originalItems')<>jsonb_array_length(original_seal->'items') OR jsonb_array_length(b->'effectiveItems')<>jsonb_array_length(original_seal->'items') THEN RAISE EXCEPTION 'independently approved complete basis required'; END IF;
  FOR item IN SELECT value FROM jsonb_array_elements(b->'originalItems') LOOP
   effective:=b->'effectiveItems'->idx;
   IF item->'identity' IS DISTINCT FROM original_seal#>ARRAY['items',idx::text,'instance'] OR NOT EXISTS(SELECT 1 FROM question_instances WHERE id=item#>>'{identity,id}' AND version=(item#>>'{identity,version}')::integer AND sha256=item#>>'{identity,sha256}' AND body->'body'=jsonb_set(item,'{identity}',(item->'identity')-'sha256')) OR NOT EXISTS(SELECT 1 FROM question_instances WHERE id=effective#>>'{identity,id}' AND version=(effective#>>'{identity,version}')::integer AND sha256=effective#>>'{identity,sha256}' AND body->'body'=jsonb_set(effective,'{identity}',(effective->'identity')-'sha256')) OR NOT correction_equivalent_json(item,effective) OR NOT correction_replacement_authorized(b,idx) THEN RAISE EXCEPTION 'equivalent accurate instance required'; END IF;idx:=idx+1;
  END LOOP;
  IF r.evidence_kind='assessment' AND (idx<>5 OR jsonb_array_length(answers)<>5 OR r.score<>(SELECT count(*) FROM jsonb_array_elements(r.correctness) x WHERE x='true'::jsonb) OR NOT (original_seal->'core' <@ (SELECT coalesce(jsonb_agg(DISTINCT x),'[]') FROM jsonb_array_elements(original_seal->'items') i CROSS JOIN LATERAL jsonb_array_elements(i->'coverage') x))) THEN RAISE EXCEPTION 'complete five-item verdict required'; END IF;
 END IF;
 FOR expected_role IN SELECT unnest(ARRAY['audit','effective']) LOOP
  IF EXISTS((SELECT d->>'kind',d->>'id',(d->>'version')::integer,d->>'sha256' FROM jsonb_array_elements(coalesce(b->CASE WHEN expected_role='audit' THEN 'auditDeps' ELSE 'effectiveDeps' END,'[]')) d) EXCEPT (SELECT kind,id,version,sha256 FROM correction_dependencies WHERE result_id=r.id AND role=expected_role)) OR EXISTS((SELECT kind,id,version,sha256 FROM correction_dependencies WHERE result_id=r.id AND role=expected_role) EXCEPT (SELECT d->>'kind',d->>'id',(d->>'version')::integer,d->>'sha256' FROM jsonb_array_elements(coalesce(b->CASE WHEN expected_role='audit' THEN 'auditDeps' ELSE 'effectiveDeps' END,'[]')) d)) THEN RAISE EXCEPTION 'complete correction dependencies required'; END IF;
 END LOOP;RETURN NULL;
END $$;
CREATE FUNCTION correction_dependency_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP<>'INSERT' OR EXISTS(SELECT 1 FROM correction_results WHERE id=NEW.result_id AND sealed) THEN RAISE EXCEPTION 'immutable correction dependency'; END IF;RETURN NEW;
END $$;
CREATE FUNCTION correction_notice_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN IF NOT learning_evidence_owner(NEW.evidence_kind,NEW.evidence_id,NEW.owner_user_id) OR NEW.result_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM correction_results WHERE id=NEW.result_id AND owner_user_id=NEW.owner_user_id AND evidence_kind=NEW.evidence_kind AND evidence_id=NEW.evidence_id AND case_id=NEW.case_id AND sealed) THEN RAISE EXCEPTION 'owned notification source required'; END IF;RETURN NEW;END $$;
CREATE FUNCTION correction_event_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE p correction_plans;r correction_results;a assessment_attempts;completion learning_events;b jsonb:=NEW.body->'body';
BEGIN
 IF NEW.subject_kind='case' AND NOT EXISTS(SELECT 1 FROM correction_cases WHERE id=NEW.subject_id AND id=NEW.case_id) OR NEW.subject_kind='plan' AND NOT EXISTS(SELECT 1 FROM correction_plans WHERE id=NEW.subject_id AND version=NEW.subject_version AND case_id=NEW.case_id) OR NEW.subject_kind='job' AND NOT EXISTS(SELECT 1 FROM correction_jobs WHERE id=NEW.subject_id AND case_id=NEW.case_id) OR NEW.subject_kind='result' AND NOT EXISTS(SELECT 1 FROM correction_results WHERE id=NEW.subject_id AND case_id=NEW.case_id) THEN RAISE EXCEPTION 'event subject mismatch'; END IF;
 IF NEW.kind='qualification_granted' THEN
  SELECT * INTO r FROM correction_results WHERE id=NEW.result_id AND id=NEW.subject_id AND sealed AND owner_user_id=NEW.owner_user_id AND status='corrected_passed' AND evidence_kind='assessment' AND passed AND score BETWEEN 4 AND 5;
  IF NOT FOUND OR NOT EXISTS(SELECT 1 FROM correction_plans WHERE id=r.plan_id AND version=r.plan_version AND status='approved') THEN RAISE EXCEPTION 'approved corrected qualification required'; END IF;
  SELECT * INTO a FROM assessment_attempts WHERE id=r.evidence_id AND owner_user_id=r.owner_user_id AND state='submitted';
  IF NOT FOUND OR a.id::text IS DISTINCT FROM b->>'evidenceAttemptId' OR b->'knowledge' IS DISTINCT FROM jsonb_build_object('id',a.knowledge_id,'version',a.knowledge_version,'sha256',a.knowledge_sha256) OR (a.mode='diagnostic') IS DISTINCT FROM (b->>'kind'='diagnostic') THEN RAISE EXCEPTION 'qualification source mismatch'; END IF;
  IF a.mode<>'diagnostic' THEN SELECT * INTO completion FROM learning_events WHERE id=(b->>'completedEventId')::uuid AND owner_user_id=a.owner_user_id AND kind='completed';IF NOT FOUND OR (completion.knowledge_id,completion.knowledge_version,completion.knowledge_sha256) IS DISTINCT FROM (a.knowledge_id,a.knowledge_version,a.knowledge_sha256) THEN RAISE EXCEPTION 'normal correction needs completion'; END IF;
  ELSIF b->'completedEventId' IS DISTINCT FROM 'null'::jsonb THEN RAISE EXCEPTION 'diagnostic must not invent completion'; END IF;
 END IF;RETURN NEW;
END $$;
CREATE TRIGGER correction_case_source BEFORE INSERT OR UPDATE OR DELETE ON correction_cases FOR EACH ROW EXECUTE FUNCTION correction_case_guard();
CREATE CONSTRAINT TRIGGER correction_case_registered AFTER INSERT OR UPDATE ON correction_cases DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION correction_case_complete();
CREATE TRIGGER correction_plan_freeze BEFORE INSERT OR UPDATE OR DELETE ON correction_plans FOR EACH ROW EXECUTE FUNCTION correction_plan_guard();
CREATE CONSTRAINT TRIGGER correction_plan_decision AFTER INSERT OR UPDATE ON correction_plans DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION correction_plan_complete();
CREATE TRIGGER correction_job_advance BEFORE UPDATE OR DELETE ON correction_jobs FOR EACH ROW EXECUTE FUNCTION correction_job_guard();
CREATE CONSTRAINT TRIGGER correction_job_audit AFTER INSERT OR UPDATE ON correction_jobs DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION correction_job_complete();
CREATE TRIGGER correction_result_freeze BEFORE UPDATE OR DELETE ON correction_results FOR EACH ROW EXECUTE FUNCTION correction_result_guard();
CREATE CONSTRAINT TRIGGER correction_result_basis AFTER INSERT OR UPDATE ON correction_results DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION correction_result_complete();
CREATE TRIGGER correction_dependency_freeze BEFORE INSERT OR UPDATE OR DELETE ON correction_dependencies FOR EACH ROW EXECUTE FUNCTION correction_dependency_guard();
CREATE TRIGGER correction_events_immutable BEFORE UPDATE OR DELETE ON correction_events FOR EACH ROW EXECUTE FUNCTION correction_immutable();
CREATE TRIGGER correction_event_subject BEFORE INSERT ON correction_events FOR EACH ROW EXECUTE FUNCTION correction_event_guard();
CREATE TRIGGER correction_receipts_immutable BEFORE UPDATE OR DELETE ON correction_idempotency FOR EACH ROW EXECUTE FUNCTION correction_immutable();
CREATE TRIGGER correction_rates_immutable BEFORE UPDATE OR DELETE ON correction_rate_limits FOR EACH ROW EXECUTE FUNCTION correction_immutable();
CREATE TRIGGER notification_source BEFORE INSERT ON notifications FOR EACH ROW EXECUTE FUNCTION correction_notice_guard();
CREATE TRIGGER notifications_immutable BEFORE UPDATE OR DELETE ON notifications FOR EACH ROW EXECUTE FUNCTION correction_immutable();
CREATE TRIGGER notification_reads_immutable BEFORE UPDATE OR DELETE ON notification_reads FOR EACH ROW EXECUTE FUNCTION correction_immutable();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$DECLARE name text;n bigint;BEGIN
 FOREACH name IN ARRAY ARRAY['correction_cases','correction_plans','correction_events','correction_jobs','correction_results','correction_dependencies','correction_idempotency','correction_rate_limits','notifications','notification_reads'] LOOP
  EXECUTE format('SELECT count(*) FROM %I',name) INTO n;IF n>0 THEN RAISE EXCEPTION 'nonempty correction workflow cannot roll back'; END IF;
 END LOOP;
END $$;
DROP TABLE notification_reads,notifications,correction_idempotency,correction_rate_limits,correction_events,correction_dependencies,correction_results,correction_jobs,correction_plans,correction_cases;
DROP FUNCTION correction_event_guard(),correction_notice_guard(),correction_dependency_guard(),correction_result_complete(),correction_equivalent_json(jsonb,jsonb),correction_replacement_authorized(jsonb,integer),correction_result_guard(),correction_job_complete(),correction_job_guard(),correction_plan_complete(),correction_plan_proof(jsonb,uuid,jsonb),correction_instance_proof(jsonb,uuid,jsonb),correction_plan_guard(),correction_case_complete(),correction_case_guard(),correction_immutable();
DROP INDEX correction_original_asset,correction_assessment_scope,correction_practice_scope;
-- correction_enabled 与 correction_marker_guard/trigger 永久保留。
-- +goose StatementEnd
