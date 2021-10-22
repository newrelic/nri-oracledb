package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/godror/godror"
	"github.com/jmoiron/sqlx"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/log"
)

const defaultCustomSampleType = "OracleCustomSample"

// oracleMetric is a storage struct for the information needed to parse
// a metric from a query and create a newrelicMetric
type oracleMetric struct {
	name          string
	identifier    string
	metricType    metric.SourceType
	defaultMetric bool
}

// newrelicMetric is a storage struct for all the information needed
// to insert a metric into a metric set
type newrelicMetric struct {
	name       string
	metricType metric.SourceType
	value      interface{}
}

// newrelicMetricSender is a wrapper struct meant to send a metric through
// a channel along with the metadata needed to insert it into the correct
// metric set
type newrelicMetricSender struct {
	metric              *newrelicMetric
	metadata            map[string]string
	isCustom            bool
	customMetrics       []map[string]interface{}
	metricTypeOverrides map[string]metricType
}

// oracleMetricGroup is a struct that contains all the information needed
// to collect the list of metrics contained in it: the db query to retrieve
// the metrics, the list of metrics to collect from that query, and a function
// to parse the metrics into structs to send down a channel
type oracleMetricGroup struct {
	sqlQuery         func() string
	metrics          []*oracleMetric
	metricsGenerator func(*sql.Rows, []*oracleMetric, chan<- newrelicMetricSender) error
}

// Collect is a method on oracleMetricGroups which collects the metrics defined
// by the metric group and sends them down the channel passed to it
func (mg *oracleMetricGroup) Collect(db *sqlx.DB, wg *sync.WaitGroup, metricChan chan<- newrelicMetricSender) {
	defer wg.Done()

	rows, err := db.Query(mg.sqlQuery())
	if err != nil {
		log.Error("Failed to execute query %s: %s", mg.sqlQuery(), err)
		return
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			log.Error("Failed to close rows: %s", err)
		}
	}()

	if err = mg.metricsGenerator(rows, mg.metrics, metricChan); err != nil {
		log.Error("Failed to generate metrics from db response for query %s: %s", mg.sqlQuery(), err)
		return
	}
}

// This function is necessary because of how sql-mock auto-converts
// types for the sql driver. More information about the issue
// is here https://github.com/DATA-DOG/go-sqlmock/issues/133
func getInstanceIDString(originalID interface{}) string {
	switch id := originalID.(type) {
	case godror.Number:
		return id.String()
	case int:
		return strconv.Itoa(id)
	case string:
		return id
	default:
		return ""
	}
}

func columnMetricsGenerator(rows *sql.Rows, metrics []*oracleMetric, metricChan chan<- newrelicMetricSender) error {

	columnNames, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to retrieve columns from rows")
	}

	for rows.Next() {
		// Make an array of columns and an array of pointers to each element of the array
		columns := make([]interface{}, len(columnNames))
		pointers := make([]interface{}, len(columnNames))
		for i := 0; i < len(columnNames); i++ {
			pointers[i] = &columns[i]
		}

		// Scan the row into the array of pointers
		err := rows.Scan(pointers...)
		if err != nil {
			return err
		}

		// Put the values of the row into a column with the column name as the key
		rowMap := make(map[string]interface{})
		for i, column := range columnNames {
			rowMap[column] = columns[i]
		}

		// Create each metric in the list of metrics we want to collect
		for _, metric := range metrics {
			if metric.defaultMetric || args.ExtendedMetrics {
				newMetric := &newrelicMetric{
					name:       metric.name,
					metricType: metric.metricType,
					value:      rowMap[metric.identifier],
				}

				metadata := map[string]string{"instanceID": getInstanceIDString(rowMap["INST_ID"])}

				// Send the new metric down the channel
				metricChan <- newrelicMetricSender{metric: newMetric, metadata: metadata}
			}
		}
	}

	return nil
}

type customMetricGroup struct {
	Query string
}

// Collect is a method on oracleMetricGroups which collects the metrics defined
// by the metric group and sends them down the channel passed to it
func (mg *customMetricGroup) Collect(db *sqlx.DB, wg *sync.WaitGroup, metricChan chan<- newrelicMetricSender) {
	defer wg.Done()

	rows, err := db.Queryx(`SELECT INSTANCE_NUMBER FROM v$instance`)
	if err != nil {
		log.Error("Failed to execute query %s: %s", mg.Query, err)
		return
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			log.Error("Failed to close rows: %s", err)
		}
	}()

	var instanceID string
	for rows.Next() {
		err = rows.Scan(&instanceID)
		if err != nil {
			log.Error("Failed to get instance ID %s: %s", mg.Query, err)
			return
		}
	}

	rows, err = db.Queryx(mg.Query)
	if err != nil {
		log.Error("Failed to execute query %s: %s", mg.Query, err)
		return
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			log.Error("Failed to close rows: %s", err)
		}
	}()

	sender := newrelicMetricSender{
		isCustom: true,
		metadata: map[string]string{
			"instanceID": instanceID,
			"sampleName": defaultCustomSampleType,
		},
	}

	for rows.Next() {
		row := make(map[string]interface{})
		err := rows.MapScan(row)
		if err != nil {
			log.Error("Failed to scan custom query row: %s", err)
			return
		}

		convertedMetrics := make(map[string]interface{})
		for key, val := range row {
			convertedMetrics[key] = sanitizeValue(val)
		}

		sender.customMetrics = append(sender.customMetrics, convertedMetrics)
	}

	if sender.customMetrics == nil {
		log.Info("Query did not return any results: %s", mg.Query)
	}

	metricChan <- sender
}

var oracleLongRunningQueries = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT inst_id, sum(num) AS total FROM ((
      SELECT i.inst_id, 1 AS num
      FROM gv$session s, gv$instance i
      WHERE i.inst_id=s.inst_id
      AND s.status='ACTIVE'
      AND s.type <>'BACKGROUND'
      AND s.last_call_et > 60
      GROUP BY i.inst_id
    ) UNION (
      SELECT i.inst_id, 0 AS num
      FROM gv$session s, gv$instance i
      WHERE i.inst_id=s.inst_id
    ))
    GROUP BY inst_id
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "longRunningQueries",
			identifier:    "TOTAL",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: columnMetricsGenerator,
}

var oracleSGAUGATotalMemory = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT SUM(value) AS sum,inst.inst_id
    FROM GV$sesstat, GV$statname, GV$INSTANCE inst
    WHERE name = 'session uga memory max'
    AND GV$sesstat.statistic#=GV$statname.statistic#
    AND GV$sesstat.inst_id=inst.inst_id
    AND GV$statname.inst_id=inst.inst_id
    GROUP BY inst.inst_id
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "sga.ugaTotalMemoryInBytes",
			identifier:    "SUM",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: columnMetricsGenerator,
}

var oracleSGASharedPoolLibraryCacheSharableStatement = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT SUM(sqlarea.sharable_mem) AS sum,inst.inst_id
    FROM GV$sqlarea sqlarea, GV$INSTANCE inst
    WHERE sqlarea.executions > 5
    AND inst.inst_id=sqlarea.inst_id
    GROUP BY inst.inst_id
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "sga.sharedPoolLibraryCacheShareableMemoryPerStatementInBytes",
			identifier:    "SUM",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: columnMetricsGenerator,
}

var oracleSGASharedPoolLibraryCacheShareableUser = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT SUM(250 * sqlarea.users_opening) AS sum,inst.inst_id
    FROM GV$sqlarea sqlarea, GV$INSTANCE inst
    WHERE inst.inst_id=sqlarea.inst_id
    GROUP BY inst.inst_id
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "sga.sharedPoolLibraryCacheShareableMemoryPerUserInBytes",
			identifier:    "SUM",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: columnMetricsGenerator,
}

var oracleSGASharedPoolLibraryCacheReloadRatio = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT (sum(libcache.reloads)/sum(libcache.pins))  AS ratio,inst.inst_id
    FROM GV$librarycache libcache, GV$INSTANCE inst
    WHERE inst.inst_id=libcache.inst_id
    GROUP BY inst.inst_id
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "sga.sharedPoolLibraryCacheReloadRatio",
			identifier:    "RATIO",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: columnMetricsGenerator,
}

var oracleSGASharedPoolLibraryCacheHitRatio = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT libcache.gethitratio as ratio,inst.inst_id
    FROM GV$librarycache libcache, GV$INSTANCE inst
    WHERE namespace='SQL AREA'
    AND inst.inst_id=libcache.inst_id
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "sga.sharedPoolLibraryCacheHitRatio",
			identifier:    "RATIO",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: columnMetricsGenerator,
}

var oracleSGASharedPoolDictCacheRatio = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT (SUM(rcache.getmisses)/SUM(rcache.gets)) as ratio,inst.inst_id
    FROM GV$rowcache rcache, GV$INSTANCE inst
    WHERE inst.inst_id=rcache.inst_id
    GROUP BY inst.inst_id
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "sga.sharedPoolDictCacheMissRatio",
			identifier:    "RATIO",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: columnMetricsGenerator,
}

var oracleSGALogBufferSpaceWaits = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT count(wait.inst_id) as count,inst.inst_id
    FROM GV$SESSION_WAIT wait, GV$INSTANCE inst
    WHERE wait.event like 'log buffer space%'
    AND inst.inst_id=wait.inst_id
    GROUP BY inst.inst_id
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "sga.logBufferSpaceWaits",
			identifier:    "COUNT",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: columnMetricsGenerator,
}

var oracleSGALogAllocRetries = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT (rbar.value/re.value) as ratio, inst.inst_id
    FROM GV$SYSSTAT rbar, GV$SYSSTAT re, GV$INSTANCE inst
    WHERE rbar.name like 'redo buffer allocation retries'
    AND re.name like 'redo entries'
    AND re.inst_id=inst.inst_id AND rbar.inst_id=inst.inst_id
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "sga.logBufferAllocationRetriesRatio",
			identifier:    "RATIO",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: columnMetricsGenerator,
}

var oracleSGAHitRatio = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT inst.inst_id,(1 - (phy.value - lob.value - dir.value)/ses.value) as ratio
    FROM GV$SYSSTAT ses, GV$SYSSTAT lob, GV$SYSSTAT dir, GV$SYSSTAT phy, GV$INSTANCE inst
    WHERE ses.name='session logical reads'
    AND dir.name='physical reads direct'
    AND lob.name='physical reads direct (lob)'
    AND phy.name='physical reads'
    AND ses.inst_id=inst.inst_id
    AND lob.inst_id=inst.inst_id
    AND dir.inst_id=inst.inst_id
    AND phy.inst_id=inst.inst_id
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "sga.hitRatio",
			identifier:    "RATIO",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: columnMetricsGenerator,
}

var oracleSysstat = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
		SELECT sysstat.value,inst.inst_id, sysstat.name
		FROM GV$SYSSTAT sysstat, GV$INSTANCE inst
		WHERE sysstat.inst_id=inst.inst_id
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "sga.logBufferRedoAllocationRetries",
			identifier:    "redo buffer allocation retries",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "sga.logBufferRedoEntries",
			identifier:    "redo entries",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "sorts.memoryInBytes",
			identifier:    "sorts (memory)",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "sorts.diskInBytes",
			identifier:    "sorts (disk)",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricsChan chan<- newrelicMetricSender) error {

		var sysScanner struct {
			value  int
			instID int
			name   string
		}

		for rows.Next() {

			// Scan the row into a struct
			err := rows.Scan(&sysScanner.value, &sysScanner.instID, &sysScanner.name)
			if err != nil {
				return err
			}

			// Match the metric to one of the metrics we want to collect
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					if sysScanner.name == metric.identifier {
						newMetric := &newrelicMetric{
							name:       metric.name,
							value:      sysScanner.value,
							metricType: metric.metricType,
						}

						metadata := map[string]string{"instanceID": strconv.Itoa(sysScanner.instID)}

						// Send the metric down the channel
						metricsChan <- newrelicMetricSender{metadata: metadata, metric: newMetric}
						break
					}
				}
			}
		}

		return nil
	},
}

var oracleSGA = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT sga.name, sga.value,inst.inst_id
    FROM GV$SGA sga, GV$INSTANCE inst
    WHERE sga.inst_id=inst.inst_id
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "sga.fixedSizeInBytes",
			identifier:    "Fixed Size",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "sga.redoBuffersInBytes",
			identifier:    "Redo Buffers",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricsChan chan<- newrelicMetricSender) error {

		var sysScanner struct {
			value  int
			instID int
			name   string
		}

		for rows.Next() {

			// Scan the row into a struct
			err := rows.Scan(&sysScanner.name, &sysScanner.value, &sysScanner.instID)
			if err != nil {
				return err
			}

			// Match the metric to one of the metrics we want to collect
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					if sysScanner.name == metric.identifier {
						newMetric := &newrelicMetric{
							name:       metric.name,
							value:      sysScanner.value,
							metricType: metric.metricType,
						}

						metadata := map[string]string{"instanceID": strconv.Itoa(sysScanner.instID)}

						// Send the metric down the channel
						metricsChan <- newrelicMetricSender{metadata: metadata, metric: newMetric}
						break
					}
				}
			}
		}

		return nil
	},
}

var oracleRollbackSegments = oracleMetricGroup{
	sqlQuery: func() string {
		query := `SELECT
      SUM(stat.gets) AS gets,
      sum(stat.waits) AS waits,
      sum(stat.waits)/sum(stat.gets) AS ratio,
      inst.inst_id
    FROM GV$ROLLSTAT stat, GV$INSTANCE inst
    WHERE stat.inst_id=inst.inst_id
    GROUP BY inst.inst_id
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "rollbackSegments.gets",
			identifier:    "GETS",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "rollbackSegments.waits",
			identifier:    "WAITS",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "rollbackSegments.ratioWait",
			identifier:    "RATIO",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: columnMetricsGenerator,
}

var oracleRedoLogWaits = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT
      sysevent.total_waits,
      inst.inst_id,
      sysevent.event
    FROM
      GV$SYSTEM_EVENT sysevent,
      GV$INSTANCE inst
    WHERE sysevent.inst_id=inst.inst_id
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "redoLog.waits",
			identifier:    "log file parallel write",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "redoLog.logFileSwitch",
			identifier:    "log file switch completion",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "redoLog.logFileSwitchCheckpointIncomplete",
			identifier:    "log file switch (check",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "redoLog.logFileSwitchArchivingNeeded",
			identifier:    "log file switch (arch",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "sga.bufferBusyWaits",
			identifier:    "buffer busy waits",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "sga.freeBufferWaits",
			identifier:    "freeBufferWaits",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "sga.freeBufferInspected",
			identifier:    "free buffer inspected",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricsChan chan<- newrelicMetricSender) error {

		var sysScanner struct {
			totalWaits int
			instID     int
			event      string
		}

		for rows.Next() {

			// Scan the row into a struct
			err := rows.Scan(&sysScanner.totalWaits, &sysScanner.instID, &sysScanner.event)
			if err != nil {
				return err
			}

			// Match the metric to one of the metrics we want to collect
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					if strings.Contains(sysScanner.event, metric.identifier) {
						newMetric := &newrelicMetric{
							name:       metric.name,
							value:      sysScanner.totalWaits,
							metricType: metric.metricType,
						}

						metadata := map[string]string{"instanceID": strconv.Itoa(sysScanner.instID)}

						// Send the metric down the channel
						metricsChan <- newrelicMetricSender{metadata: metadata, metric: newMetric}
						break
					}
				}
			}
		}

		return nil
	},
}

var oraclePDBDatafilesOffline = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT
      sum(CASE WHEN ONLINE_STATUS IN ('ONLINE','SYSTEM','RECOVER') THEN 0 ELSE 1 END)
        AS "PDB_DATAFILES_OFFLINE",
      a.TABLESPACE_NAME
    FROM cdb_data_files a, cdb_pdbs b
    WHERE a.con_id = b.con_id
		`

		if len(tablespaceWhiteList) > 0 {
			query += `
			AND a.TABLESPACE_NAME IN (`

			for i, tablespace := range tablespaceWhiteList {
				query += fmt.Sprintf(`'%s'`, tablespace)

				if i != len(tablespaceWhiteList)-1 {
					query += ","
				}
			}

			query += ")"
		}

		query += `
		GROUP BY a.TABLESPACE_NAME`
		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "tablespace.offlinePDBDatafiles",
			identifier:    "PDB_DATAFILES_OFFLINE",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricChan chan<- newrelicMetricSender) error {

		columnNames, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("failed to retrieve columns from rows")
		}

		for rows.Next() {
			// Make an array of columns and an array of pointers to each element of the array
			columns := make([]interface{}, len(columnNames))
			pointers := make([]interface{}, len(columnNames))
			for i := 0; i < len(columnNames); i++ {
				pointers[i] = &columns[i]
			}

			// Scan the row into the array of pointers
			err := rows.Scan(pointers...)
			if err != nil {
				return err
			}

			// Put the values of the row into a column with the column name as the key
			rowMap := make(map[string]interface{})
			for i, column := range columnNames {
				rowMap[column] = columns[i]
			}

			// Create each metric in the list of metrics we want to collect
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					newMetric := &newrelicMetric{
						name:       metric.name,
						metricType: metric.metricType,
						value:      rowMap[metric.identifier],
					}

					metadata := map[string]string{"tablespace": rowMap["TABLESPACE_NAME"].(string)}

					// Send the new metric down the channel
					metricChan <- newrelicMetricSender{metric: newMetric, metadata: metadata}
				}
			}
		}

		return nil
	},
}

var oracleCDBDatafilesOffline = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT
      sum(CASE WHEN ONLINE_STATUS IN ('ONLINE', 'SYSTEM','RECOVER') THEN 0 ELSE 1 END)
        AS "CDB_DATAFILES_OFFLINE" ,
      TABLESPACE_NAME
    FROM dba_data_files
		`

		if len(tablespaceWhiteList) > 0 {
			query += `
			WHERE TABLESPACE_NAME IN (`

			for i, tablespace := range tablespaceWhiteList {
				query += fmt.Sprintf(`'%s'`, tablespace)

				if i != len(tablespaceWhiteList)-1 {
					query += ","
				}
			}

			query += ")"
		}

		query += `
		GROUP BY TABLESPACE_NAME`
		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "tablespace.offlineCDBDatafiles",
			identifier:    "CDB_DATAFILES_OFFLINE",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricChan chan<- newrelicMetricSender) error {

		columnNames, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("failed to retrieve columns from rows")
		}

		for rows.Next() {
			// Make an array of columns and an array of pointers to each element of the array
			columns := make([]interface{}, len(columnNames))
			pointers := make([]interface{}, len(columnNames))
			for i := 0; i < len(columnNames); i++ {
				pointers[i] = &columns[i]
			}

			// Scan the row into the array of pointers
			err := rows.Scan(pointers...)
			if err != nil {
				return err
			}

			// Put the values of the row into a column with the column name as the key
			rowMap := make(map[string]interface{})
			for i, column := range columnNames {
				rowMap[column] = columns[i]
			}

			// Create each metric in the list of metrics we want to collect
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					newMetric := &newrelicMetric{
						name:       metric.name,
						metricType: metric.metricType,
						value:      rowMap[metric.identifier],
					}

					metadata := map[string]string{"tablespace": rowMap["TABLESPACE_NAME"].(string)}

					// Send the new metric down the channel
					metricChan <- newrelicMetricSender{metric: newMetric, metadata: metadata}
				}
			}
		}

		return nil
	},
}

var oracleLockedAccounts = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT
      INST_ID, LOCKED_ACCOUNTS
    FROM
    (	SELECT count(1) AS "LOCKED_ACCOUNTS"
      FROM
        cdb_users a,
        cdb_pdbs b
      WHERE a.con_id = b.con_id
        AND username IN ('SYS', 'SYSTEM', 'DBSNMP')
        AND a.account_status != 'OPEN'
    ) l,
    gv$instance i
		`

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "lockedAccounts",
			identifier:    "LOCKED_ACCOUNTS",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricChan chan<- newrelicMetricSender) error {

		columnNames, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("failed to retrieve columns from rows")
		}

		for rows.Next() {
			// Make an array of columns and an array of pointers to each element of the array
			columns := make([]interface{}, len(columnNames))
			pointers := make([]interface{}, len(columnNames))
			for i := 0; i < len(columnNames); i++ {
				pointers[i] = &columns[i]
			}

			// Scan the row into the array of pointers
			err := rows.Scan(pointers...)
			if err != nil {
				return err
			}

			// Put the values of the row into a column with the column name as the key
			rowMap := make(map[string]interface{})
			for i, column := range columnNames {
				rowMap[column] = columns[i]
			}

			// Create each metric in the list of metrics we want to collect
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					newMetric := &newrelicMetric{
						name:       metric.name,
						metricType: metric.metricType,
						value:      rowMap[metric.identifier],
					}

					metadata := map[string]string{"instanceID": getInstanceIDString(rowMap["INST_ID"])}

					// Send the new metric down the channel
					metricChan <- newrelicMetricSender{metric: newMetric, metadata: metadata}
				}
			}
		}

		return nil
	},
}

var oraclePDBNonWrite = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT TABLESPACE_NAME, sum(CASE WHEN ONLINE_STATUS IN ('ONLINE','SYSTEM','RECOVER') THEN 0 ELSE 1 END) AS "PDB_NON_WRITE_MODE"
    FROM cdb_data_files a, cdb_pdbs b
    WHERE a.con_id = b.con_id
    `
		if len(tablespaceWhiteList) > 0 {
			query += `
			AND TABLESPACE_NAME IN (`

			for i, tablespace := range tablespaceWhiteList {
				query += fmt.Sprintf(`'%s'`, tablespace)

				if i != len(tablespaceWhiteList)-1 {
					query += ","
				}
			}

			query += ")"
		}

		query += `
		GROUP BY TABLESPACE_NAME`
		return query

	},

	metrics: []*oracleMetric{
		{
			name:          "tablespace.pdbDatafilesNonWrite",
			identifier:    "PDB_NON_WRITE_MODE",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricChan chan<- newrelicMetricSender) error {

		columnNames, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("failed to retrieve columns from rows")
		}

		for rows.Next() {
			// Make an array of columns and an array of pointers to each element of the array
			columns := make([]interface{}, len(columnNames))
			pointers := make([]interface{}, len(columnNames))
			for i := 0; i < len(columnNames); i++ {
				pointers[i] = &columns[i]
			}

			// Scan the row into the array of pointers
			err := rows.Scan(pointers...)
			if err != nil {
				return err
			}

			// Put the values of the row into a column with the column name as the key
			rowMap := make(map[string]interface{})
			for i, column := range columnNames {
				rowMap[column] = columns[i]
			}

			// Create each metric in the list of metrics we want to collect
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					newMetric := &newrelicMetric{
						name:       metric.name,
						metricType: metric.metricType,
						value:      rowMap[metric.identifier],
					}

					metadata := map[string]string{"tablespace": rowMap["TABLESPACE_NAME"].(string)}

					// Send the new metric down the channel
					metricChan <- newrelicMetricSender{metric: newMetric, metadata: metadata}
				}
			}
		}

		return nil
	},
}

var oracleTablespaceMetrics = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
			SELECT a.TABLESPACE_NAME,
				a.USED_PERCENT,
				a.USED_SPACE AS "USED",
				a.TABLESPACE_SIZE AS "SIZE",
				b.TABLESPACE_OFFLINE AS "OFFLINE"
			FROM DBA_TABLESPACE_USAGE_METRICS a
			JOIN (
				SELECT
					TABLESPACE_NAME,
					MAX( CASE WHEN status = 'OFFLINE' THEN 1 ELSE 0 END) AS "TABLESPACE_OFFLINE"
				FROM DBA_TABLESPACES
				GROUP BY TABLESPACE_NAME
			) b
			ON a.TABLESPACE_NAME = b.TABLESPACE_NAME`

		if len(tablespaceWhiteList) > 0 {
			query += `
			WHERE a.TABLESPACE_NAME IN (`

			for i, tablespace := range tablespaceWhiteList {
				query += fmt.Sprintf(`'%s'`, tablespace)

				if i != len(tablespaceWhiteList)-1 {
					query += ","
				}
			}

			query += ")"
		}

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "tablespace.spaceConsumedInBytes",
			identifier:    "USED",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "tablespace.spaceReservedInBytes",
			identifier:    "SIZE",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "tablespace.spaceUsedPercentage",
			identifier:    "USED_PERCENT",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "tablespace.isOffline",
			identifier:    "OFFLINE",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},

	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricChan chan<- newrelicMetricSender) error {

		columnNames, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("failed to retrieve columns from rows")
		}

		for rows.Next() {
			// Make an array of columns and an array of pointers to each element of the array
			columns := make([]interface{}, len(columnNames))
			pointers := make([]interface{}, len(columnNames))
			for i := 0; i < len(columnNames); i++ {
				pointers[i] = &columns[i]
			}

			// Scan the row into the array of pointers
			err := rows.Scan(pointers...)
			if err != nil {
				return err
			}

			// Put the values of the row into a column with the column name as the key
			rowMap := make(map[string]interface{})
			for i, column := range columnNames {
				rowMap[column] = columns[i]
			}

			// Create each metric in the list of metrics we want to collect
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					newMetric := &newrelicMetric{
						name:       metric.name,
						metricType: metric.metricType,
						value:      rowMap[metric.identifier],
					}

					metadata := map[string]string{"tablespace": rowMap["TABLESPACE_NAME"].(string)}

					// Send the new metric down the channel
					metricChan <- newrelicMetricSender{metric: newMetric, metadata: metadata}
				}
			}
		}

		return nil
	},
}

var globalNameInstanceMetric = oracleMetricGroup{
	sqlQuery: func() string {
		query := `
    SELECT
      t1.INST_ID,
      t2.GLOBAL_NAME
		FROM
      (SELECT INST_ID FROM gv$instance) t1,
      (SELECT GLOBAL_NAME FROM global_name) t2
    `

		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "globalName",
			identifier:    "GLOBAL_NAME",
			metricType:    metric.ATTRIBUTE,
			defaultMetric: true,
		},
	},

	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricChan chan<- newrelicMetricSender) error {

		type pgaRow struct {
			instID int
			value  string
		}
		for rows.Next() {

			// Scan the row into a struct
			var tempPgaRow pgaRow
			err := rows.Scan(&tempPgaRow.instID, &tempPgaRow.value)
			if err != nil {
				return err
			}

			// Match the metric to one of the metrics we want to collect
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					newMetric := &newrelicMetric{
						name:       metric.name,
						value:      tempPgaRow.value,
						metricType: metric.metricType,
					}

					metadata := map[string]string{"instanceID": strconv.Itoa(tempPgaRow.instID)}

					// Send the new metric down the channel
					metricChan <- newrelicMetricSender{metric: newMetric, metadata: metadata}
					break

				}
			}
		}

		return nil
	},
}

var globalNameTablespaceMetric = oracleMetricGroup{
	sqlQuery: func() string {
		query := `SELECT
		t1.TABLESPACE_NAME,
		t2.GLOBAL_NAME
		FROM (SELECT TABLESPACE_NAME FROM DBA_TABLESPACES) t1,
		(SELECT GLOBAL_NAME FROM global_name) t2`
		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "globalName",
			identifier:    "GLOBAL_NAME",
			metricType:    metric.ATTRIBUTE,
			defaultMetric: true,
		},
	},

	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricChan chan<- newrelicMetricSender) error {

		type pgaRow struct {
			tableName string
			value     string
		}
		for rows.Next() {

			// Scan the row into a struct
			var tempPgaRow pgaRow
			err := rows.Scan(&tempPgaRow.tableName, &tempPgaRow.value)
			if err != nil {
				return err
			}

			// Match the metric to one of the metrics we want to collect
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					newMetric := &newrelicMetric{
						name:       metric.name,
						value:      tempPgaRow.value,
						metricType: metric.metricType,
					}

					metadata := map[string]string{"tablespace": tempPgaRow.tableName}

					// Send the new metric down the channel
					metricChan <- newrelicMetricSender{metric: newMetric, metadata: metadata}
					break

				}
			}
		}

		return nil
	},
}

var dbIDInstanceMetric = oracleMetricGroup{
	sqlQuery: func() string {
		query := `SELECT
		t1.INST_ID,
		t2.DBID
		FROM (SELECT INST_ID FROM gv$instance) t1,
		(SELECT DBID FROM v$database) t2`
		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "dbID",
			identifier:    "DBID",
			metricType:    metric.ATTRIBUTE,
			defaultMetric: true,
		},
	},

	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricChan chan<- newrelicMetricSender) error {

		type pgaRow struct {
			instID int
			value  string
		}
		for rows.Next() {

			// Scan the row into a struct
			var tempPgaRow pgaRow
			err := rows.Scan(&tempPgaRow.instID, &tempPgaRow.value)
			if err != nil {
				return err
			}

			// Match the metric to one of the metrics we want to collect
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					newMetric := &newrelicMetric{
						name:       metric.name,
						value:      tempPgaRow.value,
						metricType: metric.metricType,
					}

					metadata := map[string]string{"instanceID": strconv.Itoa(tempPgaRow.instID)}

					// Send the new metric down the channel
					metricChan <- newrelicMetricSender{metric: newMetric, metadata: metadata}
					break

				}
			}
		}

		return nil
	},
}

var dbIDTablespaceMetric = oracleMetricGroup{
	sqlQuery: func() string {
		query := `SELECT
		t1.TABLESPACE_NAME,
		t2.DBID
		FROM (SELECT TABLESPACE_NAME FROM DBA_TABLESPACES) t1,
		(SELECT DBID FROM v$database) t2`
		return query
	},

	metrics: []*oracleMetric{
		{
			name:          "dbID",
			identifier:    "DBID",
			metricType:    metric.ATTRIBUTE,
			defaultMetric: true,
		},
	},

	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricChan chan<- newrelicMetricSender) error {

		type pgaRow struct {
			tableName string
			value     string
		}
		for rows.Next() {

			// Scan the row into a struct
			var tempPgaRow pgaRow
			err := rows.Scan(&tempPgaRow.tableName, &tempPgaRow.value)
			if err != nil {
				return err
			}

			// Match the metric to one of the metrics we want to collect
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					newMetric := &newrelicMetric{
						name:       metric.name,
						value:      tempPgaRow.value,
						metricType: metric.metricType,
					}

					metadata := map[string]string{"tablespace": tempPgaRow.tableName}

					// Send the new metric down the channel
					metricChan <- newrelicMetricSender{metric: newMetric, metadata: metadata}
					break

				}
			}
		}

		return nil
	},
}

var oracleReadWriteMetrics = oracleMetricGroup{
	sqlQuery: func() string {
		return `
		SELECT
			INST_ID,
			SUM(PHYRDS) AS "PhysicalReads",
			SUM(PHYWRTS) AS "PhysicalWrites",
			SUM(PHYBLKRD) AS "PhysicalBlockReads",
			SUM(PHYBLKWRT) AS "PhysicalBlockWrites",
			SUM(READTIM) * 10 AS "ReadTime",
			SUM(WRITETIM) * 10 AS "WriteTime"
		FROM gv$filestat
		GROUP BY INST_ID`
	},

	metrics: []*oracleMetric{
		{
			name:          "disk.reads",
			identifier:    "PhysicalReads",
			metricType:    metric.RATE,
			defaultMetric: true,
		},
		{
			name:          "disk.writes",
			identifier:    "PhysicalWrites",
			metricType:    metric.RATE,
			defaultMetric: true,
		},
		{
			name:          "disk.blocksRead",
			identifier:    "PhysicalBlockReads",
			metricType:    metric.RATE,
			defaultMetric: true,
		},
		{
			name:          "disk.blocksWritten",
			identifier:    "PhysicalBlockWrites",
			metricType:    metric.RATE,
			defaultMetric: true,
		},
		{
			name:          "disk.readTimeInMilliseconds",
			identifier:    "ReadTime",
			metricType:    metric.RATE,
			defaultMetric: true,
		},
		{
			name:          "disk.writeTimeInMilliseconds",
			identifier:    "WriteTime",
			metricType:    metric.RATE,
			defaultMetric: true,
		},
	},

	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricChan chan<- newrelicMetricSender) error {

		columnNames, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("failed to get column names from rows")
		}

		for rows.Next() {
			// Create an array of columns and an array of pointers to the elements of the columns
			columns := make([]interface{}, len(columnNames))
			pointers := make([]interface{}, len(columnNames))
			for i := 0; i < len(columnNames); i++ {
				pointers[i] = &columns[i]
			}

			// Scan the row into the array of columns
			err := rows.Scan(pointers...)
			if err != nil {
				return fmt.Errorf("failed to parse row: %s", err)
			}

			// Put the values into a map indexed by column name
			rowMap := make(map[string]interface{})
			for i, column := range columnNames {
				rowMap[column] = columns[i]
			}

			// Create each new metric
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					newMetric := &newrelicMetric{
						name:       metric.name,
						metricType: metric.metricType,
						value:      rowMap[metric.identifier],
					}

					idString := getInstanceIDString(rowMap["INST_ID"])

					metadata := map[string]string{"instanceID": idString}

					// Send the metric down the channel
					metricChan <- newrelicMetricSender{metric: newMetric, metadata: metadata}
				}
			}
		}

		return nil
	},
}

var oraclePgaMetrics = oracleMetricGroup{
	sqlQuery: func() string {
		return `SELECT INST_ID, NAME, VALUE FROM gv$pgastat`
	},
	metrics: []*oracleMetric{
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
		{
			name:          "memory.pgaMaxSizeInBytes",
			metricType:    metric.GAUGE,
			defaultMetric: true,
			identifier:    "global memory bound",
		},
	},
	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricChan chan<- newrelicMetricSender) error {

		type pgaRow struct {
			instID int
			name   string
			value  float64
		}
		for rows.Next() {

			// Scan the row into a struct
			var tempPgaRow pgaRow
			err := rows.Scan(&tempPgaRow.instID, &tempPgaRow.name, &tempPgaRow.value)
			if err != nil {
				return err
			}

			// Match the metric to one of the metrics we want to collect
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					if tempPgaRow.name == metric.identifier {
						newMetric := &newrelicMetric{
							name:       metric.name,
							value:      tempPgaRow.value,
							metricType: metric.metricType,
						}

						metadata := map[string]string{"instanceID": strconv.Itoa(tempPgaRow.instID)}

						// Send the new metric down the channel
						metricChan <- newrelicMetricSender{metric: newMetric, metadata: metadata}
						break

					}
				}
			}
		}

		return nil
	},
}

var oracleSysMetrics = oracleMetricGroup{
	sqlQuery: func() string {
		return `
		SELECT
			INST_ID,
			METRIC_NAME,
			VALUE
		FROM gv$sysmetric`
	},

	metrics: []*oracleMetric{
		{
			name:          "memory.bufferCacheHitRatio",
			identifier:    "Buffer Cache Hit Ratio",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "memory.sortsRatio",
			identifier:    "Memory Sorts Ratio",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "memory.redoAllocationHitRatio",
			identifier:    "Redo Allocation Hit Ratio",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "query.transactionsPerSecond",
			identifier:    "User Transaction Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "query.physicalReadsPerTransaction",
			identifier:    "Physical Reads Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "query.physicalWritesPerTransaction",
			identifier:    "Physical Writes Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "disk.physicalReadsPerSecond",
			identifier:    "Physical Reads Direct Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "query.physicalReadsPerTransaction",
			identifier:    "Physical Reads Direct Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "disk.physicalWritesPerSecond",
			identifier:    "Physical Writes Direct Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "query.physicalWritesPerTransaction",
			identifier:    "Physical Writes Direct Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "disk.physicalLobsReadsPerSecond",
			identifier:    "Physical Reads Direct Lobs Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "query.physicalLobsReadsPerTransaction",
			identifier:    "Physical Reads Direct Lobs Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "disk.physicalLobsWritesPerSecond",
			identifier:    "Physical Writes Direct Lobs Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "query.physicalLobsWritesPerTransaction",
			identifier:    "Physical Writes Direct Lobs Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "memory.redoGeneratedBytesPerSecond",
			identifier:    "Redo Generated Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "memory.redoGeneratedBytesPerTransaction",
			identifier:    "Redo Generated Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.logonsPerTransaction",
			identifier:    "Logons Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.openCursorsPerSecond",
			identifier:    "Open Cursors Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.openCursorsPerTransaction",
			identifier:    "Open Cursors Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.userCommitsPerSecond",
			identifier:    "User Commits Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.userCommitsPercentage",
			identifier:    "User Commits Percentage",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.userRollbacksPerSecond",
			identifier:    "User Rollbacks Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.userRollbacksPercentage",
			identifier:    "User Rollbacks Percentage",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.userCallsPerSecond",
			identifier:    "User Calls Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.userCallsPerTransaction",
			identifier:    "User Calls Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.recursiveCallsPerSecond",
			identifier:    "Recursive Calls Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.recursiveCallsPerTransaction",
			identifier:    "Recursive Calls Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.logicalReadsPerSecond",
			identifier:    "Logical Reads Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.logicalReadsPerTransaction",
			identifier:    "Logical Reads Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.dbwrCheckpointsPerSecond",
			identifier:    "DBWR Checkpoints Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.backgroundCheckpointsPerSecond",
			identifier:    "Background Checkpoints Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.redoWritesPerSecond",
			identifier:    "Redo Writes Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.redoWritesPerTransaction",
			identifier:    "Redo Writes Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.longTableScansPerSecond",
			identifier:    "Long Table Scans Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.longTableScansPerTransaction",
			identifier:    "Long Table Scans Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.totalTableScansPerSecond",
			identifier:    "Total Table Scans Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "db.totalTableScansPerTransaction",
			identifier:    "Total Table Scans Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.fullIndexScansPerSecond",
			identifier:    "Full Index Scans Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.fullIndexScansPerTransaction",
			identifier:    "Full Index Scans Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.totalIndexScansPerSecond",
			identifier:    "Total Index Scans Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "db.totalIndexScansPerTransaction",
			identifier:    "Total Index Scans Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.totalParseCountPerSecond",
			identifier:    "Total Parse Count Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.totalParseCountPerTransaction",
			identifier:    "Total Parse Count Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.hardParseCountPerSecond",
			identifier:    "Hard Parse Count Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.hardParseCountPerTransaction",
			identifier:    "Hard Parse Count Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.parseFailureCountPerSecond",
			identifier:    "Parse Failure Count Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.parseFailureCountPerTransaction",
			identifier:    "Parse Failure Count Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.cursorCacheHitsPerAttempts",
			identifier:    "Cursor Cache Hit Ratio",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "disk.sortPerSecond",
			identifier:    "Disk Sort Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "disk.sortPerTransaction",
			identifier:    "Disk Sort Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.rowsPerSort",
			identifier:    "Rows Per Sort",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.softParseRatio",
			identifier:    "Soft Parse Ratio",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.userCallsRatio",
			identifier:    "User Calls Ratio",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.hostCpuUtilization",
			identifier:    "Host CPU Utilization (%)",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "network.trafficBytePerSecond",
			identifier:    "Network Traffic Volume Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "db.enqueueTimeoutsPerSecond",
			identifier:    "Enqueue Timeouts Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.enqueueTimeoutsPerTransaction",
			identifier:    "Enqueue Timeouts Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.enqueueWaitsPerSecond",
			identifier:    "Enqueue Waits Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.enqueueWaitsPerTransaction",
			identifier:    "Enqueue Waits Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.enqueueDeadlocksPerSecond",
			identifier:    "Enqueue Deadlocks Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.enqueueDeadlocksPerTransaction",
			identifier:    "Enqueue Deadlocks Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.enqueueRequestsPerSecond",
			identifier:    "Enqueue Requests Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.enqueueRequestsPerTransaction",
			identifier:    "Enqueue Requests Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.blockGetsPerSecond",
			identifier:    "DB Block Gets Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.blockGetsPerTransaction",
			identifier:    "DB Block Gets Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.consistentReadGetsPerSecond",
			identifier:    "Consistent Read Gets Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.blockChangesPerSecond",
			identifier:    "DB Block Changes Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.consistentReadGetsPerTransaction",
			identifier:    "Consistent Read Gets Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.blockChangesPerTransaction",
			identifier:    "DB Block Changes Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.consistentReadChangesPerSecond",
			identifier:    "Consistent Read Changes Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.consistentReadChangesPerTransaction",
			identifier:    "Consistent Read Changes Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.cpuUsagePerSecond",
			identifier:    "CPU Usage Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "db.cpuUsagePerTransaction",
			identifier:    "CPU Usage Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.crBlocksCreatedPerSecond",
			identifier:    "CR Blocks Created Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.crBlocksCreatedPerTransaction",
			identifier:    "CR Blocks Created Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.crUndoRecordsAppliedPerSecond",
			identifier:    "CR Undo Records Applied Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.crUndoRecordsAppliedPerTransaction",
			identifier:    "CR Undo Records Applied Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.userRollbackUndoRecordsAppliedPerSecond",
			identifier:    "User Rollback UndoRec Applied Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.userRollbackUndoRecordsAppliedPerTransaction",
			identifier:    "User Rollback Undo Records Applied Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.leafNodeSplitsPerSecond",
			identifier:    "Leaf Node Splits Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.leafNodeSplitsPerTransaction",
			identifier:    "Leaf Node Splits Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.branchNodeSplitsPerSecond",
			identifier:    "Branch Node Splits Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.branchNodeSplitsPerTransaction",
			identifier:    "Branch Node Splits Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "disk.physicalReadIoRequestsPerSecond",
			identifier:    "Physical Read Total IO Requests Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "disk.physicalReadBytesPerSecond",
			identifier:    "Physical Read Total Bytes Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "db.GcCrBlockRecievedPerSecond",
			identifier:    "GC CR Block Received Per Second",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.GcCrBlockRecievedPerTransaction",
			identifier:    "GC CR Block Received Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.GcCurrentBlockReceivedPerSecond",
			identifier:    "GC Current Block Received Per Second",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.GcCurrentBlockReceivedPerTransaction",
			identifier:    "GC Current Block Received Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.globalCacheAverageCrGetTime",
			identifier:    "Global Cache Average CR Get Time",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.globalCacheAverageCurrentGetTime",
			identifier:    "Global Cache Average Current Get Time",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "disk.physicalWriteTotalIoRequestsPerSecond",
			identifier:    "Physical Write Total IO Requests Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "memory.globalCacheBlocksCorrupted",
			identifier:    "Global Cache Blocks Corrupted",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "memory.globalCacheBlocksLost",
			identifier:    "Global Cache Blocks Lost",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.currentLogons",
			identifier:    "Current Logons Count",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.currentOpenCursors",
			identifier:    "Current Open Cursors Count",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.userLimitPercentage",
			identifier:    "User Limit %",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.sqlServiceResponseTime",
			identifier:    "SQL Service Response Time",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "db.waitTimeRatio",
			identifier:    "Database Wait Time Ratio",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.cpuTimeRatio",
			identifier:    "Database CPU Time Ratio",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.responseTimePerTransaction",
			identifier:    "Response Time Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.rowCacheHitRatio",
			identifier:    "Row Cache Hit Ratio",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.rowCacheMissRatio",
			identifier:    "Row Cache Miss Ratio",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.libraryCacheHitRatio",
			identifier:    "Library Cache Hit Ratio",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.libraryCacheMissRatio",
			identifier:    "Library Cache Miss Ratio",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.sharedPoolFreePercentage",
			identifier:    "Shared Pool Free %",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.pgaCacheHitPercentage",
			identifier:    "PGA Cache Hit %",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.processLimitPercentage",
			identifier:    "Process Limit %",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.sessionLimitPercentage",
			identifier:    "Session Limit %",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.executionsPerTransaction",
			identifier:    "Executions Per Txn",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.executionsPerSecond",
			identifier:    "Executions Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "db.TransactionsPerLogon",
			identifier:    "Txns Per Logon",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.databaseCpuTimePerSecond",
			identifier:    "Database Time Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "disk.physicalWriteBytesPerSecond",
			identifier:    "Physical Write Total Bytes Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "disk.physicalWriteIoRequestsPerSecond",
			identifier:    "Physical Write IO Requests Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.blockChangesPerUserCall",
			identifier:    "DB Block Changes Per User Call",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.blockGetsPerUserCall",
			identifier:    "DB Block Gets Per User Call",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.executionsPerUserCall",
			identifier:    "Executions Per User Call",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "disk.logicalReadsPerUserCall",
			identifier:    "Logical Reads Per User Call",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.sortsPerUserCall",
			identifier:    "Total Sorts Per User Call",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.tableScansPerUserCall",
			identifier:    "Total Table Scans Per User Call",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.osLoad",
			identifier:    "Current OS Load",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.streamsPoolUsagePercentage",
			identifier:    "Streams Pool Usage Percentage",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "network.ioMegabytesPerSecond",
			identifier:    "I/O Megabytes per Second",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "network.ioRequestsPerSecond",
			identifier:    "I/O Requests per Second",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
		{
			name:          "db.averageActiveSessions",
			identifier:    "Average Active Sessions",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.activeSerialSessions",
			identifier:    "Active Serial Sessions",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.activeParallelSessions",
			identifier:    "Active Parallel Sessions",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.backgroundCpuUsagePerSecond",
			identifier:    "Background CPU Usage Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.backgroundTimePerSecond",
			identifier:    "Background Time Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.hostCpuUsagePerSecond",
			identifier:    "Host CPU Usage Per Sec",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "disk.tempSpaceUsedInBytes",
			identifier:    "Temp Space Used",
			metricType:    metric.GAUGE,
			defaultMetric: false,
		},
		{
			name:          "db.sessionCount",
			identifier:    "Session Count",
			metricType:    metric.GAUGE,
			defaultMetric: true,
		},
	},
	metricsGenerator: func(rows *sql.Rows, metrics []*oracleMetric, metricsChan chan<- newrelicMetricSender) error {

		var sysScanner struct {
			instID     int
			metricName string
			value      float64
		}

		for rows.Next() {

			// Scan the row into a struct
			err := rows.Scan(&sysScanner.instID, &sysScanner.metricName, &sysScanner.value)
			if err != nil {
				return err
			}

			// Match the metric to one of the metrics we want to collect
			for _, metric := range metrics {
				if metric.defaultMetric || args.ExtendedMetrics {
					if sysScanner.metricName == metric.identifier {
						newMetric := &newrelicMetric{
							name:       metric.name,
							value:      sysScanner.value,
							metricType: metric.metricType,
						}

						metadata := map[string]string{"instanceID": strconv.Itoa(sysScanner.instID)}

						// Send the metric down the channel
						metricsChan <- newrelicMetricSender{metadata: metadata, metric: newMetric}
						break
					}
				}
			}
		}

		return nil
	},
}
