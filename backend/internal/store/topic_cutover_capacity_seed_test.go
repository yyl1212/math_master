package store_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"testing"
	"time"
)

// These rows and writers are private to the legacy capacity test. All original
// canonical factories, table constraints and deferred evidence guards apply.
type capacitySeedRow struct {
	ID          string          `json:"id"`
	Owner       string          `json:"owner"`
	Knowledge   string          `json:"kid"`
	Version     int             `json:"version"`
	SHA         string          `json:"sha"`
	Publication string          `json:"publication"`
	Seal        json.RawMessage `json:"seal"`
	Bytes       string          `json:"bytes"`
	Digest      string          `json:"digest"`
	At          time.Time       `json:"at"`
}

type capacitySeedTimes struct {
	Events, Records, Dependencies, Commit time.Duration
}

func writeCapacitySeedBatch(ctx context.Context, db *sql.DB, values []capacitySeedRow) (out capacitySeedTimes, err error) {
	if len(values) == 0 || len(values) > 2000 {
		return out, errors.New("capacity seed batch must contain 1 to 2000 events")
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return out, err
	}
	ids := make([]string, len(values))
	for i, value := range values {
		ids[i] = value.ID
	}
	idBytes, err := json.Marshal(ids)
	if err != nil {
		return out, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	const selected = `jsonb_to_recordset($1::jsonb) x(id uuid,owner uuid,kid text,version integer,sha text,publication text,seal jsonb,bytes text,digest text,at timestamptz)`
	start := time.Now()
	if _, err = tx.ExecContext(ctx, `INSERT INTO learning_events(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,kind,seal,seal_bytes,seal_sha256,recorded_at) SELECT id,owner,kid,version,sha,publication,'started',seal,decode(bytes,'base64'),digest,at FROM `+selected, string(raw)); err != nil {
		return out, err
	}
	out.Events = time.Since(start)
	start = time.Now()
	// Separate statements see the inserted rows. Ordinality retains the input
	// order; only the small ID list crosses the wire again.
	const batch = `SELECT e.*,x.n FROM jsonb_array_elements_text($1::jsonb) WITH ORDINALITY x(id,n) JOIN learning_events e ON e.id=x.id::uuid`
	if _, err = tx.ExecContext(ctx, `INSERT INTO learning_records(owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,started_event_id,started_at) SELECT owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,id,recorded_at FROM (`+batch+`) b ORDER BY n`, string(idBytes)); err != nil {
		return out, err
	}
	out.Records = time.Since(start)
	start = time.Now()
	// Dependency derivation depends on exact knowledge/units/assets, not actor,
	// event ID or time. Derive once per equal material using an actual seal;
	// every original per-event dependency/approval guard still runs at commit.
	if _, err = tx.ExecContext(ctx, `WITH batch AS MATERIALIZED (`+batch+`),
 materials AS MATERIALIZED (
  SELECT DISTINCT ON (seal#>'{body,knowledge}',seal#>'{body,units}',seal#>'{body,assets}')
   seal,seal#>'{body,knowledge}' AS knowledge,seal#>'{body,units}' AS units,seal#>'{body,assets}' AS assets FROM batch),
 deps AS MATERIALIZED (
  SELECT m.knowledge,m.units,m.assets,d.kind,d.id,d.version,d.sha256
  FROM materials m CROSS JOIN LATERAL learning_seal_dependencies(m.seal,true) d)
 INSERT INTO learning_evidence_dependencies(evidence_kind,evidence_id,owner_user_id,kind,id,version,sha256)
 SELECT 'learning-event',b.id,b.owner_user_id,d.kind,d.id,d.version,d.sha256
 FROM batch b JOIN deps d ON d.knowledge=b.seal#>'{body,knowledge}' AND d.units=b.seal#>'{body,units}' AND d.assets=b.seal#>'{body,assets}'
 ORDER BY b.n,d.kind,d.id,d.version,d.sha256`, string(idBytes)); err != nil {
		return out, err
	}
	out.Dependencies = time.Since(start)
	start = time.Now()
	err = tx.Commit()
	out.Commit = time.Since(start)
	return out, err
}

func capacitySeedSample(t *testing.T) (*workflowFixture, []capacitySeedRow) {
	t.Helper()
	f := newWorkflowFixture(t)
	capacity, err := testutil.CapacityContent("../content/testdata", "../../../content/catalogue/domains.json")
	if err != nil {
		t.Fatal(err)
	}
	in := capacity.Inputs[0]
	for _, k := range in.Package.Knowledge {
		link := f.Input().SourceMap[0]
		link.Knowledge = content.VersionRef{ID: k.ID, Version: k.Version}
		link.LegacyID = k.ID
		in.SourceMap = append(in.SourceMap, link)
	}
	sub := f.ApprovedInput(in)
	p := f.Activate(f.Prepare(sub, nil), nil)
	var out []capacitySeedRow
	for _, k := range in.Package.Knowledge[:2] {
		ref := question.Identity{ID: k.ID, Version: k.Version}
		if err := f.db.QueryRow(`SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=$2`, ref.ID, ref.Version).Scan(&ref.SHA256); err != nil {
			t.Fatal(err)
		}
		for index, owner := range []string{f.ids["author_b"], f.ids["author_a"], f.ids["admin_a"]} {
			id := f.ID()
			at := time.Date(2026, 1, 1, 0, 0, 0, len(out)*1000, time.UTC)
			units, assets := []question.Identity{}, []question.AssetRef{}
			if index == 1 {
				for _, u := range in.Package.Units {
					if u.Knowledge.ID == k.ID {
						var unit question.Identity
						unit.ID, unit.Version = u.ID, u.Version
						if err := f.db.QueryRow("SELECT sha256 FROM unit_versions WHERE id=$1 AND version=$2", unit.ID, unit.Version).Scan(&unit.SHA256); err != nil {
							t.Fatal(err)
						}
						units = append(units, unit)
						break
					}
				}
			}
			if index == 2 {
				for _, a := range in.Package.Assets {
					if a.Knowledge.ID == k.ID {
						assets = append(assets, question.AssetRef{ID: a.ID, SHA256: a.SHA256})
						break
					}
				}
			}
			raw, digest, err := learning.CanonicalLearningEvent(learning.EventSeal{ID: id, ActorID: owner, Knowledge: ref, KnowledgePublicationID: p.ID, Units: units, Assets: assets, Kind: "started", RecordedAt: at})
			if err != nil {
				t.Fatal(err)
			}
			out = append(out, capacitySeedRow{ID: id, Owner: owner, Knowledge: ref.ID, Version: ref.Version, SHA: ref.SHA256, Publication: p.ID, Seal: raw, Bytes: base64.StdEncoding.EncodeToString(raw), Digest: digest, At: at})
		}
	}
	return f, out
}

// Catches dropping, misbinding or re-encoding canonical facts when the writer
// derives records/dependencies from events rather than repeated input JSON.
func TestTopicCapacitySeedRoundTrip(t *testing.T) {
	f, values := capacitySeedSample(t)
	if _, err := writeCapacitySeedBatch(context.Background(), f.db, values); err != nil {
		t.Fatal(err)
	}
	if f.count(`SELECT count(*) FROM learning_events`) != 6 || f.count(`SELECT count(*) FROM learning_records`) != 6 || f.count(`SELECT count(*) FROM learning_evidence_dependencies`) != 12 {
		t.Fatal("batch omitted or duplicated facts")
	}
	for _, want := range values {
		var raw []byte
		var digest, owner, knowledge, publication string
		var version int
		var at time.Time
		if err := f.db.QueryRow(`SELECT e.seal_bytes,e.seal_sha256,r.owner_user_id::text,r.knowledge_id,r.knowledge_version,e.knowledge_publication_id,r.started_at FROM learning_events e JOIN learning_records r ON r.started_event_id=e.id WHERE e.id=$1`, want.ID).Scan(&raw, &digest, &owner, &knowledge, &version, &publication, &at); err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(want.Seal)
		if !bytes.Equal(raw, want.Seal) || digest != hex.EncodeToString(hash[:]) || digest != want.Digest || owner != want.Owner || knowledge != want.Knowledge || version != want.Version || publication != want.Publication || !at.Equal(want.At) {
			t.Fatal("canonical bytes, digest, identity or timestamp changed")
		}
		if f.count(`SELECT count(*) FROM learning_evidence_dependencies WHERE evidence_kind='learning-event' AND evidence_id=$1 AND owner_user_id=$2 AND kind='knowledge' AND id=$3 AND version=$4 AND sha256=$5`, want.ID, want.Owner, want.Knowledge, want.Version, want.SHA) != 1 {
			t.Fatal("dependency crossed user or knowledge")
		}
	}
}

// Immediate constraints and deferred evidence/approval guards must both abort
// the whole batch, including facts inserted before the invalid final row.
func TestTopicCapacitySeedRollback(t *testing.T) {
	for _, failure := range []string{"digest", "approval"} {
		t.Run(failure, func(t *testing.T) {
			f, values := capacitySeedSample(t)
			bad := &values[len(values)-1]
			if failure == "digest" {
				bad.Digest = values[0].Digest
			} else {
				var seal learning.EventSeal
				// Keep a valid canonical envelope and existing publication ID, but
				// that source contains no approved knowledge: only commit rejects it.
				var decoded struct {
					Body learning.EventSeal `json:"body"`
				}
				if err := json.Unmarshal(bad.Seal, &decoded); err != nil {
					t.Fatal(err)
				}
				seal = decoded.Body
				seal.KnowledgePublicationID = f.ID()
				f.exec(`INSERT INTO publication_snapshots(id,catalogue_version,status) VALUES($1,1,'published')`, seal.KnowledgePublicationID)
				raw, digest, err := learning.CanonicalLearningEvent(seal)
				if err != nil {
					t.Fatal(err)
				}
				bad.Publication, bad.Seal, bad.Bytes, bad.Digest = seal.KnowledgePublicationID, raw, base64.StdEncoding.EncodeToString(raw), digest
			}
			times, err := writeCapacitySeedBatch(context.Background(), f.db, values)
			if err == nil {
				t.Fatal("invalid batch accepted")
			}
			if failure == "approval" && times.Commit == 0 {
				t.Fatal("approval case did not reach the original deferred guard")
			}
			if f.count(`SELECT count(*) FROM learning_events`) != 0 || f.count(`SELECT count(*) FROM learning_records`) != 0 || f.count(`SELECT count(*) FROM learning_evidence_dependencies`) != 0 {
				t.Fatal("failed batch left partial facts")
			}
		})
	}
}

func TestTopicCapacitySeedBatchBound(t *testing.T) {
	// Invalid sizes must fail before acquiring a database connection or
	// allocating/serializing an unbounded event JSON payload.
	for _, size := range []int{0, 2001} {
		if _, err := writeCapacitySeedBatch(context.Background(), nil, make([]capacitySeedRow, size)); err == nil {
			t.Fatal("unbounded or empty batch accepted", size)
		}
	}
}
