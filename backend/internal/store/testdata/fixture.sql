-- Test-only publication fixture. Never use as a production seed or approval workflow.
UPDATE publication_snapshots SET status='published';
INSERT INTO publication_heads(singleton,snapshot_id) SELECT true,id FROM publication_snapshots LIMIT 1;
