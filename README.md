# New Relic Infrastructure Integration for Oracle Database

The New Relic Infrastructure Integration for Oracle Database monitors key performance metrics for Oracle Database.

See our [documentation web site](https://docs.newrelic.com/docs/integrations/host-integrations/host-integrations-list/oracledb-monitoring-integration) for more details.

## Requirements

Have a working installation of the Oracle Instant Client. Installation instructions [here](http://www.oracle.com/technetwork/database/database-technologies/instant-client/downloads/index.html)

## Configuration

A user with the necessary permissions to collect all the metrics and inventory can be configured as follows
```sql
alter session set "_ORACLE_SCRIPT"=true;
CREATE USER <username> IDENTIFIED BY "<password>";
GRANT CONNECT TO <username>;
GRANT SELECT ON gv_$sysmetric TO <username>;
GRANT SELECT ON gv_$pgastat TO <username>;
GRANT SELECT ON gv_$instance TO <username>;
GRANT SELECT ON gv_$filestat TO <username>;
GRANT SELECT ON gv_$parameter TO <username>;
GRANT SELECT ON sys.dba_data_files TO <username>;
```

## Installation

- install the [New Relic Infrastructure Agent](https://docs.newrelic.com/docs/infrastructure/new-relic-infrastructure/installation/install-infrastructure-linux)
- download and exctract the archive file for the `Oracle Database` integration
- build the integration as described above
- copy `oracledb-definition.yml` to `/var/db/newrelic-infra/newrelic-integrations`
- copy the binary in `bin/` that matches your target OS/architecture into `/var/db/newrelic-infra/newrelic-integrations`
- add execute permissions for the binary file
- copy `oracledb-config.yml.sample` into `/etc/newrelic-infra/integrations.d`, rename it to `oracledb-config.yml`, and edit it to represent the environment you are monitoring
- install the [Oracle Instant Client](http://www.oracle.com/technetwork/database/database-technologies/instant-client/downloads/index.html)

## Usage

To configure the plugin, edit `oracledb-config.yml` to add the OracleDB connection information. If extended metrics are required, set `extended_metrics: true`. Once configuration is complete, restart the Infrastructure agent. 

You can view your data in Insights by creating your own custom NRQL queries. To do so, use **OracleDatabaseSample** and **OracleTablespaceSample** event types.

## Compatibility

* Supported OS: No limitations
* oracledb versions: 11.2+

## Integration Development usage

The OracleDB integration uses the `goracle` package to connect to an Oracle database. The package uses go bindings to the C library
ODPI-C, which complicates the process for cross compiling since both go code and C code need be compiled for the target OS and architecture. 
To help simplify that process, the `make` target `cross-compile-all` has been defined which uses `xgo` to compile the integration for Linux, 
Mac, and Windows for both `amd64` and `386` architectures. `xgo` requires a working docker installation on the compiling machine. Further
installation instructions for `xgo` can be found [here](https://github.com/karalabe/xgo). Once `xgo` is installed, simply run `make cross-compile-all` 
to compile the integration for all architectures. The compiled binaries can all be found in the `bin/` directory.

The integration can also be run locally. If run locally, you must have a working installation of the Oracle Instant Client.

* Go to the directory of the OracleDB integration and build it
```bash 
$ make
```

* The command above will execute tests for the OracleDB integration and build an executable file called `nri-oracledb` in the `bin/` directory.
```bash
$ ./bin/nri-oracledb
```

* If you want to know more about the usage of `./nri-oracledb`, check
```bash
$ ./bin/nri-oracledb --help
```

For managing external dependencies [govendor tool](https://github.com/kardianos/govendor) is used. It is required to lock all external dependencies to specific version (if possible) into vendor directory.
