-- +goose Up
-- Asset IDs are package aliases; fixed unit versions bind exact immutable asset digests.
CREATE TABLE unit_asset_bindings (
 unit_id text NOT NULL, unit_version integer NOT NULL, asset_id text NOT NULL, asset_sha256 text NOT NULL,
 PRIMARY KEY(unit_id,unit_version,asset_id),
 FOREIGN KEY(unit_id,unit_version) REFERENCES unit_versions(id,version),
 FOREIGN KEY(asset_sha256) REFERENCES assets(sha256)
);
-- Backfill existing P1 drafts. Conflicting digests for the same unit version fail migration.
INSERT INTO unit_asset_bindings
SELECT DISTINCT u.id,u.version,ids.id,a.value->>'sha256'
FROM unit_versions u
CROSS JOIN LATERAL jsonb_array_elements_text(u.body->'assetIds') ids(id)
JOIN package_members m ON m.kind='unit' AND m.id=u.id AND m.version=u.version
JOIN imported_packages p ON p.id=m.package_id AND p.version=m.package_version
CROSS JOIN LATERAL jsonb_array_elements(p.body->'assets') a(value)
WHERE a.value->>'id'=ids.id;
CREATE TRIGGER unit_asset_bindings_immutable BEFORE UPDATE OR DELETE ON unit_asset_bindings FOR EACH ROW EXECUTE FUNCTION reject_content_update();
-- +goose Down
DROP TABLE unit_asset_bindings;
