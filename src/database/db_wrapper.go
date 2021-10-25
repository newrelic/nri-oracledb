package database

import (
	"database/sql"
	"github.com/jmoiron/sqlx"
)

type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	ScannedRowsCount() int
	MapScan(dest map[string]interface{}) error
	Columns() ([]string, error)
	Close() error
}

type DBWrapper struct {
	db *sqlx.DB
}

func NewDBWrapper(db *sqlx.DB) DBWrapper {
	return DBWrapper{db: db}
}

func (d *DBWrapper) Query(query string, args ...interface{}) (*RowsWrapper, error) {
	rows, err := d.db.Query(query, args...)
	return &RowsWrapper{rows: rows}, err
}

func (d *DBWrapper) Queryx(query string, args ...interface{}) (*RowsxWrapper, error) {
	rows, err := d.db.Queryx(query, args...)
	return &RowsxWrapper{rows: rows}, err
}

type RowsWrapper struct {
	count int
	rows  *sql.Rows
}

func (r *RowsWrapper) Next() bool {
	n := r.rows.Next()
	if n {
		r.count++
	}
	return n
}

// ScannedRowsCount returns the number of rows iterated with Next
func (r *RowsWrapper) ScannedRowsCount() int {
	return r.count
}

func (r *RowsWrapper) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *RowsWrapper) Columns() ([]string, error) {
	return r.rows.Columns()
}

func (r *RowsWrapper) MapScan(_ map[string]interface{}) error {
	return nil
}

func (r *RowsWrapper) Close() error {
	return r.rows.Close()
}

type RowsxWrapper struct {
	count int
	rows  *sqlx.Rows
}

func (rx *RowsxWrapper) Next() bool {
	n := rx.rows.Next()
	if n {
		rx.count++
	}
	return n
}

func (rx *RowsxWrapper) Scan(dest ...interface{}) error {
	return rx.rows.Scan(dest...)
}

// ScannedRowsCount returns the number of rows iterated with Next
func (rx *RowsxWrapper) ScannedRowsCount() int {
	return rx.count
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
