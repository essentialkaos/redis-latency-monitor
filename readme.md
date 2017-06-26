## Redis Latency Monitor [![Build Status](https://travis-ci.org/essentialkaos/redis-latency-monitor.svg?branch=master)](https://travis-ci.org/essentialkaos/redis-latency-monitor) [![Go Report Card](https://goreportcard.com/badge/github.com/essentialkaos/redis-latency-monitor)](https://goreportcard.com/report/github.com/essentialkaos/redis-latency-monitor) [![License](https://gh.kaos.io/ekol.svg)](https://essentialkaos.com/ekol)

Tiny Redis client for latency measurement. Utility show `PING` command latency or connection latency in milliseconds (_one thousandth of a second_).

### Usage demo

[![demo](https://gh.kaos.io/redis-latency-monitor-210.gif)](#usage-demo)

### Installation

#### From source

Before the initial install allows git to use redirects for [pkg.re](https://github.com/essentialkaos/pkgre) service (reason why you should do this described [here](https://github.com/essentialkaos/pkgre#git-support)):

```
git config --global http.https://pkg.re.followRedirects true
```

To build the `redis-latency-monitor` from scratch, make sure you have a working Go 1.6+ workspace ([instructions](https://golang.org/doc/install)), then:

```
go get github.com/essentialkaos/redis-latency-monitor
```

If you want to update `redis-latency-monitor` to latest stable release, do:

```
go get -u github.com/essentialkaos/redis-latency-monitor
```

#### From ESSENTIAL KAOS Public repo for RHEL6/CentOS6

```bash
[sudo] yum install -y https://yum.kaos.io/6/release/x86_64/kaos-repo-8.0-0.el6.noarch.rpm
[sudo] yum install redis-latency-monitor
```

#### From ESSENTIAL KAOS Public repo for RHEL7/CentOS7

```bash
[sudo] yum install -y https://yum.kaos.io/7/release/x86_64/kaos-repo-8.0-0.el7.noarch.rpm
[sudo] yum install redis-latency-monitor
```

#### Prebuilt binaries

You can download prebuilt binaries for Linux and OS X from [EK Apps Repository](https://apps.kaos.io/redis-latency-monitor/latest).

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

| Repository | Status |
|------------|--------|
| Stable | [![Build Status](https://travis-ci.org/essentialkaos/redis-latency-monitor.svg?branch=master)](https://travis-ci.org/essentialkaos/redis-latency-monitor) |
| Unstable | [![Build Status](https://travis-ci.org/essentialkaos/redis-latency-monitor.svg?branch=develop)](https://travis-ci.org/essentialkaos/redis-latency-monitor) |

### License

[EKOL](https://essentialkaos.com/ekol)
