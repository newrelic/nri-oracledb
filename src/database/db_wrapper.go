package database

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/log"
)

type DBWrapper struct {
	db *sqlx.DB
}

func NewDBWrapper(db *sqlx.DB) DBWrapper {
	return DBWrapper{db: db}
}

func (d *DBWrapper) Query(query string, args ...interface{}) (*RowsWrapper, error) {
	rows, err := d.db.Query(query, args...)
	return &RowsWrapper{rows: rows, executedQuery: query}, err
}

func (d *DBWrapper) Queryx(query string, args ...interface{}) (*RowsxWrapper, error) {
	rows, err := d.db.Queryx(query, args...)
	return &RowsxWrapper{rows: rows, executedQuery: query}, err
}

type RowsWrapper struct {
	count         int
	rows          *sql.Rows
	executedQuery string
}

func (r *RowsWrapper) Next() bool {
	n := r.rows.Next()
	if !n && r.count < 1 {
		log.Warn("Query did not return any results: %s", r.executedQuery)
	}
	r.count++
	return n
}

func (r *RowsWrapper) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *RowsWrapper) Columns() ([]string, error) {
	return r.rows.Columns()
}

func (r *RowsWrapper) Close() error {
	return r.rows.Close()
}

type RowsxWrapper struct {
	count         int
	rows          *sqlx.Rows
	executedQuery string
}

func (rx *RowsxWrapper) Next() bool {
	n := rx.rows.Next()
	if !n && rx.count < 1 {
		log.Warn("Query did not return any results: %s", rx.executedQuery)
	}
	rx.count++
	return n
}

func (rx *RowsxWrapper) Scan(dest ...interface{}) error {
	return rx.rows.Scan(dest...)
}

func (rx *RowsxWrapper) MapScan(dest map[string]interface{}) error {
	return rx.rows.MapScan(dest)
}

func (rx *RowsxWrapper) Columns() ([]string, error) {
	return rx.rows.Columns()
}

func (rx *RowsxWrapper) Close() error {
	return rx.rows.Close()
}
