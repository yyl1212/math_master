from pathlib import Path
import json,runpy
runpy.run_path('/private/tmp/math-master-p5b-source-split-diagnostic.py')
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master');o=Path('/private/tmp/math-master-p5b-capacity-overlay.json');overlay=json.loads(o.read_text())
p=Path(overlay['Replace'][str(r/'backend/internal/store/correction_jobs.go')]);s=p.read_text();a=s.index('func (s *Store) correctionSystemTx(');b=s.index('\n}',a)+2;block=s[a:b]
old='tx, e := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})'
new='''conn,e:=s.db.Conn(ctx);if e!=nil{return correctionError(e)};defer conn.Close()
 tx, e := conn.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})'''
assert old in block;block=block.replace(old,new,1)
needle='defer tx.Rollback()';assert needle in block;block=block.replace(needle,needle+'\n ctx=context.WithValue(ctx,correctionConnectionKey{},correctionTransactionConnection{tx:tx,conn:conn})',1)
s=s[:a]+block+s[b:];p.write_text(s)
p=Path(overlay['Replace'][str(r/'backend/internal/store/correction_process.go')]);s=p.read_text().replace('readAccount(ctx, tx, meta.Owner,','correctionReadAccount(ctx, tx, meta.Owner,');p.write_text(s)
code='''package store
import("context";"database/sql";"errors";"github.com/jackc/pgx/v5";"github.com/jackc/pgx/v5/stdlib";"github.com/yyl1212/math_master/backend/internal/auth")
type correctionConnectionKey struct{}
type correctionTransactionConnection struct{tx *sql.Tx;conn *sql.Conn}
func correctionReadAccount(ctx context.Context,tx *sql.Tx,id string,lock bool)(accountRow,error){
 link,ok:=ctx.Value(correctionConnectionKey{}).(correctionTransactionConnection);if !ok||link.tx!=tx{return readAccount(ctx,tx,id,lock)}
 var a accountRow;handled:=false
 err:=link.conn.Raw(func(driverConn any)(err error){
 c,ok:=driverConn.(*stdlib.Conn);if !ok{return nil};handled=true
 q:="SELECT id::text,username,password_phc,credential_version,must_change_password FROM auth_users WHERE id=$1";if lock{q+=" FOR UPDATE"}
 batch:=&pgx.Batch{};batch.Queue(q,id);batch.Queue("SELECT role FROM auth_user_roles WHERE user_id=$1 ORDER BY CASE role WHEN 'learner' THEN 1 WHEN 'editor' THEN 2 WHEN 'reviewer' THEN 3 WHEN 'admin' THEN 4 END",id)
 results:=c.Conn().SendBatch(ctx,batch);defer func(){if e:=results.Close();err==nil{err=e}}()
 if err=results.QueryRow().Scan(&a.User.ID,&a.User.Username,&a.PHC,&a.Version,&a.User.MustChangePassword);err!=nil{if errors.Is(err,pgx.ErrNoRows){return auth.ErrNotFound};return err}
 rows,err:=results.Query();if err!=nil{return err};defer rows.Close();a.User.Roles=make([]auth.Role,0,4)
 for rows.Next(){var role auth.Role;if err=rows.Scan(&role);err!=nil{return err};a.User.Roles=append(a.User.Roles,role)};return rows.Err()
 });if !handled&&err==nil{return readAccount(ctx,tx,id,lock)};return a,err
}
'''
p=Path('/private/tmp/p5b_capacity_account_batch.go');p.write_text(code);overlay['Replace'][str(r/'backend/internal/store/correction_capacity_account_batch.go')]=str(p);o.write_text(json.dumps(overlay))
print('Prepared temporary pgx account/role two-statement batch on the exact SQL transaction connection; both snapshots/order retained, no data reused, standard-driver fallback.')
