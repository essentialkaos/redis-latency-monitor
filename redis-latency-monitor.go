package main

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2017 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"pkg.re/essentialkaos/ek.v9/fmtc"
	"pkg.re/essentialkaos/ek.v9/fmtutil/table"
	"pkg.re/essentialkaos/ek.v9/log"
	"pkg.re/essentialkaos/ek.v9/options"
	"pkg.re/essentialkaos/ek.v9/timeutil"
	"pkg.re/essentialkaos/ek.v9/usage"

	"github.com/montanaflynn/stats"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	APP  = "Redis Latency Monitor"
	VER  = "1.0.0"
	DESC = "Tiny Redis client for latency measurement"
)

const (
	OPT_HOST     = "H:host"
	OPT_PORT     = "P:port"
	OPT_AUTH     = "a:password"
	OPT_TIMEOUT  = "t:timeout"
	OPT_INTERVAL = "i:interval"
	OPT_OUTPUT   = "o:output"
	OPT_NO_COLOR = "nc:no-color"
	OPT_HELP     = "h:help"
	OPT_VER      = "v:version"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// optMap is map with options
var optMap = options.Map{
	OPT_HOST:     {Value: "127.0.0.1"},
	OPT_PORT:     {Value: "6379"},
	OPT_TIMEOUT:  {Type: options.INT, Value: 3, Min: 1, Max: 300},
	OPT_AUTH:     {},
	OPT_INTERVAL: {Type: options.INT, Value: 60, Min: 1, Max: 3600},
	OPT_OUTPUT:   {},
	OPT_NO_COLOR: {Type: options.BOOL},
	OPT_HELP:     {Type: options.BOOL, Alias: "u:usage"},
	OPT_VER:      {Type: options.BOOL, Alias: "ver"},
}

// conn is connection to Redis
var conn net.Conn

// ////////////////////////////////////////////////////////////////////////////////// //

// main is main function
func main() {
	_, errs := options.Parse(optMap)

	if len(errs) != 0 {
		for _, err := range errs {
			printError(err.Error())
		}

		os.Exit(1)
	}

	if options.GetB(OPT_NO_COLOR) {
		fmtc.DisableColors = true
	}

	if options.GetB(OPT_VER) {
		showAbout()
		return
	}

	if options.GetB(OPT_HELP) {
		showUsage()
		return
	}

	setupLogger()
	connect()
}

// setupLogger setup logger
func setupLogger() {
	if !options.Has(OPT_OUTPUT) {
		log.Set("/dev/null", 0)
		return
	}

	err := log.Set(options.GetS(OPT_OUTPUT), 0644)

	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	log.EnableBufIO(250 * time.Millisecond)
}

// connect connect to Readis and start measurement loop
func connect() {
	var err error

	log.Aux("%s %s started", APP, VER)

	host := options.GetS(OPT_HOST) + ":" + options.GetS(OPT_PORT)
	timeout := time.Second * time.Duration(options.GetI(OPT_TIMEOUT))

	log.Info("Connecting to %s with %v timeout", host, timeout)

	conn, err = net.DialTimeout("tcp", host, timeout)

	if err != nil {
		printErrorAndExit(err.Error())
	}

	log.Info("Successfully connected to Redis")

	if options.GetS(OPT_AUTH) != "" {
		conn.Write([]byte("AUTH " + options.GetS(OPT_AUTH) + "\n"))
		log.Info("Authentication command sent to Redis")
	}

	measure()
}

// measure measure latency
func measure() {
	var measurements []float64
	var t *table.Table
	var count int

	buf := bufio.NewReader(conn)
	interval := time.Duration(options.GetI(OPT_INTERVAL)) * time.Second

	pretty := !options.Has(OPT_OUTPUT)

	if pretty {
		table.HeaderCapitalize = true
		t = table.NewTable("Date & Time", "Samples", "Min", "Max", "Mean", "Median", "StdDev", "Perc 95", "Perc 99")
		t.SetSizes(23, 8, 8, 8, 8, 8, 8, 8)
		t.SetAlignments(2, 2, 2, 2, 2, 2, 2, 2)
	}

	last := time.Now()

	for {
		start := time.Now()

		execCommand(buf)

		dur := float64(time.Since(start)) / float64(time.Millisecond)
		measurements = append(measurements, dur)

		if time.Since(last) >= interval {
			last = start

			printMeasurements(t, measurements, pretty)

			if pretty {
				count++

				if count == 10 {
					t.Separator()
					count = 0
				}
			}

			measurements = nil
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// execCommand execute command and read output
func execCommand(buf *bufio.Reader) {
	_, err := conn.Write([]byte("PING\n"))

	if err != nil {
		printErrorAndExit("Can't send PING command to Redis: %v", err)
	}

	_, err = buf.ReadString('\n')

	if err != nil && err != io.EOF {
		printErrorAndExit("Can't read Redis response: %v", err)
	}
}

// printMeasurements calculate and print measurements
func printMeasurements(t *table.Table, measurements []float64, pretty bool) {
	min, _ := stats.Min(measurements)
	max, _ := stats.Max(measurements)
	men, _ := stats.Mean(measurements)
	med, _ := stats.Median(measurements)
	mgh, _ := stats.Midhinge(measurements)
	sdv, _ := stats.StandardDeviation(measurements)
	p95, _ := stats.Percentile(measurements, 95.0)
	p99, _ := stats.Percentile(measurements, 99.0)

	if pretty {
		t.Print(
			timeutil.Format(time.Now(), "%Y/%m/%d %H:%M:%S.%K"),
			fmt.Sprintf("%d", len(measurements)),
			fmt.Sprintf("%.03gms", min),
			fmt.Sprintf("%.03gms", max),
			fmt.Sprintf("%.03gms", men),
			fmt.Sprintf("%.03gms", med),
			fmt.Sprintf("%.03gms", mgh),
			fmt.Sprintf("%.03gms", sdv),
			fmt.Sprintf("%.03gms", p95),
			fmt.Sprintf("%.03gms", p99),
		)
	} else {
		log.Info(
			"Samples: %d | Min: %5.03gms | Max: %5.03gms | Mean: %5.03gms | Median: %5.03gms | Midhinge: %5.03gms | StdDev: %5.03gms | Perc95: %5.03gms | Perc99: %5.03gms",
			len(measurements), min, max, men, med, mgh, sdv, p95, p99,
		)
	}
}

// printErrorAndExit print error message and exit from utility
func printErrorAndExit(f string, a ...interface{}) {
	if options.Has(OPT_OUTPUT) {
		log.Crit(f, a...)
	} else {
		printError(f, a...)
	}

	shutdown(1)
}

// printError prints error message to console
func printError(f string, a ...interface{}) {
	fmtc.Fprintf(os.Stderr, "{r}"+f+"{!}\n", a...)
}

// shutdown close connection to Redis and exit from utility
func shutdown(code int) {
	if conn != nil {
		conn.Close()
	}

	if options.Has(OPT_OUTPUT) {
		log.Flush()
	}

	os.Exit(1)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// showUsage print usage info
func showUsage() {
	info := usage.NewInfo("")

	info.AddOption(OPT_HOST, "Server hostname {s-}(127.0.0.1 by default){!}", "ip/host")
	info.AddOption(OPT_PORT, "Server port {s-}(6379 by default){!}", "port")
	info.AddOption(OPT_AUTH, "Password to use when connecting to the server", "password")
	info.AddOption(OPT_TIMEOUT, "Connection timeout in seconds {s-}(3 by default){!}", "1-300")
	info.AddOption(OPT_INTERVAL, "Interval in seconds {s-}(60 by default){!}", "1-3600")
	info.AddOption(OPT_OUTPUT, "Path to output file")
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

	info.AddExample(
		"-H 192.168.0.123 -P 6821 -t 15",
		"Start monitoring instance on 192.168.0.123:6821 with 15 second timeout",
	)

	info.Render()
}

// showAbout print info about version
func showAbout() {
	about := &usage.About{
		App:     APP,
		Version: VER,
		Desc:    DESC,
		Year:    2006,
		Owner:   "ESSENTIAL KAOS",
		License: "Essential Kaos Open Source License <https://essentialkaos.com/ekol>",
	}

	about.Render()
}
