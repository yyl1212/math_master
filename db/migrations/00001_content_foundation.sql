-- +goose Up
CREATE TABLE catalogue_versions (version integer PRIMARY KEY CHECK(version>0), sha256 text NOT NULL UNIQUE, body jsonb NOT NULL, UNIQUE(version,sha256));
CREATE TABLE domains (catalogue_version integer NOT NULL REFERENCES catalogue_versions(version), id text NOT NULL, position integer NOT NULL, body jsonb NOT NULL, PRIMARY KEY(catalogue_version,id), UNIQUE(catalogue_version,position));
CREATE TABLE topics (catalogue_version integer NOT NULL, id text NOT NULL, domain_id text NOT NULL, body jsonb NOT NULL, PRIMARY KEY(catalogue_version,id), FOREIGN KEY(catalogue_version,domain_id) REFERENCES domains(catalogue_version,id));
CREATE TABLE domain_relations (catalogue_version integer NOT NULL, source_id text NOT NULL, target_id text NOT NULL, PRIMARY KEY(catalogue_version,source_id,target_id), FOREIGN KEY(catalogue_version,source_id) REFERENCES domains(catalogue_version,id), FOREIGN KEY(catalogue_version,target_id) REFERENCES domains(catalogue_version,id));
CREATE TABLE knowledge (id text PRIMARY KEY);
CREATE TABLE knowledge_versions (id text NOT NULL REFERENCES knowledge(id), version integer NOT NULL CHECK(version>0), sha256 text NOT NULL, body jsonb NOT NULL, PRIMARY KEY(id,version));
CREATE TABLE knowledge_relations (source_id text NOT NULL, source_version integer NOT NULL, kind text NOT NULL CHECK(kind IN ('prerequisite','derivation','related')), target_id text NOT NULL, target_version integer NOT NULL, PRIMARY KEY(source_id,source_version,kind,target_id,target_version), FOREIGN KEY(source_id,source_version) REFERENCES knowledge_versions(id,version), FOREIGN KEY(target_id,target_version) REFERENCES knowledge_versions(id,version));
CREATE TABLE unit_versions (id text NOT NULL, version integer NOT NULL CHECK(version>0), sha256 text NOT NULL, body jsonb NOT NULL, knowledge_id text NOT NULL, knowledge_version integer NOT NULL, PRIMARY KEY(id,version), FOREIGN KEY(knowledge_id,knowledge_version) REFERENCES knowledge_versions(id,version));
CREATE TABLE path_versions (id text NOT NULL, version integer NOT NULL CHECK(version>0), sha256 text NOT NULL, body jsonb NOT NULL, PRIMARY KEY(id,version));
CREATE TABLE path_nodes (path_id text NOT NULL, path_version integer NOT NULL, position integer NOT NULL, knowledge_id text NOT NULL, knowledge_version integer NOT NULL, PRIMARY KEY(path_id,path_version,position), UNIQUE(path_id,path_version,knowledge_id), FOREIGN KEY(path_id,path_version) REFERENCES path_versions(id,version), FOREIGN KEY(knowledge_id,knowledge_version) REFERENCES knowledge_versions(id,version));
CREATE TABLE assets (sha256 text PRIMARY KEY, bytes bytea NOT NULL CHECK(octet_length(bytes)<=1048576));
CREATE TABLE imported_packages (id text NOT NULL, version integer NOT NULL CHECK(version>0), sha256 text NOT NULL UNIQUE, catalogue_version integer NOT NULL REFERENCES catalogue_versions(version), catalogue_sha256 text NOT NULL, body jsonb NOT NULL, PRIMARY KEY(id,version), FOREIGN KEY(catalogue_version,catalogue_sha256) REFERENCES catalogue_versions(version,sha256));
CREATE TABLE package_members (
 package_id text NOT NULL, package_version integer NOT NULL, kind text NOT NULL, id text NOT NULL, version integer NOT NULL,
 knowledge_id text, unit_id text, path_id text, asset_sha256 text,
 PRIMARY KEY(package_id,package_version,kind,id), UNIQUE(package_id,package_version,kind,id,version),
 FOREIGN KEY(package_id,package_version) REFERENCES imported_packages(id,version),
 FOREIGN KEY(knowledge_id,version) REFERENCES knowledge_versions(id,version), FOREIGN KEY(unit_id,version) REFERENCES unit_versions(id,version), FOREIGN KEY(path_id,version) REFERENCES path_versions(id,version), FOREIGN KEY(asset_sha256) REFERENCES assets(sha256),
 CHECK ((kind='knowledge' AND knowledge_id IS NOT NULL AND knowledge_id=id AND unit_id IS NULL AND path_id IS NULL AND asset_sha256 IS NULL) OR
 (kind='unit' AND unit_id IS NOT NULL AND unit_id=id AND knowledge_id IS NULL AND path_id IS NULL AND asset_sha256 IS NULL) OR
 (kind='path' AND path_id IS NOT NULL AND path_id=id AND knowledge_id IS NULL AND unit_id IS NULL AND asset_sha256 IS NULL) OR
 (kind='asset' AND asset_sha256 IS NOT NULL AND knowledge_id IS NULL AND unit_id IS NULL AND path_id IS NULL AND version=1))
);
CREATE TABLE publication_snapshots (id text PRIMARY KEY, catalogue_version integer NOT NULL REFERENCES catalogue_versions(version), status text NOT NULL CHECK(status IN ('draft','published','withdrawn')));
CREATE TABLE publication_members (snapshot_id text NOT NULL REFERENCES publication_snapshots(id), package_id text NOT NULL, package_version integer NOT NULL, kind text NOT NULL, id text NOT NULL, version integer NOT NULL, availability text NOT NULL CHECK(availability IN ('active','withdrawn')), PRIMARY KEY(snapshot_id,kind,id), FOREIGN KEY(package_id,package_version,kind,id,version) REFERENCES package_members(package_id,package_version,kind,id,version));
CREATE TABLE publication_heads (singleton boolean PRIMARY KEY DEFAULT true CHECK(singleton), snapshot_id text NOT NULL REFERENCES publication_snapshots(id));
CREATE INDEX publication_member_lookup ON publication_members(snapshot_id,kind,id,availability);
CREATE INDEX knowledge_prerequisites ON knowledge_relations(source_id,source_version) WHERE kind='prerequisite';
-- +goose StatementBegin
CREATE FUNCTION reject_content_update() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'immutable content'; END $$;
-- +goose StatementEnd
CREATE TRIGGER catalogue_immutable BEFORE UPDATE ON catalogue_versions FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER knowledge_immutable BEFORE UPDATE ON knowledge_versions FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER unit_immutable BEFORE UPDATE ON unit_versions FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER path_immutable BEFORE UPDATE ON path_versions FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER asset_immutable BEFORE UPDATE ON assets FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER package_immutable BEFORE UPDATE ON imported_packages FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER domains_immutable BEFORE UPDATE ON domains FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER topics_immutable BEFORE UPDATE ON topics FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER domain_relations_immutable BEFORE UPDATE ON domain_relations FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER knowledge_relations_immutable BEFORE UPDATE ON knowledge_relations FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER path_nodes_immutable BEFORE UPDATE ON path_nodes FOR EACH ROW EXECUTE FUNCTION reject_content_update();
CREATE TRIGGER package_members_immutable BEFORE UPDATE ON package_members FOR EACH ROW EXECUTE FUNCTION reject_content_update();
-- +goose Down
DROP TABLE publication_heads,publication_members,publication_snapshots,package_members,imported_packages,assets,path_nodes,path_versions,unit_versions,knowledge_relations,knowledge_versions,knowledge,domain_relations,topics,domains,catalogue_versions;
DROP FUNCTION reject_content_update();
