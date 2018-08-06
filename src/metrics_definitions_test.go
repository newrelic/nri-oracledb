package main

import (
	"reflect"
	"sync"
	"testing"

	"github.com/newrelic/infra-integrations-sdk/data/metric"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/kr/pretty"
)

func TestOracleTablespaceMetrics(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Error(err)
	}

	mock.ExpectQuery(".*").WillReturnRows(
		sqlmock.NewRows([]string{"TABLESPACE_NAME", "USED", "OFFLINE", "SIZE", "USED_PERCENT"}).
			AddRow("testtablespace", 1234, 0, 4321, 12),
	)

	var wg sync.WaitGroup
	metricChan := make(chan newrelicMetricSender, 10)

	wg.Add(1)
	go oracleTablespaceMetrics.Collect(db, &wg, metricChan)
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

	wg.Add(1)
	go oracleReadWriteMetrics.Collect(db, &wg, metricChan)
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
				metricType: metric.GAUGE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
		{
			metric: &newrelicMetric{
				name:       "disk.writes",
				value:      int64(23),
				metricType: metric.GAUGE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
		{
			metric: &newrelicMetric{
				name:       "disk.blocksRead",
				value:      int64(34),
				metricType: metric.GAUGE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
		{
			metric: &newrelicMetric{
				name:       "disk.blocksWritten",
				value:      int64(45),
				metricType: metric.GAUGE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
		{
			metric: &newrelicMetric{
				name:       "disk.readTime",
				value:      int64(56),
				metricType: metric.GAUGE,
			},
			metadata: map[string]string{
				"instanceID": "1",
			},
		},
		{
			metric: &newrelicMetric{
				name:       "disk.writeTime",
				value:      int64(67),
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

	wg.Add(1)
	go oraclePgaMetrics.Collect(db, &wg, metricChan)
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
				name:       "memory.pgaMaxSize",
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

	wg.Add(1)
	var generatedMetrics []newrelicMetricSender
	go oracleSysMetrics.Collect(db, &wg, metricChan)
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
