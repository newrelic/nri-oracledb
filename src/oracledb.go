//go:generate goversioninfo
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/godror/godror"
	"github.com/godror/godror/dsn"
	"github.com/jmoiron/sqlx"
	sdkArgs "github.com/newrelic/infra-integrations-sdk/args"
	"github.com/newrelic/infra-integrations-sdk/integration"
	"github.com/newrelic/infra-integrations-sdk/log"
	"github.com/newrelic/nri-oracledb/src/database"
)

type argumentList struct {
	sdkArgs.DefaultArgumentList
	ServiceName           string `default:"" help:"The Oracle service name"`
	Username              string `default:"" help:"The OracleDB connection user name"`
	Password              string `default:"" help:"The OracleDB connection password"`
	IsSysDBA              bool   `default:"false" help:"Is the user a SysDBA"`
	IsSysOper             bool   `default:"false" help:"Is the user a SysOper"`
	Hostname              string `default:"127.0.0.1" help:"The OracleDB connection host name"`
	Tablespaces           string `default:"" help:"JSON Array of Tablespaces to collect. If empty will collect all tablespaces."`
	Port                  string `default:"1521" help:"The OracleDB connection port"`
	ExtendedMetrics       bool   `default:"false" help:"Enable extended metrics"`
	MaxOpenConnections    int    `default:"5" help:"Maximum number of connections opened by the integration"`
	ConnectionString      string `default:"" help:"An advanced connection string. Takes precedence over host, port, and service name"`
	CustomMetricsQuery    string `default:"" help:"A SQL query to collect custom metrics. Must have the columns metric_name, metric_type, and metric_value. Additional columns are added as attributes"`
	CustomMetricsConfig   string `default:"" help:"YAML configuration file with one or more custom SQL queries to collect"`
	DisableConnectionPool bool   `default:"false" help:"Disables connection pooling. It may make the integration run slower but may reduce issues with not being able to execute queries due to ORA-24459 (failure to get new connection)"`
	ShowVersion           bool   `default:"false" help:"Print build information and exit"`
}

const (
	integrationName = "com.newrelic.oracledb"
)

var (
	args                argumentList
	tablespaceWhiteList []string
	integrationVersion  = "0.0.0"
	gitCommit           = ""
	buildDate           = ""
)

func main() {
	// Create Integration
	i, err := integration.New(integrationName, integrationVersion, integration.Args(&args))
	exitOnErr(err)

	if args.ShowVersion {
		fmt.Printf(
			"New Relic %s integration Version: %s, Platform: %s, GoVersion: %s, GitCommit: %s, BuildDate: %s\n",
			strings.Title(strings.Replace(integrationName, "com.newrelic.", "", 1)),
			integrationVersion,
			fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
			runtime.Version(),
			gitCommit,
			buildDate)
		os.Exit(0)
	}

	// parse tablespace whitelist
	err = parseTablespaceWhitelist()
	exitOnErr(err)

	db, err := sqlx.Open("godror", getConnectionString())
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

	dbWrapper := database.NewDBWrapper(db)

	instanceLookUp, err := createInstanceIDLookup(dbWrapper)
	exitOnErr(err)

	if args.HasMetrics() {
		populaterWg.Add(1)
		mc := metricsCollector{
			integration:         i,
			db:                  dbWrapper,
			wg:                  &populaterWg,
			instanceLookUp:      instanceLookUp,
			customMetricsQuery:  args.CustomMetricsQuery,
			customMetricsConfig: args.CustomMetricsConfig,
		}
		go mc.collectMetrics()
	}

	if args.HasInventory() {
		populaterWg.Add(1)
		go collectInventory(dbWrapper, &populaterWg, i, instanceLookUp)
	}

	populaterWg.Wait()

	exitOnErr(i.Publish())
}

func getConnectionString() string {
	var connString string
	if args.ConnectionString == "" {
		connString = fmt.Sprintf("%s:%s/%s", args.Hostname, args.Port, args.ServiceName)
	} else {
		connString = strings.ReplaceAll(args.ConnectionString, " ", "")
	}

	return godror.ConnectionParams{
		StandaloneConnection: args.DisableConnectionPool,
		CommonParams: dsn.CommonParams{
			Username:      args.Username,
			Password:      dsn.NewPassword(args.Password),
			ConnectString: connString,
		},
		PoolParams: dsn.PoolParams{
			MinSessions:      0,
			MaxSessions:      args.MaxOpenConnections,
			SessionIncrement: 1,
		},
		ConnParams: dsn.ConnParams{
			IsSysDBA:  args.IsSysDBA,
			IsSysOper: args.IsSysOper,
		},
	}.StringWithPassword()
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

func createInstanceIDLookup(db database.DBWrapper) (map[string]string, error) {
	const instanceQuery = `SELECT
		INSTANCE_NAME, INST_ID
		FROM gv$instance`

	rows, err := db.Query(instanceQuery)
	if err != nil {
		log.Error("Failed running query: %s", formatQueryForLogging(instanceQuery))
		return nil, err
	}

	defer func() {
		checkAndLogEmptyQueryResult(instanceQuery, rows)
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

func checkAndLogEmptyQueryResult(executedQuery string, rows database.Rows) {
	if rows.ScannedRowsCount() == 0 {
		log.Warn("Query did not return any results: %s", formatQueryForLogging(executedQuery))
	}
}

func formatQueryForLogging(query string) string {
	return strings.Join(strings.Fields(query), " ")
}
