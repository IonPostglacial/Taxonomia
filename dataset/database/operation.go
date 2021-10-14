package database

import "database/sql"

type DatabaseOperation struct {
	db  *sql.DB
	tx  *sql.Tx
	err error
}

func NewDatabaseOperation(db *sql.DB) *DatabaseOperation {
	tx, err := db.Begin()
	return &DatabaseOperation{db: db, tx: tx, err: err}
}

func (op *DatabaseOperation) HasFailed() bool {
	return op.err != nil
}

func (op *DatabaseOperation) Close() {
	if op.HasFailed() {
		op.tx.Rollback()
	} else {
		op.tx.Commit()
	}
}

func (op *DatabaseOperation) Error() error {
	return op.err
}

func (op *DatabaseOperation) TryPrepare(query string) (stmt *sql.Stmt) {
	if op.err != nil {
		return nil
	}
	stmt, op.err = op.tx.Prepare(query)
	return
}

func (op *DatabaseOperation) TryExec(stmt *sql.Stmt, args ...interface{}) (res sql.Result) {
	if op.err != nil {
		return nil
	}
	res, op.err = stmt.Exec(args...)
	return
}

func (op *DatabaseOperation) TryQuery(stmt *sql.Stmt, args ...interface{}) (rows *sql.Rows) {
	if op.err != nil {
		return nil
	}
	rows, op.err = stmt.Query(args...)
	return
}
