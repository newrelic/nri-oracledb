package main

import (
	"errors"
	"github.com/newrelic/nri-oracledb/src/database"
	"reflect"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func Test_parseTablespaceWhitelist(t *testing.T) {
	testCases := []struct {
		name string
		arg  string
		want []string
	}{
		{
			"No Whitelist",
			"",
			nil,
		},
		{
			"Whitelist",
			`["one", "two", "three"]`,
			[]string{"one", "two", "three"},
		},
		{
			"Empty Whitelist",
			`[]`,
			[]string{},
		},
	}

	for _, tc := range testCases {
		args.Tablespaces = tc.arg
		tablespaceWhiteList = nil
		if err := parseTablespaceWhitelist(); err != nil {
			t.Errorf("Test Case %s Failed: Unexpected error: %s", tc.name, err.Error())
			t.FailNow()
		}

		if !reflect.DeepEqual(tablespaceWhiteList, tc.want) {
			t.Errorf("Test Case %s Failed: Expected '%+v', got '%+v'", tc.name, tc.want, tablespaceWhiteList)
		}
	}
}

func Test_createInstanceIDLookup_QueryFail(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	mock.
		ExpectQuery(`SELECT
		INSTANCE_NAME, INST_ID
		FROM gv\$instance`).
		WillReturnError(errors.New("error"))

	sqlxDb := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDb)

	_, err = createInstanceIDLookup(dbWrapper)
	if err == nil {
		t.Error("Did not return expected error")
	}
}

func Test_createInstanceIDLookup(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	mock.
		ExpectQuery(`SELECT
		INSTANCE_NAME, INST_ID
		FROM gv\$instance`).
		WillReturnRows(
			sqlmock.NewRows([]string{"INSTANCE_NAME", "INST_ID"}).
				AddRow("one", 1).
				AddRow("two", 2).
				AddRow("three", 3),
		)

	expected := map[string]string{
		"1": "one",
		"2": "two",
		"3": "three",
	}

	sqlxDb := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDb)

	out, err := createInstanceIDLookup(dbWrapper)
	if err != nil {
		t.Errorf("Unexpected Error %s", err.Error())
		t.FailNow()
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("Expected %+v got %+v", expected, out)
	}
}
