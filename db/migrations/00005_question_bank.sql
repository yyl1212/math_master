-- +goose Up
-- 独立题库契约，不修改已有数学内容表及迁移。
CREATE TABLE question_workspaces (
 id uuid PRIMARY KEY CHECK((id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$') IS TRUE),owner_user_id uuid NOT NULL REFERENCES auth_users(id),
 catalogue_version integer NOT NULL REFERENCES catalogue_versions(version),package jsonb NOT NULL CHECK((jsonb_typeof(package)='object' AND octet_length(package::text)<=4194304) IS TRUE),
 source_map jsonb NOT NULL DEFAULT '[]' CHECK((jsonb_typeof(source_map)='array' AND octet_length(source_map::text)<=262144) IS TRUE),
 legacy_unattributed boolean NOT NULL DEFAULT false,base_submission_id uuid,
 revision bigint NOT NULL CHECK((revision>=1) IS TRUE),status text NOT NULL DEFAULT 'editing' CHECK((status IN ('editing','submitted')) IS TRUE),
 gate jsonb NOT NULL DEFAULT '{}',created_at timestamptz NOT NULL DEFAULT clock_timestamp(),updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX question_workspaces_owner ON question_workspaces(owner_user_id,created_at,id);
CREATE TABLE question_workspace_authors (workspace_id uuid NOT NULL REFERENCES question_workspaces(id),user_id uuid NOT NULL REFERENCES auth_users(id),PRIMARY KEY(workspace_id,user_id));
CREATE TABLE question_packages (
 id text NOT NULL CHECK((id ~ '^[a-z][a-z0-9]*(-[a-z0-9]+)*$' AND length(id)<=64) IS TRUE),version integer NOT NULL CHECK((version>0) IS TRUE),sha256 text NOT NULL CHECK((sha256 ~ '^[0-9a-f]{64}$') IS TRUE),body jsonb NOT NULL,body_bytes bytea NOT NULL,
 CHECK((encode(sha256(body_bytes),'hex')=sha256) IS TRUE), CHECK((convert_from(body_bytes,'UTF8')::jsonb=body) IS TRUE), CHECK((jsonb_typeof(body)='object') IS TRUE),
 catalogue_version integer NOT NULL,catalogue_sha256 text NOT NULL,
 instance_identities jsonb NOT NULL CHECK((jsonb_typeof(instance_identities)='array') IS TRUE),source_map jsonb NOT NULL DEFAULT '[]' CHECK((jsonb_typeof(source_map)='array') IS TRUE),
 author_ids jsonb NOT NULL DEFAULT '[]' CHECK((jsonb_typeof(author_ids)='array') IS TRUE),legacy_unattributed boolean NOT NULL DEFAULT true,
 sealed boolean NOT NULL DEFAULT false,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(id,version),UNIQUE(id,version,sha256),FOREIGN KEY(catalogue_version,catalogue_sha256) REFERENCES catalogue_versions(version,sha256),
 CHECK((octet_length(body_bytes)<=2097152) IS TRUE),CHECK((body->>'purpose'='question-package-v1') IS TRUE),
 CHECK((jsonb_typeof(body#>'{body,templates}')='array' AND jsonb_typeof(body#>'{body,fixedQuestions}')='array' AND jsonb_typeof(body#>'{body,blueprints}')='array') IS TRUE),
 CHECK((body->'body'->>'kind'='question-bank' AND body->'body'->>'schemaVersion'='1') IS TRUE),
 CHECK((body->'body'->>'id'=id AND (body->'body'->>'version')::integer=version) IS TRUE)
);
CREATE TABLE question_templates (
 id text NOT NULL CHECK((id ~ '^[a-z][a-z0-9]*(-[a-z0-9]+)*$' AND length(id)<=64) IS TRUE),version integer NOT NULL CHECK((version>0) IS TRUE),sha256 text NOT NULL CHECK((sha256 ~ '^[0-9a-f]{64}$') IS TRUE),body jsonb NOT NULL,body_bytes bytea NOT NULL,
 CHECK((encode(sha256(body_bytes),'hex')=sha256) IS TRUE), CHECK((convert_from(body_bytes,'UTF8')::jsonb=body) IS TRUE), CHECK((jsonb_typeof(body)='object') IS TRUE),
 knowledge_id text NOT NULL,knowledge_version integer NOT NULL,
 PRIMARY KEY(id,version),UNIQUE(id,version,sha256),FOREIGN KEY(knowledge_id,knowledge_version) REFERENCES knowledge_versions(id,version),
 CHECK((octet_length(body_bytes)<=2097152) IS TRUE),CHECK((body->>'purpose'='question-template-v1') IS TRUE),
 CHECK((body->'body'->>'id'=id AND (body->'body'->>'version')::integer=version) IS TRUE),
 CHECK((body->'body'->'knowledge'->>'id'=knowledge_id AND (body->'body'->'knowledge'->>'version')::integer=knowledge_version) IS TRUE)
);
CREATE TABLE question_instances (
 id text NOT NULL CHECK(((id ~ '^[a-z][a-z0-9]*(-[a-z0-9]+)*$' AND length(id)<=64) OR id ~ '^qi-[0-9a-f]{64}$') IS TRUE),
 version integer NOT NULL CHECK((version>0) IS TRUE),sha256 text NOT NULL CHECK((sha256 ~ '^[0-9a-f]{64}$') IS TRUE),body jsonb NOT NULL,body_bytes bytea NOT NULL,
 CHECK((encode(sha256(body_bytes),'hex')=sha256) IS TRUE), CHECK((convert_from(body_bytes,'UTF8')::jsonb=body) IS TRUE), CHECK((jsonb_typeof(body)='object') IS TRUE),origin text NOT NULL CHECK((origin IN ('fixed','template')) IS TRUE),
 knowledge_id text NOT NULL,knowledge_version integer NOT NULL,
 template_id text,template_version integer,template_sha256 text,generator_version integer,verifier_version integer,
 parameters jsonb NOT NULL CHECK((jsonb_typeof(parameters)='array') IS TRUE),parameter_bytes bytea NOT NULL,parameter_sha256 text NOT NULL CHECK((parameter_sha256 ~ '^[0-9a-f]{64}$') IS TRUE),
 sealed boolean NOT NULL DEFAULT false,
 PRIMARY KEY(id,version),UNIQUE(id,version,sha256),UNIQUE(template_id,template_version,template_sha256,parameter_sha256),
 FOREIGN KEY(knowledge_id,knowledge_version) REFERENCES knowledge_versions(id,version),FOREIGN KEY(template_id,template_version,template_sha256) REFERENCES question_templates(id,version,sha256),
 CHECK((octet_length(body_bytes)<=4194304) IS TRUE),CHECK((body->>'purpose'='question-instance-body-v1') IS TRUE),
 CHECK((body->'body'->'identity'->>'id'=id AND (body->'body'->'identity'->>'version')::integer=version AND body->'body'->>'origin'=origin) IS TRUE),
 CHECK((body->'body'->'body'->'knowledge'->>'id'=knowledge_id AND (body->'body'->'body'->'knowledge'->>'version')::integer=knowledge_version) IS TRUE),
 CHECK((parameters=convert_from(parameter_bytes,'UTF8')::jsonb AND encode(sha256(parameter_bytes),'hex')=parameter_sha256 AND parameters=body->'body'->'parameters') IS TRUE),
 CHECK(((origin='fixed' AND template_id IS NULL AND template_version IS NULL AND template_sha256 IS NULL AND generator_version IS NULL AND verifier_version IS NULL AND parameters='[]' AND body->'body'->'template'='null' AND body->'body'->'generatorVersion'='null' AND body->'body'->'verifierVersion'='null') OR
 (origin='template' AND id ~ '^qi-[0-9a-f]{64}$' AND version=1 AND template_id IS NOT NULL AND template_version IS NOT NULL AND template_sha256 IS NOT NULL AND generator_version=1 AND verifier_version=1 AND
 body->'body'->'template'->>'id'=template_id AND (body->'body'->'template'->>'version')::integer=template_version AND body->'body'->'template'->>'sha256'=template_sha256 AND (body->'body'->>'generatorVersion')::integer=generator_version AND (body->'body'->>'verifierVersion')::integer=verifier_version)) IS TRUE)
);
CREATE TABLE question_blueprints (
 id text NOT NULL CHECK((id ~ '^[a-z][a-z0-9]*(-[a-z0-9]+)*$' AND length(id)<=64) IS TRUE),version integer NOT NULL CHECK((version>0) IS TRUE),sha256 text NOT NULL CHECK((sha256 ~ '^[0-9a-f]{64}$') IS TRUE),body jsonb NOT NULL,body_bytes bytea NOT NULL,
 CHECK((encode(sha256(body_bytes),'hex')=sha256) IS TRUE), CHECK((convert_from(body_bytes,'UTF8')::jsonb=body) IS TRUE), CHECK((jsonb_typeof(body)='object') IS TRUE),
 knowledge_id text NOT NULL,knowledge_version integer NOT NULL,sealed boolean NOT NULL DEFAULT false,
 PRIMARY KEY(id,version),UNIQUE(id,version,sha256),FOREIGN KEY(knowledge_id,knowledge_version) REFERENCES knowledge_versions(id,version),
 CHECK((octet_length(body_bytes)<=2097152) IS TRUE),CHECK((body->>'purpose'='question-blueprint-v1') IS TRUE),
 CHECK((body->'body'->>'id'=id AND (body->'body'->>'version')::integer=version) IS TRUE),
 CHECK((body->'body'->'knowledge'->>'id'=knowledge_id AND (body->'body'->'knowledge'->>'version')::integer=knowledge_version) IS TRUE),
 CHECK((body->'body'->>'ruleVersion'='1' AND body->'body'->>'questionCount'='5' AND body->'body'->>'passCount'='4') IS TRUE)
);
CREATE TABLE question_instance_coverage (
 instance_id text NOT NULL,instance_version integer NOT NULL,knowledge_id text NOT NULL,knowledge_version integer NOT NULL,objective_index integer NOT NULL CHECK((objective_index>=0) IS TRUE),
 PRIMARY KEY(instance_id,instance_version,knowledge_id,knowledge_version,objective_index),
 FOREIGN KEY(instance_id,instance_version) REFERENCES question_instances(id,version),FOREIGN KEY(knowledge_id,knowledge_version) REFERENCES knowledge_versions(id,version)
);
CREATE TABLE question_blueprint_sources (
 blueprint_id text NOT NULL,blueprint_version integer NOT NULL,kind text NOT NULL CHECK((kind IN ('template','fixed')) IS TRUE),id text NOT NULL,version integer NOT NULL CHECK((version>0) IS TRUE),template_id text,instance_id text,
 PRIMARY KEY(blueprint_id,blueprint_version,kind,id),FOREIGN KEY(blueprint_id,blueprint_version) REFERENCES question_blueprints(id,version),
 FOREIGN KEY(template_id,version) REFERENCES question_templates(id,version),FOREIGN KEY(instance_id,version) REFERENCES question_instances(id,version),
 CHECK(((kind='template' AND template_id=id AND template_id IS NOT NULL AND instance_id IS NULL) OR (kind='fixed' AND instance_id=id AND instance_id IS NOT NULL AND template_id IS NULL)) IS TRUE)
);
CREATE TABLE question_submissions (
 id uuid PRIMARY KEY CHECK((id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$') IS TRUE),workspace_id uuid NOT NULL REFERENCES question_workspaces(id),owner_user_id uuid NOT NULL REFERENCES auth_users(id),revision bigint NOT NULL CHECK((revision>=1) IS TRUE),
 package_id text NOT NULL,package_version integer NOT NULL,package_sha256 text NOT NULL,catalogue_version integer NOT NULL,catalogue_sha256 text NOT NULL,
 frozen_body jsonb NOT NULL,frozen_bytes bytea NOT NULL CHECK((octet_length(frozen_bytes)<=4194304) IS TRUE),frozen_digest text NOT NULL CHECK((frozen_digest ~ '^[0-9a-f]{64}$') IS TRUE),gate jsonb NOT NULL,
 status text NOT NULL DEFAULT 'pending' CHECK((status IN ('pending','approved','returned')) IS TRUE),sealed boolean NOT NULL DEFAULT false,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 UNIQUE(workspace_id,revision),FOREIGN KEY(package_id,package_version,package_sha256) REFERENCES question_packages(id,version,sha256),FOREIGN KEY(catalogue_version,catalogue_sha256) REFERENCES catalogue_versions(version,sha256),
 CHECK((encode(sha256(frozen_bytes),'hex')=frozen_digest AND convert_from(frozen_bytes,'UTF8')::jsonb=frozen_body) IS TRUE),CHECK((frozen_body->>'purpose'='question-submission-v1') IS TRUE),
 CHECK((frozen_body#>>'{body,body,catalogueVersion}'=catalogue_version::text AND frozen_body#>>'{body,body,catalogueSha256}'=catalogue_sha256) IS TRUE),
 CHECK((frozen_body#>>'{body,body,questionPackage,id}'=package_id AND frozen_body#>>'{body,body,questionPackage,version}'=package_version::text) IS TRUE),
 CHECK((frozen_body#>>'{body,body,frozenDigest}'='') IS TRUE),
 CHECK((jsonb_typeof(frozen_body#>'{body,instances}')='array' AND jsonb_typeof(frozen_body#>'{body,body,authorIds}')='array' AND jsonb_typeof(frozen_body#>'{body,body,sourceMap}')='array' AND jsonb_typeof(frozen_body#>'{body,body,resolved}')='array' AND jsonb_typeof(frozen_body#>'{body,body,objectives}')='array' AND jsonb_typeof(frozen_body#>'{body,body,generation}')='array' AND jsonb_typeof(frozen_body#>'{body,body,instanceIdentities}')='array' AND jsonb_typeof(frozen_body#>'{body,body,coverage}')='array' AND jsonb_typeof(frozen_body#>'{body,body,generatorVersions}')='array' AND jsonb_typeof(frozen_body#>'{body,body,verifierVersions}')='array') IS TRUE)
);
ALTER TABLE question_workspaces ADD FOREIGN KEY(base_submission_id) REFERENCES question_submissions(id);
CREATE INDEX question_submissions_status ON question_submissions(status,created_at,id);
CREATE TABLE question_submission_authors (submission_id uuid NOT NULL REFERENCES question_submissions(id),user_id uuid NOT NULL REFERENCES auth_users(id),PRIMARY KEY(submission_id,user_id));
CREATE TABLE question_submission_members (
 submission_id uuid NOT NULL REFERENCES question_submissions(id),kind text NOT NULL CHECK((kind IN ('template','instance','blueprint')) IS TRUE),id text NOT NULL,version integer NOT NULL,sha256 text NOT NULL,
 template_id text,instance_id text,blueprint_id text,PRIMARY KEY(submission_id,kind,id),
 FOREIGN KEY(template_id,version,sha256) REFERENCES question_templates(id,version,sha256),FOREIGN KEY(instance_id,version,sha256) REFERENCES question_instances(id,version,sha256),FOREIGN KEY(blueprint_id,version,sha256) REFERENCES question_blueprints(id,version,sha256),
 CHECK(((kind='template' AND template_id IS NOT NULL AND template_id=id AND instance_id IS NULL AND blueprint_id IS NULL) OR (kind='instance' AND instance_id IS NOT NULL AND instance_id=id AND template_id IS NULL AND blueprint_id IS NULL) OR (kind='blueprint' AND blueprint_id IS NOT NULL AND blueprint_id=id AND template_id IS NULL AND instance_id IS NULL)) IS TRUE)
);
CREATE TABLE question_review_decisions (
 id uuid PRIMARY KEY CHECK((id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$') IS TRUE),submission_id uuid NOT NULL UNIQUE REFERENCES question_submissions(id),reviewer_user_id uuid NOT NULL REFERENCES auth_users(id),
 frozen_digest text NOT NULL CHECK((frozen_digest ~ '^[0-9a-f]{64}$') IS TRUE),decision text NOT NULL CHECK((decision IN ('approve','return')) IS TRUE),
 checks jsonb NOT NULL,independence_note text NOT NULL DEFAULT '',generation_note text NOT NULL DEFAULT '',note text NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 CHECK((char_length(note) BETWEEN 10 AND 1000 AND note ~ '[^[:space:]]' AND octet_length(note)<=3000) IS TRUE),
 CHECK((decision='return' OR (char_length(independence_note) BETWEEN 10 AND 1000 AND independence_note ~ '[^[:space:]]' AND octet_length(independence_note)<=3000 AND checks @> '{"mathematics":true,"explanations":true,"objectives":true,"sources":true,"illustrations":true,"generation":true}')) IS TRUE),
 CHECK((octet_length(generation_note)<=3000 AND char_length(generation_note)<=1000) IS TRUE)
);
CREATE TABLE question_publications (
 id uuid PRIMARY KEY CHECK((id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$') IS TRUE),base_knowledge_head text REFERENCES publication_snapshots(id),base_question_head uuid REFERENCES question_publications(id),
 catalogue_version integer NOT NULL,catalogue_sha256 text NOT NULL,manifest jsonb NOT NULL,manifest_bytes bytea NOT NULL CHECK((octet_length(manifest_bytes)<=8388608) IS TRUE),
 manifest_sha256 text NOT NULL CHECK((manifest_sha256 ~ '^[0-9a-f]{64}$') IS TRUE),diff jsonb NOT NULL,changes jsonb NOT NULL CHECK((jsonb_typeof(changes)='array') IS TRUE),
 creator_user_id uuid NOT NULL REFERENCES auth_users(id),status text NOT NULL DEFAULT 'prepared' CHECK((status IN ('prepared','published')) IS TRUE),sealed boolean NOT NULL DEFAULT false,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 FOREIGN KEY(catalogue_version,catalogue_sha256) REFERENCES catalogue_versions(version,sha256),
 CHECK((encode(sha256(manifest_bytes),'hex')=manifest_sha256 AND convert_from(manifest_bytes,'UTF8')::jsonb=manifest) IS TRUE),CHECK((manifest->>'purpose'='question-manifest-v1') IS TRUE),
 CHECK((manifest#>>'{body,catalogueVersion}'=catalogue_version::text AND manifest#>>'{body,catalogueSha256}'=catalogue_sha256) IS TRUE),
 CHECK(((manifest#>>'{body,baseKnowledgeHead}') IS NOT DISTINCT FROM base_knowledge_head AND (manifest#>>'{body,baseQuestionHead}') IS NOT DISTINCT FROM base_question_head::text) IS TRUE)
);
CREATE TABLE question_publication_members (
 publication_id uuid NOT NULL REFERENCES question_publications(id),kind text NOT NULL CHECK((kind IN ('template','instance','blueprint')) IS TRUE),id text NOT NULL,version integer NOT NULL,sha256 text NOT NULL,
 package_id text NOT NULL,package_version integer NOT NULL,template_id text,instance_id text,blueprint_id text,
 submission_id uuid NOT NULL REFERENCES question_submissions(id),review_id uuid NOT NULL REFERENCES question_review_decisions(id),evidence jsonb NOT NULL,
 PRIMARY KEY(publication_id,kind,id),FOREIGN KEY(package_id,package_version) REFERENCES question_packages(id,version),
 FOREIGN KEY(template_id,version,sha256) REFERENCES question_templates(id,version,sha256),FOREIGN KEY(instance_id,version,sha256) REFERENCES question_instances(id,version,sha256),FOREIGN KEY(blueprint_id,version,sha256) REFERENCES question_blueprints(id,version,sha256),
 CHECK(((kind='template' AND template_id IS NOT NULL AND template_id=id AND instance_id IS NULL AND blueprint_id IS NULL) OR (kind='instance' AND instance_id IS NOT NULL AND instance_id=id AND template_id IS NULL AND blueprint_id IS NULL) OR (kind='blueprint' AND blueprint_id IS NOT NULL AND blueprint_id=id AND template_id IS NULL AND instance_id IS NULL)) IS TRUE)
);
CREATE TABLE question_heads (singleton boolean PRIMARY KEY DEFAULT true CHECK((singleton) IS TRUE),publication_id uuid NOT NULL REFERENCES question_publications(id));
CREATE TABLE question_withdrawals (
 id uuid PRIMARY KEY CHECK((id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$') IS TRUE),kind text NOT NULL CHECK((kind IN ('template','instance','blueprint')) IS TRUE),target_id text NOT NULL,target_version integer NOT NULL CHECK((target_version>0) IS TRUE),sha256 text NOT NULL CHECK((sha256 ~ '^[0-9a-f]{64}$') IS TRUE),
 actor_user_id uuid NOT NULL REFERENCES auth_users(id),reason text NOT NULL,request_id text NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),UNIQUE(kind,target_id,target_version),
 CHECK((char_length(reason) BETWEEN 10 AND 1000 AND reason ~ '[^[:space:]]' AND octet_length(reason)<=3000) IS TRUE)
);
CREATE TABLE question_events (
 id uuid PRIMARY KEY CHECK((id::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$') IS TRUE),actor_user_id uuid NOT NULL REFERENCES auth_users(id),action text NOT NULL,object_kind text NOT NULL,object_id text NOT NULL,
 before_digest text,after_digest text,before_state text,after_state text,reason text NOT NULL DEFAULT '',request_id text NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),CHECK((octet_length(reason)<=3000) IS TRUE)
);
CREATE TABLE question_idempotency (
 actor_user_id uuid NOT NULL REFERENCES auth_users(id),route text NOT NULL CHECK((route IN ('createDraft','saveDraft','adoptDraft','submitDraft','reviseSubmission','decideReview','prepareRelease','activateRelease','withdrawVersion')) IS TRUE),
 key uuid NOT NULL CHECK((key::text ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$') IS TRUE),request_sha256 text NOT NULL CHECK((request_sha256 ~ '^[0-9a-f]{64}$') IS TRUE),
 result_bytes bytea NOT NULL CHECK((octet_length(result_bytes)<=4194304) IS TRUE),created_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(actor_user_id,route,key),CHECK((jsonb_typeof(convert_from(result_bytes,'UTF8')::jsonb) IS NOT NULL) IS TRUE)
);
-- +goose StatementBegin
CREATE FUNCTION question_seal_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' OR OLD.sealed OR NOT NEW.sealed OR (to_jsonb(NEW)-'sealed') IS DISTINCT FROM (to_jsonb(OLD)-'sealed') THEN RAISE EXCEPTION 'immutable question record'; END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION question_relation_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE frozen boolean;
BEGIN
 IF TG_OP<>'INSERT' THEN RAISE EXCEPTION 'immutable question relation'; END IF;
 IF TG_TABLE_NAME='question_instance_coverage' THEN SELECT sealed INTO frozen FROM question_instances WHERE id=NEW.instance_id AND version=NEW.instance_version FOR SHARE;
 ELSIF TG_TABLE_NAME='question_blueprint_sources' THEN SELECT sealed INTO frozen FROM question_blueprints WHERE id=NEW.blueprint_id AND version=NEW.blueprint_version FOR SHARE;
 ELSIF TG_TABLE_NAME IN ('question_submission_authors','question_submission_members') THEN SELECT sealed INTO frozen FROM question_submissions WHERE id=NEW.submission_id FOR SHARE;
 ELSIF TG_TABLE_NAME='question_publication_members' THEN SELECT sealed INTO frozen FROM question_publications WHERE id=NEW.publication_id FOR SHARE;
 ELSE SELECT status='submitted' INTO frozen FROM question_workspaces WHERE id=NEW.workspace_id FOR SHARE;
 END IF;
 IF frozen THEN RAISE EXCEPTION 'sealed question relation'; END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION question_fixed_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE s record; expected jsonb; actual jsonb;
BEGIN
 IF TG_TABLE_NAME='question_instances' THEN
  SELECT * INTO s FROM question_instances WHERE id=NEW.id AND version=NEW.version;
  IF NOT s.sealed THEN RAISE EXCEPTION 'unsealed instance'; END IF;
  SELECT coalesce(jsonb_agg(jsonb_build_object('knowledge',jsonb_build_object('id',knowledge_id,'version',knowledge_version),'index',objective_index) ORDER BY knowledge_id,knowledge_version,objective_index),'[]') INTO actual FROM question_instance_coverage WHERE instance_id=s.id AND instance_version=s.version;
  SELECT coalesce(jsonb_agg(jsonb_build_object('knowledge',c->'knowledge','index',i) ORDER BY c->'knowledge'->>'id',(c->'knowledge'->>'version')::integer,i),'[]') INTO expected FROM jsonb_array_elements(s.body#>'{body,body,coverage}') c CROSS JOIN LATERAL jsonb_array_elements(c->'objectiveIndices') i;
  IF actual<>expected THEN RAISE EXCEPTION 'fixed coverage mismatch'; END IF;
 ELSIF TG_TABLE_NAME='question_blueprints' THEN
  SELECT * INTO s FROM question_blueprints WHERE id=NEW.id AND version=NEW.version;
  IF NOT s.sealed THEN RAISE EXCEPTION 'unsealed blueprint'; END IF;
  SELECT coalesce(jsonb_agg(jsonb_build_object('kind',kind,'ref',jsonb_build_object('id',id,'version',version)) ORDER BY kind,id,version),'[]') INTO actual FROM question_blueprint_sources WHERE blueprint_id=s.id AND blueprint_version=s.version;
  SELECT coalesce(jsonb_agg(v ORDER BY v->>'kind',v->'ref'->>'id',(v->'ref'->>'version')::integer),'[]') INTO expected FROM jsonb_array_elements(s.body#>'{body,sources}') v;
  IF actual<>expected THEN RAISE EXCEPTION 'fixed blueprint source mismatch'; END IF;
  IF EXISTS(SELECT 1 FROM question_blueprint_sources b JOIN question_instances i ON b.kind='fixed' AND i.id=b.id AND i.version=b.version WHERE b.blueprint_id=s.id AND b.blueprint_version=s.version AND (i.origin<>'fixed' OR NOT i.sealed)) THEN RAISE EXCEPTION 'invalid fixed source'; END IF;
 ELSE
  SELECT * INTO s FROM question_packages WHERE id=NEW.id AND version=NEW.version;
  IF NOT s.sealed THEN RAISE EXCEPTION 'unsealed package'; END IF;
  IF EXISTS(SELECT 1 FROM jsonb_array_elements(s.body#>'{body,templates}') v LEFT JOIN question_templates t ON t.id=v->>'id' AND t.version=(v->>'version')::integer WHERE t.id IS NULL OR t.body->'body'<>v) OR
   EXISTS(SELECT 1 FROM jsonb_array_elements(s.body#>'{body,blueprints}') v LEFT JOIN question_blueprints b ON b.id=v->>'id' AND b.version=(v->>'version')::integer WHERE b.id IS NULL OR NOT b.sealed OR b.body->'body'<>v) OR
   EXISTS(SELECT 1 FROM jsonb_array_elements(s.instance_identities) v LEFT JOIN question_instances i ON i.id=v->>'id' AND i.version=(v->>'version')::integer AND i.sha256=v->>'sha256' WHERE i.id IS NULL OR NOT i.sealed) THEN RAISE EXCEPTION 'fixed package member mismatch'; END IF;
  IF EXISTS(SELECT 1 FROM jsonb_array_elements(s.body#>'{body,fixedQuestions}') v LEFT JOIN question_instances i ON i.id=v->>'id' AND i.version=(v->>'version')::integer WHERE i.id IS NULL OR i.origin<>'fixed' OR i.body#>'{body,body}'<>v->'body' OR NOT s.instance_identities @> jsonb_build_array(jsonb_build_object('id',i.id,'version',i.version,'sha256',i.sha256))) THEN RAISE EXCEPTION 'fixed package question mismatch'; END IF;
 END IF;
 RETURN NULL;
END $$;
CREATE FUNCTION question_workspace_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'immutable workspace'; END IF;
 IF NEW.id<>OLD.id OR NEW.owner_user_id<>OLD.owner_user_id OR NEW.created_at<>OLD.created_at OR NEW.base_submission_id IS DISTINCT FROM OLD.base_submission_id OR (OLD.legacy_unattributed AND NOT NEW.legacy_unattributed) THEN RAISE EXCEPTION 'workspace provenance mismatch'; END IF;
 IF OLD.status='submitted' THEN
  IF NEW.status<>'editing' OR NEW.revision<>OLD.revision+1 OR (to_jsonb(NEW)-'status'-'revision'-'updated_at') IS DISTINCT FROM (to_jsonb(OLD)-'status'-'revision'-'updated_at') OR NOT EXISTS(SELECT 1 FROM question_submissions WHERE workspace_id=OLD.id AND revision=OLD.revision AND status='returned') THEN RAISE EXCEPTION 'invalid returned workspace'; END IF;
 ELSIF NEW.status='submitted' THEN
  IF NEW.revision<>OLD.revision OR (to_jsonb(NEW)-'status'-'updated_at') IS DISTINCT FROM (to_jsonb(OLD)-'status'-'updated_at') THEN RAISE EXCEPTION 'invalid workspace submission'; END IF;
 ELSE
  IF NEW.revision NOT IN (OLD.revision,OLD.revision+1) OR (NEW.revision=OLD.revision AND (to_jsonb(NEW)-'gate'-'updated_at') IS DISTINCT FROM (to_jsonb(OLD)-'gate'-'updated_at')) THEN RAISE EXCEPTION 'invalid workspace revision'; END IF;
 END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION question_workspace_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE w question_workspaces;
BEGIN
 SELECT * INTO w FROM question_workspaces WHERE id=NEW.id;
 IF w.status='submitted' AND NOT EXISTS(SELECT 1 FROM question_submissions WHERE workspace_id=w.id AND revision=w.revision AND owner_user_id=w.owner_user_id AND sealed AND status IN ('pending','approved')) THEN RAISE EXCEPTION 'incomplete workspace submission'; END IF;
 RETURN NULL;
END $$;
CREATE FUNCTION question_submission_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' OR (to_jsonb(NEW)-'status'-'sealed') IS DISTINCT FROM (to_jsonb(OLD)-'status'-'sealed') THEN RAISE EXCEPTION 'immutable frozen submission'; END IF;
 IF OLD.sealed THEN IF NOT NEW.sealed OR OLD.status<>'pending' OR NEW.status NOT IN ('approved','returned') THEN RAISE EXCEPTION 'final submission'; END IF;
 ELSE IF NOT NEW.sealed OR NEW.status<>'pending' THEN RAISE EXCEPTION 'invalid seal'; END IF; END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION question_submission_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE s question_submissions; p question_packages; expected jsonb; actual jsonb;
BEGIN
 SELECT * INTO s FROM question_submissions WHERE id=NEW.id;
 SELECT * INTO p FROM question_packages WHERE id=s.package_id AND version=s.package_version;
 IF NOT s.sealed OR NOT p.sealed OR s.frozen_body#>'{body,body,questionPackage}'<>p.body->'body' THEN RAISE EXCEPTION 'unsealed submission package'; END IF;
 SELECT coalesce(jsonb_agg(user_id::text ORDER BY user_id),'[]') INTO actual FROM question_submission_authors WHERE submission_id=s.id;
 SELECT coalesce(jsonb_agg(v ORDER BY v),'[]') INTO expected FROM jsonb_array_elements(s.frozen_body#>'{body,body,authorIds}') v;
 IF actual<>expected OR NOT actual @> jsonb_build_array(s.owner_user_id::text) OR NOT actual @> p.author_ids OR (p.legacy_unattributed AND s.frozen_body#>>'{body,body,legacyUnattributed}'<>'true') THEN RAISE EXCEPTION 'frozen author mismatch'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('kind',kind,'id',id,'version',version,'sha256',sha256) ORDER BY kind,id),'[]') INTO actual FROM question_submission_members WHERE submission_id=s.id;
 SELECT coalesce(jsonb_agg(v ORDER BY v->>'kind',v->>'id'),'[]') INTO expected FROM (
  SELECT jsonb_build_object('kind','template','id',t.id,'version',t.version,'sha256',t.sha256) v FROM jsonb_array_elements(p.body#>'{body,templates}') j JOIN question_templates t ON t.id=j->>'id' AND t.version=(j->>'version')::integer
  UNION ALL SELECT jsonb_build_object('kind','blueprint','id',b.id,'version',b.version,'sha256',b.sha256) FROM jsonb_array_elements(p.body#>'{body,blueprints}') j JOIN question_blueprints b ON b.id=j->>'id' AND b.version=(j->>'version')::integer
  UNION ALL SELECT j || '{"kind":"instance"}' FROM jsonb_array_elements(p.instance_identities) j
 ) q;
 IF actual<>expected THEN RAISE EXCEPTION 'frozen member mismatch'; END IF;
 IF s.frozen_body#>'{body,body,instanceIdentities}'<>p.instance_identities THEN RAISE EXCEPTION 'frozen instance identity mismatch'; END IF;
 SELECT coalesce(jsonb_agg(v->'identity' ORDER BY v->'identity'->>'id',(v->'identity'->>'version')::integer),'[]') INTO expected FROM jsonb_array_elements(s.frozen_body#>'{body,instances}') v;
 SELECT coalesce(jsonb_agg(v ORDER BY v->>'id',(v->>'version')::integer),'[]') INTO actual FROM jsonb_array_elements(p.instance_identities) v;
 IF expected<>actual THEN RAISE EXCEPTION 'frozen instance count mismatch'; END IF;
 IF EXISTS(SELECT 1 FROM jsonb_array_elements(s.frozen_body#>'{body,instances}') v JOIN question_instances i ON i.id=v->'identity'->>'id' AND i.version=(v->'identity'->>'version')::integer WHERE i.body->'body'<>jsonb_set(v,'{identity}',(v->'identity')-'sha256')) THEN RAISE EXCEPTION 'frozen instance body mismatch'; END IF;
 IF s.status IN ('pending','approved') AND NOT EXISTS(SELECT 1 FROM question_workspaces WHERE id=s.workspace_id AND status='submitted' AND revision=s.revision AND owner_user_id=s.owner_user_id) THEN RAISE EXCEPTION 'workspace submission state mismatch'; END IF;
 IF NOT (s.frozen_body#>'{body,body,sourceMap}') @> p.source_map THEN RAISE EXCEPTION 'frozen source responsibility mismatch'; END IF;
 IF s.status='pending' AND EXISTS(SELECT 1 FROM question_review_decisions WHERE submission_id=s.id) OR s.status<>'pending' AND NOT EXISTS(SELECT 1 FROM question_review_decisions WHERE submission_id=s.id AND frozen_digest=s.frozen_digest AND decision=CASE s.status WHEN 'approved' THEN 'approve' ELSE 'return' END) THEN RAISE EXCEPTION 'review state mismatch'; END IF;
 RETURN NULL;
END $$;
CREATE FUNCTION question_review_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE s question_submissions;
BEGIN
 SELECT * INTO s FROM question_submissions WHERE id=NEW.submission_id FOR SHARE;
 IF NOT s.sealed OR s.status<>'pending' OR s.frozen_digest<>NEW.frozen_digest THEN RAISE EXCEPTION 'invalid review target'; END IF;
 IF NEW.decision='approve' AND EXISTS(SELECT 1 FROM question_submission_authors WHERE submission_id=s.id AND user_id=NEW.reviewer_user_id) THEN RAISE EXCEPTION 'author cannot approve'; END IF;
 IF NEW.decision='approve' AND jsonb_array_length(s.frozen_body#>'{body,body,questionPackage,templates}')=0 AND NOT (char_length(NEW.generation_note) BETWEEN 10 AND 1000 AND NEW.generation_note ~ '[^[:space:]]') THEN RAISE EXCEPTION 'generation note required'; END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION question_review_complete() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM question_submissions WHERE id=NEW.submission_id AND sealed AND frozen_digest=NEW.frozen_digest AND status=CASE NEW.decision WHEN 'approve' THEN 'approved' ELSE 'returned' END) THEN RAISE EXCEPTION 'review state mismatch'; END IF;
 RETURN NULL;
END $$;
CREATE FUNCTION question_publication_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' OR (to_jsonb(NEW)-'status'-'sealed') IS DISTINCT FROM (to_jsonb(OLD)-'status'-'sealed') THEN RAISE EXCEPTION 'immutable question manifest'; END IF;
 IF OLD.sealed THEN IF NOT NEW.sealed OR OLD.status<>'prepared' OR NEW.status<>'published' THEN RAISE EXCEPTION 'immutable published state'; END IF;
 ELSE IF NOT NEW.sealed OR OLD.status<>NEW.status THEN RAISE EXCEPTION 'invalid manifest seal'; END IF; END IF;
 RETURN NEW;
END $$;
CREATE FUNCTION question_publication_complete() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE p question_publications; actual jsonb; expected jsonb;
BEGIN
 SELECT * INTO p FROM question_publications WHERE id=NEW.id;
 IF NOT p.sealed THEN RAISE EXCEPTION 'unsealed manifest'; END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('identity',jsonb_build_object('kind',kind,'id',id,'version',version,'sha256',sha256,'packageId',package_id,'packageVersion',package_version),'evidence',evidence) ORDER BY kind,id),'[]') INTO actual FROM question_publication_members WHERE publication_id=p.id;
 SELECT coalesce(jsonb_agg(v ORDER BY v->'identity'->>'kind',v->'identity'->>'id'),'[]') INTO expected FROM jsonb_array_elements(p.manifest#>'{body,members}') v;
 IF actual<>expected THEN RAISE EXCEPTION 'manifest member mismatch'; END IF;
 IF EXISTS(SELECT 1 FROM question_publication_members m JOIN question_submissions s ON s.id=m.submission_id JOIN question_review_decisions r ON r.id=m.review_id WHERE m.publication_id=p.id AND (s.status<>'approved' OR r.submission_id<>s.id OR r.decision<>'approve' OR r.frozen_digest<>s.frozen_digest OR NOT EXISTS(SELECT 1 FROM question_submission_members sm WHERE sm.submission_id=s.id AND sm.kind=m.kind AND sm.id=m.id AND sm.version=m.version AND sm.sha256=m.sha256))) THEN RAISE EXCEPTION 'invalid approval evidence'; END IF;
 RETURN NULL;
END $$;
CREATE FUNCTION question_head_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM question_publications WHERE id=NEW.publication_id AND sealed AND status='published') THEN RAISE EXCEPTION 'unpublished head'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER question_packages_immutable BEFORE UPDATE OR DELETE ON question_packages FOR EACH ROW EXECUTE FUNCTION question_seal_guard();
CREATE CONSTRAINT TRIGGER question_packages_complete AFTER INSERT OR UPDATE ON question_packages DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION question_fixed_complete();
CREATE TRIGGER question_instances_immutable BEFORE UPDATE OR DELETE ON question_instances FOR EACH ROW EXECUTE FUNCTION question_seal_guard();
CREATE CONSTRAINT TRIGGER question_instances_complete AFTER INSERT OR UPDATE ON question_instances DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION question_fixed_complete();
CREATE TRIGGER question_blueprints_immutable BEFORE UPDATE OR DELETE ON question_blueprints FOR EACH ROW EXECUTE FUNCTION question_seal_guard();
CREATE CONSTRAINT TRIGGER question_blueprints_complete AFTER INSERT OR UPDATE ON question_blueprints DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION question_fixed_complete();
CREATE TRIGGER question_templates_immutable BEFORE UPDATE OR DELETE ON question_templates FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER question_review_decisions_immutable BEFORE UPDATE OR DELETE ON question_review_decisions FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER question_withdrawals_immutable BEFORE UPDATE OR DELETE ON question_withdrawals FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER question_events_immutable BEFORE UPDATE OR DELETE ON question_events FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER question_idempotency_immutable BEFORE UPDATE OR DELETE ON question_idempotency FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER question_workspace_authors_immutable BEFORE INSERT OR UPDATE OR DELETE ON question_workspace_authors FOR EACH ROW EXECUTE FUNCTION question_relation_guard();
CREATE TRIGGER question_instance_coverage_immutable BEFORE INSERT OR UPDATE OR DELETE ON question_instance_coverage FOR EACH ROW EXECUTE FUNCTION question_relation_guard();
CREATE TRIGGER question_blueprint_sources_immutable BEFORE INSERT OR UPDATE OR DELETE ON question_blueprint_sources FOR EACH ROW EXECUTE FUNCTION question_relation_guard();
CREATE TRIGGER question_submission_authors_immutable BEFORE INSERT OR UPDATE OR DELETE ON question_submission_authors FOR EACH ROW EXECUTE FUNCTION question_relation_guard();
CREATE TRIGGER question_submission_members_immutable BEFORE INSERT OR UPDATE OR DELETE ON question_submission_members FOR EACH ROW EXECUTE FUNCTION question_relation_guard();
CREATE TRIGGER question_publication_members_immutable BEFORE INSERT OR UPDATE OR DELETE ON question_publication_members FOR EACH ROW EXECUTE FUNCTION question_relation_guard();
CREATE TRIGGER question_workspace_immutable BEFORE UPDATE OR DELETE ON question_workspaces FOR EACH ROW EXECUTE FUNCTION question_workspace_guard();
CREATE CONSTRAINT TRIGGER question_workspace_complete AFTER INSERT OR UPDATE ON question_workspaces DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION question_workspace_complete();
CREATE TRIGGER question_submission_immutable BEFORE UPDATE OR DELETE ON question_submissions FOR EACH ROW EXECUTE FUNCTION question_submission_guard();
CREATE CONSTRAINT TRIGGER question_submission_complete AFTER INSERT OR UPDATE ON question_submissions DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION question_submission_complete();
CREATE TRIGGER question_review_target BEFORE INSERT ON question_review_decisions FOR EACH ROW EXECUTE FUNCTION question_review_guard();
CREATE CONSTRAINT TRIGGER question_review_final AFTER INSERT ON question_review_decisions DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION question_review_complete();
CREATE TRIGGER question_publication_immutable BEFORE UPDATE OR DELETE ON question_publications FOR EACH ROW EXECUTE FUNCTION question_publication_guard();
CREATE CONSTRAINT TRIGGER question_publication_complete AFTER INSERT OR UPDATE ON question_publications DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION question_publication_complete();
CREATE TRIGGER question_head_public BEFORE INSERT OR UPDATE ON question_heads FOR EACH ROW EXECUTE FUNCTION question_head_guard();
-- +goose Down
ALTER TABLE question_workspaces DROP CONSTRAINT question_workspaces_base_submission_id_fkey;
DROP TABLE question_idempotency,question_events,question_withdrawals,question_heads,question_publication_members,question_publications,question_review_decisions,question_submission_members,question_submission_authors,question_submissions,question_blueprint_sources,question_instance_coverage,question_blueprints,question_instances,question_templates,question_packages,question_workspace_authors,question_workspaces;
DROP FUNCTION question_seal_guard(),question_relation_guard(),question_fixed_complete(),question_workspace_guard(),question_workspace_complete(),question_submission_guard(),question_submission_complete(),question_review_guard(),question_review_complete(),question_publication_guard(),question_publication_complete(),question_head_guard();
