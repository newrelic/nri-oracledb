# Change Log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## 3.1.1 (2021-10-27)
### Changed
- Added warning log when a query returns no results.


## 3.1.0 (2021-09-30)
### Changed
- Moved default config.sample to [V4](https://docs.newrelic.com/docs/create-integrations/infrastructure-integrations-sdk/specifications/host-integrations-newer-configuration-format/), added a dependency for infra-agent version 1.20.0

Please notice that old [V3](https://docs.newrelic.com/docs/create-integrations/infrastructure-integrations-sdk/specifications/host-integrations-standard-configuration-format/) configuration format is deprecated, but still supported.


## 3.0.0 (2021-05-12)
### Changed
* Integration SDK has been upgrade to 3.6.7, which fixes a bug that caused scrambled metrics when integration autodiscovery was used (#67)
  - Additionally, this PR also switches to go modules, go 1.16, and upgrades the driver used to connect to the database to its latest version

Since these changes involve a change of the oracle database driver, a major version bump has been issued.
While we have not detected any breakage during our tests, we encourage users to monitor the solution to ensure their use case has not been impacted. 

## 2.5.2 (2020-12-04)
### Changed
- Added configuration option to enable disabling the connection pool. There are cases where the connection pool does not properly re-use cnnections and leads to errors getting new connections thus failing some queries. Disabling the connection pool can lead to lower performance, but removes the issue of not being able to execute some queries.

## 2.5.1 (2020-07-30)
### Fixed
- Fixed case where sample name for custoem metrics was not being set properly

## 2.5.0 (2020-06-08)
### Added
- Custom query YAML configuration

## 2.4.1 (2020-05-19)
### Fixed
- Panic on empty custom query result

## 2.4.0 (2020-04-28)
### Changed
- Custom metrics query now does not require special column names, and returns each row as a separate sample with the column names as the metric names. This fixes issues with overwriting metric names as well as increases flexibility of collection so that the queries are less awkward to write. This is a breaking change since metric types are no longer defineable, (numerics are assumed to be gauges) and metric names are defined by the column name.

## 2.3.1 (2020-02-28)
### Fixed
- Tablespace usage percent calculation
- Connections not being closed cleanly

## 2.3.0 (2020-02-06)
### Added
- `custom_metrics_query` to capture metrics that the integration does not query for by default
### Fixed
- CDB and PDB metrics now respect tablespace whitelist

## 2.2.0 (2019-11-18)
### Changed
- Renamed the integration executable from nr-oracledb to nri-oracledb in order to be consistent with the package naming. **Important Note:** if you have any security module rules (eg. SELinux), alerts or automation that depends on the name of this binary, these will have to be updated.

## 2.1.7 - 2019-11-13
### Added
- Windows MSI resources

## 2.1.6 - 2019-11-08
### Fixed
- Run all DB queries concurrently to avoid deadlock

## 2.1.5 - 2019-11-07
### Fixed
- Avoid panicking or blocking when inventory connection fails.

## 2.1.4 - 2019-10-28
### Fixed
- Close rows objects when finished to allow recycling of connections

## 2.1.2 - 2019-08-27
### Added
- A number of requested metrics, including RAC

## 2.1.1 - 2019-07-18
### Added
- Add `connection_string` argument to enable more custom / manual configuration

## 2.1.0 - 2019-07-18
### Added
- Expose pool connection params as arguments

## 2.0.3 - 2019-07-17
### Changed
- Updated goracle dependency

## 2.0.2 - 2019-07-16
### Fixed
- Default max datafile size to 2G

## 2.0.1 - 2019-06-20
### Added
- lockedAccounts metric
- tablespace.offlinePDBDatafiles metric
- tablespace.offlineCDBDatafiles metric
- tablespace.pdbDatafilesNonWrite  metric

## 2.0.0 - 2019-04-26
### Changed
- Updated SDK
- Made entity keys more unique
- Prefixed namespaces

## 1.1.6 - 2019-04-17
### Fixed
- Timing out when waiting for a connection

## 1.1.5 - 2019-03-19
### Added
- Don't force exit if ORACLE_HOME is unset

## 1.1.4 - 2019-03-19
### Added
- Log error message if ORACLE_HOME is unset

## 1.1.3 - 2019-02-04
### Fixed
- Use correct protocol version

## 1.1.2 - 2019-01-09
### Fixed
- Divide by zero error in query

## 1.1.1 - 2018-12-11
### Fixed
- Properly specified inventory prefix

## 1.1.0 - 2018-11-20
### Added
- DBID and Global Name to all entities

## 1.0.0 - 2018-11-16
### Changed
- Updated to version 1.0.0

## 0.2.0 - 2018-11-12
### Added
- Configuration prefix for consistency

## 0.1.5 - 2018-10-23
### Changed
- Changed operating command from `inventory` or `metrics` to `all_data`

## 0.1.4 - 2018-10-16
### Fixed
- Removed a leftover misleading comment in the definition file

## 0.1.3 - 2018-10-05
### Fixed
- Instances using IDs rather than names

## 0.1.2 - 2018-10-01
### Added
- Tablespace whitelist configuration parameter
- Hard coded limit for the number of tablespaces that can be collected

## 0.1.1 - 2018-09-13
### Changed
- Renamed nr-oracledb-config.yml.template to oracledb-config.yml.sample
- Renamed nr-oracledb-definition.yml to oracledb-definition.yml

## 0.1.0 - 2018-08-30
### Added
- Initial version: Includes Metrics and Inventory data
