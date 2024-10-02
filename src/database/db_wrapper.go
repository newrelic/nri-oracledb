package database

import (
	"database/sql"
	"errors"

	"github.com/newrelic/infra-integrations-sdk/v3/log"

	"github.com/jmoiron/sqlx"
)

var ErrNotImplemented = errors.New("method not implemented")

type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	ScannedRowsCount() int
	MapScan(dest map[string]interface{}) error
	Columns() ([]string, error)
}

type DBWrapper struct {
	db *sqlx.DB
}

func NewDBWrapper(db *sqlx.DB) DBWrapper {
	return DBWrapper{db: db}
}

func (d *DBWrapper) Query(query string, args ...interface{}) (*RowsWrapper, error) {
	rows, err := d.db.Query(query, args...)
	return &RowsWrapper{Rows: rows}, err
}

func (d *DBWrapper) Queryx(query string, args ...interface{}) (*RowsxWrapper, error) {
	rows, err := d.db.Queryx(query, args...)
	return &RowsxWrapper{Rows: rows}, err
}

type RowsWrapper struct {
	count int
	*sql.Rows
}

func (r *RowsWrapper) Next() bool {
	n := r.Rows.Next()
	if n {
		r.count++
	}
	return n
}

func (r *RowsWrapper) Close() {
	if r.Rows == nil {
		return
	}
	err := r.Rows.Close()
	if err != nil {
		log.Error("Failed to close rows: %s", err)
	}
}

// ScannedRowsCount returns the number of rows iterated with Next
func (r *RowsWrapper) ScannedRowsCount() int {
	return r.count
}

func (r *RowsWrapper) MapScan(_ map[string]interface{}) error {
	return ErrNotImplemented
}

type RowsxWrapper struct {
	count int
	*sqlx.Rows
}

func (rx *RowsxWrapper) Next() bool {
	n := rx.Rows.Next()
	if n {
		rx.count++
	}
	return n
}

// ScannedRowsCount returns the number of rows iterated with Next
func (rx *RowsxWrapper) ScannedRowsCount() int {
	return rx.count
}

func (rx *RowsxWrapper) Close() {
	if rx.Rows == nil {
		return
	}
	err := rx.Rows.Close()
	if err != nil {
		log.Error("Failed to close rows: %s", err)
	}
}
