package main

import (
	"sync"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
)

func populateMetrics(metricChan <-chan newrelicMetricSender, wg *sync.WaitGroup, i *integration.Integration) {
	defer wg.Done()

	tsMetricSets := make(map[string]*metric.Set)
	instanceMetricSets := make(map[string]*metric.Set)

	for {
		metricSender, ok := <-metricChan
		if !ok {
			return
		}

		metric := metricSender.metric

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

func getOrCreateMetricSet(entityIdentifier string, entityType string, m map[string]*metric.Set, i *integration.Integration) *metric.Set {
	set, ok := m[entityIdentifier]
	if ok {
		return set
	}

	e, _ := i.Entity(entityIdentifier, entityType) //can't error if both name and namespace are defined
	var newSet *metric.Set
	if entityType == "instance" {
		newSet = e.NewMetricSet("OracleDatabaseSample", metric.Attr("entityName", "instance:instance"+entityIdentifier), metric.Attr("displayName", "instance"+entityIdentifier))
	} else if entityType == "tablespace" {
		newSet = e.NewMetricSet("OracleTablespaceSample", metric.Attr("entityName", "tablespace:"+entityIdentifier), metric.Attr("displayName", entityIdentifier))
	} else {
		panic("invalid entity type, unreachable code")
	}

	m[entityIdentifier] = newSet

	return newSet
}
