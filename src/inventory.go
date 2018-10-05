package main

import (
	"database/sql"
	"strconv"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
)

// collectInventory queries the database for the inventory items, then populates
// the integration with the results
func collectInventory(db *sql.DB, wg *sync.WaitGroup, i *integration.Integration, instanceLookUp map[string]string) {
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
		log.Error("Failed to collect inventory: %s", err)
	}

	for rows.Next() {

		// Scan the row into a struct
		var inventoryResultRow inventoryRow
		err := rows.Scan(&inventoryResultRow.instID, &inventoryResultRow.name, &inventoryResultRow.value, &inventoryResultRow.description)
		if err != nil {
			log.Error("Failed to scan inventory row: %s", err)
			continue
		}

		// Retrieve or create the instance entity
		instanceID := strconv.Itoa(inventoryResultRow.instID)
		instanceName := func() string {
			if name, ok := instanceLookUp[instanceID]; ok {
				return name
			}

			return instanceID
		}()

		e, err := i.Entity(instanceName, "instance")
		if err != nil {
			log.Error("Failed to get instance entity %d", inventoryResultRow.instID)
			continue
		}

		// Create inventory entry
		if err := e.SetInventoryItem(inventoryResultRow.name, "value", inventoryResultRow.value); err != nil {
			log.Error("Failed to set value for %s", inventoryResultRow.name)
		}
		if err := e.SetInventoryItem(inventoryResultRow.name, "description", inventoryResultRow.description); err != nil {
			log.Error("Failed to set description for %s", inventoryResultRow.name)
		}
	}
}
