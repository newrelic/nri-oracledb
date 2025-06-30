<a href="https://opensource.newrelic.com/oss-category/#community-plus"><picture><source media="(prefers-color-scheme: dark)" srcset="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/dark/Community_Plus.png"><source media="(prefers-color-scheme: light)" srcset="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Plus.png"><img alt="New Relic Open Source community plus project banner." src="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Plus.png"></picture></a> 

# New Relic integration for Oracle Database

The New Relic integration for Oracle Database monitors key performance metrics for Oracle Database.

## Requirements

* A working installation of the Oracle Instant Client. Installation instructions [here](http://www.oracle.com/technetwork/database/database-technologies/instant-client/downloads/index.html).
* A user with the necessary permissions to collect all the metrics and inventory can be configured as follows:

* For standalone databases, use the following commands:

```sql
ALTER SESSION set "_Oracle_SCRIPT"=true;
CREATE USER <username> IDENTIFIED BY "<user_password>";
GRANT CONNECT TO <username>;
```

* For multi-tenant databases,  log in to the root database as an administrator use the following commands:

```sql
CREATE USER <username> IDENTIFIED BY "<user_password>";
ALTER USER <username> SET CONTAINER_DATA=ALL CONTAINER=CURRENT;
GRANT CONNECT TO <username>;
```
* Grant `SELECT` privileges to the user on the following global views

```sql
GRANT SELECT ON cdb_data_files TO <username>;
GRANT SELECT ON cdb_pdbs TO <username>;
GRANT SELECT ON cdb_users TO <username>;
GRANT SELECT ON dba_users to <username>;
GRANT SELECT ON gv_$sysmetric TO <username>;
GRANT SELECT ON gv_$pgastat TO <username>;
GRANT SELECT ON gv_$instance TO <username>;
GRANT SELECT ON gv_$filestat TO <username>;
GRANT SELECT ON gv_$parameter TO <username>;
GRANT SELECT ON sys.dba_data_files TO <username>;
GRANT SELECT ON DBA_TABLESPACES TO <username>;
GRANT SELECT ON DBA_TABLESPACE_USAGE_METRICS TO <username>;
GRANT SELECT ON gv_$session TO <username>;
GRANT SELECT ON gv_$sesstat TO <username>;
GRANT SELECT ON gv_$statname TO <username>;
GRANT SELECT ON gv_$rowcache TO <username>;
GRANT SELECT ON gv_$sga TO <username>;
GRANT SELECT ON gv_$sysstat TO <username>;
GRANT SELECT ON v_$database TO <username>;
GRANT SELECT ON gv_$librarycache TO <username>;
GRANT SELECT ON gv_$sqlarea TO <username>;
GRANT SELECT ON gv_$system_event TO <username>;
GRANT SELECT ON dba_tablespaces TO <username>;
GRANT SELECT ON gv_$session_wait TO <username>;
GRANT SELECT ON gv_$rollstat TO <username>;
GRANT SELECT ON v_$instance TO <username>;
```

* For Oracle Container Databases greater than version 12.1 user must be given access to global view for PDB containers

```sql
GRANT SELECT ON gv$con_sysmetric TO <username>;
```

## Installation and usage

For installation and usage instructions, see our [documentation web site](https://docs.newrelic.com/docs/integrations/host-integrations/host-integrations-list/oracledb-monitoring-integration).

## Compatibility

* Supported OS: Linux x86_64
* oracledb versions: 11.2+

## Building

Golang is required to build the integration. We recommend Golang 1.11 or higher.

After cloning this repository, go to the directory of the Oracle DB integration and build it:

```bash
$ make
```

The command above executes the tests for the Oracle DB integration and builds an executable file called `nri-oracledb` under the `bin` directory. 

To start the integration, run `nri-oracledb`:

```bash
$ ./bin/nri-oracledb
```

If you want to know more about usage of `./bin/nri-oracledb`, pass the `-help` parameter:

```bash
$ ./bin/nri-oracledb -help
```

External dependencies are managed through the [govendor tool](https://github.com/kardianos/govendor). Locking all external dependencies to a specific version (if possible) into the vendor directory is required.

## Testing

To run the tests execute:

```bash
$ make test
```

## Support

Should you need assistance with New Relic products, you are in good hands with several support diagnostic tools and support channels.

> New Relic offers NRDiag, [a client-side diagnostic utility](https://docs.newrelic.com/docs/using-new-relic/cross-product-functions/troubleshooting/new-relic-diagnostics) that automatically detects common problems with New Relic agents. If NRDiag detects a problem, it suggests troubleshooting steps. NRDiag can also automatically attach troubleshooting data to a New Relic Support ticket.

If the issue has been confirmed as a bug or is a Feature request, please file a Github issue.

**Support Channels**

* [New Relic Documentation](https://docs.newrelic.com): Comprehensive guidance for using our platform
* [New Relic Community](https://forum.newrelic.com): The best place to engage in troubleshooting questions
* [New Relic Developer](https://developer.newrelic.com/): Resources for building a custom observability applications
* [New Relic University](https://learn.newrelic.com/): A range of online training for New Relic users of every level
* [New Relic Technical Support](https://support.newrelic.com/) 24/7/365 ticketed support. Read more about our [Technical Support Offerings](https://docs.newrelic.com/docs/licenses/license-information/general-usage-licenses/support-plan).

## Privacy

At New Relic we take your privacy and the security of your information seriously, and are committed to protecting your information. We must emphasize the importance of not sharing personal data in public forums, and ask all users to scrub logs and diagnostic information for sensitive information, whether personal, proprietary, or otherwise.

We define “Personal Data” as any information relating to an identified or identifiable individual, including, for example, your name, phone number, post code or zip code, Device ID, IP address, and email address.

For more information, review [New Relic’s General Data Privacy Notice](https://newrelic.com/termsandconditions/privacy).

## Contribute

We encourage your contributions to improve this project! Keep in mind that when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant. You only have to sign the CLA one time per project.

If you have any questions, or to execute our corporate CLA (which is required if your contribution is on behalf of a company), drop us an email at opensource@newrelic.com.

**A note about vulnerabilities**

As noted in our [security policy](../../security/policy), New Relic is committed to the privacy and security of our customers and their data. We believe that providing coordinated disclosure by security researchers and engaging with the security community are important means to achieve our security goals.

If you believe you have found a security vulnerability in this project or any of New Relic's products or websites, we welcome and greatly appreciate you reporting it to New Relic through [HackerOne](https://hackerone.com/newrelic).

If you would like to contribute to this project, review [these guidelines](./CONTRIBUTING.md).

To all contributors, we thank you!  Without your contribution, this project would not be what it is today.

## License

nri-oracledb is licensed under the [MIT](/LICENSE) License.

