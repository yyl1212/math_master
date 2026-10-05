-- +goose Up
-- 管理员兼审核者可批准本人提交；普通审核者保持独立性要求。
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION content_review_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE s content_submissions;
BEGIN
 SELECT * INTO s FROM content_submissions WHERE id=NEW.submission_id FOR SHARE;
 IF NOT s.sealed OR s.status<>'pending' OR s.frozen_digest<>NEW.frozen_digest THEN RAISE EXCEPTION 'invalid review target'; END IF;
 IF EXISTS(SELECT 1 FROM content_submission_authors WHERE submission_id=s.id AND user_id=NEW.reviewer_user_id)
 AND NOT EXISTS(SELECT 1 FROM auth_users u WHERE u.id=NEW.reviewer_user_id AND NOT u.must_change_password
 AND EXISTS(SELECT 1 FROM auth_user_roles r WHERE r.user_id=u.id AND r.role='admin')
 AND EXISTS(SELECT 1 FROM auth_user_roles r WHERE r.user_id=u.id AND r.role='reviewer'))
 THEN RAISE EXCEPTION 'administrator and reviewer required for author approval'; END IF;
 RETURN NEW;
END $$;
CREATE OR REPLACE FUNCTION question_review_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE s question_submissions;
BEGIN
 SELECT * INTO s FROM question_submissions WHERE id=NEW.submission_id FOR SHARE;
 IF NOT s.sealed OR s.status<>'pending' OR s.frozen_digest<>NEW.frozen_digest THEN RAISE EXCEPTION 'invalid review target'; END IF;
 IF EXISTS(SELECT 1 FROM question_submission_authors WHERE submission_id=s.id AND user_id=NEW.reviewer_user_id)
 AND NOT EXISTS(SELECT 1 FROM auth_users u WHERE u.id=NEW.reviewer_user_id AND NOT u.must_change_password
 AND EXISTS(SELECT 1 FROM auth_user_roles r WHERE r.user_id=u.id AND r.role='admin')
 AND EXISTS(SELECT 1 FROM auth_user_roles r WHERE r.user_id=u.id AND r.role='reviewer'))
 THEN RAISE EXCEPTION 'administrator and reviewer required for author approval'; END IF;
 IF NEW.decision='approve' AND jsonb_array_length(s.frozen_body#>'{body,body,questionPackage,templates}')=0 AND NOT (char_length(NEW.generation_note) BETWEEN 10 AND 1000 AND NEW.generation_note ~ '[^[:space:]]') THEN RAISE EXCEPTION 'generation note required'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
-- +goose Down
-- 保留已产生的自审历史，禁止恢复为无法解释这些记录的旧数据库规则。
-- +goose StatementBegin
LOCK TABLE content_review_decisions, question_review_decisions IN ACCESS EXCLUSIVE MODE;
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM content_review_decisions r JOIN content_submission_authors a ON a.submission_id=r.submission_id AND a.user_id=r.reviewer_user_id WHERE r.decision='approve')
 OR EXISTS(SELECT 1 FROM question_review_decisions r JOIN question_submission_authors a ON a.submission_id=r.submission_id AND a.user_id=r.reviewer_user_id WHERE r.decision='approve')
 THEN RAISE EXCEPTION 'administrator self-review history prevents schema rollback'; END IF;
END $$;
CREATE OR REPLACE FUNCTION content_review_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE s content_submissions;
BEGIN
 SELECT * INTO s FROM content_submissions WHERE id=NEW.submission_id FOR SHARE;
 IF NOT s.sealed OR s.status<>'pending' OR s.frozen_digest<>NEW.frozen_digest THEN RAISE EXCEPTION 'invalid review target'; END IF;
 IF NEW.decision='approve' AND EXISTS(SELECT 1 FROM content_submission_authors WHERE submission_id=s.id AND user_id=NEW.reviewer_user_id) THEN RAISE EXCEPTION 'author cannot approve'; END IF;
 RETURN NEW;
END $$;
CREATE OR REPLACE FUNCTION question_review_guard() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE s question_submissions;
BEGIN
 SELECT * INTO s FROM question_submissions WHERE id=NEW.submission_id FOR SHARE;
 IF NOT s.sealed OR s.status<>'pending' OR s.frozen_digest<>NEW.frozen_digest THEN RAISE EXCEPTION 'invalid review target'; END IF;
 IF NEW.decision='approve' AND EXISTS(SELECT 1 FROM question_submission_authors WHERE submission_id=s.id AND user_id=NEW.reviewer_user_id) THEN RAISE EXCEPTION 'author cannot approve'; END IF;
 IF NEW.decision='approve' AND jsonb_array_length(s.frozen_body#>'{body,body,questionPackage,templates}')=0 AND NOT (char_length(NEW.generation_note) BETWEEN 10 AND 1000 AND NEW.generation_note ~ '[^[:space:]]') THEN RAISE EXCEPTION 'generation note required'; END IF;
 RETURN NEW;
END $$;
-- +goose StatementEnd
