package postgres

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// New opens a PostgreSQL connection pool using the given DSN.
func New(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return db, nil
}
