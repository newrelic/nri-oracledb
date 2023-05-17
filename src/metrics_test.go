package main

import (
	"path/filepath"
	"sync"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/persist"
	"github.com/newrelic/nri-oracledb/src/database"
)

func TestCollectMetrics(t *testing.T) {
	i, err := integration.New("oracletest", "0.0.1")
	if err != nil {
		t.Error(err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	mock.MatchExpectationsInOrder(false)

	columns := []string{"INST_ID", "PhysicalReads", "PhysicalWrites", "PhysicalBlockReads", "PhysicalBlockWrites", "ReadTime", "WriteTime"}
	mock.ExpectQuery(`.*PHYRDS.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow("1", 12, 23, 34, 45, 56, 67),
	)

	columns = []string{"INST_ID", "NAME", "VALUE"}
	mock.ExpectQuery(`.*pgastat.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow("1", "total PGA inuse", 135),
	)

	columns = []string{"INST_ID", "METRIC_NAME", "VALUE"}
	mock.ExpectQuery(`.*\$sysmetric.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow("1", "Buffer Cache Hit Ratio", 0.5),
	)

	columns = []string{"TABLESPACE_NAME", "USED", "OFFLINE", "SIZE", "USED_PERCENT"}
	mock.ExpectQuery(`.*TABLESPACE_NAME.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow("testtablespace", 11, 0, 123, 12),
	)

	lookup := map[string]string{
		"1": "MyInstance",
	}

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	var populaterWg sync.WaitGroup
	populaterWg.Add(1)
	mc := metricsCollector{
		integration:    i,
		db:             dbWrapper,
		wg:             &populaterWg,
		instanceLookUp: lookup,
	}
	go mc.collect()
	populaterWg.Wait()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestCollectPDBMetrics(t *testing.T) {
	args = argumentList{
		SysMetricsSource: "PDB",
	}
	defer func() { args = argumentList{} }()

	i, err := integration.New("oracletest", "0.0.1")
	if err != nil {
		t.Error(err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	mock.MatchExpectationsInOrder(false)

	columns := []string{"INST_ID", "PhysicalReads", "PhysicalWrites", "PhysicalBlockReads", "PhysicalBlockWrites", "ReadTime", "WriteTime"}
	mock.ExpectQuery(`.*PHYRDS.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow("1", 12, 23, 34, 45, 56, 67),
	)

	columns = []string{"INST_ID", "NAME", "VALUE"}
	mock.ExpectQuery(`.*pgastat.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow("1", "total PGA inuse", 135),
	)

	columns = []string{"INST_ID", "METRIC_NAME", "VALUE"}
	mock.ExpectQuery(`.*\$con_sysmetric.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow("1", "CPU Usage Per Sec", 10.0),
	)

	columns = []string{"TABLESPACE_NAME", "USED", "OFFLINE", "SIZE", "USED_PERCENT"}
	mock.ExpectQuery(`.*TABLESPACE_NAME.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow("testtablespace", 11, 0, 123, 12),
	)

	lookup := map[string]string{
		"1": "MyInstance",
	}

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	var populaterWg sync.WaitGroup
	populaterWg.Add(1)
	mc := metricsCollector{
		integration:    i,
		db:             dbWrapper,
		wg:             &populaterWg,
		instanceLookUp: lookup,
	}
	go mc.collect()
	populaterWg.Wait()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestCollectAllMetrics(t *testing.T) {
	args = argumentList{
		SysMetricsSource: "All",
	}
	defer func() { args = argumentList{} }()

	i, err := integration.New("oracletest", "0.0.1")
	if err != nil {
		t.Error(err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	mock.MatchExpectationsInOrder(false)

	columns := []string{"INST_ID", "PhysicalReads", "PhysicalWrites", "PhysicalBlockReads", "PhysicalBlockWrites", "ReadTime", "WriteTime"}
	mock.ExpectQuery(`.*PHYRDS.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow("1", 12, 23, 34, 45, 56, 67),
	)

	columns = []string{"INST_ID", "NAME", "VALUE"}
	mock.ExpectQuery(`.*pgastat.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow("1", "total PGA inuse", 135),
	)

	columns = []string{"INST_ID", "METRIC_NAME", "VALUE"}
	mock.ExpectQuery(`.*\$con_sysmetric.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow("1", "CPU Usage Per Sec", 10.0),
	)

	columns = []string{"INST_ID", "METRIC_NAME", "VALUE"}
	mock.ExpectQuery(`.*\$sysmetric.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow("1", "Buffer Cache Hit Ratio", 0.5),
	)

	columns = []string{"TABLESPACE_NAME", "USED", "OFFLINE", "SIZE", "USED_PERCENT"}
	mock.ExpectQuery(`.*TABLESPACE_NAME.*`).WillReturnRows(
		sqlmock.NewRows(columns).AddRow("testtablespace", 11, 0, 123, 12),
	)

	lookup := map[string]string{
		"1": "MyInstance",
	}

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	var populaterWg sync.WaitGroup
	populaterWg.Add(1)
	mc := metricsCollector{
		integration:    i,
		db:             dbWrapper,
		wg:             &populaterWg,
		instanceLookUp: lookup,
	}
	go mc.collect()
	populaterWg.Wait()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestGetOrCreateMetricSet(t *testing.T) {
	args = argumentList{
		Hostname:	 "testhost",//nolint
		Port:        "1234",
		ServiceName: "testServiceName",
	}
	defer func() { args = argumentList{} }()

	testCases := []struct {
		inputEntityID     string
		inputEntityType   string
		inputMap          map[string]*metric.Set
		expectedMetricSet string
	}{
		{
			inputEntityID:   "1",
			inputEntityType: "instance",
			inputMap: map[string]*metric.Set{
				"1": metric.NewSet("NewEventSample", persist.NewInMemoryStore()),
			},
			expectedMetricSet: `{"event_type":"NewEventSample"}`,
		},
		{
			inputEntityID:     "MyInstance",
			inputEntityType:   "instance",
			inputMap:          map[string]*metric.Set{},
			expectedMetricSet: `{"displayName":"MyInstance","entityName":"ora-instance:MyInstance","event_type":"OracleDatabaseSample","reportingEndpoint":"testhost:1234"}`,
		},
		{
			inputEntityID:     "testtablespace",
			inputEntityType:   "tablespace",
			inputMap:          map[string]*metric.Set{},
			expectedMetricSet: `{"displayName":"testtablespace","entityName":"ora-tablespace:testtablespace","event_type":"OracleTablespaceSample","reportingEndpoint":"testhost:1234"}`,
		},
	}

	i, _ := integration.New("oracletest", "0.0.1")
	for _, tc := range testCases {
		ms := getOrCreateMetricSet(tc.inputEntityID, tc.inputEntityType, tc.inputMap, i)
		marshalled, err := ms.MarshalJSON()
		if err != nil {
			t.Error(err)
		}

		if string(marshalled) != tc.expectedMetricSet {
			t.Errorf("Expected metric set %s, got %s", tc.expectedMetricSet, string(marshalled))
		}
	}
}

func TestPopulateMetrics(t *testing.T) {
	args = argumentList{
		Hostname:    "testhost",
		Port:        "1234",
		ServiceName: "testServiceName",
	}
	defer func() { args = argumentList{} }()

	testCases := []struct {
		inputMetric  newrelicMetricSender
		expectedJSON string
	}{
		{
			inputMetric: newrelicMetricSender{
				metric: &newrelicMetric{
					name:       "testmetric",
					metricType: metric.GAUGE,
					value:      123.0,
				},
				metadata: map[string]string{
					"tablespace": "testtbname",
				},
			},
			expectedJSON: `{"name":"oracletest","protocol_version":"3","integration_version":"0.0.1","data":[{"entity":{"name":"testtbname","type":"ora-tablespace","id_attributes":[{"Key":"endpoint","Value":"testhost:1234"},{"Key":"serviceName","Value":"testServiceName"}]},"metrics":[{"displayName":"testtbname","entityName":"ora-tablespace:testtbname","event_type":"OracleTablespaceSample","reportingEndpoint":"testhost:1234","testmetric":123}],"inventory":{},"events":[]}]}`,
		},
		{
			inputMetric: newrelicMetricSender{
				metric: &newrelicMetric{
					name:       "testmetric",
					metricType: metric.ATTRIBUTE,
					value:      "testattr",
				},
				metadata: map[string]string{
					"instanceID": "1",
				},
			},
			expectedJSON: `{"name":"oracletest","protocol_version":"3","integration_version":"0.0.1","data":[{"entity":{"name":"MyInstance","type":"ora-instance","id_attributes":[{"Key":"endpoint","Value":"testhost:1234"},{"Key":"serviceName","Value":"testServiceName"}]},"metrics":[{"displayName":"MyInstance","entityName":"ora-instance:MyInstance","event_type":"OracleDatabaseSample","reportingEndpoint":"testhost:1234","testmetric":"testattr"}],"inventory":{},"events":[]}]}`,
		},
	}

	for _, tc := range testCases {
		i, _ := integration.New("oracletest", "0.0.1")
		metricChan := make(chan newrelicMetricSender)

		lookup := map[string]string{
			"1": "MyInstance",
		}

		go func() {
			metricChan <- tc.inputMetric
			close(metricChan)
		}()

		populateMetrics(metricChan, i, lookup)

		marshalled, err := i.MarshalJSON()
		if err != nil {
			t.Error(err)
		}

		if string(marshalled) != tc.expectedJSON {
			t.Errorf("Expected %s, got %s", tc.expectedJSON, marshalled)
		}

	}
}

func Test_collectTableSpaces_NoWhitelist_Ok(t *testing.T) {
	i, err := integration.New("oracletest", "0.0.1")
	if err != nil {
		t.Error(err)
	}

	args = argumentList{
		Hostname:    "testhost",
		Port:        "1234",
		ServiceName: "testServiceName",
	}
	defer func() { args = argumentList{} }()

	tablespaceWhiteList = nil
	tablespaceCollections := []oracleMetricGroup{
		oracleTablespaceMetrics,
		globalNameTablespaceMetric,
		dbIDTablespaceMetric,
		oracleCDBDatafilesOffline,
		oraclePDBDatafilesOffline,
		oraclePDBNonWrite,
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	lookup := map[string]string{
		"1": "MyInstance",
	}

	mock.ExpectQuery(`.*FROM DBA_TABLESPACE_USAGE_METRICS.*`).WillReturnRows(
		sqlmock.NewRows([]string{"TABLESPACE_NAME", "USED", "OFFLINE", "SIZE", "USED_PERCENT"}).
			AddRow("testtablespace", 1234, 0, 4321, 12),
	)

	metricChan := make(chan newrelicMetricSender, 10)
	var collectorWg sync.WaitGroup

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	collectorWg.Add(1)
	mc := metricsCollector{
		integration:    i,
		db:             dbWrapper,
		wg:             &collectorWg,
		instanceLookUp: lookup,
	}
	go mc.collectTableSpaces(&collectorWg, metricChan, tablespaceCollections)

	collectorWg.Wait()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Expectations not met: %s", err.Error())
	}
}

func Test_PopulateMetrics_FromCustomQueryFile(t *testing.T) {
	qf, err := filepath.Abs(filepath.Join("..", "test", "fixtures", "custom_query_multi.yml"))
	if err != nil {
		t.Error(err)
	}

	args = argumentList{
		Hostname:            "testhost",
		Port:                "1234",
		ServiceName:         "testServiceName",
		CustomMetricsConfig: qf,
	}
	defer func() { args = argumentList{} }()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	mock.MatchExpectationsInOrder(false)

	// there are 2 queries, so this query will be called 2 times
	columns := []string{"val1"}
	mock.ExpectQuery("SELECT.*FROM v\\$instance").WillReturnRows(
		sqlmock.NewRows(columns).AddRow("1"),
	)

	mock.ExpectQuery("SELECT.*FROM v\\$instance").WillReturnRows(
		sqlmock.NewRows(columns).AddRow("1"),
	)

	// queries from query file
	columns = []string{"val1", "val2"}
	mock.ExpectQuery("SELECT.*FROM numbers.*").WillReturnRows(
		sqlmock.NewRows(columns).AddRow("one", "two"),
	)

	mock.ExpectQuery("SELECT.*FROM somewhere.*").WillReturnRows(
		sqlmock.NewRows(columns).AddRow("something", "otherthing"),
	)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)

	var wg sync.WaitGroup
	wg.Add(1)

	ch := make(chan newrelicMetricSender)

	results := []newrelicMetricSender{}

	PopulateCustomMetricsFromFile(dbWrapper, &wg, ch, args.CustomMetricsConfig)

	go func() {
		wg.Wait()
		close(ch)
	}()

	for result := range ch {
		results = append(results, result)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %v", len(results))
	}
}

func Test_CollectMetrics_SkippedList(t *testing.T) {
	i, err := integration.New("oracletest", "0.0.1")
	if err != nil {
		t.Error(err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	mock.ExpectQuery(`.*GV\$sesstat, GV\$statname, GV\$INSTANCE.*`)

	dbWrapper := database.NewDBWrapper(sqlx.NewDb(db, "sqlmock"))
	var populaterWg sync.WaitGroup

	skipMetricGroup := []string{"sgauga_total_memory"}

	populaterWg.Add(1)
	mc := metricsCollector{
		integration:       i,
		db:                dbWrapper,
		wg:                &populaterWg,
		instanceLookUp:    map[string]string{"1": "MyInstance"},
		skipMetricsGroups: skipMetricGroup,
	}
	go mc.collect()
	populaterWg.Wait()

	if err := mock.ExpectationsWereMet(); err == nil {
		t.Errorf("Metrics group should be excluded from collection: %s", err)
	}
}
