package main

import (
	"database/sql"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
)

// collectMetrics spins off goroutines for each of the metric groups, which
// send their metrics to the populateMetrics goroutine
func collectMetrics(db *sql.DB, populaterWg *sync.WaitGroup, i *integration.Integration) {
	defer populaterWg.Done()

	var collectorWg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 100) // large buffer for speed

	// Create a goroutine for each of the metric groups to collect
	collectorWg.Add(4)
	go oracleReadWriteMetrics.Collect(db, &collectorWg, metricChan)
	go oraclePgaMetrics.Collect(db, &collectorWg, metricChan)
	go oracleSysMetrics.Collect(db, &collectorWg, metricChan)
	go oracleTablespaceMetrics.Collect(db, &collectorWg, metricChan)

	// When the metric groups are finished collecting, close the channel
	go func() {
		collectorWg.Wait()
		close(metricChan)
	}()

	// Create a goroutine to read from the metric channel and insert the metrics
	populateMetrics(metricChan, i)
}

// populateMetrics reads metrics from the metricChan, then populates the correct
// metric set with the read metric
func populateMetrics(metricChan <-chan newrelicMetricSender, i *integration.Integration) {

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
			err := ms.SetMetric(metric.name, metric.value, metric.metricType)
			if err != nil {
				logger.Errorf("Failed to set metric %s", metric.name)
			}
		} else if instanceName, ok := metricSender.metadata["instanceID"]; ok {
			ms := getOrCreateMetricSet(instanceName, "instance", instanceMetricSets, i)
			err := ms.SetMetric(metric.name, metric.value, metric.metricType)
			if err != nil {
				logger.Errorf("Failed to set metric %s", metric.name)
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
	e, _ := i.Entity(entityIdentifier, entityType) //can't error if both name and namespace are defined
	var newSet *metric.Set
	if entityType == "instance" {
		newSet = e.NewMetricSet("OracleDatabaseSample", metric.Attr("entityName", "instance:instance"+entityIdentifier), metric.Attr("displayName", "instance"+entityIdentifier))
	} else if entityType == "tablespace" {
		newSet = e.NewMetricSet("OracleTablespaceSample", metric.Attr("entityName", "tablespace:"+entityIdentifier), metric.Attr("displayName", entityIdentifier))
	} else {
		panic("invalid entity type, unreachable code")
	}

	// Put the new metric set the map
	m[entityIdentifier] = newSet

	return newSet
}
