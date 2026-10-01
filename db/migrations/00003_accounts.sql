-- +goose Up
CREATE TABLE auth_users (
 id uuid PRIMARY KEY,
 username text NOT NULL UNIQUE CHECK (username ~ '^[a-z0-9_]{3,32}$'),
 password_phc text NOT NULL CHECK (length(password_phc) BETWEEN 1 AND 512),
 credential_version bigint NOT NULL DEFAULT 1 CHECK (credential_version > 0),
 must_change_password boolean NOT NULL DEFAULT false,
 default_role text NOT NULL DEFAULT 'learner' CHECK (default_role = 'learner'),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE auth_user_roles (
 user_id uuid NOT NULL REFERENCES auth_users(id),
 role text NOT NULL CHECK (role IN ('learner','editor','reviewer','admin')),
 PRIMARY KEY(user_id,role)
);
-- NO ACTION is deferred: registration inserts a user before its required role.
ALTER TABLE auth_users ADD CONSTRAINT auth_users_learner_fk
 FOREIGN KEY(id,default_role) REFERENCES auth_user_roles(user_id,role)
 ON DELETE NO ACTION DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE auth_sessions (
 token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash)=32),
 user_id uuid NOT NULL REFERENCES auth_users(id),
 credential_version bigint NOT NULL CHECK (credential_version>0),
 csrf bytea NOT NULL CHECK (octet_length(csrf)=32),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 last_seen_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 absolute_expires_at timestamptz NOT NULL,
 revoked_at timestamptz,
 reauthenticated_at timestamptz,
 CHECK (absolute_expires_at>created_at)
);
CREATE INDEX auth_sessions_user_created_idx ON auth_sessions(user_id,created_at,token_hash);
CREATE INDEX auth_sessions_cleanup_idx ON auth_sessions(absolute_expires_at,last_seen_at,revoked_at);
CREATE TABLE auth_preauth (
 token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash)=32),
 csrf bytea NOT NULL CHECK (octet_length(csrf)=32),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 expires_at timestamptz NOT NULL,
 consumed_at timestamptz,
 CHECK (expires_at>created_at)
);
CREATE INDEX auth_preauth_cleanup_idx ON auth_preauth(expires_at,consumed_at);
CREATE TABLE auth_rate_limits (
 scope text NOT NULL,key text NOT NULL,window_start timestamptz NOT NULL,
 window_end timestamptz NOT NULL CHECK (window_end>window_start),
 attempts integer NOT NULL CHECK (attempts>0),
 PRIMARY KEY(scope,key,window_start)
);
CREATE INDEX auth_rate_limits_cleanup_idx ON auth_rate_limits(window_end);
CREATE TABLE auth_audit_events (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 action text NOT NULL,
 actor_id uuid REFERENCES auth_users(id),target_id uuid REFERENCES auth_users(id),
 before_roles text[] NOT NULL DEFAULT '{}',after_roles text[] NOT NULL DEFAULT '{}',
 reason text NOT NULL DEFAULT '',ownership_note text NOT NULL DEFAULT '',
 username_hash bytea CHECK (username_hash IS NULL OR octet_length(username_hash)=32),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),request_id text NOT NULL,
 CHECK ((action='login_failed' AND username_hash IS NOT NULL AND actor_id IS NULL AND target_id IS NULL AND reason='' AND ownership_note='') OR (action<>'login_failed' AND username_hash IS NULL))
);
CREATE INDEX auth_audit_events_target_idx ON auth_audit_events(target_id,id);
-- +goose StatementBegin
CREATE FUNCTION reject_auth_audit_update() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 RAISE EXCEPTION 'immutable account audit';
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER auth_audit_events_immutable BEFORE UPDATE OR DELETE ON auth_audit_events
 FOR EACH ROW EXECUTE FUNCTION reject_auth_audit_update();
CREATE TABLE auth_bootstrap (
 singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
 user_id uuid NOT NULL REFERENCES auth_users(id),created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
-- +goose Down
-- Empty isolated test databases only. Application rollbacks retain real accounts.
DROP TABLE auth_bootstrap;
DROP TABLE auth_audit_events;
DROP FUNCTION reject_auth_audit_update();
DROP TABLE auth_rate_limits;
DROP TABLE auth_preauth;
DROP TABLE auth_sessions;
ALTER TABLE auth_users DROP CONSTRAINT auth_users_learner_fk;
DROP TABLE auth_user_roles;
DROP TABLE auth_users;
