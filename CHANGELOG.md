# Change Log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

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
