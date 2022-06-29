package main

import (
	"reflect"
	"sync"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/kr/pretty"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/nri-oracledb/src/database"
)

func TestOracleTablespaceMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	mock.ExpectQuery(`SELECT a.TABLESPACE_NAME.*`).WillReturnRows(
		sqlmock.NewRows([]string{"TABLESPACE_NAME", "USED", "OFFLINE", "SIZE", "USED_PERCENT"}).
			AddRow("testtablespace", 1234, 0, 4321, 12),
	)

	var wg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 10)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	wg.Add(1)
	go oracleTablespaceMetrics.Collect(dbWrapper, &wg, metricChan)
	go func() {
		wg.Wait()
		close(metricChan)
	}()
	var generatedMetrics []newrelicMetricSender
	for {
		newMetric, ok := <-metricChan
		if !ok {
			break
		}
		generatedMetrics = append(generatedMetrics, newMetric)
	}

	expectedMetrics := []newrelicMetricSender{
		{
			metric: &newrelicMetric{
				name:       "tablespace.spaceUsedPercentage",
				value:      int64(12),
				metricType: metric.GAUGE,
			},
			metadata: map[string]string{
				"tablespace": "testtablespace",
			},
		},
		{
			metric: &newrelicMetric{
				name:       "tablespace.isOffline",
				value:      int64(0),
				metricType: metric.GAUGE,
			},
			metadata: map[string]string{
				"tablespace": "testtablespace",
			},
		},
	}

	if !reflect.DeepEqual(expectedMetrics, generatedMetrics) {
		t.Errorf("failed to get expected metric: %s", pretty.Diff(expectedMetrics, generatedMetrics))
	}
}

func Test_dbIDTablespaceMetric(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	mock.ExpectQuery(`SELECT t1.TABLESPACE_NAME, t2.DBID.*`).WillReturnRows(
		sqlmock.NewRows([]string{"TABLESPACE_NAME", "DBID"}).
			AddRow("testtablespace", 12345),
	)

	var wg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 10)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	wg.Add(1)
	go dbIDTablespaceMetric.Collect(dbWrapper, &wg, metricChan)
	go func() {
		wg.Wait()
		close(metricChan)
	}()
	var generatedMetrics []newrelicMetricSender
	for {
		newMetric, ok := <-metricChan
		if !ok {
			break
		}
		generatedMetrics = append(generatedMetrics, newMetric)
	}

	expectedMetrics := []newrelicMetricSender{
		{
			metric: &newrelicMetric{
				name:       "dbID",
				value:      "12345",
				metricType: metric.ATTRIBUTE,
			},
			metadata: map[string]string{
				"tablespace": "testtablespace",
			},
		},
	}

	if !reflect.DeepEqual(expectedMetrics, generatedMetrics) {
		t.Errorf("failed to get expected metric: %s", pretty.Diff(expectedMetrics, generatedMetrics))
	}
}

func Test_globalNameTablespaceMetric(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}
	mock.ExpectQuery(`SELECT t1.TABLESPACE_NAME, t2.GLOBAL_NAME.*`).WillReturnRows(
		sqlmock.NewRows([]string{"TABLESPACE_NAME", "GLOBAL_NAME"}).
			AddRow("testtablespace", "global_name"),
	)

	var wg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 10)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	wg.Add(1)
	go globalNameTablespaceMetric.Collect(dbWrapper, &wg, metricChan)
	go func() {
		wg.Wait()
		close(metricChan)
	}()
	var generatedMetrics []newrelicMetricSender
	for {
		newMetric, ok := <-metricChan
		if !ok {
			break
		}
		generatedMetrics = append(generatedMetrics, newMetric)
	}

	expectedMetrics := []newrelicMetricSender{
		{
			metric: &newrelicMetric{
				name:       "globalName",
				value:      "global_name",
				metricType: metric.ATTRIBUTE,
			},
			metadata: map[string]string{
				"tablespace": "testtablespace",
			},
		},
	}

	if !reflect.DeepEqual(expectedMetrics, generatedMetrics) {
		t.Errorf("failed to get expected metric: %s", pretty.Diff(expectedMetrics, generatedMetrics))
	}
}

func Test_dbIDInstanceMetric(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	mock.ExpectQuery(`SELECT t1.INST_ID, t2.DBID.*`).WillReturnRows(
		sqlmock.NewRows([]string{"INST_ID", "DBID"}).
			AddRow(1, 12345),
	)

	var wg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 10)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	wg.Add(1)
	go dbIDInstanceMetric.Collect(dbWrapper, &wg, metricChan)
	go func() {
		wg.Wait()
		close(metricChan)
	}()
	var generatedMetrics []newrelicMetricSender
	for {
		newMetric, ok := <-metricChan
		if !ok {
			break
		}
		generatedMetrics = append(generatedMetrics, newMetric)
	}

	expectedMetrics := []newrelicMetricSender{
		{
			metric: &newrelicMetric{
				name:       "dbID",
				value:      "12345",
				metricType: metric.ATTRIBUTE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
	}

	if !reflect.DeepEqual(expectedMetrics, generatedMetrics) {
		t.Errorf("failed to get expected metric: %s", pretty.Diff(expectedMetrics, generatedMetrics))
	}
}

func Test_globalNameInstanceMetric(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	mock.ExpectQuery(`SELECT t1.INST_ID, t2.GLOBAL_NAME.*`).WillReturnRows(
		sqlmock.NewRows([]string{"INST_ID", "GLOBAL_NAME"}).
			AddRow(1, "global_name"),
	)

	var wg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 10)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	wg.Add(1)
	go globalNameInstanceMetric.Collect(dbWrapper, &wg, metricChan)
	go func() {
		wg.Wait()
		close(metricChan)
	}()
	var generatedMetrics []newrelicMetricSender
	for {
		newMetric, ok := <-metricChan
		if !ok {
			break
		}
		generatedMetrics = append(generatedMetrics, newMetric)
	}

	expectedMetrics := []newrelicMetricSender{
		{
			metric: &newrelicMetric{
				name:       "globalName",
				value:      "global_name",
				metricType: metric.ATTRIBUTE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
	}

	if !reflect.DeepEqual(expectedMetrics, generatedMetrics) {
		t.Errorf("failed to get expected metric: %s", pretty.Diff(expectedMetrics, generatedMetrics))
	}
}

func TestOracleTablespaceGlobalNameMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}
	mock.ExpectQuery(`SELECT t1.TABLESPACE_NAME, t2.GLOBAL_NAME.*`).WillReturnRows(
		sqlmock.NewRows([]string{"TABLESPACE_NAME", "GLOBAL_NAME"}).
			AddRow("testtablespace", "global_name"),
	)

	var wg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 10)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	wg.Add(1)
	go globalNameTablespaceMetric.Collect(dbWrapper, &wg, metricChan)
	go func() {
		wg.Wait()
		close(metricChan)
	}()
	var generatedMetrics []newrelicMetricSender
	for {
		newMetric, ok := <-metricChan
		if !ok {
			break
		}
		generatedMetrics = append(generatedMetrics, newMetric)
	}

	expectedMetrics := []newrelicMetricSender{
		{
			metric: &newrelicMetric{
				name:       "globalName",
				value:      "global_name",
				metricType: metric.ATTRIBUTE,
			},
			metadata: map[string]string{
				"tablespace": "testtablespace",
			},
		},
	}

	if !reflect.DeepEqual(expectedMetrics, generatedMetrics) {
		t.Errorf("failed to get expected metric: %s", pretty.Diff(expectedMetrics, generatedMetrics))
	}
}

func TestOracleTablespaceMetrics_Whitelist(t *testing.T) {
	tablespaceWhiteList = []string{"testtablespace", "othertablespace"}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	mock.ExpectQuery(`.*WHERE a.TABLESPACE_NAME IN \('testtablespace','othertablespace'\).*`).WillReturnRows(
		sqlmock.NewRows([]string{"TABLESPACE_NAME", "USED", "OFFLINE", "SIZE", "USED_PERCENT"}).
			AddRow("testtablespace", 1234, 0, 4321, 12),
	)

	var wg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 10)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	wg.Add(1)
	go oracleTablespaceMetrics.Collect(dbWrapper, &wg, metricChan)
	go func() {
		wg.Wait()
		close(metricChan)
	}()
	var generatedMetrics []newrelicMetricSender
	for {
		newMetric, ok := <-metricChan
		if !ok {
			break
		}
		generatedMetrics = append(generatedMetrics, newMetric)
	}

	expectedMetrics := []newrelicMetricSender{
		{
			metric: &newrelicMetric{
				name:       "tablespace.spaceUsedPercentage",
				value:      int64(12),
				metricType: metric.GAUGE,
			},
			metadata: map[string]string{
				"tablespace": "testtablespace",
			},
		},
		{
			metric: &newrelicMetric{
				name:       "tablespace.isOffline",
				value:      int64(0),
				metricType: metric.GAUGE,
			},
			metadata: map[string]string{
				"tablespace": "testtablespace",
			},
		},
	}

	if !reflect.DeepEqual(expectedMetrics, generatedMetrics) {
		t.Errorf("failed to get expected metric: %s", pretty.Diff(expectedMetrics, generatedMetrics))
	}
}

func TestOracleReadWriteMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	mock.ExpectQuery(".*").WillReturnRows(
		sqlmock.NewRows([]string{"INST_ID", "PhysicalReads", "PhysicalWrites", "PhysicalBlockReads", "PhysicalBlockWrites", "ReadTime", "WriteTime"}).
			AddRow("1", 12, 23, 34, 45, 56, 67),
	)

	var wg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 100)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	wg.Add(1)
	go oracleReadWriteMetrics.Collect(dbWrapper, &wg, metricChan)
	go func() {
		wg.Wait()
		close(metricChan)
	}()
	var generatedMetrics []newrelicMetricSender
	for i := 0; i < 6; i++ {
		generatedMetrics = append(generatedMetrics, <-metricChan)
	}

	expectedMetrics := []newrelicMetricSender{
		{
			metric: &newrelicMetric{
				name:       "disk.reads",
				value:      int64(12),
				metricType: metric.RATE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
		{
			metric: &newrelicMetric{
				name:       "disk.writes",
				value:      int64(23),
				metricType: metric.RATE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
		{
			metric: &newrelicMetric{
				name:       "disk.blocksRead",
				value:      int64(34),
				metricType: metric.RATE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
		{
			metric: &newrelicMetric{
				name:       "disk.blocksWritten",
				value:      int64(45),
				metricType: metric.RATE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
		{
			metric: &newrelicMetric{
				name:       "disk.readTimeInMilliseconds",
				value:      int64(56),
				metricType: metric.RATE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
		{
			metric: &newrelicMetric{
				name:       "disk.writeTimeInMilliseconds",
				value:      int64(67),
				metricType: metric.RATE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
	}

	if !reflect.DeepEqual(expectedMetrics, generatedMetrics) {
		t.Errorf("failed to get expected metric: %s", pretty.Diff(expectedMetrics, generatedMetrics))
	}

}

func TestOraclePgaMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	mock.ExpectQuery(".*").WillReturnRows(
		sqlmock.NewRows([]string{"INST_ID", "NAME", "VALUE"}).
			AddRow("1", "global memory bound", 1234.0),
	)

	var wg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 100)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	wg.Add(1)
	go oraclePgaMetrics.Collect(dbWrapper, &wg, metricChan)
	go func() {
		wg.Wait()
		close(metricChan)
	}()

	var generatedMetrics []newrelicMetricSender
	for {
		newMetric, ok := <-metricChan
		if !ok {
			break
		}
		generatedMetrics = append(generatedMetrics, newMetric)
	}

	expectedMetrics := []newrelicMetricSender{
		{
			metric: &newrelicMetric{
				name:       "memory.pgaMaxSizeInBytes",
				value:      1234.0,
				metricType: metric.GAUGE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
	}

	if !reflect.DeepEqual(expectedMetrics, generatedMetrics) {
		t.Errorf("failed to get expected metric: %s", pretty.Diff(expectedMetrics, generatedMetrics))
	}

}

func TestOracleSysMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	mock.ExpectQuery(".*").WillReturnRows(
		sqlmock.NewRows([]string{"INST_ID", "METRIC_NAME", "VALUE"}).
			AddRow("1", "Buffer Cache Hit Ratio", 0.5),
	)

	var wg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 10)

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	dbWrapper := database.NewDBWrapper(sqlxDB)
	wg.Add(1)
	var generatedMetrics []newrelicMetricSender
	go oracleSysMetrics.Collect(dbWrapper, &wg, metricChan)
	go func() {
		wg.Wait()
		close(metricChan)
	}()

	for {
		metric, ok := <-metricChan
		if !ok {
			break
		}

		generatedMetrics = append(generatedMetrics, metric)

	}

	expectedMetrics := []newrelicMetricSender{
		{
			metric: &newrelicMetric{
				name:       "memory.bufferCacheHitRatio",
				value:      0.5,
				metricType: metric.GAUGE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
	}

	if !reflect.DeepEqual(expectedMetrics, generatedMetrics) {
		t.Errorf("failed to get expected metric: %s", pretty.Diff(expectedMetrics, generatedMetrics))
	}

}

func TestInMetrics(t *testing.T) {
	samplemetrics := []*oracleMetric{
		{
			name:          "memory.pgaInUseInBytes",
			metricType:    metric.GAUGE,
			defaultMetric: false,
			identifier:    "total PGA inuse",
		},
		{
			name:          "memory.pgaAllocatedInBytes",
			metricType:    metric.GAUGE,
			defaultMetric: false,
			identifier:    "total PGA allocated",
		},
		{
			name:          "memory.pgaFreeableInBytes",
			metricType:    metric.GAUGE,
			defaultMetric: false,
			identifier:    "total freeable PGA memory",
		},
	}

	expectedResult := ` METRIC_NAME IN ('total PGA inuse','total PGA allocated','total freeable PGA memory')`

	generatedResult := inMetrics("METRIC_NAME", samplemetrics)

	if !reflect.DeepEqual(expectedResult, generatedResult) {
		t.Errorf("failed to get expected result: %s", pretty.Diff(expectedResult, generatedResult))
	}
}

func TestInWhiteList(t *testing.T) {
	tablespaceWhiteList = []string{"SYSTEM", "USER", "TABLESPACE1"}

	tests := []struct {
		field          string
		addWhere       bool
		grouped        bool
		expectedResult string
	}{
		{"TABLESPACE_NAME", true, true, ` WHERE TABLESPACE_NAME IN ('SYSTEM','USER','TABLESPACE1') GROUP BY TABLESPACE_NAME`},
		{"a.TABLESPACE_NAME", true, false, ` WHERE a.TABLESPACE_NAME IN ('SYSTEM','USER','TABLESPACE1')`},
		{"b.TABLESPACE_NAME", false, true, ` AND b.TABLESPACE_NAME IN ('SYSTEM','USER','TABLESPACE1') GROUP BY b.TABLESPACE_NAME`},
		{"c.TABLESPACE_NAME", false, false, ` AND c.TABLESPACE_NAME IN ('SYSTEM','USER','TABLESPACE1')`},
	}

	for _, test := range tests {
		generatedResult := inWhitelist(test.field, test.addWhere, test.grouped)
		if !reflect.DeepEqual(test.expectedResult, generatedResult) {
			t.Errorf("failed to get expected result: %s", pretty.Diff(test.expectedResult, generatedResult))
		}
	}
}
