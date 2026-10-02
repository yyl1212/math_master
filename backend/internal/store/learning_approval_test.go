package store_test

import (
	"encoding/json"
	"testing"
)

// Frozen pre-optimization proof; compare decisions, not an implementation shape.
const learningApprovalReference = `CREATE FUNCTION learning_approval_reference(snap text,k text,object_id text,v integer,h text) RETURNS boolean LANGUAGE sql STABLE AS $$
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
$$;`

func TestLearningApprovalProofMatchesOriginal(t *testing.T) {
	f := newLearningFixture(t)
	f.publishLearningGraph()
	if _, err := f.db.ExecContext(f.ctx, learningApprovalReference); err != nil {
		t.Fatal(err)
	}
	var original []byte
	if err := f.db.QueryRowContext(f.ctx, `SELECT manifest_bytes FROM content_publication_manifests WHERE snapshot_id=$1`, *f.KHead()).Scan(&original); err != nil {
		t.Fatal(err)
	}
	rows, err := f.db.QueryContext(f.ctx, `SELECT e#>>'{identity,kind}',e#>>'{identity,id}',(e#>>'{identity,version}')::integer,e#>>'{identity,sha256}' FROM content_publication_manifests cm CROSS JOIN LATERAL jsonb_array_elements(cm.manifest->'members') e WHERE cm.snapshot_id=$1`, *f.KHead())
	if err != nil {
		t.Fatal(err)
	}
	type member struct {
		kind, id, sha string
		version       int
	}
	members := []member{}
	for rows.Next() {
		var m member
		if err = rows.Scan(&m.kind, &m.id, &m.version, &m.sha); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		members = append(members, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, m := range members {
		var actual, original bool
		err = f.db.QueryRowContext(f.ctx, `SELECT learning_content_approved($1,$2,$3,$4,$5),learning_approval_reference($1,$2,$3,$4,$5)`, *f.KHead(), m.kind, m.id, m.version, m.sha).Scan(&actual, &original)
		if err != nil || !actual || !original {
			t.Fatal("real approved member changed", m, actual, original, err)
		}
		kinds[m.kind] = true
	}
	if len(kinds) != 4 {
		t.Fatal("all four genuine content kinds must be compared", kinds)
	}
	cases := []struct {
		name string
		want bool
		edit func(map[string]any, map[string]any)
	}{
		{"canonical", true, nil},
		{"wrong-version", false, func(m, e map[string]any) { e["identity"].(map[string]any)["version"] = 2 }},
		{"wrong-sha", false, func(m, e map[string]any) {
			e["identity"].(map[string]any)["sha256"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}},
		{"wrong-decision", false, func(m, e map[string]any) {
			e["evidence"].(map[string]any)["decisionId"] = "99999999-9999-4999-8999-999999999999"
		}},
		{"wrong-submission", false, func(m, e map[string]any) {
			e["evidence"].(map[string]any)["submissionId"] = "99999999-9999-4999-8999-999999999999"
		}},
		{"wrong-frozen-digest", false, func(m, e map[string]any) {
			e["evidence"].(map[string]any)["frozenDigest"] = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		}},
		{"missing-id", false, func(m, e map[string]any) { delete(e["identity"].(map[string]any), "id") }},
		{"array-id", false, func(m, e map[string]any) { e["identity"].(map[string]any)["id"] = []any{f.knowledge.ID} }},
		{"object-id", false, func(m, e map[string]any) {
			e["identity"].(map[string]any)["id"] = map[string]any{"value": f.knowledge.ID}
		}},
		{"null-identity", false, func(m, e map[string]any) { e["identity"] = nil }},
		{"string-version-retains-original-decision", true, func(m, e map[string]any) { e["identity"].(map[string]any)["version"] = "1" }},
		{"decimal-version-retains-original-rejection", false, func(m, e map[string]any) { e["identity"].(map[string]any)["version"] = json.Number("1.0") }},
		{"missing-identity-before-valid", true, func(m, e map[string]any) {
			m["members"] = append([]any{map[string]any{"evidence": nil}}, m["members"].([]any)...)
		}},
		{"array-id-before-valid", true, func(m, e map[string]any) {
			m["members"] = append([]any{map[string]any{"identity": map[string]any{"id": []any{f.knowledge.ID}}}}, m["members"].([]any)...)
		}},
		{"object-id-before-valid", true, func(m, e map[string]any) {
			m["members"] = append([]any{map[string]any{"identity": map[string]any{"id": map[string]any{"value": f.knowledge.ID}}}}, m["members"].([]any)...)
		}},
		{"invalid-duplicate-before-valid", true, func(m, e map[string]any) {
			duplicate := map[string]any{"identity": map[string]any{"kind": "knowledge", "id": f.knowledge.ID, "version": 2, "sha256": f.knowledge.SHA256}, "evidence": e["evidence"]}
			m["members"] = append([]any{duplicate}, m["members"].([]any)...)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var manifest map[string]any
			if err := json.Unmarshal(original, &manifest); err != nil {
				t.Fatal(err)
			}
			var entry map[string]any
			for _, member := range manifest["members"].([]any) {
				candidate := member.(map[string]any)
				identity := candidate["identity"].(map[string]any)
				if identity["kind"] == "knowledge" && identity["id"] == f.knowledge.ID {
					entry = candidate
					break
				}
			}
			if entry == nil {
				t.Fatal("real approved fixture member missing")
			}
			if tc.edit != nil {
				tc.edit(manifest, entry)
			}
			raw, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			snapshot := f.ID()
			if _, err = f.db.ExecContext(f.ctx, `INSERT INTO publication_snapshots(id,catalogue_version,status) SELECT $1,catalogue_version,status FROM publication_snapshots WHERE id=$2`, snapshot, *f.KHead()); err != nil {
				t.Fatal(err)
			}
			if _, err = f.db.ExecContext(f.ctx, `INSERT INTO publication_members(snapshot_id,package_id,package_version,kind,id,version,availability) SELECT $1,package_id,package_version,kind,id,version,availability FROM publication_members WHERE snapshot_id=$2`, snapshot, *f.KHead()); err != nil {
				t.Fatal(err)
			}
			if _, err = f.db.ExecContext(f.ctx, `INSERT INTO content_publication_manifests(snapshot_id,base_head,manifest,manifest_bytes,sha256,diff,creator_user_id,created_at) SELECT $1,base_head,$3::jsonb,$4,encode(sha256($4),'hex'),diff,creator_user_id,created_at FROM content_publication_manifests WHERE snapshot_id=$2`, snapshot, *f.KHead(), string(raw), raw); err != nil {
				t.Fatal(err)
			}
			var actual, expected bool
			err = f.db.QueryRowContext(f.ctx, `SELECT learning_content_approved($1,'knowledge',$2,$3,$4),learning_approval_reference($1,'knowledge',$2,$3,$4)`, snapshot, f.knowledge.ID, f.knowledge.Version, f.knowledge.SHA256).Scan(&actual, &expected)
			if err != nil || actual != expected || actual != tc.want {
				t.Fatal("exact proof changed", actual, expected, tc.want, err)
			}
		})
	}
}
