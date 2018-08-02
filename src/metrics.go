package main

import (
	"fmt"
	"sync"

	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
)

func metricsWorker(metricChan <-chan newRelicMetricSender, wg *sync.WaitGroup, i *integration.Integration) {
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
			ms := getOrCreateTablespaceMetricSet(tsName, tsMetricSets, i)
			err := ms.SetMetric(metric.name, metric.value, metric.metricType)
			if err != nil {
				fmt.Println(err)
			}
		}

		if instanceName, ok := metricSender.metadata["instanceID"]; ok {
			ms := getOrCreateInstanceMetricSet(instanceName, instanceMetricSets, i)
			err := ms.SetMetric(metric.name, metric.value, metric.metricType)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func getOrCreateInstanceMetricSet(instID string, m map[string]*metric.Set, i *integration.Integration) *metric.Set {
	set, ok := m[instID]
	if ok {
		return set
	}

	e, _ := i.Entity(instID, "instance") //can't error if both name and namespace are defined
	newSet := e.NewMetricSet("OracleDatabaseSample", metric.Attr("entityName", "instance:instance"+instID), metric.Attr("displayName", "instance"+instID))

	m[instID] = newSet

	return newSet
}

func getOrCreateTablespaceMetricSet(tablespaceName string, m map[string]*metric.Set, i *integration.Integration) *metric.Set {
	set, ok := m[tablespaceName]
	if ok {
		return set
	}

	e, _ := i.Entity(tablespaceName, "tablespace") //can't error if both name and namespace are defined
	newSet := e.NewMetricSet("OracleTablespaceSample", metric.Attr("entityName", "tablespace:"+tablespaceName), metric.Attr("displayName", tablespaceName))

	m[tablespaceName] = newSet

	return newSet
}
