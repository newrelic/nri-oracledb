package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
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
	Tablespaces     string `default:"" help:"JSON Array of Tablespaces to collect. If empty will collect all tablespaces."`
	Port            string `default:"1521" help:"The OracleDB connection port"`
	ExtendedMetrics bool   `default:"false" help:"Enable extended metrics"`
}

const (
	integrationName    = "com.newrelic.oracledb"
	integrationVersion = "0.1.2"
)

var (
	args                argumentList
	tableSpaceWhiteList []string
)

func main() {
	// Create Integration
	i, err := integration.New(integrationName, integrationVersion, integration.Args(&args))
	exitOnErr(err)

	// parse tablespace whitelist
	err = parseTablespaceWhitelist()
	exitOnErr(err)

	db, err := sql.Open("goracle", getConnectionString())
	exitOnErr(err)

	defer func() {
		if err := db.Close(); err != nil {
			log.Error("Failed to close database")
		}
	}()

	err = db.Ping()
	exitOnErr(err)

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

	exitOnErr(i.Publish())
}

func getConnectionString() string {

	cp := goracle.ConnectionParams{
		Username:    args.Username,
		Password:    args.Password,
		SID:         fmt.Sprintf("%s:%s/%s", args.Hostname, args.Port, args.ServiceName),
		IsSysDBA:    args.IsSysDBA,
		IsSysOper:   args.IsSysOper,
		MaxSessions: 8,
	}

	return cp.StringWithPassword()
}

func exitOnErr(err error) {
	if err != nil {
		log.Error("%s", err.Error())
		os.Exit(1)
	}
}

func parseTablespaceWhitelist() error {
	if args.Tablespaces == "" {
		tableSpaceWhiteList = nil
	}

	return json.Unmarshal([]byte(args.Tablespaces), &tableSpaceWhiteList)
}
