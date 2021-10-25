package main

import (
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/godror/godror"
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/nri-oracledb/src/database"
)

func TestPopulateInventory(t *testing.T) {
	db, mock, err := sqlmock.New()
	defer db.Close() // nolint
	if err != nil {
		t.Error(err)
	}

	args = argumentList{
		Hostname:    "testhost",
		Port:        "1234",
		ServiceName: "testServiceName",
	}
	defer func() { args = argumentList{} }()

	columns := []string{"INST_ID", "NAME", "VALUE", "DESCRIPTION"}
	mock.ExpectQuery(`.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow(1, "testname", "testvalue", "this is a test"),
	)

	var wg sync.WaitGroup
	i, _ := integration.New("oracletest", "0.0.1")

	lookup := map[string]string{
		"1": "MyInstance",
	}

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	wg.Add(1)
	go collectInventory(dbWrapper, &wg, i, lookup)
	wg.Wait()

	marshalled, _ := i.MarshalJSON()

	expectedMarshalled := `{"name":"oracletest","protocol_version":"3","integration_version":"0.0.1","data":[{"entity":{"name":"MyInstance","type":"ora-instance","id_attributes":[{"Key":"endpoint","Value":"testhost:1234"},{"Key":"serviceName","Value":"testServiceName"}]},"metrics":[],"inventory":{"testname":{"description":"this is a test","value":"testvalue"}},"events":[]}]}`
	if string(marshalled) != expectedMarshalled {
		t.Errorf("Expected %s, got %s", expectedMarshalled, marshalled)
	}

}
