-- +goose Up
-- 显式启用 P4b；旧数学正文、摘要和迁移不变。
-- Goose 回退删除版本行；独立启用标记保留在原迁移元数据，Down 不允许静默降级曝光保护。
ALTER TABLE goose_db_version ADD COLUMN IF NOT EXISTS learning_enabled boolean NOT NULL DEFAULT false;
UPDATE goose_db_version SET learning_enabled=true WHERE version_id=0;

CREATE UNIQUE INDEX learning_knowledge_identity_sha ON knowledge_versions(id,version,sha256);
CREATE UNIQUE INDEX learning_unit_identity_sha ON unit_versions(id,version,sha256);
CREATE UNIQUE INDEX learning_path_identity_sha ON path_versions(id,version,sha256);
CREATE INDEX learning_candidate_knowledge ON question_instances(knowledge_id,knowledge_version,id,version,sha256);
CREATE INDEX learning_coverage_knowledge ON question_instance_coverage(knowledge_id,knowledge_version,instance_id,instance_version,objective_index);

-- +goose StatementBegin
CREATE FUNCTION learning_content_approved(snap text,k text,object_id text,v integer,h text) RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT EXISTS(
  SELECT 1 FROM publication_members m JOIN publication_snapshots p ON p.id=m.snapshot_id AND p.status='published'
  JOIN content_publication_manifests cm ON cm.snapshot_id=p.id
  CROSS JOIN LATERAL jsonb_array_elements(cm.manifest->'members') entry
  JOIN content_review_decisions r ON r.id::text=entry#>>'{evidence,decisionId}' AND r.decision='approve'
  JOIN content_submissions s ON s.id=r.submission_id AND s.sealed AND s.status='approved' AND s.frozen_digest=r.frozen_digest
  JOIN content_submission_members sm ON sm.submission_id=s.id AND sm.kind=m.kind AND sm.id=m.id AND sm.version=m.version AND sm.sha256=h
  WHERE m.snapshot_id=snap AND m.kind=k AND m.id=object_id AND m.version=v AND m.availability='active'
   AND entry#>>'{identity,kind}'=k AND entry#>>'{identity,id}'=object_id AND entry#>>'{identity,version}'=v::text AND entry#>>'{identity,sha256}'=h
   AND entry#>>'{evidence,submissionId}'=s.id::text AND entry#>>'{evidence,frozenDigest}'=s.frozen_digest);
$$;
CREATE FUNCTION learning_material_hash(seal jsonb) RETURNS text LANGUAGE sql IMMUTABLE STRICT AS $$
 SELECT encode(sha256(convert_to(jsonb_build_object('knowledge',seal#>'{body,knowledge}','units',seal#>'{body,units}','assets',seal#>'{body,assets}')::text,'UTF8')),'hex');
$$;
-- +goose StatementEnd

CREATE TABLE learning_events (
 id uuid PRIMARY KEY CHECK(id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),owner_user_id uuid NOT NULL REFERENCES auth_users(id),
 knowledge_id text NOT NULL,knowledge_version integer NOT NULL,knowledge_sha256 text NOT NULL,
 knowledge_publication_id text NOT NULL REFERENCES publication_snapshots(id),kind text NOT NULL CHECK(kind IN ('started','completed')),
 seal jsonb NOT NULL,seal_bytes bytea NOT NULL CHECK(octet_length(seal_bytes)<=4194304),seal_sha256 text NOT NULL CHECK(seal_sha256 ~ '^[0-9a-f]{64}$'),
 material_sha256 text GENERATED ALWAYS AS (learning_material_hash(seal)) STORED,recorded_at timestamptz NOT NULL,
 UNIQUE(id,owner_user_id),UNIQUE(owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,kind,material_sha256),
 FOREIGN KEY(knowledge_id,knowledge_version,knowledge_sha256) REFERENCES knowledge_versions(id,version,sha256),
 CHECK((encode(sha256(seal_bytes),'hex')=seal_sha256 AND convert_from(seal_bytes,'UTF8')::jsonb=seal AND seal->>'purpose'='learning-event-v1') IS TRUE),
 CHECK((seal#>>'{body,id}'=id::text AND seal#>>'{body,actorId}'=owner_user_id::text AND seal#>>'{body,kind}'=kind AND (seal#>>'{body,recordedAt}')::timestamptz=recorded_at) IS TRUE)
);
CREATE UNIQUE INDEX learning_first_started ON learning_events(owner_user_id,knowledge_id,knowledge_version,knowledge_sha256) WHERE kind='started';
CREATE INDEX learning_event_history ON learning_events(owner_user_id,recorded_at DESC,id DESC);
CREATE TABLE learning_records (
 owner_user_id uuid NOT NULL REFERENCES auth_users(id),knowledge_id text NOT NULL,knowledge_version integer NOT NULL,knowledge_sha256 text NOT NULL,
 started_event_id uuid NOT NULL,started_at timestamptz NOT NULL,completed_event_id uuid,completed_at timestamptz,
 PRIMARY KEY(owner_user_id,knowledge_id,knowledge_version,knowledge_sha256),
 FOREIGN KEY(knowledge_id,knowledge_version,knowledge_sha256) REFERENCES knowledge_versions(id,version,sha256),
 FOREIGN KEY(started_event_id,owner_user_id) REFERENCES learning_events(id,owner_user_id) DEFERRABLE INITIALLY DEFERRED,
 FOREIGN KEY(completed_event_id,owner_user_id) REFERENCES learning_events(id,owner_user_id) DEFERRABLE INITIALLY DEFERRED,
 CHECK((completed_event_id IS NULL)=(completed_at IS NULL)),CHECK(completed_at IS NULL OR completed_at>=started_at)
);
CREATE TABLE learning_path_enrollments (
 id uuid PRIMARY KEY CHECK(id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),owner_user_id uuid NOT NULL REFERENCES auth_users(id),
 path_id text NOT NULL,path_version integer NOT NULL,path_sha256 text NOT NULL,knowledge_publication_id text NOT NULL REFERENCES publication_snapshots(id),
 total_nodes integer NOT NULL CHECK(total_nodes BETWEEN 1 AND 1000),created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(id,owner_user_id),UNIQUE(owner_user_id,path_id,path_version,path_sha256),
 FOREIGN KEY(path_id,path_version,path_sha256) REFERENCES path_versions(id,version,sha256)
);
CREATE INDEX learning_enrollment_history ON learning_path_enrollments(owner_user_id,created_at DESC,id DESC);
CREATE TABLE learning_path_nodes (
 enrollment_id uuid NOT NULL REFERENCES learning_path_enrollments(id),position integer NOT NULL CHECK(position BETWEEN 0 AND 999),
 knowledge_id text NOT NULL,knowledge_version integer NOT NULL,knowledge_sha256 text NOT NULL,
 PRIMARY KEY(enrollment_id,position),UNIQUE(enrollment_id,knowledge_id),
 FOREIGN KEY(knowledge_id,knowledge_version,knowledge_sha256) REFERENCES knowledge_versions(id,version,sha256)
);
CREATE TABLE learner_exposure_state (
 owner_user_id uuid PRIMARY KEY REFERENCES auth_users(id),sequence bigint NOT NULL DEFAULT 0 CHECK(sequence>=0),updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE learner_answer_exposures (
 owner_user_id uuid NOT NULL REFERENCES auth_users(id),kind text NOT NULL CHECK(kind IN ('template','instance')),
 id text NOT NULL CHECK(id ~ '^[a-z][a-z0-9-]{0,63}$' OR (kind='instance' AND id ~ '^qi-[0-9a-f]{64}$')),
 version integer NOT NULL CHECK(version>0),sha256 text NOT NULL CHECK(sha256 ~ '^[0-9a-f]{64}$'),sequence bigint NOT NULL CHECK(sequence>0),exposed_at timestamptz NOT NULL,
 PRIMARY KEY(owner_user_id,kind,id,version,sha256)
);
-- 草稿曝光允许早于发布：数学身份不能 FK 到尚未存在的 question 行。
CREATE INDEX learning_exposure_reverse ON learner_answer_exposures(owner_user_id,exposed_at,kind,id,version,sha256);
CREATE TABLE learner_question_views (
 owner_user_id uuid NOT NULL REFERENCES auth_users(id),instance_id text NOT NULL,instance_version integer NOT NULL,instance_sha256 text NOT NULL,
 first_seen_at timestamptz NOT NULL,last_seen_at timestamptz NOT NULL CHECK(last_seen_at>=first_seen_at),
 PRIMARY KEY(owner_user_id,instance_id,instance_version,instance_sha256),
 FOREIGN KEY(instance_id,instance_version,instance_sha256) REFERENCES question_instances(id,version,sha256)
);
CREATE TABLE assessment_attempts (
 id uuid PRIMARY KEY CHECK(id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),owner_user_id uuid NOT NULL REFERENCES auth_users(id),
 knowledge_id text NOT NULL,knowledge_version integer NOT NULL,knowledge_sha256 text NOT NULL,
 knowledge_publication_id text NOT NULL REFERENCES publication_snapshots(id),question_publication_id uuid NOT NULL REFERENCES question_publications(id),
 blueprint_id text NOT NULL,blueprint_version integer NOT NULL,blueprint_sha256 text NOT NULL,mode text NOT NULL CHECK(mode IN ('node','diagnostic','review')),
 rule_version integer NOT NULL CHECK(rule_version=1),core jsonb NOT NULL CHECK((jsonb_typeof(core)='array' AND jsonb_array_length(core) BETWEEN 1 AND 8) IS TRUE),seed text NOT NULL CHECK(seed ~ '^[0-9a-f]{64}$'),
 seal jsonb NOT NULL,seal_bytes bytea NOT NULL CHECK(octet_length(seal_bytes)<=4194304),seal_sha256 text NOT NULL CHECK(seal_sha256 ~ '^[0-9a-f]{64}$'),sealed boolean NOT NULL DEFAULT false,
 state text NOT NULL DEFAULT 'active' CHECK(state IN ('active','submitted','abandoned','expired')),created_at timestamptz NOT NULL,expires_at timestamptz NOT NULL,terminal_at timestamptz,
 creation_exposure_sequence bigint NOT NULL CHECK(creation_exposure_sequence>=0),submission_exposure_sequence bigint CHECK(submission_exposure_sequence>=creation_exposure_sequence),
 UNIQUE(id,owner_user_id),FOREIGN KEY(knowledge_id,knowledge_version,knowledge_sha256) REFERENCES knowledge_versions(id,version,sha256),
 FOREIGN KEY(blueprint_id,blueprint_version,blueprint_sha256) REFERENCES question_blueprints(id,version,sha256),
 CHECK(expires_at=created_at+interval '24 hours'),CHECK((state='active')=(terminal_at IS NULL)),CHECK(terminal_at IS NULL OR terminal_at>=created_at),
 CHECK((state='submitted')=(submission_exposure_sequence IS NOT NULL)),
 CHECK((encode(sha256(seal_bytes),'hex')=seal_sha256 AND convert_from(seal_bytes,'UTF8')::jsonb=seal AND seal->>'purpose'='assessment-attempt-v1') IS TRUE)
);
CREATE UNIQUE INDEX learning_one_active_assessment ON assessment_attempts(owner_user_id) WHERE state='active';
CREATE INDEX learning_recent_submission ON assessment_attempts(owner_user_id,terminal_at DESC,id DESC) WHERE state='submitted';
CREATE INDEX learning_attempt_evidence ON assessment_attempts(owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,terminal_at DESC,id);
CREATE TABLE assessment_items (
 attempt_id uuid NOT NULL REFERENCES assessment_attempts(id),position integer NOT NULL CHECK(position BETWEEN 1 AND 5),
 instance_id text NOT NULL,instance_version integer NOT NULL,instance_sha256 text NOT NULL,binding jsonb NOT NULL,
 PRIMARY KEY(attempt_id,position),UNIQUE(attempt_id,instance_id,instance_version,instance_sha256),UNIQUE(attempt_id,position,instance_id,instance_version,instance_sha256),
 FOREIGN KEY(instance_id,instance_version,instance_sha256) REFERENCES question_instances(id,version,sha256),
 CHECK((binding->>'position'=position::text AND binding#>>'{instance,id}'=instance_id AND binding#>>'{instance,version}'=instance_version::text AND binding#>>'{instance,sha256}'=instance_sha256) IS TRUE)
);
CREATE TABLE assessment_answers (
 attempt_id uuid NOT NULL,owner_user_id uuid NOT NULL,position integer NOT NULL,instance_id text NOT NULL,instance_version integer NOT NULL,instance_sha256 text NOT NULL,
 answer jsonb NOT NULL CHECK((jsonb_typeof(answer)='object') IS TRUE),PRIMARY KEY(attempt_id,position),
 FOREIGN KEY(attempt_id,owner_user_id) REFERENCES assessment_attempts(id,owner_user_id),
 FOREIGN KEY(attempt_id,position,instance_id,instance_version,instance_sha256) REFERENCES assessment_items(attempt_id,position,instance_id,instance_version,instance_sha256)
);
CREATE TABLE assessment_results (
 attempt_id uuid PRIMARY KEY,owner_user_id uuid NOT NULL,rule_version integer NOT NULL CHECK(rule_version=1),outcome text NOT NULL CHECK(outcome IN ('passed','failed','affected')),
 sealed boolean NOT NULL DEFAULT false,score integer,passed boolean,original_reasons jsonb NOT NULL CHECK((jsonb_typeof(original_reasons)='array') IS TRUE),progress jsonb NOT NULL,submitted_at timestamptz NOT NULL,
 FOREIGN KEY(attempt_id,owner_user_id) REFERENCES assessment_attempts(id,owner_user_id),
 CHECK(((outcome='affected' AND score IS NULL AND passed IS NULL) OR (outcome IN ('passed','failed') AND score BETWEEN 0 AND 5 AND passed=(score>=4) AND ((outcome='passed')=passed))) IS TRUE)
);
CREATE TABLE practice_attempts (
 id uuid PRIMARY KEY CHECK(id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),owner_user_id uuid NOT NULL REFERENCES auth_users(id),
 knowledge_id text NOT NULL,knowledge_version integer NOT NULL,knowledge_sha256 text NOT NULL,
 knowledge_publication_id text NOT NULL REFERENCES publication_snapshots(id),question_publication_id uuid NOT NULL REFERENCES question_publications(id),
 seal jsonb NOT NULL,seal_bytes bytea NOT NULL CHECK(octet_length(seal_bytes)<=4194304),seal_sha256 text NOT NULL CHECK(seal_sha256 ~ '^[0-9a-f]{64}$'),
 state text NOT NULL DEFAULT 'active' CHECK(state IN ('active','answered','revealed','abandoned','expired')),created_at timestamptz NOT NULL,expires_at timestamptz NOT NULL,terminal_at timestamptz,answer jsonb,correct boolean,
 UNIQUE(id,owner_user_id),FOREIGN KEY(knowledge_id,knowledge_version,knowledge_sha256) REFERENCES knowledge_versions(id,version,sha256),
 CHECK(expires_at=created_at+interval '24 hours'),CHECK((state='active')=(terminal_at IS NULL)),CHECK(terminal_at IS NULL OR terminal_at>=created_at),
 CHECK(((state='answered' AND answer IS NOT NULL AND correct IS NOT NULL) OR (state<>'answered' AND answer IS NULL AND correct IS NULL)) IS TRUE),
 CHECK((encode(sha256(seal_bytes),'hex')=seal_sha256 AND convert_from(seal_bytes,'UTF8')::jsonb=seal AND seal->>'purpose'='practice-attempt-v1') IS TRUE)
);
CREATE UNIQUE INDEX learning_one_active_practice ON practice_attempts(owner_user_id) WHERE state='active';
CREATE INDEX learning_practice_history ON practice_attempts(owner_user_id,created_at DESC,id DESC);
CREATE TABLE learning_qualification_events (
 id uuid PRIMARY KEY CHECK(id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),owner_user_id uuid NOT NULL REFERENCES auth_users(id),knowledge_id text NOT NULL,knowledge_version integer NOT NULL,knowledge_sha256 text NOT NULL,
 kind text NOT NULL CHECK(kind IN ('normal','diagnostic')),attempt_id uuid NOT NULL,completed_event_id uuid,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(knowledge_id,knowledge_version,knowledge_sha256) REFERENCES knowledge_versions(id,version,sha256),
 FOREIGN KEY(attempt_id,owner_user_id) REFERENCES assessment_attempts(id,owner_user_id),
 FOREIGN KEY(completed_event_id,owner_user_id) REFERENCES learning_events(id,owner_user_id),CHECK((kind='diagnostic')=(completed_event_id IS NULL)),
 UNIQUE NULLS NOT DISTINCT(owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,kind,attempt_id,completed_event_id)
);
CREATE INDEX learning_qualification_history ON learning_qualification_events(owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,created_at DESC,id);
CREATE TABLE learning_unlocks (
 owner_user_id uuid NOT NULL REFERENCES auth_users(id),knowledge_id text NOT NULL,knowledge_version integer NOT NULL,knowledge_sha256 text NOT NULL,knowledge_publication_id text NOT NULL REFERENCES publication_snapshots(id),
 source_kind text NOT NULL CHECK(source_kind IN ('learning-event','enrollment','assessment')),source_id uuid NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(owner_user_id,knowledge_id),FOREIGN KEY(knowledge_id,knowledge_version,knowledge_sha256) REFERENCES knowledge_versions(id,version,sha256)
);
CREATE TABLE learning_evidence_dependencies (
 evidence_kind text NOT NULL CHECK(evidence_kind IN ('learning-event','practice','assessment')),evidence_id uuid NOT NULL,owner_user_id uuid NOT NULL REFERENCES auth_users(id),
 kind text NOT NULL CHECK(kind IN ('knowledge','unit','asset','template','instance','blueprint')),id text NOT NULL,version integer,sha256 text NOT NULL CHECK(sha256 ~ '^[0-9a-f]{64}$'),
 CHECK((kind='asset' AND version IS NULL) OR (kind<>'asset' AND version>0)),
 UNIQUE NULLS NOT DISTINCT(evidence_kind,evidence_id,kind,id,version,sha256)
);
CREATE INDEX learning_dependency_owner ON learning_evidence_dependencies(owner_user_id,evidence_kind,evidence_id);
CREATE INDEX learning_dependency_restriction ON learning_evidence_dependencies(kind,id,version,sha256,evidence_kind,evidence_id,owner_user_id);
CREATE TABLE learning_idempotency (
 owner_user_id uuid NOT NULL REFERENCES auth_users(id),action text NOT NULL CHECK(action IN ('startKnowledge','completeKnowledge','enrollPath','createPractice','answerPractice','revealPractice','abandonPractice','createAssessment','submitAssessment','abandonAssessment')),
 target text NOT NULL CHECK(octet_length(target) BETWEEN 1 AND 80),key uuid NOT NULL CHECK(key::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'),request_sha256 text NOT NULL CHECK(request_sha256 ~ '^[0-9a-f]{64}$'),
 receipt jsonb NOT NULL,receipt_bytes bytea NOT NULL CHECK(octet_length(receipt_bytes)<=4096),created_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(owner_user_id,action,key),
 CHECK((convert_from(receipt_bytes,'UTF8')::jsonb=receipt AND jsonb_typeof(receipt)='object' AND receipt-'resourceKind'-'resourceId'-'status'='{}' AND receipt->>'resourceKind' IN ('knowledge','enrollment','practice','assessment') AND receipt->>'status' IN ('200','201') AND receipt->>'resourceId' IS NOT NULL) IS TRUE)
);

-- +goose StatementBegin
CREATE FUNCTION learning_item_proof(item jsonb,k jsonb,kpub text,qpub uuid) RETURNS boolean LANGUAGE plpgsql STABLE AS $$
DECLARE i question_instances; m question_publication_members; u jsonb; a jsonb; refs jsonb; covered jsonb;
BEGIN
 SELECT * INTO i FROM question_instances WHERE id=item#>>'{instance,id}' AND version=(item#>>'{instance,version}')::integer AND sha256=item#>>'{instance,sha256}';
 IF NOT FOUND OR NOT i.sealed OR i.knowledge_id<>k->>'id' OR i.knowledge_version<>(k->>'version')::integer THEN RETURN false; END IF;
 SELECT * INTO m FROM question_publication_members WHERE publication_id=qpub AND kind='instance' AND id=i.id AND version=i.version AND sha256=i.sha256;
 IF NOT FOUND OR NOT EXISTS(SELECT 1 FROM question_publications WHERE id=qpub AND sealed AND status='published') OR item->'approval' IS DISTINCT FROM m.evidence OR item->'template' IS DISTINCT FROM i.body#>'{body,template}' THEN RETURN false; END IF;
 IF NOT EXISTS(SELECT 1 FROM question_review_decisions r JOIN question_submissions s ON s.id=r.submission_id JOIN question_submission_members sm ON sm.submission_id=s.id AND sm.kind='instance' AND sm.id=i.id AND sm.version=i.version AND sm.sha256=i.sha256 WHERE r.id=m.review_id AND s.id=m.submission_id AND r.decision='approve' AND s.status='approved' AND s.sealed AND r.frozen_digest=s.frozen_digest AND m.evidence->>'frozenDigest'=s.frozen_digest) THEN RETURN false; END IF;
 SELECT coalesce(jsonb_agg(objective_index ORDER BY objective_index),'[]') INTO covered FROM question_instance_coverage WHERE instance_id=i.id AND instance_version=i.version AND knowledge_id=k->>'id' AND knowledge_version=(k->>'version')::integer;
 IF item->'coverage' IS DISTINCT FROM covered OR item->'assets' IS DISTINCT FROM i.body#>'{body,body,assets}' OR jsonb_typeof(item->'units') IS DISTINCT FROM 'array' THEN RETURN false; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('id',value->'id','version',value->'version') ORDER BY ord),'[]') INTO refs FROM jsonb_array_elements(item->'units') WITH ORDINALITY e(value,ord);
 IF refs IS DISTINCT FROM i.body#>'{body,body,units}' THEN RETURN false; END IF;
 FOR u IN SELECT value FROM jsonb_array_elements(item->'units') LOOP
  IF NOT EXISTS(SELECT 1 FROM unit_versions WHERE id=u->>'id' AND version=(u->>'version')::integer AND sha256=u->>'sha256') OR NOT learning_content_approved(kpub,'unit',u->>'id',(u->>'version')::integer,u->>'sha256') THEN RETURN false; END IF;
 END LOOP;
 FOR a IN SELECT value FROM jsonb_array_elements(item->'assets') LOOP
  IF NOT learning_content_approved(kpub,'asset',a->>'id',1,a->>'sha256') THEN RETURN false; END IF;
 END LOOP;
 RETURN true;
END;
$$;
CREATE FUNCTION learning_seal_proof(s jsonb,kid text,kv integer,kh text,kpub text,qpub uuid,kind text) RETURNS boolean LANGUAGE plpgsql STABLE AS $$
DECLARE b jsonb:=s->'body';i jsonb;bp question_blueprints;member question_publication_members;
BEGIN
 IF (b->>'kind'=kind AND b->'knowledge'=jsonb_build_object('id',kid,'version',kv,'sha256',kh) AND b->>'knowledgePublicationId'=kpub AND b->>'questionPublicationId'=qpub::text AND b->>'ruleVersion'='1' AND b->>'seed' ~ '^[0-9a-f]{64}$' AND jsonb_typeof(b->'items')='array') IS NOT TRUE THEN RETURN false; END IF;
 IF NOT learning_content_approved(kpub,'knowledge',kid,kv,kh) THEN RETURN false; END IF;
 IF kind='practice' THEN
  IF (jsonb_array_length(b->'items')=1 AND b->'mode'='null' AND b->'blueprint'='null' AND b->'core'='[]') IS NOT TRUE THEN RETURN false; END IF;
 ELSE
  IF (jsonb_array_length(b->'items')=5 AND b->>'mode' IN ('node','diagnostic','review') AND jsonb_typeof(b->'core')='array' AND jsonb_array_length(b->'core') BETWEEN 1 AND 8) IS NOT TRUE THEN RETURN false; END IF;
  SELECT * INTO bp FROM question_blueprints WHERE id=b#>>'{blueprint,id}' AND version=(b#>>'{blueprint,version}')::integer AND sha256=b#>>'{blueprint,sha256}' AND sealed AND knowledge_id=kid AND knowledge_version=kv;
  IF NOT FOUND OR bp.body#>'{body,coreObjectiveIndices}' IS DISTINCT FROM b->'core' THEN RETURN false; END IF;
  SELECT * INTO member FROM question_publication_members qpm WHERE qpm.publication_id=qpub AND qpm.kind='blueprint' AND qpm.id=bp.id AND qpm.version=bp.version AND qpm.sha256=bp.sha256;
  IF NOT FOUND THEN RETURN false; END IF;
 END IF;
 FOR i IN SELECT value FROM jsonb_array_elements(b->'items') LOOP
  IF NOT learning_item_proof(i,b->'knowledge',kpub,qpub) THEN RETURN false; END IF;
  IF kind='assessment' AND NOT EXISTS(SELECT 1 FROM question_blueprint_sources src JOIN question_instances qi ON qi.id=i#>>'{instance,id}' AND qi.version=(i#>>'{instance,version}')::integer WHERE src.blueprint_id=bp.id AND src.blueprint_version=bp.version AND ((src.kind='instance' AND src.id=qi.id AND src.version=qi.version) OR (src.kind='template' AND src.id=qi.template_id AND src.version=qi.template_version))) THEN RETURN false; END IF;
 END LOOP;
 IF kind='assessment' AND NOT ((b->'core') <@ (SELECT coalesce(jsonb_agg(DISTINCT goal),'[]') FROM jsonb_array_elements(b->'items') e CROSS JOIN LATERAL jsonb_array_elements(e->'coverage') goal)) THEN RETURN false; END IF;
 RETURN true;
END;
$$;
CREATE FUNCTION learning_answer_valid(body jsonb,a jsonb,allow_skip boolean) RETURNS boolean LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE raw text;value text;fmt text;
BEGIN
 IF jsonb_typeof(a) IS DISTINCT FROM 'object' THEN RETURN false; END IF;
 IF a->>'kind'='skipped' THEN RETURN allow_skip AND a='{"kind":"skipped"}'; END IF;
 IF body->>'type'='single_choice' THEN
  RETURN (a->>'kind'='choice' AND jsonb_typeof(a->'choiceId')='string' AND a-'kind'-'choiceId'='{}' AND EXISTS(SELECT 1 FROM jsonb_array_elements(body->'choices') opt WHERE opt->'id'=a->'choiceId')) IS TRUE;
 END IF;
 IF (body->>'type'='numeric' AND a->>'kind'='numeric' AND jsonb_typeof(a->'raw')='string' AND a-'kind'-'raw'='{}') IS NOT TRUE THEN RETURN false; END IF;
 raw:=a->>'raw';IF char_length(raw)>128 THEN RETURN false; END IF;
 value:=btrim(raw,E' \t\n\r\v\f\u0085\u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000');fmt:=body->>'answerFormat';
 IF fmt='percentage' THEN RETURN (value ~ '^[+-]?([0-9]+(\.[0-9]*)?|\.[0-9]+)%$') IS TRUE; END IF;
 IF fmt<>'rational' THEN RETURN false; END IF;
 IF value ~ '^[+-]?[0-9]+[ \t]*/[ \t]*[+-]?[0-9]+$' THEN RETURN btrim(split_part(value,'/',2),E' \t') !~ '^[+-]?0+$'; END IF;
 RETURN (value ~ '^[+-]?([0-9]+(\.[0-9]*)?|\.[0-9]+)$') IS TRUE;
END;
$$;
CREATE FUNCTION learning_attempt_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'immutable learning attempt'; END IF;
 IF TG_TABLE_NAME='assessment_attempts' THEN
  IF NOT OLD.sealed THEN
   IF NOT NEW.sealed OR NEW.state<>'active' OR (to_jsonb(NEW)-'sealed') IS DISTINCT FROM (to_jsonb(OLD)-'sealed') THEN RAISE EXCEPTION 'invalid assessment seal'; END IF;RETURN NEW;
  END IF;
 END IF;
 IF OLD.state<>'active' OR NEW.state='active' THEN RAISE EXCEPTION 'terminal learning attempt'; END IF;
 IF TG_TABLE_NAME='assessment_attempts' THEN
  IF (to_jsonb(NEW)-'state'-'terminal_at'-'submission_exposure_sequence') IS DISTINCT FROM (to_jsonb(OLD)-'state'-'terminal_at'-'submission_exposure_sequence') THEN RAISE EXCEPTION 'immutable assessment basis'; END IF;
 ELSE
  IF (to_jsonb(NEW)-'state'-'terminal_at'-'answer'-'correct') IS DISTINCT FROM (to_jsonb(OLD)-'state'-'terminal_at'-'answer'-'correct') THEN RAISE EXCEPTION 'immutable practice basis'; END IF;
 END IF;
 RETURN NEW;
END;
$$;
CREATE FUNCTION learning_attempt_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE p assessment_attempts;pr practice_attempts;actual jsonb;cnt integer;r assessment_results;b jsonb;
BEGIN
 IF TG_TABLE_NAME='practice_attempts' THEN
  SELECT * INTO pr FROM practice_attempts WHERE id=NEW.id;
  IF NOT learning_dependencies_match('practice',pr.id,pr.owner_user_id,pr.seal,false) THEN RAISE EXCEPTION 'incomplete practice dependencies'; END IF;
  IF NOT learning_seal_proof(pr.seal,pr.knowledge_id,pr.knowledge_version,pr.knowledge_sha256,pr.knowledge_publication_id,pr.question_publication_id,'practice') THEN RAISE EXCEPTION 'invalid published practice basis'; END IF;
  IF pr.state='answered' THEN
   SELECT qi.body#>'{body,body}' INTO b FROM question_instances qi WHERE qi.id=pr.seal#>>'{body,items,0,instance,id}' AND qi.version=(pr.seal#>>'{body,items,0,instance,version}')::integer;
   IF NOT learning_answer_valid(b,pr.answer,false) THEN RAISE EXCEPTION 'invalid practice answer'; END IF;
  END IF;
  IF pr.state IN ('answered','revealed') AND (pr.terminal_at>=pr.expires_at OR clock_timestamp()>=pr.expires_at) THEN RAISE EXCEPTION 'learning attempt expired' USING ERRCODE='M0001'; END IF;
  IF pr.state='expired' AND pr.terminal_at<pr.expires_at THEN RAISE EXCEPTION 'early expiry'; END IF;
  RETURN NULL;
 END IF;
 SELECT * INTO p FROM assessment_attempts WHERE id=NEW.id;
 IF NOT learning_dependencies_match('assessment',p.id,p.owner_user_id,p.seal,false) THEN RAISE EXCEPTION 'incomplete assessment dependencies'; END IF;
 IF NOT p.sealed OR NOT learning_seal_proof(p.seal,p.knowledge_id,p.knowledge_version,p.knowledge_sha256,p.knowledge_publication_id,p.question_publication_id,'assessment') OR p.seal#>>'{body,mode}'<>p.mode OR p.seal#>'{body,core}' IS DISTINCT FROM p.core OR p.seal#>>'{body,seed}'<>p.seed OR p.seal#>'{body,blueprint}' IS DISTINCT FROM jsonb_build_object('id',p.blueprint_id,'version',p.blueprint_version,'sha256',p.blueprint_sha256) THEN RAISE EXCEPTION 'invalid assessment basis'; END IF;
 SELECT coalesce(jsonb_agg(binding ORDER BY position),'[]') INTO actual FROM assessment_items WHERE attempt_id=p.id;
 IF actual IS DISTINCT FROM p.seal#>'{body,items}' THEN RAISE EXCEPTION 'fixed five items mismatch'; END IF;
 SELECT count(*) INTO cnt FROM assessment_answers WHERE attempt_id=p.id;
 IF p.state='submitted' THEN
  IF cnt<>5 THEN RAISE EXCEPTION 'five answers required'; END IF;
  SELECT * INTO r FROM assessment_results WHERE attempt_id=p.id;
  IF NOT FOUND OR NOT r.sealed OR r.owner_user_id<>p.owner_user_id OR r.submitted_at<>p.terminal_at OR (r.progress->'knowledge'=p.seal#>'{body,knowledge}' AND jsonb_typeof(r.progress->'qualificationGranted')='boolean' AND jsonb_typeof(r.progress->'newlyUnlocked')='array') IS NOT TRUE THEN RAISE EXCEPTION 'complete result required'; END IF;
  IF p.terminal_at>=p.expires_at OR clock_timestamp()>=p.expires_at THEN RAISE EXCEPTION 'learning attempt expired' USING ERRCODE='M0001'; END IF;
 ELSE
  IF cnt<>0 OR EXISTS(SELECT 1 FROM assessment_results WHERE attempt_id=p.id) THEN RAISE EXCEPTION 'unsubmitted assessment has answers'; END IF;
  IF p.state='expired' AND p.terminal_at<p.expires_at THEN RAISE EXCEPTION 'early expiry'; END IF;
 END IF;
 RETURN NULL;
END;
$$;
CREATE FUNCTION learning_attempt_child_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE p assessment_attempts;b jsonb;
BEGIN
 IF TG_TABLE_NAME='assessment_results' AND TG_OP='UPDATE' THEN
  IF OLD.sealed OR NOT NEW.sealed OR (to_jsonb(NEW)-'sealed'-'progress') IS DISTINCT FROM (to_jsonb(OLD)-'sealed'-'progress') THEN RAISE EXCEPTION 'immutable original result'; END IF;
  SELECT * INTO p FROM assessment_attempts WHERE id=NEW.attempt_id;
  IF p.state NOT IN ('active','submitted') THEN RAISE EXCEPTION 'result cannot seal abandoned attempt'; END IF;RETURN NEW;
 END IF;
 IF TG_OP<>'INSERT' THEN RAISE EXCEPTION 'immutable attempt child'; END IF;
 SELECT * INTO p FROM assessment_attempts WHERE id=NEW.attempt_id;
 IF TG_TABLE_NAME='assessment_items' THEN
  IF p.sealed THEN RAISE EXCEPTION 'sealed assessment item'; END IF;
 ELSE
  IF p.state<>'active' THEN RAISE EXCEPTION 'terminal assessment child'; END IF;
  IF TG_TABLE_NAME='assessment_answers' THEN
   SELECT body#>'{body,body}' INTO b FROM question_instances WHERE id=NEW.instance_id AND version=NEW.instance_version AND sha256=NEW.instance_sha256;
   IF NOT learning_answer_valid(b,NEW.answer,true) THEN RAISE EXCEPTION 'invalid formal answer'; END IF;
  END IF;
 END IF;
 RETURN NEW;
END;
$$;
CREATE FUNCTION learning_child_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE p assessment_attempts;
BEGIN
 SELECT * INTO p FROM assessment_attempts WHERE id=NEW.attempt_id;
 IF p.state<>'submitted' OR NOT EXISTS(SELECT 1 FROM assessment_results WHERE attempt_id=p.id AND sealed) OR (SELECT count(*) FROM assessment_answers WHERE attempt_id=p.id)<>5 THEN RAISE EXCEPTION 'partial submitted result'; END IF;
 RETURN NULL;
END;
$$;
CREATE FUNCTION learning_record_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'immutable first learning times'; END IF;
 IF (to_jsonb(NEW)-'completed_event_id'-'completed_at') IS DISTINCT FROM (to_jsonb(OLD)-'completed_event_id'-'completed_at') OR OLD.completed_event_id IS NOT NULL THEN RAISE EXCEPTION 'immutable first learning times'; END IF;RETURN NEW;
END;
$$;
CREATE FUNCTION learning_event_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE e learning_events;r learning_records;u jsonb;a jsonb;
BEGIN
 SELECT * INTO e FROM learning_events WHERE id=NEW.id;
 IF NOT learning_dependencies_match('learning-event',e.id,e.owner_user_id,e.seal,true) THEN RAISE EXCEPTION 'incomplete learning dependencies'; END IF;
 IF (e.seal#>'{body,knowledge}'=jsonb_build_object('id',e.knowledge_id,'version',e.knowledge_version,'sha256',e.knowledge_sha256) AND e.seal#>>'{body,knowledgePublicationId}'=e.knowledge_publication_id AND jsonb_typeof(e.seal#>'{body,units}')='array' AND jsonb_typeof(e.seal#>'{body,assets}')='array') IS NOT TRUE OR NOT learning_content_approved(e.knowledge_publication_id,'knowledge',e.knowledge_id,e.knowledge_version,e.knowledge_sha256) THEN RAISE EXCEPTION 'invalid learning source'; END IF;
 FOR u IN SELECT value FROM jsonb_array_elements(e.seal#>'{body,units}') LOOP
  IF NOT learning_content_approved(e.knowledge_publication_id,'unit',u->>'id',(u->>'version')::integer,u->>'sha256') OR NOT EXISTS(SELECT 1 FROM unit_versions WHERE id=u->>'id' AND version=(u->>'version')::integer AND sha256=u->>'sha256' AND knowledge_id=e.knowledge_id AND knowledge_version=e.knowledge_version) THEN RAISE EXCEPTION 'invalid learning unit'; END IF;
 END LOOP;
 FOR a IN SELECT value FROM jsonb_array_elements(e.seal#>'{body,assets}') LOOP
  IF NOT learning_content_approved(e.knowledge_publication_id,'asset',a->>'id',1,a->>'sha256') THEN RAISE EXCEPTION 'invalid learning asset'; END IF;
 END LOOP;
 SELECT * INTO r FROM learning_records WHERE owner_user_id=e.owner_user_id AND knowledge_id=e.knowledge_id AND knowledge_version=e.knowledge_version AND knowledge_sha256=e.knowledge_sha256;
 IF NOT FOUND OR r.started_at>e.recorded_at OR (e.kind='started' AND (r.started_event_id<>e.id OR r.started_at<>e.recorded_at)) THEN RAISE EXCEPTION 'learning event needs explicit start'; END IF;
 RETURN NULL;
END;
$$;
CREATE FUNCTION learning_record_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE e learning_events;
BEGIN
 SELECT * INTO e FROM learning_events WHERE id=NEW.started_event_id AND owner_user_id=NEW.owner_user_id;
 IF NOT FOUND OR e.kind<>'started' OR (e.knowledge_id,e.knowledge_version,e.knowledge_sha256,e.recorded_at) IS DISTINCT FROM (NEW.knowledge_id,NEW.knowledge_version,NEW.knowledge_sha256,NEW.started_at) THEN RAISE EXCEPTION 'invalid first learning event'; END IF;
 IF NEW.completed_event_id IS NOT NULL THEN
  SELECT * INTO e FROM learning_events WHERE id=NEW.completed_event_id AND owner_user_id=NEW.owner_user_id;
  IF NOT FOUND OR e.kind<>'completed' OR (e.knowledge_id,e.knowledge_version,e.knowledge_sha256,e.recorded_at) IS DISTINCT FROM (NEW.knowledge_id,NEW.knowledge_version,NEW.knowledge_sha256,NEW.completed_at) THEN RAISE EXCEPTION 'invalid first completion'; END IF;
 END IF;
 RETURN NULL;
END;
$$;
CREATE FUNCTION learning_path_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE p learning_path_enrollments;pid uuid;
BEGIN
 IF TG_TABLE_NAME='learning_path_nodes' THEN pid:=NEW.enrollment_id;ELSE pid:=NEW.id;END IF;
 SELECT * INTO p FROM learning_path_enrollments WHERE id=pid;
 IF NOT learning_content_approved(p.knowledge_publication_id,'path',p.path_id,p.path_version,p.path_sha256) OR p.total_nodes<>(SELECT count(*) FROM learning_path_nodes WHERE enrollment_id=p.id) OR p.total_nodes<>(SELECT count(*) FROM path_nodes WHERE path_id=p.path_id AND path_version=p.path_version) THEN RAISE EXCEPTION 'fixed route denominator mismatch'; END IF;
 IF EXISTS(SELECT 1 FROM learning_path_nodes n WHERE n.enrollment_id=p.id AND (NOT EXISTS(SELECT 1 FROM path_nodes pn WHERE pn.path_id=p.path_id AND pn.path_version=p.path_version AND pn.position=n.position AND pn.knowledge_id=n.knowledge_id AND pn.knowledge_version=n.knowledge_version) OR NOT learning_content_approved(p.knowledge_publication_id,'knowledge',n.knowledge_id,n.knowledge_version,n.knowledge_sha256))) THEN RAISE EXCEPTION 'fixed route node mismatch'; END IF;
 RETURN NULL;
END;
$$;
CREATE FUNCTION learning_evidence_owner(kind text,eid uuid,uid uuid) RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT CASE kind WHEN 'learning-event' THEN EXISTS(SELECT 1 FROM learning_events WHERE id=eid AND owner_user_id=uid) WHEN 'practice' THEN EXISTS(SELECT 1 FROM practice_attempts WHERE id=eid AND owner_user_id=uid) WHEN 'assessment' THEN EXISTS(SELECT 1 FROM assessment_attempts WHERE id=eid AND owner_user_id=uid) WHEN 'enrollment' THEN EXISTS(SELECT 1 FROM learning_path_enrollments WHERE id=eid AND owner_user_id=uid) ELSE false END;
$$;
CREATE FUNCTION learning_grant_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE p assessment_attempts;e learning_events;r assessment_results;
BEGIN
 IF TG_TABLE_NAME='learning_unlocks' THEN
  IF NOT learning_evidence_owner(NEW.source_kind,NEW.source_id,NEW.owner_user_id) OR NOT learning_content_approved(NEW.knowledge_publication_id,'knowledge',NEW.knowledge_id,NEW.knowledge_version,NEW.knowledge_sha256) THEN RAISE EXCEPTION 'invalid historical unlock source'; END IF;RETURN NULL;
 END IF;
 SELECT * INTO p FROM assessment_attempts WHERE id=NEW.attempt_id AND owner_user_id=NEW.owner_user_id;
 SELECT * INTO r FROM assessment_results WHERE attempt_id=p.id;
 IF p.state<>'submitted' OR r.outcome IS DISTINCT FROM 'passed' OR (p.knowledge_id,p.knowledge_version,p.knowledge_sha256) IS DISTINCT FROM (NEW.knowledge_id,NEW.knowledge_version,NEW.knowledge_sha256) OR (NEW.kind='diagnostic')<>(p.mode='diagnostic') THEN RAISE EXCEPTION 'invalid passing qualification'; END IF;
 IF NEW.kind='normal' THEN
  SELECT * INTO e FROM learning_events WHERE id=NEW.completed_event_id AND owner_user_id=NEW.owner_user_id AND kind='completed';
  IF NOT FOUND OR (e.knowledge_id,e.knowledge_version,e.knowledge_sha256) IS DISTINCT FROM (NEW.knowledge_id,NEW.knowledge_version,NEW.knowledge_sha256) THEN RAISE EXCEPTION 'normal qualification needs completion'; END IF;
 END IF;RETURN NULL;
END;
$$;
CREATE FUNCTION learning_dependency_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE s jsonb;
BEGIN
 IF NOT learning_evidence_owner(NEW.evidence_kind,NEW.evidence_id,NEW.owner_user_id) THEN RAISE EXCEPTION 'foreign evidence owner'; END IF;

 SELECT CASE NEW.evidence_kind WHEN 'learning-event' THEN (SELECT seal FROM learning_events WHERE id=NEW.evidence_id) WHEN 'practice' THEN (SELECT seal FROM practice_attempts WHERE id=NEW.evidence_id) ELSE (SELECT seal FROM assessment_attempts WHERE id=NEW.evidence_id) END INTO s;
 IF NOT EXISTS(SELECT 1 FROM learning_seal_dependencies(s,NEW.evidence_kind='learning-event') d WHERE d.kind=NEW.kind AND d.id=NEW.id AND d.version IS NOT DISTINCT FROM NEW.version AND d.sha256=NEW.sha256) THEN RAISE EXCEPTION 'dependency not in fixed evidence'; END IF;
 IF NOT (CASE NEW.kind WHEN 'knowledge' THEN EXISTS(SELECT 1 FROM knowledge_versions WHERE id=NEW.id AND version=NEW.version AND sha256=NEW.sha256) WHEN 'unit' THEN EXISTS(SELECT 1 FROM unit_versions WHERE id=NEW.id AND version=NEW.version AND sha256=NEW.sha256) WHEN 'asset' THEN EXISTS(SELECT 1 FROM assets WHERE sha256=NEW.sha256) WHEN 'template' THEN EXISTS(SELECT 1 FROM question_templates WHERE id=NEW.id AND version=NEW.version AND sha256=NEW.sha256) WHEN 'instance' THEN EXISTS(SELECT 1 FROM question_instances WHERE id=NEW.id AND version=NEW.version AND sha256=NEW.sha256) WHEN 'blueprint' THEN EXISTS(SELECT 1 FROM question_blueprints WHERE id=NEW.id AND version=NEW.version AND sha256=NEW.sha256) ELSE false END) THEN RAISE EXCEPTION 'inexact evidence dependency'; END IF;RETURN NULL;
END;
$$;
CREATE FUNCTION learning_counter_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'immutable learner exposure history'; END IF;
 IF TG_OP='UPDATE' THEN
  IF NEW.owner_user_id<>OLD.owner_user_id OR NEW.sequence<OLD.sequence THEN RAISE EXCEPTION 'exposure sequence decreased'; END IF;
  IF TG_TABLE_NAME='learner_exposure_state' THEN
   IF NEW.updated_at<OLD.updated_at THEN RAISE EXCEPTION 'exposure clock decreased'; END IF;
  ELSIF (to_jsonb(NEW)-'sequence'-'exposed_at') IS DISTINCT FROM (to_jsonb(OLD)-'sequence'-'exposed_at') OR NEW.exposed_at<OLD.exposed_at THEN RAISE EXCEPTION 'exposure projection changed identity or time'; END IF;
 END IF;
 IF TG_TABLE_NAME='learner_answer_exposures' AND NOT EXISTS(SELECT 1 FROM learner_exposure_state WHERE owner_user_id=NEW.owner_user_id AND sequence>=NEW.sequence) THEN RAISE EXCEPTION 'exposure exceeds account sequence'; END IF;RETURN NEW;
END;
$$;
CREATE FUNCTION learning_view_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' OR (to_jsonb(NEW)-'last_seen_at') IS DISTINCT FROM (to_jsonb(OLD)-'last_seen_at') OR NEW.last_seen_at<OLD.last_seen_at THEN RAISE EXCEPTION 'immutable first question view'; END IF;RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION learning_seal_dependencies(s jsonb,is_event boolean) RETURNS TABLE(kind text,id text,version integer,sha256 text) LANGUAGE sql IMMUTABLE STRICT AS $$
 WITH b AS(SELECT s->'body' v), items AS(SELECT item FROM b CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN is_event THEN '[]'::jsonb ELSE v->'items' END) item),
 units AS(SELECT u FROM b CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN is_event THEN v->'units' ELSE '[]'::jsonb END) u UNION SELECT u FROM items CROSS JOIN LATERAL jsonb_array_elements(item->'units') u),
 assets AS(SELECT a FROM b CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN is_event THEN v->'assets' ELSE '[]'::jsonb END) a UNION SELECT a FROM items CROSS JOIN LATERAL jsonb_array_elements(item->'assets') a)
 SELECT 'knowledge',v#>>'{knowledge,id}',(v#>>'{knowledge,version}')::integer,v#>>'{knowledge,sha256}' FROM b
 UNION SELECT 'blueprint',v#>>'{blueprint,id}',(v#>>'{blueprint,version}')::integer,v#>>'{blueprint,sha256}' FROM b WHERE NOT is_event AND v#>>'{blueprint,id}' IS NOT NULL
 UNION SELECT 'instance',item#>>'{instance,id}',(item#>>'{instance,version}')::integer,item#>>'{instance,sha256}' FROM items
 UNION SELECT 'template',item#>>'{template,id}',(item#>>'{template,version}')::integer,item#>>'{template,sha256}' FROM items WHERE item#>>'{template,id}' IS NOT NULL
 UNION SELECT 'unit',u->>'id',(u->>'version')::integer,u->>'sha256' FROM units
 UNION SELECT 'asset',a->>'id',NULL::integer,a->>'sha256' FROM assets;
$$;
CREATE FUNCTION learning_dependencies_match(ek text,eid uuid,uid uuid,s jsonb,is_event boolean) RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT NOT EXISTS((SELECT * FROM learning_seal_dependencies(s,is_event)) EXCEPT (SELECT kind,id,version,sha256 FROM learning_evidence_dependencies WHERE evidence_kind=ek AND evidence_id=eid AND owner_user_id=uid))
 AND NOT EXISTS((SELECT kind,id,version,sha256 FROM learning_evidence_dependencies WHERE evidence_kind=ek AND evidence_id=eid AND owner_user_id=uid) EXCEPT (SELECT * FROM learning_seal_dependencies(s,is_event)));
$$;
-- +goose StatementEnd

CREATE TRIGGER learning_assessment_guard BEFORE UPDATE OR DELETE ON assessment_attempts FOR EACH ROW EXECUTE FUNCTION learning_attempt_guard();
CREATE TRIGGER learning_practice_guard BEFORE UPDATE OR DELETE ON practice_attempts FOR EACH ROW EXECUTE FUNCTION learning_attempt_guard();
CREATE CONSTRAINT TRIGGER learning_assessment_complete AFTER INSERT OR UPDATE ON assessment_attempts DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION learning_attempt_complete();
CREATE CONSTRAINT TRIGGER learning_practice_complete AFTER INSERT OR UPDATE ON practice_attempts DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION learning_attempt_complete();
CREATE TRIGGER learning_item_guard BEFORE INSERT OR UPDATE OR DELETE ON assessment_items FOR EACH ROW EXECUTE FUNCTION learning_attempt_child_guard();
CREATE TRIGGER learning_answer_guard BEFORE INSERT OR UPDATE OR DELETE ON assessment_answers FOR EACH ROW EXECUTE FUNCTION learning_attempt_child_guard();
CREATE TRIGGER learning_result_guard BEFORE INSERT OR UPDATE OR DELETE ON assessment_results FOR EACH ROW EXECUTE FUNCTION learning_attempt_child_guard();
CREATE CONSTRAINT TRIGGER learning_answer_complete AFTER INSERT ON assessment_answers DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION learning_child_complete();
CREATE CONSTRAINT TRIGGER learning_result_complete AFTER INSERT OR UPDATE ON assessment_results DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION learning_child_complete();
CREATE TRIGGER learning_record_guard BEFORE UPDATE OR DELETE ON learning_records FOR EACH ROW EXECUTE FUNCTION learning_record_guard();
CREATE CONSTRAINT TRIGGER learning_record_complete AFTER INSERT OR UPDATE ON learning_records DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION learning_record_complete();
CREATE TRIGGER learning_event_immutable BEFORE UPDATE OR DELETE ON learning_events FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE CONSTRAINT TRIGGER learning_event_complete AFTER INSERT ON learning_events DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION learning_event_complete();
CREATE TRIGGER learning_enrollment_immutable BEFORE UPDATE OR DELETE ON learning_path_enrollments FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER learning_path_node_immutable BEFORE UPDATE OR DELETE ON learning_path_nodes FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE CONSTRAINT TRIGGER learning_enrollment_complete AFTER INSERT ON learning_path_enrollments DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION learning_path_complete();
CREATE CONSTRAINT TRIGGER learning_path_node_complete AFTER INSERT ON learning_path_nodes DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION learning_path_complete();
CREATE TRIGGER learning_unlock_immutable BEFORE UPDATE OR DELETE ON learning_unlocks FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER learning_qualification_immutable BEFORE UPDATE OR DELETE ON learning_qualification_events FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE CONSTRAINT TRIGGER learning_unlock_source AFTER INSERT ON learning_unlocks DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION learning_grant_complete();
CREATE CONSTRAINT TRIGGER learning_qualification_source AFTER INSERT ON learning_qualification_events DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION learning_grant_complete();
CREATE TRIGGER learning_dependency_immutable BEFORE UPDATE OR DELETE ON learning_evidence_dependencies FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE CONSTRAINT TRIGGER learning_dependency_source AFTER INSERT ON learning_evidence_dependencies DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION learning_dependency_complete();
CREATE TRIGGER learning_counter_guard BEFORE UPDATE OR DELETE ON learner_exposure_state FOR EACH ROW EXECUTE FUNCTION learning_counter_guard();
CREATE TRIGGER learning_exposure_guard BEFORE INSERT OR UPDATE OR DELETE ON learner_answer_exposures FOR EACH ROW EXECUTE FUNCTION learning_counter_guard();
CREATE TRIGGER learning_view_guard BEFORE UPDATE OR DELETE ON learner_question_views FOR EACH ROW EXECUTE FUNCTION learning_view_guard();
CREATE TRIGGER learning_idempotency_immutable BEFORE UPDATE OR DELETE ON learning_idempotency FOR EACH ROW EXECUTE FUNCTION reject_content_update();

-- +goose Down
DROP TABLE learning_idempotency,learning_evidence_dependencies,learning_unlocks,learning_qualification_events,practice_attempts,assessment_results,assessment_answers,assessment_items,assessment_attempts,learner_question_views,learner_answer_exposures,learner_exposure_state,learning_path_nodes,learning_path_enrollments,learning_records,learning_events;
DROP FUNCTION learning_dependencies_match(text,uuid,uuid,jsonb,boolean),learning_seal_dependencies(jsonb,boolean),learning_view_guard(),learning_counter_guard(),learning_dependency_complete(),learning_grant_complete(),learning_evidence_owner(text,uuid,uuid),learning_path_complete(),learning_record_complete(),learning_event_complete(),learning_record_guard(),learning_child_complete(),learning_attempt_child_guard(),learning_attempt_complete(),learning_attempt_guard(),learning_answer_valid(jsonb,jsonb,boolean),learning_seal_proof(jsonb,text,integer,text,text,uuid,text),learning_item_proof(jsonb,jsonb,text,uuid),learning_material_hash(jsonb),learning_content_approved(text,text,text,integer,text);
DROP INDEX learning_coverage_knowledge,learning_candidate_knowledge,learning_path_identity_sha,learning_unit_identity_sha,learning_knowledge_identity_sha;
