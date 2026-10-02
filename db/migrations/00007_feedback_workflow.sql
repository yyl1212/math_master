-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION feedback_shape(j jsonb, keys text[]) RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
 SELECT coalesce(jsonb_typeof(j)='object' AND j ?& keys AND j-keys='{}',false)
$$;
CREATE FUNCTION feedback_uuid(v text) RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
 SELECT coalesce(v ~ '^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$',false)
$$;
CREATE FUNCTION feedback_identity(j jsonb, instance boolean DEFAULT false) RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
 SELECT feedback_shape(j,ARRAY['id','version','sha256']) AND coalesce(jsonb_typeof(j->'id')='string' AND jsonb_typeof(j->'sha256')='string' AND j->>'sha256' ~ '^[0-9a-f]{64}$' AND jsonb_typeof(j->'version')='number' AND j->>'version' ~ '^[1-9][0-9]{0,9}$' AND (j->>'version')::numeric<=2147483647 AND (j->>'id' ~ '^[a-z][a-z0-9-]{0,63}$' OR instance AND j->>'id' ~ '^qi-[0-9a-f]{64}$'),false)
$$;
CREATE FUNCTION feedback_asset(j jsonb) RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
 SELECT feedback_shape(j,ARRAY['id','sha256']) AND coalesce(jsonb_typeof(j->'id')='string' AND j->>'id' ~ '^[a-z][a-z0-9-]{0,63}$' AND jsonb_typeof(j->'sha256')='string' AND j->>'sha256' ~ '^[0-9a-f]{64}$',false)
$$;
CREATE FUNCTION feedback_target_shape(t jsonb,s jsonb) RETURNS boolean LANGUAGE plpgsql IMMUTABLE AS $$
BEGIN
 IF NOT feedback_shape(t,ARRAY['kind','identity','area','part']) OR NOT feedback_shape(s,ARRAY['kind','publicationId','attemptId','position']) THEN RETURN false; END IF;
 IF t->>'kind'='site' THEN RETURN (t->'identity'='null' AND t->'part'='null' AND t->>'area' IN ('home','knowledge_map','learning_center','account','review','other') AND s->>'kind'='site' AND s->'publicationId'='null' AND s->'attemptId'='null' AND s->'position'='null') IS TRUE; END IF;
 IF t->'area' IS DISTINCT FROM 'null'::jsonb OR NOT feedback_identity(t->'identity',t->>'kind'='instance') THEN RETURN false; END IF;
 IF t->>'kind' IN ('knowledge','path') THEN
  IF NOT (s->>'kind'='publication' AND feedback_uuid(s->>'publicationId') AND s->'attemptId'='null' AND s->'position'='null') IS TRUE THEN RETURN false; END IF;
 ELSIF t->>'kind'='instance' THEN
  IF NOT (s->'publicationId'='null' AND feedback_uuid(s->>'attemptId') AND jsonb_typeof(s->'position')='number' AND ((s->>'kind'='practice' AND s->>'position'='1') OR (s->>'kind'='assessment' AND s->>'position' IN ('1','2','3','4','5')))) IS TRUE THEN RETURN false; END IF;
 ELSE RETURN false; END IF;
 IF t->'part'='null' THEN RETURN true; END IF;
 IF NOT feedback_shape(t->'part',ARRAY['kind','unit','asset']) THEN RETURN false; END IF;
 RETURN ((t#>>'{part,kind}'='unit' AND t->>'kind'='knowledge' AND feedback_identity(t#>'{part,unit}') AND t#>'{part,asset}'='null') OR (t#>>'{part,kind}'='asset' AND t->>'kind' IN ('knowledge','instance') AND t#>'{part,unit}'='null' AND feedback_asset(t#>'{part,asset}'))) IS TRUE;
END $$;
CREATE FUNCTION feedback_content_clean(k text,i jsonb) RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT NOT EXISTS(SELECT 1 FROM content_withdrawals WHERE kind=k AND ((k='asset' AND sha256=i->>'sha256') OR (k<>'asset' AND target_id=i->>'id' AND target_version=(i->>'version')::integer AND sha256=i->>'sha256')))
$$;
CREATE FUNCTION feedback_target_proof(t jsonb,s jsonb,owner uuid,creating boolean) RETURNS boolean LANGUAGE plpgsql STABLE AS $$
DECLARE item jsonb;k jsonb;kp text;qp uuid;p jsonb:=t->'part';v integer;
BEGIN
 IF NOT feedback_target_shape(t,s) OR NOT EXISTS(SELECT 1 FROM auth_users WHERE id=owner) THEN RETURN false; END IF;
 IF t->>'kind'='site' THEN RETURN true; END IF;
 IF t->>'kind' IN ('knowledge','path') THEN
  kp:=s->>'publicationId';v:=(t#>>'{identity,version}')::integer;
  IF creating AND NOT EXISTS(SELECT 1 FROM publication_heads WHERE singleton AND snapshot_id=kp) THEN RETURN false; END IF;
  IF NOT learning_content_approved(kp,t->>'kind',t#>>'{identity,id}',v,t#>>'{identity,sha256}') OR (creating AND NOT feedback_content_clean(t->>'kind',t->'identity')) THEN RETURN false; END IF;
  IF t->>'kind'='knowledge' THEN
   IF NOT EXISTS(SELECT 1 FROM knowledge_versions WHERE id=t#>>'{identity,id}' AND version=v AND sha256=t#>>'{identity,sha256}') THEN RETURN false; END IF;
  ELSE
   IF NOT EXISTS(SELECT 1 FROM path_versions WHERE id=t#>>'{identity,id}' AND version=v AND sha256=t#>>'{identity,sha256}') THEN RETURN false; END IF;
  END IF;
  IF p='null' THEN RETURN true; END IF;
  IF p->>'kind'='unit' THEN
   RETURN EXISTS(SELECT 1 FROM unit_versions WHERE id=p#>>'{unit,id}' AND version=(p#>>'{unit,version}')::integer AND sha256=p#>>'{unit,sha256}' AND knowledge_id=t#>>'{identity,id}' AND knowledge_version=v) AND learning_content_approved(kp,'unit',p#>>'{unit,id}',(p#>>'{unit,version}')::integer,p#>>'{unit,sha256}') AND (NOT creating OR feedback_content_clean('unit',p->'unit'));
  END IF;
  RETURN learning_content_approved(kp,'asset',p#>>'{asset,id}',1,p#>>'{asset,sha256}') AND (NOT creating OR feedback_content_clean('asset',p->'asset')) AND EXISTS(SELECT 1 FROM publication_members m JOIN imported_packages ip ON ip.id=m.package_id AND ip.version=m.package_version CROSS JOIN LATERAL jsonb_array_elements(ip.body->'assets') a WHERE m.snapshot_id=kp AND m.kind='asset' AND m.id=p#>>'{asset,id}' AND a->>'id'=m.id AND a->>'sha256'=p#>>'{asset,sha256}' AND a#>>'{knowledge,id}'=t#>>'{identity,id}' AND a#>>'{knowledge,version}'=v::text);
 END IF;
 IF s->>'kind'='practice' THEN
  SELECT seal#>'{body,items,0}',seal#>'{body,knowledge}',knowledge_publication_id,question_publication_id INTO item,k,kp,qp FROM practice_attempts WHERE id=(s->>'attemptId')::uuid AND owner_user_id=owner;
 ELSE
  SELECT i.binding,a.seal#>'{body,knowledge}',a.knowledge_publication_id,a.question_publication_id INTO item,k,kp,qp FROM assessment_attempts a JOIN assessment_items i ON i.attempt_id=a.id AND i.position=(s->>'position')::integer WHERE a.id=(s->>'attemptId')::uuid AND a.owner_user_id=owner AND a.sealed;
 END IF;
 IF item IS NULL OR item->'instance' IS DISTINCT FROM t->'identity' OR NOT learning_item_proof(item,k,kp,qp) THEN RETURN false; END IF;
 RETURN p='null' OR p->>'kind'='asset' AND EXISTS(SELECT 1 FROM jsonb_array_elements(item->'assets') a WHERE a=p->'asset');
END $$;
CREATE FUNCTION feedback_text(v text,minimum integer,maximum integer) RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
 SELECT char_length(v) BETWEEN minimum AND maximum AND (v='' AND minimum=0 OR v ~ '[^[:space:]\u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]')
$$;
-- +goose StatementEnd
CREATE TABLE feedback_tickets (
 id uuid PRIMARY KEY CHECK(feedback_uuid(id::text)),owner_user_id uuid NOT NULL REFERENCES auth_users(id),target jsonb NOT NULL,source jsonb NOT NULL,
 original_title text NOT NULL CHECK(feedback_text(original_title,1,120)),original_location text NOT NULL CHECK(feedback_text(original_location,0,400)),category text NOT NULL CHECK(category IN ('math_error','unclear_explanation','typo','accessibility','technical_issue','suggestion')),
 status text NOT NULL CHECK(status IN ('new','processing','waiting_details','resolved','closed')),sequence bigint NOT NULL CHECK(sequence BETWEEN 1 AND 9007199254740991),current_resolution jsonb,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),CHECK(feedback_target_shape(target,source)),CHECK(target->>'kind'<>'site' OR category NOT IN ('math_error','unclear_explanation'))
);
CREATE INDEX feedback_owner_list ON feedback_tickets(owner_user_id,created_at DESC,id DESC);
CREATE INDEX feedback_queue_list ON feedback_tickets(status,category,created_at DESC,id DESC);
CREATE INDEX feedback_all_list ON feedback_tickets(created_at DESC,id DESC);
CREATE TABLE feedback_events (
 ticket_id uuid NOT NULL REFERENCES feedback_tickets(id),sequence bigint NOT NULL CHECK(sequence BETWEEN 1 AND 9007199254740991),actor_user_id uuid NOT NULL REFERENCES auth_users(id),
 kind text NOT NULL CHECK(kind IN ('created','replied','transitioned')),from_status text,to_status text NOT NULL CHECK(to_status IN ('new','processing','waiting_details','resolved','closed')),
 message text NOT NULL CHECK(feedback_text(message,1,4000)),resolution jsonb,effective_resolution jsonb,recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),request_id text NOT NULL CHECK(octet_length(request_id) BETWEEN 1 AND 200),PRIMARY KEY(ticket_id,sequence)
);
ALTER TABLE feedback_tickets ADD CONSTRAINT feedback_current_event FOREIGN KEY(id,sequence) REFERENCES feedback_events(ticket_id,sequence) DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE feedback_idempotency (
 actor_user_id uuid NOT NULL REFERENCES auth_users(id),action text NOT NULL CHECK(action IN ('create','reply','transition')),resource text NOT NULL CHECK(octet_length(resource) BETWEEN 1 AND 512),
 key uuid NOT NULL CHECK(feedback_uuid(key::text)),request_sha256 text NOT NULL CHECK(request_sha256 ~ '^[0-9a-f]{64}$'),ticket_id uuid NOT NULL,event_sequence bigint NOT NULL,receipt jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(actor_user_id,action,resource,key),FOREIGN KEY(ticket_id,event_sequence) REFERENCES feedback_events(ticket_id,sequence)
);
CREATE TABLE feedback_rate_limits (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),actor_user_id uuid NOT NULL REFERENCES auth_users(id),scope text NOT NULL CHECK(scope IN ('create','reply','transition')),consumed_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX feedback_rate_window ON feedback_rate_limits(actor_user_id,scope,consumed_at);
-- +goose StatementBegin
CREATE FUNCTION feedback_resolution_proof(t feedback_tickets,r jsonb,destination text) RETURNS boolean LANGUAGE plpgsql STABLE AS $$
DECLARE k text:=r->>'kind';root text:=t.target->>'kind';obj jsonb:=t.target->'identity';part text:=t.target#>>'{part,kind}';w jsonb:=r->'withdrawal';rep jsonb:=r->'replacement';kind text;rid jsonb;pub text;oldknowledge text;oldversion integer;knowledge_id text;knowledge_version integer;
BEGIN
 IF NOT feedback_shape(r,ARRAY['kind','withdrawal','replacement','duplicateOf']) THEN RETURN false; END IF;
 IF k IN ('clarified','service_fixed','not_reproducible','out_of_scope','suggestion_recorded') THEN
  RETURN (r->'withdrawal'='null' AND r->'replacement'='null' AND r->'duplicateOf'='null' AND ((destination='resolved' AND k IN ('clarified','service_fixed') AND (k<>'service_fixed' OR root='site')) OR (destination='closed' AND k IN ('not_reproducible','out_of_scope','suggestion_recorded')))) IS TRUE;
 END IF;
 IF k='duplicate' THEN RETURN destination='closed' AND r->'withdrawal'='null' AND r->'replacement'='null' AND feedback_uuid(r->>'duplicateOf') AND EXISTS(SELECT 1 FROM feedback_tickets other WHERE other.id=(r->>'duplicateOf')::uuid AND other.id<>t.id AND other.target=t.target AND other.category=t.category); END IF;
 IF k NOT IN ('withdrawn','revision_published') OR destination<>'resolved' OR r->'duplicateOf' IS DISTINCT FROM 'null'::jsonb OR NOT feedback_shape(w,ARRAY['space','id']) OR NOT feedback_uuid(w->>'id') THEN RETURN false; END IF;
 kind:=coalesce(part,root);IF part='unit' THEN obj:=t.target#>'{part,unit}';ELSIF part='asset' THEN obj:=t.target#>'{part,asset}';END IF;
 IF w->>'space'='content' THEN
  IF NOT EXISTS(SELECT 1 FROM content_withdrawals WHERE id=(w->>'id')::uuid AND kind=feedback_resolution_proof.kind AND sha256=obj->>'sha256' AND (kind='asset' OR target_id=obj->>'id' AND target_version=(obj->>'version')::integer)) THEN RETURN false; END IF;
 ELSIF w->>'space'='question' THEN
  IF kind<>'instance' OR NOT EXISTS(SELECT 1 FROM question_withdrawals WHERE id=(w->>'id')::uuid AND kind='instance' AND target_id=obj->>'id' AND target_version=(obj->>'version')::integer AND sha256=obj->>'sha256') THEN RETURN false; END IF;
 ELSE RETURN false; END IF;
 IF k='withdrawn' THEN RETURN rep='null'; END IF;
 IF NOT feedback_shape(rep,ARRAY['kind','identity','asset','publicationId']) OR rep->>'kind'<>kind OR NOT feedback_uuid(rep->>'publicationId') THEN RETURN false; END IF;
 pub:=rep->>'publicationId';rid:=rep->'identity';
 IF kind='asset' THEN
  IF rep->'identity' IS DISTINCT FROM 'null'::jsonb OR NOT feedback_asset(rep->'asset') OR rep#>>'{asset,sha256}'=obj->>'sha256' OR NOT EXISTS(SELECT 1 FROM publication_heads WHERE snapshot_id=pub) OR NOT learning_content_approved(pub,'asset',rep#>>'{asset,id}',1,rep#>>'{asset,sha256}') OR NOT feedback_content_clean('asset',rep->'asset') THEN RETURN false; END IF;
 ELSE
  IF rep->'asset' IS DISTINCT FROM 'null'::jsonb OR NOT feedback_identity(rid,kind='instance') OR rid=obj THEN RETURN false; END IF;
  IF kind='instance' THEN
   IF rid->>'id'=obj->>'id' OR NOT EXISTS(SELECT 1 FROM question_heads WHERE publication_id=pub::uuid) OR NOT EXISTS(SELECT 1 FROM question_publications WHERE id=pub::uuid AND sealed AND status='published') THEN RETURN false; END IF;
   SELECT i.knowledge_id,i.knowledge_version INTO oldknowledge,oldversion FROM question_instances i WHERE i.id=obj->>'id' AND i.version=(obj->>'version')::integer;
   RETURN EXISTS(SELECT 1 FROM question_instances i JOIN question_publication_members m ON m.publication_id=pub::uuid AND m.kind='instance' AND m.id=i.id AND m.version=i.version AND m.sha256=i.sha256 JOIN question_review_decisions d ON d.id=m.review_id AND d.decision='approve' JOIN question_submissions s ON s.id=m.submission_id AND s.sealed AND s.status='approved' AND s.frozen_digest=d.frozen_digest WHERE i.id=rid->>'id' AND i.version=(rid->>'version')::integer AND i.sha256=rid->>'sha256' AND i.sealed AND i.knowledge_id=oldknowledge AND i.knowledge_version>=oldversion AND NOT EXISTS(SELECT 1 FROM question_withdrawals qw WHERE qw.kind='instance' AND qw.target_id=i.id AND qw.target_version=i.version) AND NOT EXISTS(SELECT 1 FROM question_withdrawals qw WHERE qw.kind='template' AND qw.target_id=i.template_id AND qw.target_version=i.template_version));
  END IF;
  IF kind IN ('knowledge','path') AND rid->>'id'<>obj->>'id' THEN RETURN false; END IF;
  IF NOT EXISTS(SELECT 1 FROM publication_heads WHERE snapshot_id=pub) OR NOT learning_content_approved(pub,kind,rid->>'id',(rid->>'version')::integer,rid->>'sha256') OR NOT feedback_content_clean(kind,rid) THEN RETURN false; END IF;
 END IF;
 -- Unit/asset revisions must belong to the corresponding stable knowledge identity.
 IF kind='unit' THEN RETURN EXISTS(SELECT 1 FROM unit_versions u WHERE u.id=rid->>'id' AND u.version=(rid->>'version')::integer AND u.sha256=rid->>'sha256' AND u.knowledge_id=t.target#>>'{identity,id}' AND u.knowledge_version>=(t.target#>>'{identity,version}')::integer); END IF;
 IF kind='asset' THEN
  IF root='knowledge' THEN oldknowledge:=t.target#>>'{identity,id}';oldversion:=(t.target#>>'{identity,version}')::integer;ELSE SELECT i.knowledge_id,i.knowledge_version INTO oldknowledge,oldversion FROM question_instances i WHERE i.id=t.target#>>'{identity,id}' AND i.version=(t.target#>>'{identity,version}')::integer;END IF;
  RETURN EXISTS(SELECT 1 FROM publication_members m JOIN imported_packages p ON p.id=m.package_id AND p.version=m.package_version CROSS JOIN LATERAL jsonb_array_elements(p.body->'assets') a WHERE m.snapshot_id=pub AND m.kind='asset' AND m.id=rep#>>'{asset,id}' AND a->>'id'=m.id AND a->>'sha256'=rep#>>'{asset,sha256}' AND a#>>'{knowledge,id}'=oldknowledge AND (a#>>'{knowledge,version}')::integer>=oldversion);
 END IF;
 RETURN true;
END $$;
CREATE FUNCTION feedback_ticket_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN RAISE EXCEPTION 'immutable feedback report'; END IF;
 IF TG_OP='INSERT' THEN
  IF NEW.status<>'new' OR NEW.sequence<>1 OR NEW.current_resolution IS NOT NULL OR NEW.created_at<>NEW.updated_at OR NOT feedback_target_proof(NEW.target,NEW.source,NEW.owner_user_id,true) THEN RAISE EXCEPTION 'invalid feedback source'; END IF;
 ELSE
  IF (NEW.id,NEW.owner_user_id,NEW.target,NEW.source,NEW.original_title,NEW.original_location,NEW.category,NEW.created_at) IS DISTINCT FROM (OLD.id,OLD.owner_user_id,OLD.target,OLD.source,OLD.original_title,OLD.original_location,OLD.category,OLD.created_at) OR NEW.sequence<>OLD.sequence+1 OR NEW.updated_at<OLD.updated_at THEN RAISE EXCEPTION 'immutable feedback report or invalid projection'; END IF;
 END IF;RETURN NEW;
END $$;
CREATE FUNCTION feedback_event_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE t feedback_tickets;prev feedback_events;expected text;staff boolean;
BEGIN
 IF TG_OP<>'INSERT' THEN RAISE EXCEPTION 'immutable feedback event'; END IF;
 SELECT * INTO t FROM feedback_tickets WHERE id=NEW.ticket_id FOR UPDATE;IF NOT FOUND THEN RAISE EXCEPTION 'missing feedback ticket'; END IF;
 SELECT * INTO prev FROM feedback_events WHERE ticket_id=NEW.ticket_id ORDER BY sequence DESC LIMIT 1;
 IF NOT FOUND THEN
  IF NEW.sequence<>1 OR NEW.kind<>'created' OR NEW.actor_user_id<>t.owner_user_id OR NEW.from_status IS NOT NULL OR NEW.to_status<>'new' OR NEW.resolution IS NOT NULL OR NEW.effective_resolution IS NOT NULL OR NEW.recorded_at<>t.created_at THEN RAISE EXCEPTION 'invalid feedback creation'; END IF;
  RETURN NEW;
 END IF;
 IF NEW.sequence<>prev.sequence+1 OR NEW.from_status IS DISTINCT FROM prev.to_status OR NEW.recorded_at<prev.recorded_at THEN RAISE EXCEPTION 'invalid feedback sequence'; END IF;
 IF NEW.actor_user_id=t.owner_user_id THEN
  expected:=CASE WHEN prev.to_status IN ('new','processing') THEN prev.to_status ELSE 'processing' END;
  IF NEW.kind<>'replied' OR NEW.to_status<>expected OR NEW.resolution IS NOT NULL OR NEW.effective_resolution IS NOT NULL THEN RAISE EXCEPTION 'invalid owner reply'; END IF;
 ELSE
  SELECT EXISTS(SELECT 1 FROM auth_user_roles WHERE user_id=NEW.actor_user_id AND role IN ('reviewer','admin')) AND NOT EXISTS(SELECT 1 FROM auth_users WHERE id=NEW.actor_user_id AND must_change_password) INTO staff;
  IF NOT staff THEN RAISE EXCEPTION 'feedback handler required'; END IF;
  IF NEW.to_status=prev.to_status THEN
   IF NEW.kind<>'replied' OR NEW.resolution IS NOT NULL OR NEW.effective_resolution IS DISTINCT FROM prev.effective_resolution THEN RAISE EXCEPTION 'invalid same-state reply'; END IF;
  ELSE
   IF NEW.kind<>'transitioned' OR NOT ((prev.to_status='new' AND NEW.to_status IN ('processing','waiting_details','closed')) OR (prev.to_status='processing' AND NEW.to_status IN ('waiting_details','resolved','closed')) OR (prev.to_status='waiting_details' AND NEW.to_status IN ('processing','resolved','closed')) OR (prev.to_status IN ('resolved','closed') AND NEW.to_status='processing')) THEN RAISE EXCEPTION 'invalid feedback transition'; END IF;
   IF NEW.to_status IN ('resolved','closed') THEN
    IF NOT feedback_resolution_proof(t,NEW.resolution,NEW.to_status) OR NEW.effective_resolution IS DISTINCT FROM NEW.resolution THEN RAISE EXCEPTION 'invalid feedback resolution'; END IF;
   ELSIF NEW.resolution IS NOT NULL OR NEW.effective_resolution IS NOT NULL THEN RAISE EXCEPTION 'invalid nonterminal resolution'; END IF;
  END IF;
 END IF;RETURN NEW;
END $$;
CREATE FUNCTION feedback_projection_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE t feedback_tickets;e feedback_events;first_time timestamptz;tid uuid;
BEGIN
 IF TG_TABLE_NAME='feedback_tickets' THEN tid:=NEW.id;ELSE tid:=NEW.ticket_id;END IF;
 SELECT * INTO t FROM feedback_tickets WHERE id=tid;
 SELECT * INTO e FROM feedback_events WHERE ticket_id=tid ORDER BY sequence DESC LIMIT 1;
 IF NOT FOUND OR (t.sequence,t.status,t.current_resolution,t.updated_at) IS DISTINCT FROM (e.sequence,e.to_status,e.effective_resolution,e.recorded_at) THEN RAISE EXCEPTION 'feedback projection mismatch'; END IF;
 SELECT recorded_at INTO first_time FROM feedback_events WHERE ticket_id=tid AND sequence=1;
 IF first_time IS DISTINCT FROM t.created_at THEN RAISE EXCEPTION 'feedback creation missing'; END IF;RETURN NULL;
END $$;
CREATE FUNCTION feedback_receipt_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE t feedback_tickets;e feedback_events;m jsonb:=NEW.receipt->'ticket';
BEGIN
 IF TG_OP<>'INSERT' THEN RAISE EXCEPTION 'immutable feedback receipt'; END IF;
 SELECT * INTO t FROM feedback_tickets WHERE id=NEW.ticket_id;SELECT * INTO e FROM feedback_events WHERE ticket_id=NEW.ticket_id AND sequence=NEW.event_sequence;
 IF NOT feedback_shape(NEW.receipt,ARRAY['status','ticket']) OR NOT feedback_shape(m,ARRAY['id','target','label','category','status','sequence','createdAt','updatedAt','resolutionKind','targetValidity','canHandle']) OR m->>'id'<>t.id::text OR m->'target'<>t.target OR m->>'category'<>t.category OR m->>'status'<>e.to_status OR (m->>'sequence')::bigint<>e.sequence OR (m->>'createdAt')::timestamptz<>t.created_at OR (m->>'updatedAt')::timestamptz<>e.recorded_at OR m->>'resolutionKind' IS DISTINCT FROM e.effective_resolution->>'kind' OR jsonb_typeof(m->'label')<>'string' OR jsonb_typeof(m->'canHandle')<>'boolean' OR m->>'targetValidity' NOT IN ('current','historical','withdrawn') OR NEW.actor_user_id<>e.actor_user_id OR (NEW.action='create' AND (NEW.resource<>'tickets' OR NEW.event_sequence<>1 OR NEW.receipt->>'status'<>'201')) OR (NEW.action<>'create' AND (NEW.resource<>t.id::text OR NEW.receipt->>'status'<>'200')) THEN RAISE EXCEPTION 'invalid safe feedback receipt'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER feedback_ticket_guard BEFORE INSERT OR UPDATE OR DELETE ON feedback_tickets FOR EACH ROW EXECUTE FUNCTION feedback_ticket_guard();
CREATE TRIGGER feedback_event_guard BEFORE INSERT OR UPDATE OR DELETE ON feedback_events FOR EACH ROW EXECUTE FUNCTION feedback_event_guard();
CREATE TRIGGER feedback_receipt_guard BEFORE INSERT OR UPDATE OR DELETE ON feedback_idempotency FOR EACH ROW EXECUTE FUNCTION feedback_receipt_guard();
CREATE CONSTRAINT TRIGGER feedback_ticket_projection AFTER INSERT OR UPDATE ON feedback_tickets DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION feedback_projection_guard();
CREATE CONSTRAINT TRIGGER feedback_event_projection AFTER INSERT ON feedback_events DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION feedback_projection_guard();
-- +goose Down
-- +goose StatementBegin
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM feedback_tickets) OR EXISTS(SELECT 1 FROM feedback_events) OR EXISTS(SELECT 1 FROM feedback_idempotency) OR EXISTS(SELECT 1 FROM feedback_rate_limits) THEN RAISE EXCEPTION 'feedback data must be retained; rollback binary only'; END IF;
END $$;
-- +goose StatementEnd
DROP TABLE feedback_idempotency,feedback_rate_limits;
ALTER TABLE feedback_tickets DROP CONSTRAINT feedback_current_event;
DROP TRIGGER feedback_ticket_guard ON feedback_tickets;
DROP TRIGGER feedback_ticket_projection ON feedback_tickets;
DROP TRIGGER feedback_event_guard ON feedback_events;
DROP TRIGGER feedback_event_projection ON feedback_events;
DROP FUNCTION feedback_projection_guard(),feedback_event_guard(),feedback_ticket_guard(),feedback_receipt_guard(),feedback_resolution_proof(feedback_tickets,jsonb,text);
DROP TABLE feedback_events,feedback_tickets;
DROP FUNCTION feedback_target_proof(jsonb,jsonb,uuid,boolean),feedback_content_clean(text,jsonb),feedback_target_shape(jsonb,jsonb),feedback_text(text,integer,integer),feedback_identity(jsonb,boolean),feedback_asset(jsonb),feedback_uuid(text),feedback_shape(jsonb,text[]);
