from pathlib import Path
import json,runpy
runpy.run_path('/private/tmp/math-master-p5b-native-source-diagnostic.py')
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master');o=Path('/private/tmp/math-master-p5b-capacity-overlay.json');overlay=json.loads(o.read_text())
p=Path(overlay['Replace'][str(r/'backend/internal/store/notification_write.go')]);s=p.read_text();a=s.index('\tif !question.ValidID(owner)',s.index('func notificationAppend('));b=s.index('\n\t_, e :=',a);validation=s[a:b]
s=s[:a]+'\tif e:=notificationValidateSource(owner,in);e!=nil{return e}'+s[b:]
pos=s.index('func notificationAppend(');s=s[:pos]+'func notificationValidateSource(owner string,in notification.Source)error {\n'+validation+'\nreturn nil\n}\n'+s[pos:];p.write_text(s)
p=Path('/private/tmp/p5b_capacity_account_batch.go');s=p.read_text().replace('internal/auth")','internal/auth";"github.com/yyl1212/math_master/backend/internal/notification")')
a=s.index('func correctionReadAccount(')
s=s[:a]+'''func correctionReadAccount(ctx context.Context,tx *sql.Tx,id string,lock bool)(accountRow,error){return correctionAccountBatch(ctx,tx,id,lock,nil)}
func correctionNotifyAccount(ctx context.Context,tx *sql.Tx,owner string,in notification.Source)(accountRow,error){if e:=notificationValidateSource(owner,in);e!=nil{return accountRow{},e};return correctionAccountBatch(ctx,tx,owner,false,&in)}
'''+s[a:].replace('func correctionReadAccount(ctx context.Context,tx *sql.Tx,id string,lock bool)','func correctionAccountBatch(ctx context.Context,tx *sql.Tx,id string,lock bool,notice *notification.Source)',1)
s=s.replace('return readAccount(ctx,tx,id,lock)}','if notice!=nil{if e:=notificationAppend(ctx,tx,id,*notice);e!=nil{return accountRow{},e}};return readAccount(ctx,tx,id,lock)}')
s=s.replace('batch:=&pgx.Batch{};batch.Queue(q,id);','''batch:=&pgx.Batch{};if notice!=nil{batch.Queue(`INSERT INTO notifications(owner_user_id,dedup_key,type,evidence_kind,evidence_id,case_id,result_id) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(dedup_key) DO NOTHING`,id,notice.DedupKey,notice.Type,notice.Evidence.Kind,notice.Evidence.ID,notice.CaseID,notice.ResultID)};batch.Queue(q,id);''')
s=s.replace('if err=results.QueryRow().Scan','if notice!=nil{if _,err=results.Exec();err!=nil{return err}}\n if err=results.QueryRow().Scan')
p.write_text(s)
p=Path(overlay['Replace'][str(r/'backend/internal/store/correction_process.go')]);s=p.read_text();a=s.index('\tif e = notificationAppend(ctx, tx, meta.Owner, notification.Source{DedupKey: "result:"');b=s.index('\n\tif e != nil {',s.index('account, e = correctionReadAccount',a))
s=s[:a]+''' // The notice and final identity remain separate SQL statements in order.
 account,e=correctionNotifyAccount(ctx,tx,meta.Owner,notification.Source{DedupKey:"result:"+rid,Type:typ,Evidence:ref,CaseID:j.Meta.CaseID,ResultID:&rid})'''+s[b:];p.write_text(s)
o.write_text(json.dumps(overlay))
print('Prepared temporary three-statement batch: original notice INSERT, final account SELECT, then role SELECT; validation unchanged, final authorization and job lock/time fence remain after it.')
