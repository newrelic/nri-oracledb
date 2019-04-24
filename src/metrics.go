package main

import (
	"database/sql"
	"fmt"
	"os"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
)

// collectMetrics spins off goroutines for each of the metric groups, which
// send their metrics to the populateMetrics goroutine
func collectMetrics(db *sql.DB, populaterWg *sync.WaitGroup, i *integration.Integration, instanceLookUp map[string]string) {
	defer populaterWg.Done()

	var collectorWg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 100) // large buffer for speed

	// Create a goroutine for each of the metric groups to collect
	collectorWg.Add(5)
	go oracleReadWriteMetrics.Collect(db, &collectorWg, metricChan)
	go oraclePgaMetrics.Collect(db, &collectorWg, metricChan)
	go oracleSysMetrics.Collect(db, &collectorWg, metricChan)
	go globalNameInstanceMetric.Collect(db, &collectorWg, metricChan)
	go dbIDInstanceMetric.Collect(db, &collectorWg, metricChan)

	// Separate logic is needed to see if we should even collect tablespaces
	collectTableSpaces(db, &collectorWg, metricChan)

	// When the metric groups are finished collecting, close the channel
	go func() {
		collectorWg.Wait()
		close(metricChan)
	}()

	// Create a goroutine to read from the metric channel and insert the metrics
	populateMetrics(metricChan, i, instanceLookUp)
}

// populateMetrics reads metrics from the metricChan, then populates the correct
// metric set with the read metric
func populateMetrics(metricChan <-chan newrelicMetricSender, i *integration.Integration, instanceLookUp map[string]string) {

	// Create storage maps for tablespace and instance metric sets
	tsMetricSets := make(map[string]*metric.Set)
	instanceMetricSets := make(map[string]*metric.Set)

	for {
		metricSender, ok := <-metricChan
		if !ok {
			return // return if the channel is closed
		}

		metric := metricSender.metric

		// If the metric belongs to a tablespace, otherwise it belongs to an instance
		if tsName, ok := metricSender.metadata["tablespace"]; ok {
			ms := getOrCreateMetricSet(tsName, "tablespace", tsMetricSets, i)
			if err := ms.SetMetric(metric.name, metric.value, metric.metricType); err != nil {
				log.Error("Failed to set metric %s", metric.name)
			}
		} else if instanceID, ok := metricSender.metadata["instanceID"]; ok {
			instanceName := func() string {
				if name, ok := instanceLookUp[instanceID]; ok {
					return name
				}

				return instanceID
			}()

			ms := getOrCreateMetricSet(instanceName, "instance", instanceMetricSets, i)
			if err := ms.SetMetric(metric.name, metric.value, metric.metricType); err != nil {
				log.Error("Failed to set metric %s", metric.name)
			}
		}
	}
}

// getOrCreateMetricSet either retrieves a metric set from a map or creates the metric set
// and inserts it into the map.
func getOrCreateMetricSet(entityIdentifier string, entityType string, m map[string]*metric.Set, i *integration.Integration) *metric.Set {

	// If the metric set already exists, return it
	set, ok := m[entityIdentifier]
	if ok {
		return set
	}

	// If the metric set doesn't exist, get the entity for it and create a new metric set
	endpointIDAttr := integration.IDAttribute{Key: "endpoint", Value: fmt.Sprintf("%s:%s", args.Hostname, args.Port)}
	serviceIDAttr := integration.IDAttribute{Key: "serviceName", Value: args.ServiceName}
  e, _ := i.EntityReportedVia( //can't error if both name and namespace are defined
    fmt.Sprintf("%s:%s", args.Hostname, args.Port), 
    entityIdentifier, 
    fmt.Sprintf("ora-%s", entityType), 
    endpointIDAttr, 
    serviceIDAttr,
  ) 

	var newSet *metric.Set
	if entityType == "instance" {
		newSet = e.NewMetricSet("OracleDatabaseSample", metric.Attr("entityName", "ora-instance:"+entityIdentifier), metric.Attr("displayName", entityIdentifier))
	} else if entityType == "tablespace" {
		newSet = e.NewMetricSet("OracleTablespaceSample", metric.Attr("entityName", "ora-tablespace:"+entityIdentifier), metric.Attr("displayName", entityIdentifier))
	} else {
		log.Error("Unreachable code")
		os.Exit(1)
	}

	// Put the new metric set the map
	m[entityIdentifier] = newSet

	return newSet
}

// maxTablespaces is the maximum amount of Tablespaces that can be collect.
// If there are more than this number of Tablespaces then collection of
// Tablespaces will fail.
const maxTablespaces = 200
const tablespaceCountQuery = `SELECT count(1) FROM DBA_TABLESPACES WHERE TABLESPACE_NAME <> 'TEMP'`

func collectTableSpaces(db *sql.DB, wg *sync.WaitGroup, metricChan chan<- newrelicMetricSender) {
	// Get count from database
	if tablespaceWhiteList == nil {
		tablespaceCount, err := queryNumTablespaces(db)
		if err != nil {
			log.Error("Unable to determine the number of tablespaces due to '%s'. Skipping tablespace collection", err.Error())
			return
		}

		if tablespaceCount > maxTablespaces {
			log.Error("There are %d tablespaces in collection, the maximum amount of tablespaces to collect is %d. Use the tablespace whitelist configuration parameter to limit collection size.", tablespaceCount, maxTablespaces)
			return
		}
	} else if length := len(tablespaceWhiteList); length > maxTablespaces {
		log.Error("There are %d tablespaces in collection, the maximum amount of tablespaces to collect is %d. Use the tablespace whitelist configuration parameter to limit collection size.", length, maxTablespaces)
		return
	} else if len(tablespaceWhiteList) == 0 {
		log.Info("No tablespaces specified, skipping tablespace collection.")
		return
	}

	wg.Add(3)
	go oracleTablespaceMetrics.Collect(db, wg, metricChan)
	go globalNameTablespaceMetric.Collect(db, wg, metricChan)
	go dbIDTablespaceMetric.Collect(db, wg, metricChan)

}

func queryNumTablespaces(db *sql.DB) (int, error) {
	rows, err := db.Query(tablespaceCountQuery)
	if err != nil {
		return 0, err
	}

	var count int
	if rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return 0, err
		}

	}

	return count, nil
}
