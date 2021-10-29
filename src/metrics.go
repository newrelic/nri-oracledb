package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"sync"

	"github.com/godror/godror"
	"github.com/newrelic/infra-integrations-sdk/data/attribute"
	nrmetric "github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-oracledb/src/database"
	"gopkg.in/yaml.v2"
)

// collectMetrics spins off goroutines for each of the metric groups, which
// send their metrics to the populateMetrics goroutine
func collectMetrics(db database.DBWrapper, populaterWg *sync.WaitGroup, i *integration.Integration, instanceLookUp map[string]string, customMetricsQuery string, customMetricsConfig string) {
	defer populaterWg.Done()

	var collectorWg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 100) // large buffer for speed

	// Separate logic is needed to see if we should even collect tablespaces
	// Collect tablespaces first so the list query completes before other queries are run
	collectorWg.Add(1)
	go collectTableSpaces(db, &collectorWg, metricChan)

	// Create a goroutine for each of the metric groups to collect
	baseCollections := []oracleMetricGroup{
		oracleCDBDatafilesOffline,
		oraclePDBDatafilesOffline,
		oraclePDBNonWrite,
		oracleLockedAccounts,
		oracleReadWriteMetrics,
		oraclePgaMetrics,
		oracleSysMetrics,
		globalNameInstanceMetric,
		dbIDInstanceMetric,
		oracleLongRunningQueries,
		oracleSGAUGATotalMemory,
		oracleSGASharedPoolLibraryCacheSharableStatement,
		oracleSGASharedPoolLibraryCacheShareableUser,
		oracleSGASharedPoolLibraryCacheReloadRatio,
		oracleSGASharedPoolLibraryCacheHitRatio,
		oracleSGASharedPoolDictCacheRatio,
		oracleSGASharedPoolDictCacheRatio,
		oracleSGALogBufferSpaceWaits,
		oracleSGALogAllocRetries,
		oracleSGAHitRatio,
		oracleSysstat,
		oracleSGA,
		oracleRollbackSegments,
		oracleRedoLogWaits,
	}

	for _, collection := range baseCollections {
		collectorWg.Add(1)
		c := collection
		go c.Collect(db, &collectorWg, metricChan)
	}

	if customMetricsQuery != "" {
		custom := customMetricGroup{customMetricsQuery}
		collectorWg.Add(1)
		go custom.Collect(db, &collectorWg, metricChan)
	}

	if customMetricsConfig != "" {
		collectorWg.Add(1)
		go PopulateCustomMetricsFromFile(db, &collectorWg, metricChan, customMetricsConfig)
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
		} else if metricSender.isCustom {
			instanceID := metricSender.metadata["instanceID"]
			instanceName := func() string {
				if name, ok := instanceLookUp[instanceID]; ok {
					return name
				}

				return instanceID
			}()

			sampleName := metricSender.metadata["sampleName"]

			for _, row := range metricSender.customMetrics {
				ms := createCustomMetricSet(sampleName, instanceName, i)
				for key, val := range row {
					sanitized := sanitizeValue(val)
					inferredMetricType := func() nrmetric.SourceType {
						if t, ok := metricSender.metricTypeOverrides[key]; ok {
							return nrmetric.SourceType(t)
						}
						return inferMetricType(sanitized)
					}()

					err := ms.SetMetric(key, sanitized, inferredMetricType)
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

func sanitizeValue(val interface{}) interface{} {
	switch v := val.(type) {
	case string, float32, float64, int, int32, int64:
		return v
	case godror.Number:
		num, err := strconv.ParseFloat(string(v), 64)
		if err != nil {
			log.Error("Failed to convert %s to a number")
			return 0
		}
		return num
	default:
		log.Warn("Unknown metric type %T. Falling back to sending as string", val)
		return fmt.Sprintf("%v", v)
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
		newSet = e.NewMetricSet("OracleDatabaseSample", attribute.Attr("entityName", "ora-instance:"+entityIdentifier), attribute.Attr("displayName", entityIdentifier))
	} else if entityType == "tablespace" {
		newSet = e.NewMetricSet("OracleTablespaceSample", attribute.Attr("entityName", "ora-tablespace:"+entityIdentifier), attribute.Attr("displayName", entityIdentifier))
	} else {
		log.Error("Unreachable code")
		os.Exit(1)
	}

	// Put the new metric set the map
	m[entityIdentifier] = newSet

	return newSet
}

func createCustomMetricSet(sampleName string, instanceID string, i *integration.Integration) *nrmetric.Set {
	endpointIDAttr := integration.IDAttribute{Key: "endpoint", Value: fmt.Sprintf("%s:%s", args.Hostname, args.Port)}
	serviceIDAttr := integration.IDAttribute{Key: "serviceName", Value: args.ServiceName}
	e, _ := i.EntityReportedVia( //can't error if both name and namespace are defined
		fmt.Sprintf("%s:%s", args.Hostname, args.Port),
		instanceID,
		"ora-instance",
		endpointIDAttr,
		serviceIDAttr,
	)

	return e.NewMetricSet(sampleName, attribute.Attr("entityName", "ora-instance:"+instanceID), attribute.Attr("displayName", instanceID))
}

// maxTablespaces is the maximum amount of Tablespaces that can be collect.
// If there are more than this number of Tablespaces then collection of
// Tablespaces will fail.
const maxTablespaces = 200
const tablespaceCountQuery = `SELECT count(1) FROM DBA_TABLESPACES WHERE TABLESPACE_NAME <> 'TEMP'`

func collectTableSpaces(db database.DBWrapper, wg *sync.WaitGroup, metricChan chan<- newrelicMetricSender) {
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

func queryNumTablespaces(db database.DBWrapper) (int, error) {
	rows, err := db.Query(tablespaceCountQuery)
	if err != nil {
		return 0, err
	}
	defer func() {
		checkAndLogEmptyQueryResult(tablespaceCountQuery, rows)
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

// PopulateCustomMetricsFromFile collects metrics defined by a custom config file
func PopulateCustomMetricsFromFile(db database.DBWrapper, wg *sync.WaitGroup, metricChan chan<- newrelicMetricSender, configFile string) {
	defer wg.Done()

	contents, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Error("Failed to read custom config file: %s", err)
		return
	}

	var customYAML customMetricsYAML
	err = yaml.Unmarshal(contents, &customYAML)
	if err != nil {
		log.Error("Failed to unmarshal custom config file: %s", err)
		return
	}

	// Semaphore to run 10 custom queries concurrently
	sem := make(chan struct{}, 10)
	for _, config := range customYAML.Queries {
		sem <- struct{}{}
		wg.Add(1)
		go func(cfg customMetricsConfig) {
			defer wg.Done()
			defer func() {
				<-sem
			}()

			CollectCustomConfig(db, metricChan, cfg)
		}(config)
	}
}

// CollectCustomConfig collects metrics defined by a custom config
func CollectCustomConfig(db database.DBWrapper, metricChan chan<- newrelicMetricSender, cfg customMetricsConfig) {
	instanceQuery := `SELECT INSTANCE_NUMBER FROM v$instance`
	instanceRows, err := db.Queryx(instanceQuery)
	if err != nil {
		log.Error("Failed to execute query %s: %s", formatQueryForLogging(instanceQuery), err)
		return
	}
	defer func() {
		checkAndLogEmptyQueryResult(instanceQuery, instanceRows)
		err := instanceRows.Close()
		if err != nil {
			log.Error("Failed to close rows: %s", err)
		}
	}()

	var instanceID string
	for instanceRows.Next() {
		err = instanceRows.Scan(&instanceID)
		if err != nil {
			log.Error("Failed to get instance ID %s: %s", formatQueryForLogging(instanceQuery), err)
			return
		}
	}

	rows, err := db.Queryx(cfg.Query)
	if err != nil {
		log.Error("Could not execute database query %s: %s", formatQueryForLogging(cfg.Query), err.Error())
		return
	}
	defer func() {
		checkAndLogEmptyQueryResult(cfg.Query, rows)
		_ = rows.Close()
	}()

	sampleName := func() string {
		if cfg.SampleName == "" {
			return defaultCustomSampleType
		}
		return cfg.SampleName
	}()

	sender := newrelicMetricSender{
		isCustom: true,
		metadata: map[string]string{
			"instanceID": instanceID,
			"sampleName": sampleName,
		},
		metricTypeOverrides: cfg.MetricTypes,
		customMetrics:       make([]map[string]interface{}, 0),
	}

	for rows.Next() {
		row := make(map[string]interface{})
		err := rows.MapScan(row)
		if err != nil {
			log.Error("Failed to scan custom query row: %s", err)
			return
		}

		sender.customMetrics = append(sender.customMetrics, row)
	}

	metricChan <- sender
}

type metricType nrmetric.SourceType

func (m *metricType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw string
	err := unmarshal(&raw)
	if err != nil {
		return err
	}

	st, err := nrmetric.SourceTypeForName(raw)
	if err != nil {
		return err
	}

	*m = metricType(st)
	return nil
}

type customMetricsYAML struct {
	Queries []customMetricsConfig
}

type customMetricsConfig struct {
	Query       string                `yaml:"query"`
	MetricTypes map[string]metricType `yaml:"metric_types"`
	SampleName  string                `yaml:"sample_name"`
}
