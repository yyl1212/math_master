from pathlib import Path
import json,runpy
runpy.run_path('/private/tmp/math-master-p5b-phase-diagnostic.py')
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master');o=Path('/private/tmp/math-master-p5b-capacity-overlay.json');overlay=json.loads(o.read_text())
for name in ['correction_results.go','correction_process.go']:
 p=Path(overlay['Replace'][str(r/'backend/internal/store'/name)]);s=p.read_text()
 if name=='correction_results.go':
  start=s.index('\t// One statement keeps the same per-row FK');end=s.index('\n}\n\nfunc correctionPositionMap',start)
  s=s[:start]+'''
 eventRaw,eventHash,e:=correction.Canonical("correction-command-v1",map[string]any{"digest":h});if e!=nil{return "",e}
 eventID,e:=workflowID();if e!=nil{return "",e}
 // The parent was inserted by the previous statement. Every row guard and
 // deferred completeness check remains active. Producer dependencies finish
 // the inserts before sealing; the following notice sees the sealed result.
 _,e=tx.ExecContext(ctx,`WITH dependencies AS (
 INSERT INTO correction_dependencies(result_id,role,kind,id,version,sha256,positions) SELECT $1,d.role,d.kind,d.id,d.version,d.sha256,d.positions FROM jsonb_to_recordset($2::jsonb) AS d(role text,kind text,id text,version integer,sha256 text,positions integer[]) RETURNING result_id
 ),audit AS (
 INSERT INTO correction_events(id,subject_kind,subject_id,subject_version,case_id,kind,sequence,actor_user_id,body,body_bytes,body_digest)
 SELECT $3,'result',$1,NULL,$4,'result_sealed',1,NULL,$5,$6,$7 FROM (SELECT count(*) FROM dependencies) ready RETURNING subject_id
 ) UPDATE correction_results SET sealed=true FROM audit WHERE correction_results.id=$1 AND audit.subject_id=correction_results.id`,id,body(rows),eventID,j.Meta.CaseID,string(eventRaw),eventRaw,eventHash)
 return id,e
'''+s[end:]
 else:
  start=s.index('\t_, e = tx.ExecContext(ctx, `UPDATE correction_jobs SET cursor_kind');end=s.index('\n}\nfunc correctionProcessEvidence',start)
  s=s[:start]+'''
 raw,h,e:=correction.Canonical("correction-command-v1",map[string]any{"cursor":correction.ScanKey{Kind:ref.Kind,ID:ref.ID},"processedCount":j.Meta.ProcessedCount+1});if e!=nil{return e}
 eventID,e:=workflowID();if e!=nil{return e}
 _,e=tx.ExecContext(ctx,`WITH advanced AS (
 UPDATE correction_jobs SET cursor_kind=$2,cursor_id=$3,processed_count=processed_count+1,sequence=$4 WHERE id=$1 RETURNING id
 ) INSERT INTO correction_events(id,subject_kind,subject_id,subject_version,case_id,kind,sequence,actor_user_id,body,body_bytes,body_digest)
 SELECT $5,'job',advanced.id,NULL,$6,'job_continued',$4,NULL,$7,$8,$9 FROM advanced`,l.JobID,ref.Kind,ref.ID,seq,eventID,j.Meta.CaseID,string(raw),raw,h)
 return e
'''+s[end:]
 p.write_text(s)
o.write_text(json.dumps(overlay))
print('Prepared temporary-only result dependency/audit/seal and cursor/audit producer groups; no guards/migration/product changed.')
