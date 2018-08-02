package main

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/integration"
)

func inventoryWorker(db *sqlx.DB, wg *sync.WaitGroup, i *integration.Integration) {
	wg.Done()

	sqlQuery := `
		SELECT
			INST_ID,
			NAME,
			VALUE,
			DESCRIPTION
		FROM gv$parameter
		UNION
		SELECT 
			INST_ID,
			'version',
			VERSION,
			'OracleDB version'
		FROM gv$instance `

	type inventoryRow struct {
		instID      int
		name        string
		value       interface{}
		description string
	}

	rows, err := db.Queryx(sqlQuery)
	if err != nil {
		fmt.Printf("failed to collect inventory: %s", err)
	}

	for rows.Next() {
		var tempRow inventoryRow
		rows.Scan(&tempRow.instID, &tempRow.name, &tempRow.value, &tempRow.description)

		e, err := i.Entity(strconv.Itoa(tempRow.instID), "instance")
		if err != nil {
			fmt.Printf("failed to get instance entity %d", tempRow.instID)
		}
		e.SetInventoryItem(tempRow.name, "value", tempRow.value)
		e.SetInventoryItem(tempRow.name, "description", tempRow.description)

	}

}
