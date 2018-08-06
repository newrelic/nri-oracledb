package main

import (
	"database/sql"
	"fmt"
	"sync"

	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	goracle "gopkg.in/goracle.v2"
)

type argumentList struct {
	sdkArgs.DefaultArgumentList
	ServiceName     string `default:"" help:"The Oracle service name"`
	Username        string `default:"" help:"The OracleDB connection user name"`
	Password        string `default:"" help:"The OracleDB connection password"`
	IsSysDBA        bool   `default:"false" help:"Is the user a SysDBA"`
	IsSysOper       bool   `default:"false" help:"Is the user a SysOper"`
	Hostname        string `default:"127.0.0.1" help:"The OracleDB connection host name"`
	Port            string `default:"1521" help:"The OracleDB connection port"`
	ExtendedMetrics bool   `default:"false" help:"Enable extended metrics"`
}

const (
	integrationName    = "com.newrelic.oracledb"
	integrationVersion = "0.1.0"
)

var (
	args   argumentList
	logger log.Logger
)

func main() {
	// Create Integration
	i, err := integration.New(integrationName, integrationVersion, integration.Args(&args))
	panicOnErr(err)

	logger = i.Logger()

	cp := goracle.ConnectionParams{
		Username:    args.Username,
		Password:    args.Password,
		SID:         fmt.Sprintf("%s:%s/%s", args.Hostname, args.Port, args.ServiceName),
		IsSysDBA:    args.IsSysDBA,
		IsSysOper:   args.IsSysOper,
		MaxSessions: 8,
	}

	db, err := sql.Open("goracle", cp.StringWithPassword())
	defer db.Close()
	panicOnErr(err)

	err = db.Ping()
	panicOnErr(err)

	var populaterWg sync.WaitGroup

	if args.All() {
		populaterWg.Add(2)
		go collectMetrics(db, &populaterWg, i)
		go collectInventory(db, &populaterWg, i)
	} else if args.Metrics {
		populaterWg.Add(1)
		go collectMetrics(db, &populaterWg, i)
	} else if args.Inventory {
		populaterWg.Add(1)
		go collectInventory(db, &populaterWg, i)
	}

	populaterWg.Wait()

	panicOnErr(i.Publish())
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
