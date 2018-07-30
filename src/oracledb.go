package main

import (
	"database/sql"
	"fmt"

	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/data/event"
	"github.com/newrelic/infra-integrations-sdk/data/metric"
	"github.com/newrelic/infra-integrations-sdk/integration"
	_ "gopkg.in/goracle.v2"
)

type argumentList struct {
	sdkArgs.DefaultArgumentList
	SID      string `default:"" help:"The Oracle service name"`
	Username string `default:"" help:"The OracleDB connection user name"`
	Password string `default:"" help:"The OracleDB connection password"`
	Hostname string `default:"127.0.0.1" help:"The OracleDB connection host name"`
	Port     string `default:"1521" help:"The OracleDB connection port"`
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

	db, err := sql.Open("goracle", "SYS:password@10.77.17.158/orcl")
	if err != nil {
		fmt.Println("Failed to create db object")
	}

	// Create Entity, entities name must be unique
	e1, err := i.Entity("instance-1", "custom")
	panicOnErr(err)

	// Add Event
	if args.All() || args.Events {
		err = e1.AddEvent(event.New("restart", "status"))
		panicOnErr(err)
	}

	// Add Inventory item
	if args.All() || args.Inventory {
		err = e1.SetInventoryItem("instance", "version", "3.0.1")
		panicOnErr(err)
	}

	// Add Metric
	if args.All() || args.Metrics {
		m1 := e1.NewMetricSet("CustomSample")
		err = m1.SetMetric("some-data", 1000, metric.GAUGE)
		panicOnErr(err)
	}

	// Create another Entity
	e2, err := i.Entity("instance-2", "custom")
	panicOnErr(err)

	if args.All() || args.Inventory {
		err = e2.SetInventoryItem("instance", "version", "3.0.4")
		panicOnErr(err)
	}

	if args.All() || args.Metrics {
		m2 := e2.NewMetricSet("CustomSample")
		err = m2.SetMetric("some-data", 2000, metric.GAUGE)
		panicOnErr(err)
	}

	panicOnErr(i.Publish())
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
