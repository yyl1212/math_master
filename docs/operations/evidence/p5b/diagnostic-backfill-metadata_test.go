package store
import("context";"database/sql")
func CorrectionBackfillDiagnosticForTest(ctx context.Context,db *sql.DB,caseID string)([]string,error){
 tx,e:=db.BeginTx(ctx,nil);if e!=nil{return nil,e};defer tx.Rollback();c,e:=correctionReadCase(ctx,tx,caseID);if e!=nil{return nil,e};rows,e:=tx.QueryContext(ctx,"EXPLAIN (ANALYZE,BUFFERS) "+correctionBackfillTerminalSQL(c.Kind),caseID,48);if e!=nil{return nil,e};defer rows.Close();out:=[]string{};for rows.Next(){var v string;if e=rows.Scan(&v);e!=nil{return nil,e};out=append(out,v)};return out,rows.Err()
}
