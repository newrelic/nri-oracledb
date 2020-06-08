//go:generate goversioninfo
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	goracle "gopkg.in/goracle.v2"
)

type argumentList struct {
	sdkArgs.DefaultArgumentList
	ServiceName         string `default:"" help:"The Oracle service name"`
	Username            string `default:"" help:"The OracleDB connection user name"`
	Password            string `default:"" help:"The OracleDB connection password"`
	IsSysDBA            bool   `default:"false" help:"Is the user a SysDBA"`
	IsSysOper           bool   `default:"false" help:"Is the user a SysOper"`
	Hostname            string `default:"127.0.0.1" help:"The OracleDB connection host name"`
	Tablespaces         string `default:"" help:"JSON Array of Tablespaces to collect. If empty will collect all tablespaces."`
	Port                string `default:"1521" help:"The OracleDB connection port"`
	ExtendedMetrics     bool   `default:"false" help:"Enable extended metrics"`
	MaxOpenConnections  int    `default:"5" help:"Maximum number of connections opened by the integration"`
	ConnectionString    string `default:"" help:"An advanced connection string. Takes precedence over host, port, and service name"`
	CustomMetricsQuery  string `default:"" help:"A SQL query to collect custom metrics. Must have the columns metric_name, metric_type, and metric_value. Additional columns are added as attributes"`
	CustomMetricsConfig string `default:"" help:"YAML configuration file with one or more custom SQL queries to collect"`
}

const (
	integrationName    = "com.newrelic.oracledb"
	integrationVersion = "2.4.1"
)

var (
	args                argumentList
	tablespaceWhiteList []string
)

func main() {
	// Create Integration
	i, err := integration.New(integrationName, integrationVersion, integration.Args(&args))
	exitOnErr(err)

	// parse tablespace whitelist
	err = parseTablespaceWhitelist()
	exitOnErr(err)

	db, err := sqlx.Open("goracle", getConnectionString())
	exitOnErr(err)
	db.SetMaxOpenConns(args.MaxOpenConnections)

	defer func() {
		if err := db.Close(); err != nil {
			log.Error("Failed to close database")
		}
	}()

	err = db.Ping()
	exitOnErr(err)

	var populaterWg sync.WaitGroup

	instanceLookUp, err := createInstanceIDLookup(db)
	exitOnErr(err)

	if args.HasMetrics() {
		populaterWg.Add(1)
		go collectMetrics(db, &populaterWg, i, instanceLookUp, args.CustomMetricsQuery, args.CustomMetricsConfig)
	}

	if args.HasInventory() {
		populaterWg.Add(1)
		go collectInventory(db, &populaterWg, i, instanceLookUp)
	}

	populaterWg.Wait()

	exitOnErr(i.Publish())
}

func getConnectionString() string {

	sid := ""
	if args.ConnectionString == "" {
		sid = fmt.Sprintf("%s:%s/%s", args.Hostname, args.Port, args.ServiceName)
	} else {
		sid = strings.Replace(args.ConnectionString, " ", "", -1)
	}

	cp := goracle.ConnectionParams{
		Username:      args.Username,
		Password:      args.Password,
		SID:           sid,
		IsSysDBA:      args.IsSysDBA,
		IsSysOper:     args.IsSysOper,
		MinSessions:   0,
		MaxSessions:   args.MaxOpenConnections,
		PoolIncrement: 1,
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
		tablespaceWhiteList = nil
		return nil
	}

	return json.Unmarshal([]byte(args.Tablespaces), &tablespaceWhiteList)
}

func createInstanceIDLookup(db *sqlx.DB) (map[string]string, error) {
	const instanceQuery = `SELECT
		INSTANCE_NAME, INST_ID
		FROM gv$instance`

	rows, err := db.Query(instanceQuery)
	if err != nil {
		log.Error("Failed running query: %s", instanceQuery)
		return nil, err
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			log.Error("Failed to close rows: %s", err)
		}
	}()

	var instance struct {
		Name string
		ID   int
	}

	lookup := make(map[string]string)

	for rows.Next() {
		err := rows.Scan(&instance.Name, &instance.ID)
		if err != nil {
			return nil, err
		}

		stringID := getInstanceIDString(instance.ID)
		lookup[stringID] = instance.Name
	}

	return lookup, nil
}
