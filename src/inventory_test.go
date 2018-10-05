package main

import (
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/newrelic/infra-integrations-sdk/integration"
	_ "gopkg.in/goracle.v2"
)

func TestPopulateInventory(t *testing.T) {
	db, mock, err := sqlmock.New()
	defer db.Close()
	if err != nil {
		t.Error(err)
	}

	columns := []string{"INST_ID", "NAME", "VALUE", "DESCRIPTION"}
	mock.ExpectQuery(`.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow(1, "testname", "testvalue", "this is a test"),
	)

	var wg sync.WaitGroup
	i, _ := integration.New("oracletest", "0.0.1")

	lookup := map[string]string{
		"1": "MyInstance",
	}

	wg.Add(1)
	go collectInventory(db, &wg, i, lookup)
	wg.Wait()

	marshalled, err := i.MarshalJSON()

	expectedMarshalled := `{"name":"oracletest","protocol_version":"2","integration_version":"0.0.1","data":[{"entity":{"name":"MyInstance","type":"instance"},"metrics":[],"inventory":{"testname":{"description":"this is a test","value":"testvalue"}},"events":[]}]}`
	if string(marshalled) != expectedMarshalled {
		t.Errorf("Expected %s, got %s", expectedMarshalled, marshalled)
	}

}
