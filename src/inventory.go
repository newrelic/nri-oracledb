package main

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-oracledb/src/database"
)

type inventoryCollector struct {
	integration    *integration.Integration
	db             database.DBWrapper
	wg             *sync.WaitGroup
	instanceLookUp map[string]string
}

// collect queries the database for the inventory items, then populates
// the integration with the results
func (ic *inventoryCollector) collect() {
	defer ic.wg.Done()

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

	rows, err := ic.db.Query(sqlQuery)
	if err != nil {
		log.Error("Failed to collect inventory: %s", err)
		return
	}
	defer func() {
		checkAndLogEmptyQueryResult(sqlQuery, rows)
		rows.Close()
	}()

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
			if name, ok := ic.instanceLookUp[instanceID]; ok {
				return name
			}

			return instanceID
		}()

		endpointIDAttr := integration.IDAttribute{Key: "endpoint", Value: fmt.Sprintf("%s:%s", args.Hostname, args.Port)}
		serviceIDAttr := integration.IDAttribute{Key: "serviceName", Value: args.ServiceName}
		e, err := ic.integration.EntityReportedVia(
			fmt.Sprintf("%s:%s", args.Hostname, args.Port),
			instanceName,
			"ora-instance",
			endpointIDAttr,
			serviceIDAttr,
		)
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
