package main

import (
	"reflect"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/data/metric"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/kr/pretty"
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

	sqlxDb := sqlx.NewDb(db, "sqlmock")
	wg.Add(1)
	go oracleTablespaceMetrics.Collect(sqlxDb, &wg, metricChan)
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

	sqlxDb := sqlx.NewDb(db, "sqlmock")
	wg.Add(1)
	go dbIDTablespaceMetric.Collect(sqlxDb, &wg, metricChan)
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

	sqlxDb := sqlx.NewDb(db, "sqlmock")
	wg.Add(1)
	go globalNameTablespaceMetric.Collect(sqlxDb, &wg, metricChan)
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

	sqlxDb := sqlx.NewDb(db, "sqlmock")
	wg.Add(1)
	go dbIDInstanceMetric.Collect(sqlxDb, &wg, metricChan)
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

	sqlxDb := sqlx.NewDb(db, "sqlmock")
	wg.Add(1)
	go globalNameInstanceMetric.Collect(sqlxDb, &wg, metricChan)
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

	sqlxDb := sqlx.NewDb(db, "sqlmock")
	wg.Add(1)
	go globalNameTablespaceMetric.Collect(sqlxDb, &wg, metricChan)
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

func TestOracleTablespaceMetrics_Whitlist(t *testing.T) {
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

	sqlxDb := sqlx.NewDb(db, "sqlmock")
	wg.Add(1)
	go oracleTablespaceMetrics.Collect(sqlxDb, &wg, metricChan)
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

	sqlxDb := sqlx.NewDb(db, "sqlmock")
	wg.Add(1)
	go oracleReadWriteMetrics.Collect(sqlxDb, &wg, metricChan)
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

	sqlxDb := sqlx.NewDb(db, "sqlmock")
	wg.Add(1)
	go oraclePgaMetrics.Collect(sqlxDb, &wg, metricChan)
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

	sqlxDb := sqlx.NewDb(db, "sqlmock")
	wg.Add(1)
	var generatedMetrics []newrelicMetricSender
	go oracleSysMetrics.Collect(sqlxDb, &wg, metricChan)
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
