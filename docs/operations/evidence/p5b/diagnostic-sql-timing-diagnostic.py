from pathlib import Path
import json,runpy
runpy.run_path('/private/tmp/math-master-p5b-write-group-diagnostic.py')
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master');o=Path('/private/tmp/math-master-p5b-capacity-overlay.json');overlay=json.loads(o.read_text())
for source,target in list(overlay['Replace'].items()):
 if source.endswith('_test.go') or 'capacity_timing' in source:continue
 p=Path(target);s=p.read_text()
 s=s.replace('tx.QueryRowContext(ctx,','capacityDiagnosticQueryRow(ctx,tx,').replace('tx.QueryContext(ctx,','capacityDiagnosticQuery(ctx,tx,').replace('tx.ExecContext(ctx,','capacityDiagnosticExec(ctx,tx,').replace('tx.Commit()','capacityDiagnosticCommit(tx)')
 p.write_text(s)
q=Path('/private/tmp/p5b_capacity_sql_timing.go');q.write_text('''package store
import("context";"database/sql";"crypto/sha256";"fmt";"time")
func capacityDiagnosticSQLLabel(q string)string{h:=sha256.Sum256([]byte(q));return fmt.Sprintf("SQL_%x_%s",h[:6],q)}
func capacityDiagnosticQueryRow(ctx context.Context,tx *sql.Tx,q string,args ...any)*sql.Row{start:=time.Now();defer capacityDiagnosticRecord(capacityDiagnosticSQLLabel(q),start);return tx.QueryRowContext(ctx,q,args...)}
func capacityDiagnosticQuery(ctx context.Context,tx *sql.Tx,q string,args ...any)(*sql.Rows,error){start:=time.Now();defer capacityDiagnosticRecord(capacityDiagnosticSQLLabel(q),start);return tx.QueryContext(ctx,q,args...)}
func capacityDiagnosticExec(ctx context.Context,tx *sql.Tx,q string,args ...any)(sql.Result,error){start:=time.Now();defer capacityDiagnosticRecord(capacityDiagnosticSQLLabel(q),start);return tx.ExecContext(ctx,q,args...)}
func capacityDiagnosticCommit(tx *sql.Tx)error{start:=time.Now();defer capacityDiagnosticRecord("SQL_COMMIT",start);return tx.Commit()}
''')
overlay['Replace'][str(r/'backend/internal/store/correction_capacity_sql_timing.go')]=str(q);o.write_text(json.dumps(overlay))
print('Prepared temporary exact SQL latency fingerprints (no argument/credential logging), with current write-group candidate.')
