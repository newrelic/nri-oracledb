package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/integration"
)

func collectInventory(db *sql.DB, wg *sync.WaitGroup, i *integration.Integration) {
	defer wg.Done()

	const sqlQuery = `
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
		FROM gv$instance`

	type inventoryRow struct {
		instID      int
		name        string
		value       interface{}
		description string
	}

	rows, err := db.Query(sqlQuery)
	if err != nil {
		fmt.Printf("failed to collect inventory: %s", err)
	}

	for rows.Next() {

		// Scan the row into a struct
		var inventoryResultRow inventoryRow
		rows.Scan(&inventoryResultRow.instID, &inventoryResultRow.name, &inventoryResultRow.value, &inventoryResultRow.description)

		e, err := i.Entity(strconv.Itoa(inventoryResultRow.instID), "instance")
		if err != nil {
			logger.Errorf("failed to get instance entity %d", inventoryResultRow.instID)
		}
		e.SetInventoryItem(inventoryResultRow.name, "value", inventoryResultRow.value)
		e.SetInventoryItem(inventoryResultRow.name, "description", inventoryResultRow.description)

	}

}
