package store

import (
 "time"
	"context"
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/yyl1212/math_master/backend/internal/auth"
 "github.com/yyl1212/math_master/backend/internal/correction"
)

type correctionConnectionKey struct{}
type correctionTransactionConnection struct {
	tx   *sql.Tx
	conn *sql.Conn
}

// 批次只使用准确事务对应的连接；原始driver不离开Raw回调。
// 关闭所有结果后才能再次调用sql.Tx。非pgx和普通读者沿用原查询。
func correctionReadBatch(ctx context.Context, tx *sql.Tx, batch *pgx.Batch, consume func(pgx.BatchResults) error) (bool, error) {
 capacityDiagnosticStarted:=time.Now();defer capacityDiagnosticRecord("correctionReadBatch",capacityDiagnosticStarted);

	link, ok := ctx.Value(correctionConnectionKey{}).(correctionTransactionConnection)
	if !ok || link.tx != tx {
		return false, nil
	}
	supported := false
	err := link.conn.Raw(func(driverConn any) (err error) {
		conn, ok := driverConn.(*stdlib.Conn)
		if !ok {
			return nil
		}
		supported = true
		results := conn.Conn().SendBatch(ctx, batch)
		defer func() {
			if closeErr := results.Close(); err == nil {
				err = closeErr
			}
		}()
		return consume(results)
	})
	return supported, err
}

func correctionReadAccount(ctx context.Context, tx *sql.Tx, id string, lock bool) (accountRow, error) {
 capacityDiagnosticStarted:=time.Now();defer capacityDiagnosticRecord("correctionReadAccount",capacityDiagnosticStarted);

	var account accountRow
	query := "SELECT id::text,username,password_phc,credential_version,must_change_password FROM auth_users WHERE id=$1"
	if lock {
		query += " FOR UPDATE"
	}
	batch := &pgx.Batch{}
	batch.Queue(query, id)
	// 必须是账户行等待完成之后的另一条语句，保留新的角色快照。
	batch.Queue("SELECT role FROM auth_user_roles WHERE user_id=$1 ORDER BY CASE role WHEN 'learner' THEN 1 WHEN 'editor' THEN 2 WHEN 'reviewer' THEN 3 WHEN 'admin' THEN 4 END", id)
	supported, err := correctionReadBatch(ctx, tx, batch, func(results pgx.BatchResults) error {
		if err := results.QueryRow().Scan(&account.User.ID, &account.User.Username, &account.PHC, &account.Version, &account.User.MustChangePassword); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return auth.ErrNotFound
			}
			return err
		}
		rows, err := results.Query()
		if err != nil {
			return err
		}
		defer rows.Close()
		account.User.Roles = make([]auth.Role, 0, 4)
		for rows.Next() {
			var role auth.Role
			if err = rows.Scan(&role); err != nil {
				return err
			}
			account.User.Roles = append(account.User.Roles, role)
		}
		return rows.Err()
	})
	if !supported && err == nil {
		return readAccount(ctx, tx, id, lock)
	}
	return account, err
}

// worker本人策略只依赖ID与当前换密码状态；人员角色认证仍用原完整读。
func correctionReadWorkerOwner(ctx context.Context, tx *sql.Tx, id string, lock bool) (accountRow, error) {
	var a accountRow
	q := `SELECT id::text,must_change_password FROM auth_users WHERE id=$1`
	if lock {
		q += ` FOR UPDATE`
	}
	e := tx.QueryRowContext(ctx, q, id).Scan(&a.User.ID, &a.User.MustChangePassword)
	return a, workflowRowError(e)
}

func correctionWorkerOwnerAndCase(ctx context.Context,tx *sql.Tx,owner,caseID string)(correction.CaseKind,error){
 b:=&pgx.Batch{}
 b.Queue(`SELECT id::text,must_change_password FROM auth_users WHERE id=$1 FOR UPDATE`,owner)
 b.Queue(`SELECT kind FROM correction_cases WHERE id=$1 AND sealed AND EXISTS(SELECT 1 FROM auth_users WHERE id=$2 AND NOT must_change_password) FOR UPDATE`,caseID,owner)
 var a accountRow;var kind correction.CaseKind
 supported,e:=correctionReadBatch(ctx,tx,b,func(results pgx.BatchResults)error{
 if e:=results.QueryRow().Scan(&a.User.ID,&a.User.MustChangePassword);e!=nil{return workflowRowError(e)}
 if e:=correction.Authorize(a.User,correction.ReadOwnAction);e!=nil{return e}
 return workflowRowError(results.QueryRow().Scan(&kind))
 })
 if !supported && e==nil {
 a,e=correctionReadWorkerOwner(ctx,tx,owner,true);if e!=nil{return kind,e}
 if e=correction.Authorize(a.User,correction.ReadOwnAction);e!=nil{return kind,e}
 kind,e=correctionLockCaseKind(ctx,tx,caseID)
 }
 return kind,e
}
