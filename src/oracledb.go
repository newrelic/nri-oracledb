package main

import (
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"
	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/integration"
	_ "gopkg.in/goracle.v2"
)

type argumentList struct {
	sdkArgs.DefaultArgumentList
	SID             string `default:"" help:"The Oracle service name"`
	Username        string `default:"" help:"The OracleDB connection user name"`
	Password        string `default:"" help:"The OracleDB connection password"`
	Hostname        string `default:"127.0.0.1" help:"The OracleDB connection host name"`
	Port            string `default:"1521" help:"The OracleDB connection port"`
	ExtendedMetrics bool   `default:"false" help:"Enable extended metrics"`
}

const (
	integrationName    = "com.newrelic.oracledb"
	integrationVersion = "0.1.0"
)

var (
	args argumentList
)

func main() {
	// Create Integration
	i, err := integration.New(integrationName, integrationVersion, integration.Args(&args))
	panicOnErr(err)

	if args.All() || args.Metrics {

		db, err := sqlx.Open("goracle", "oracle://orcladmin:password@ora12-ref-1.bluemedora.localnet/DB12C?connectionClass=&poolIncrement=1&poolMaxSessions=8&poolMinSessions=0&sysdba=0&sysoper=0&standaloneConnection=0")
		if err != nil {
			fmt.Println(err)
		}

		err = db.Ping()
		if err != nil {
			fmt.Println(err)
		}

		var wg sync.WaitGroup
		var workerWg sync.WaitGroup
		metricChan := make(chan newRelicMetricSender)
		workerWg.Add(1)
		go metricsWorker(metricChan, &workerWg, i)

		wg.Add(4)
		go oracleReadWriteMetrics.Collect(db, &wg, metricChan)
		go oraclePgaMetrics.Collect(db, &wg, metricChan)
		go oracleSysMetrics.Collect(db, &wg, metricChan)
		go oracleTablespaceMetrics.Collect(db, &wg, metricChan)

		go func() {
			wg.Wait()
			close(metricChan)
		}()

		workerWg.Wait()
	}

	panicOnErr(i.Publish())
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
