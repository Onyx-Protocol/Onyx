package pg

import (
	"database/sql"

	"golang.org/x/net/context"

	"chain/log"
)

type Logger struct {
	DB
	ctx context.Context
}

func (l *Logger) Query(query string, args ...interface{}) (*sql.Rows, error) {
	log.Write(l.ctx, log.KeyMessage, "db query", "query", query, "args", args)
	return l.DB.Query(query, args...)
}

func (l *Logger) QueryRow(query string, args ...interface{}) *sql.Row {
	log.Write(l.ctx, log.KeyMessage, "db query row", "query", query, "args", args)
	return l.DB.QueryRow(query, args...)
}

func (l *Logger) Exec(query string, args ...interface{}) (sql.Result, error) {
	log.Write(l.ctx, log.KeyMessage, "db exec", "query", query, "args", args)
	return l.DB.Exec(query, args...)
}

func (l *Logger) Begin() (Tx, error) {
	log.Write(l.ctx, log.KeyMessage, "db begin tx")
	return begin(l.DB)
}

func (l *Logger) Commit() error {
	log.Write(l.ctx, log.KeyMessage, "db commit tx")
	return l.DB.(Committer).Commit()
}

func (l *Logger) Rollback() error {
	log.Write(l.ctx, log.KeyMessage, "db rollback tx")
	return l.DB.(Committer).Rollback()
}
