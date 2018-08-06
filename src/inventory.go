package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/integration"
)

// collectInventory queries the database for the inventory items, then populates
// the integration with the results
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

	// inventoryRow represents a single row in the database response to sqlQuery
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

		// Retrieve or create the instance entity
		e, err := i.Entity(strconv.Itoa(inventoryResultRow.instID), "instance")
		if err != nil {
			logger.Errorf("failed to get instance entity %d", inventoryResultRow.instID)
		}

		// Create inventory entry
		e.SetInventoryItem(inventoryResultRow.name, "value", inventoryResultRow.value)
		e.SetInventoryItem(inventoryResultRow.name, "description", inventoryResultRow.description)
	}
}
