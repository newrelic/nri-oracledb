package database

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestRowsWrapper_Next(t *testing.T) {
	const query = `SELECT a.TABLESPACE_NAME.*`

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name          string
		returnedRows  *sqlmock.Rows
		errorExpected error
		countExpected int
	}{
		{
			name: "when one row is returned the count must be 1",
			returnedRows: sqlmock.NewRows([]string{"TABLESPACE_NAME", "USED", "OFFLINE", "SIZE", "USED_PERCENT"}).
				AddRow("testtablespace", 1234, 0, 4321, 12),
			errorExpected: nil,
			countExpected: 1,
		},
		{
			name:          "when no rows are returned the count must be 0",
			returnedRows:  &sqlmock.Rows{},
			errorExpected: nil,
			countExpected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectQuery(query).WillReturnRows(tt.returnedRows)

			sqlxDb := sqlx.NewDb(db, "sqlmock")
			dbWrapper := NewDBWrapper(sqlxDb)

			rows, err := dbWrapper.Query(query)
			if !errors.Is(err, tt.errorExpected) {
				t.Errorf("Error not expected got: %w", err)
			}

			_ = rows.Next()
			if rows.ScannedRowsCount() != tt.countExpected {
				t.Errorf("Expected rows count: %d got %d", tt.countExpected, rows.count)
			}
		})
	}
}

func TestRowsxWrapper_Next(t *testing.T) {
	const query = `SELECT a.TABLESPACE_NAME.*`

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name          string
		returnedRows  *sqlmock.Rows
		errorExpected error
		countExpected int
	}{
		{
			name: "when one row is returned the count must be 1",
			returnedRows: sqlmock.NewRows([]string{"TABLESPACE_NAME", "USED", "OFFLINE", "SIZE", "USED_PERCENT"}).
				AddRow("testtablespace", 1234, 0, 4321, 12),
			errorExpected: nil,
			countExpected: 1,
		},
		{
			name:          "when no rows are returned the count must be 0",
			returnedRows:  &sqlmock.Rows{},
			errorExpected: nil,
			countExpected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectQuery(query).WillReturnRows(tt.returnedRows)

			sqlxDb := sqlx.NewDb(db, "sqlmock")
			dbWrapper := NewDBWrapper(sqlxDb)

			rowsx, err := dbWrapper.Queryx(query)
			if !errors.Is(err, tt.errorExpected) {
				t.Errorf("Error not expected got: %w", err)
			}

			_ = rowsx.Next()
			if rowsx.ScannedRowsCount() != tt.countExpected {
				t.Errorf("Expected rows count: %d got %d", tt.countExpected, rowsx.count)
			}
		})
	}
}
