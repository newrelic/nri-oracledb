package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/jmoiron/sqlx"
	nrmetric "github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
)

// collectMetrics spins off goroutines for each of the metric groups, which
// send their metrics to the populateMetrics goroutine
func collectMetrics(db *sqlx.DB, populaterWg *sync.WaitGroup, i *integration.Integration, instanceLookUp map[string]string, customMetricsQuery string) {
	defer populaterWg.Done()

	var collectorWg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 100) // large buffer for speed

	// Separate logic is needed to see if we should even collect tablespaces
	// Collect tablespaces first so the list query completes before other queries are run
	collectorWg.Add(25)
	go collectTableSpaces(db, &collectorWg, metricChan)

	// Create a goroutine for each of the metric groups to collect
	go oracleCDBDatafilesOffline.Collect(db, &collectorWg, metricChan)
	go oraclePDBDatafilesOffline.Collect(db, &collectorWg, metricChan)
	go oraclePDBNonWrite.Collect(db, &collectorWg, metricChan)
	go oracleLockedAccounts.Collect(db, &collectorWg, metricChan)
	go oracleReadWriteMetrics.Collect(db, &collectorWg, metricChan)
	go oraclePgaMetrics.Collect(db, &collectorWg, metricChan)
	go oracleSysMetrics.Collect(db, &collectorWg, metricChan)
	go globalNameInstanceMetric.Collect(db, &collectorWg, metricChan)
	go dbIDInstanceMetric.Collect(db, &collectorWg, metricChan)
	go oracleLongRunningQueries.Collect(db, &collectorWg, metricChan)
	go oracleSGAUGATotalMemory.Collect(db, &collectorWg, metricChan)
	go oracleSGASharedPoolLibraryCacheSharableStatement.Collect(db, &collectorWg, metricChan)
	go oracleSGASharedPoolLibraryCacheShareableUser.Collect(db, &collectorWg, metricChan)
	go oracleSGASharedPoolLibraryCacheReloadRatio.Collect(db, &collectorWg, metricChan)
	go oracleSGASharedPoolLibraryCacheHitRatio.Collect(db, &collectorWg, metricChan)
	go oracleSGASharedPoolDictCacheRatio.Collect(db, &collectorWg, metricChan)
	go oracleSGASharedPoolDictCacheRatio.Collect(db, &collectorWg, metricChan)
	go oracleSGALogBufferSpaceWaits.Collect(db, &collectorWg, metricChan)
	go oracleSGALogAllocRetries.Collect(db, &collectorWg, metricChan)
	go oracleSGAHitRatio.Collect(db, &collectorWg, metricChan)
	go oracleSysstat.Collect(db, &collectorWg, metricChan)
	go oracleSGA.Collect(db, &collectorWg, metricChan)
	go oracleRollbackSegments.Collect(db, &collectorWg, metricChan)
	go oracleRedoLogWaits.Collect(db, &collectorWg, metricChan)

	if customMetricsQuery != "" {
		custom := customMetricGroup{customMetricsQuery}
		collectorWg.Add(1)
		go custom.Collect(db, &collectorWg, metricChan)
	}

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
	tsMetricSets := make(map[string]*nrmetric.Set)
	instanceMetricSets := make(map[string]*nrmetric.Set)

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
				log.Error("Failed to set metric %s: %s", metric.name, err)
			}
		} else if metricSender.customMetrics != nil {
			instanceID := metricSender.metadata["instanceID"]
			instanceName := func() string {
				if name, ok := instanceLookUp[instanceID]; ok {
					return name
				}

				return instanceID
			}()

			for _, row := range metricSender.customMetrics {
				ms := createCustomMetricSet(instanceName, i)
				for key, val := range row {
					err := ms.SetMetric(key, val, inferMetricType(val))
					if err != nil {
						log.Error("Failed to set metric %s with value %v and type %T: %s", key, val, val, err)
					}
				}
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
				log.Error("Failed to set metric %s: %s", metric.name, err)
			}

		}
	}
}

func inferMetricType(val interface{}) nrmetric.SourceType {
	switch val.(type) {
	case string:
		return nrmetric.ATTRIBUTE
	case float32, float64, int, int32, int64:
		return nrmetric.GAUGE
	default:
		return nrmetric.ATTRIBUTE
	}
}

// getOrCreateMetricSet either retrieves a metric set from a map or creates the metric set
// and inserts it into the map.
func getOrCreateMetricSet(entityIdentifier string, entityType string, m map[string]*nrmetric.Set, i *integration.Integration) *nrmetric.Set {

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

	var newSet *nrmetric.Set
	if entityType == "instance" {
		newSet = e.NewMetricSet("OracleDatabaseSample", nrmetric.Attr("entityName", "ora-instance:"+entityIdentifier), nrmetric.Attr("displayName", entityIdentifier))
	} else if entityType == "tablespace" {
		newSet = e.NewMetricSet("OracleTablespaceSample", nrmetric.Attr("entityName", "ora-tablespace:"+entityIdentifier), nrmetric.Attr("displayName", entityIdentifier))
	} else {
		log.Error("Unreachable code")
		os.Exit(1)
	}

	// Put the new metric set the map
	m[entityIdentifier] = newSet

	return newSet
}

func createCustomMetricSet(instanceID string, i *integration.Integration) *nrmetric.Set {
	endpointIDAttr := integration.IDAttribute{Key: "endpoint", Value: fmt.Sprintf("%s:%s", args.Hostname, args.Port)}
	serviceIDAttr := integration.IDAttribute{Key: "serviceName", Value: args.ServiceName}
	e, _ := i.EntityReportedVia( //can't error if both name and namespace are defined
		fmt.Sprintf("%s:%s", args.Hostname, args.Port),
		instanceID,
		"ora-instance",
		endpointIDAttr,
		serviceIDAttr,
	)

	return e.NewMetricSet("OracleCustomSample", nrmetric.Attr("entityName", "ora-instance:"+instanceID), nrmetric.Attr("displayName", instanceID))
}

// maxTablespaces is the maximum amount of Tablespaces that can be collect.
// If there are more than this number of Tablespaces then collection of
// Tablespaces will fail.
const maxTablespaces = 200
const tablespaceCountQuery = `SELECT count(1) FROM DBA_TABLESPACES WHERE TABLESPACE_NAME <> 'TEMP'`

func collectTableSpaces(db *sqlx.DB, wg *sync.WaitGroup, metricChan chan<- newrelicMetricSender) {
	defer wg.Done()

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

func queryNumTablespaces(db *sqlx.DB) (int, error) {
	rows, err := db.Query(tablespaceCountQuery)
	if err != nil {
		return 0, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			log.Error("Failed to close rows: %s", err)
		}
	}()

	var count int
	if rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return 0, err
		}
	}

	return count, nil
}
