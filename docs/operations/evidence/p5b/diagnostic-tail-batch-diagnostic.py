from pathlib import Path
import json,runpy
runpy.run_path('/private/tmp/math-master-p5b-start-prefetch-diagnostic.py')
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master');o=Path('/private/tmp/math-master-p5b-capacity-overlay.json');overlay=json.loads(o.read_text())
p=Path('/private/tmp/p5b-start-prefetch.go');s=p.read_text();s=s.replace('"github.com/yyl1212/math_master/backend/internal/auth";','"github.com/yyl1212/math_master/backend/internal/auth";"github.com/yyl1212/math_master/backend/internal/notification";')
s+=r'''
type correctionTailFactsKey struct{}
type correctionTailFacts struct{tx *sql.Tx;j correctionJobRecord;now time.Time}
func correctionNotifyAndFinish(ctx context.Context,tx *sql.Tx,l correction.Lease,owner string,in notification.Source)(context.Context,error){
 if !question.ValidID(owner)||!question.ValidID(in.CaseID)||!correction.ValidEvidence(in.Evidence,true)||in.DedupKey==""||len(in.DedupKey)>512||in.ResultID!=nil&&!question.ValidID(*in.ResultID){return ctx,auth.ErrInvalidInput}
 switch in.Type{case notification.Checking,notification.Corrected,notification.Retake,notification.ReviewMaterial,notification.PathUnavailable:default:return ctx,auth.ErrInvalidInput}
 b:=&pgx.Batch{}
 b.Queue(`INSERT INTO notifications(owner_user_id,dedup_key,type,evidence_kind,evidence_id,case_id,result_id) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(dedup_key) DO NOTHING`,owner,in.DedupKey,in.Type,in.Evidence.Kind,in.Evidence.ID,in.CaseID,in.ResultID)
 b.Queue(`SELECT id::text,must_change_password FROM auth_users WHERE id=$1`,owner)
 b.Queue(`WITH locked_job AS MATERIALIZED (SELECT `+correctionJobColumns+` FROM correction_jobs WHERE id=$1 AND EXISTS(SELECT 1 FROM auth_users WHERE id=$2 AND NOT must_change_password) FOR UPDATE) SELECT locked_job.*,clock_timestamp() FROM locked_job`,l.JobID,owner)
 facts:=&correctionTailFacts{tx:tx};var account accountRow
 supported,e:=correctionReadBatch(ctx,tx,b,func(results pgx.BatchResults)error{
  if _,e:=results.Exec();e!=nil{return e};if e:=results.QueryRow().Scan(&account.User.ID,&account.User.MustChangePassword);e!=nil{return workflowRowError(e)};if e:=correction.Authorize(account.User,correction.ReadOwnAction);e!=nil{return e}
  var e error;facts.j,e=correctionScanJob(results.QueryRow(),&facts.now);return workflowRowError(e)
 });if !supported&&e==nil{if e=notificationAppend(ctx,tx,owner,in);e!=nil{return ctx,e};account,e=correctionReadWorkerOwner(ctx,tx,owner,false);if e!=nil{return ctx,e};if e=correction.Authorize(account.User,correction.ReadOwnAction);e!=nil{return ctx,e};facts.j,facts.now,e=correctionReadJobTime(ctx,tx,l.JobID)};if e!=nil{return ctx,e};return context.WithValue(ctx,correctionTailFactsKey{},facts),nil
}
'''
p.write_text(s)
p=Path(overlay['Replace'][str(r/'backend/internal/store/correction_process.go')]);s=p.read_text()
a='\tj, now, e := correctionReadJobTime(ctx, tx, l.JobID)';assert s.count(a)==1
s=s.replace(a,'''\tvar j correctionJobRecord;var now time.Time;var e error
 if facts,ok:=ctx.Value(correctionTailFactsKey{}).(*correctionTailFacts);ok&&facts.tx==tx&&facts.j.Meta.ID==l.JobID{j,now=facts.j,facts.now}else{j,now,e=correctionReadJobTime(ctx,tx,l.JobID)}''')
a=s.index('\tif e = notificationAppend(ctx, tx, meta.Owner, notification.Source{DedupKey: "result:"')
b=s.index('\treturn correctionAdvanceCursor(ctx, tx, l, ref)',a)
s=s[:a]+'''\tctx,e=correctionNotifyAndFinish(ctx,tx,l,meta.Owner,notification.Source{DedupKey:"result:"+rid,Type:typ,Evidence:ref,CaseID:j.Meta.CaseID,ResultID:&rid})
 if e!=nil{return e}
'''+s[b:]
p.write_text(s);o.write_text(json.dumps(overlay))
print('Temporary candidate tail: original notification statement, fresh minimal ReadOwn account, independently locked job/clock; account policy failure gates job lock; actual tail row consumed only in same tx/id.')
