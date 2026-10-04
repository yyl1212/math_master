package store
import("context";"database/sql";"fmt";"time";"github.com/yyl1212/math_master/backend/internal/correction")
func CorrectionCapacityDiagnosticForTest(ctx context.Context,db *sql.DB,ref correction.EvidenceRef,owner,caseID string)([]string,error){
 tx,e:=db.BeginTx(ctx,nil);if e!=nil{return nil,e};defer tx.Rollback();out:=[]string{}
 started:=time.Now();for n:=0;n<10;n++{if _,e=correctionConfigured(ctx,tx);e!=nil{return nil,e};if _,e=learningConfigured(ctx,tx);e!=nil{return nil,e};if e=questionConfigured(ctx,tx);e!=nil{return nil,e}};out=append(out,fmt.Sprintf("DIAG health 10=%s",time.Since(started)))
 meta,e:=correctionReadEvidenceMetadata(ctx,tx,ref);if e!=nil{return nil,e};started=time.Now();for n:=0;n<10;n++{if _,e=correctionOriginalBasis(ctx,tx,meta);e!=nil{return nil,e}};out=append(out,fmt.Sprintf("DIAG original basis 10=%s",time.Since(started)))
 query:=correctionApprovedPlanMetadataSQL
 started=time.Now();for n:=0;n<10;n++{rows,e:=tx.QueryContext(ctx,query,ref.Kind,ref.ID,owner,caseID);if e!=nil{return nil,e};for rows.Next(){};e=rows.Err();rows.Close();if e!=nil{return nil,e}};out=append(out,fmt.Sprintf("DIAG applicable plans 10=%s",time.Since(started)))
 rows,e:=tx.QueryContext(ctx,"EXPLAIN (ANALYZE, BUFFERS) "+query,ref.Kind,ref.ID,owner,caseID);if e!=nil{return nil,e};defer rows.Close();for rows.Next(){var v string;if e=rows.Scan(&v);e!=nil{return nil,e};out=append(out,v)};return out,rows.Err()
}

func CorrectionBackfillDiagnosticForTest(ctx context.Context,db *sql.DB,caseID string)([]string,error){
tx,e:=db.BeginTx(ctx,nil);if e!=nil{return nil,e};defer tx.Rollback();c,e:=correctionReadCase(ctx,tx,caseID);if e!=nil{return nil,e};rows,e:=tx.QueryContext(ctx,"EXPLAIN (ANALYZE,BUFFERS) "+`WITH evidence AS (`+correctionEvidenceRowsSQL+`) SELECT e.kind,e.id::text,e.owner::text FROM correction_cases c CROSS JOIN LATERAL (SELECT e.kind,e.id,e.owner FROM evidence e WHERE e.terminal AND e.kind IN ('assessment','practice') AND `+correctionCasePredicateSQL(c.Kind)+` AND NOT EXISTS(SELECT 1 FROM correction_jobs j WHERE j.source_key='terminal:'||c.id::text||':'||e.kind||':'||e.id::text) ORDER BY e.kind,e.id LIMIT $2 OFFSET 0) e WHERE c.id=$1 ORDER BY e.kind,e.id`,caseID,48);if e!=nil{return nil,e};defer rows.Close();out:=[]string{};for rows.Next(){var v string;if e=rows.Scan(&v);e!=nil{return nil,e};out=append(out,v)};return out,rows.Err()
}
