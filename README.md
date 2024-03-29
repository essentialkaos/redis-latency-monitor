<p align="center"><a href="#readme"><img src="https://gh.kaos.st/redis-latency-monitor.svg"/></a></p>

<p align="center">
  <a href="https://kaos.sh/w/redis-latency-monitor/ci"><img src="https://kaos.sh/w/redis-latency-monitor/ci.svg" alt="GitHub Actions CI Status" /></a>
  <a href="https://kaos.sh/r/redis-latency-monitor"><img src="https://kaos.sh/r/redis-latency-monitor.svg" alt="GoReportCard" /></a>
  <a href="https://kaos.sh/b/redis-latency-monitor"><img src="https://kaos.sh/b/40d24053-129b-4407-97bd-adecc66c8903.svg" alt="codebeat badge" /></a>
  <a href="https://kaos.sh/w/redis-latency-monitor/codeql"><img src="https://kaos.sh/w/redis-latency-monitor/codeql.svg" alt="GitHub Actions CodeQL Status" /></a>
  <a href="#license"><img src="https://gh.kaos.st/apache2.svg"></a>
</p>

<p align="center"><a href="#usage-demo">Usage demo</a> • <a href="#installation">Installation</a> • <a href="#usage">Usage</a> • <a href="#build-status">Build Status</a> • <a href="#license">License</a></p>

<br/>

Tiny Redis client for latency measurement. Utility show `PING` command latency or connection latency in milliseconds (_one thousandth of a second_).

### Usage demo

[![demo](https://gh.kaos.st/redis-latency-monitor-301.gif)](#usage-demo)

### Installation

#### From source

To build the `redis-latency-monitor` from scratch, make sure you have a working Go 1.17+ workspace (_[instructions](https://golang.org/doc/install)_), then:

```
go get github.com/essentialkaos/redis-latency-monitor
```

If you want to update `redis-latency-monitor` to latest stable release, do:

```
go get -u github.com/essentialkaos/redis-latency-monitor
```

#### From [ESSENTIAL KAOS Public Repository](https://yum.kaos.st)

```bash
sudo yum install -y https://yum.kaos.st/get/$(uname -r).rpm
sudo yum install redis-latency-monitor
```

#### Prebuilt binaries

You can download prebuilt binaries for Linux and macOS from [EK Apps Repository](https://apps.kaos.st/redis-latency-monitor/latest).

### Usage

```
Usage: redis-latency-monitor {options}

Utility show PING command latency or connection latency in milliseconds (one thousandth of a second).

Options

  --host, -h ip/host         Server hostname (127.0.0.1 by default)
  --port, -p port            Server port (6379 by default)
  --connect, -c              Measure connection latency instead of command latency
  --password, -a password    Password to use when connecting to the server
  --timeout, -t 1-300        Connection timeout in seconds (3 by default)
  --interval, -i 1-3600      Interval in seconds (60 by default)
  --timestamps, -T           Use unix timestamps in output
  --output, -o file          Path to output CSV file
  --error-log, -e file       Path to log with error messages
  --no-color, -nc            Disable colors in output
  --help                     Show this help message
  --version, -v              Show version

Examples

  redis-latency-monitor -h 192.168.0.123 -p 6821 -t 15
  Start monitoring instance on 192.168.0.123:6821 with 15 second timeout

  redis-latency-monitor -c -i 15 -o latency.csv
  Start connection latency monitoring with 15 second interval and save result to CSV file

```

### Build Status

| Branch | Status |
|--------|--------|
| `master` | [![CI](https://kaos.sh/w/redis-latency-monitor/ci.svg?branch=master)](https://kaos.sh/w/redis-latency-monitor/ci?query=branch:master) |
| `develop` | [![CI](https://kaos.sh/w/redis-latency-monitor/ci.svg?branch=master)](https://kaos.sh/w/redis-latency-monitor/ci?query=branch:develop) |

### License

[Apache License, Version 2.0](https://www.apache.org/licenses/LICENSE-2.0)

<p align="center"><a href="https://essentialkaos.com"><img src="https://gh.kaos.st/ekgh.svg"/></a></p>
