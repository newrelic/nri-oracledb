# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

Unreleased section should follow [Release Toolkit](https://github.com/newrelic/release-toolkit#render-markdown-and-update-markdown)

## Unreleased

### bugfix 
- Update lockedAccounts query to fetch non-cdb/pdb accounts 

## v3.10.0 - 2025-06-06

### 🛡️ Security notices
- Updated Golang version to address some vulnerabilities

## v3.9.4 - 2025-05-22

### 🐞 Bug fixes
- fix lockedAccounts metric

## v3.9.3 - 2025-03-11

### ⛓️ Dependencies
- Updated golang patch version to v1.23.6

## v3.9.2 - 2025-01-21

### ⛓️ Dependencies
- Updated golang patch version to v1.23.5

## v3.9.1 - 2024-12-03

### ⛓️ Dependencies
- Updated golang patch version to v1.23.3

## v3.9.0 - 2024-10-08

### dependency
- Upgrade go to 1.23.2

### 🚀 Enhancements
- Upgrade integrations SDK so the interval is variable and allows intervals up to 5 minutes

## v3.8.3 - 2024-09-10

### ⛓️ Dependencies
- Updated golang version to v1.23.1

## v3.8.2 - 2024-07-09

### ⛓️ Dependencies
- Updated golang version to v1.22.5

## v3.8.1 - 2024-06-18

### 🐞 Bug fixes
- A wrong custom query could cause a panic. Now the error is managed properly.

## v3.8.0 - 2024-06-17

### 🛡️ Security notices
- Updated Golang version to address some vulnerabilities
- Updated Godror dependency

### 🐞 Bug fixes
- Fixed two queries. Now the integration correctly collects `sga.logBufferRedoAllocationRetries`, `sga.logBufferRedoEntries`, `sorts.memoryInBytes`, `sorts.diskInBytes`, `sga.fixedSizeInBytes`, and `sga.redoBuffersInBytes`

## v3.7.4 - 2024-04-30

### ⛓️ Dependencies
- Updated github.com/jmoiron/sqlx to v1.4.0 - [Changelog 🔗](https://github.com/jmoiron/sqlx/releases/tag/v1.4.0)

## v3.7.3 - 2024-02-20

### ⛓️ Dependencies
- Updated github.com/newrelic/infra-integrations-sdk to v3.8.2+incompatible

## v3.7.2 - 2024-01-16

### ⛓️ Dependencies
- Updated github.com/data-dog/go-sqlmock to v1.5.2 - [Changelog 🔗](https://github.com/data-dog/go-sqlmock/releases/tag/v1.5.2)

## v3.7.2 - 2024-01-09

### ⛓️ Dependencies
- Updated github.com/data-dog/go-sqlmock to v1.5.2 - [Changelog 🔗](https://github.com/data-dog/go-sqlmock/releases/tag/v1.5.2)

## v3.7.1 - 2023-12-19

### ⛓️ Dependencies
- Updated github.com/data-dog/go-sqlmock to v1.5.1 - [Changelog 🔗](https://github.com/data-dog/go-sqlmock/releases/tag/v1.5.1)

## v3.7.0 - 2023-11-06

### 🚀 Enhancements
- publish as well for RHEL 15.5 and bookworm

## v3.6.2 - 2023-08-18

### 🐞 Bug fixes
- Restore golang docker image to buster

## v3.6.1 - 2023-08-15

### 🐞 Bug fixes
- Restore golang version to 1.19 to ensure compatibility

## v3.6.0 - 2023-08-08

### 🚀 Enhancements
- bumped golang version pinning 1.20.6

## 3.5.0 (2023-06-06)
### Added
- Add support for PDB Sys metrics by setting up the new config field `SYS_METRICS_SOURCE`.
- Include additional extended metrics in Sys metrics group.
- Bump dependencies.

## 3.4.0 (2022-06-29)
### Changed
- Removed 200 tablespace limitation. Use `TABLESPACE` config parameter to limit the number of tablespaces monitored.
- Bumped dependencies

## 3.3.0 (2022-05-05)
### Added
- Add `SKIP_METRICS_GROUPS` config: Metrics collected are group together depending on the query used to obtain the data. These metric groups are here and can be skipped from collection by adding the name of the group to SKIP_METRICS_GROUPS in Json array format. By default no group is skipped so no breaking changes are added.

### Changed
- Bumped dependencies 
- Change pipeline to use Go 1.18

## 3.2.0 (2022-02-17)
### Fixed
- Metrics `tablespace.spaceConsumedInBytes` and `tablespace.spaceReservedInBytes` previously reported in block sizes are now reported in Bytes (#94)

## 3.1.1 (2022-01-10)
### Changed
- Added warning log when a query returns no results (#87)
- Fix missing event type on custom metric config (#86)
- Strip spaces from logged result query (#88)


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
- Fixed case where sample name for custom metrics was not being set properly

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
