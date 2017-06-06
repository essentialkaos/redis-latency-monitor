package main

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2017 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bufio"
	"io"
	"net"
	"os"
	"time"

	"pkg.re/essentialkaos/ek.v9/fmtc"
	"pkg.re/essentialkaos/ek.v9/log"
	"pkg.re/essentialkaos/ek.v9/options"
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
		log.Crit(err.Error())
		shutdown(1)
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
	connbuf := bufio.NewReader(conn)

	var measurements []float64

	last := time.Now()

	for {
		start := time.Now()

		_, err := conn.Write([]byte("PING\n"))

		if err != nil {
			log.Crit("Can't send PING command to Redis: %v", err)
			shutdown(1)
		}

		_, err = connbuf.ReadString('\n')

		if err != nil && err != io.EOF {
			log.Crit("Can't read Redis response: %v", err)
			shutdown(1)
		}

		dur := float64(time.Since(start)) / float64(time.Millisecond)
		measurements = append(measurements, dur)

		if time.Since(last) > time.Minute {
			last = start

			min, _ := stats.Min(measurements)
			max, _ := stats.Max(measurements)
			med, _ := stats.Median(measurements)
			mgh, _ := stats.Midhinge(measurements)
			sdv, _ := stats.StandardDeviation(measurements)
			trm, _ := stats.Trimean(measurements)
			p95, _ := stats.Percentile(measurements, 95.0)
			p99, _ := stats.Percentile(measurements, 99.0)

			log.Info(
				"Samples: %d | Min: %5.03gms | Max: %5.03gms | Med: %5.03gms | Mgh: %5.03gms | StDev: %5.03gms | Trm: %5.03gms | P95: %5.03gms | P99: %5.03gms",
				len(measurements), min, max, med, mgh, sdv, trm, p95, p99,
			)

			measurements = nil
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// shutdown close connection to Redis and exit from utility
func shutdown(code int) {
	if conn != nil {
		conn.Close()
	}

	log.Flush()
	os.Exit(1)
}

// printError prints error message to console
func printError(f string, a ...interface{}) {
	fmtc.Fprintf(os.Stderr, "{r}"+f+"{!}\n", a...)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// showUsage print usage info
func showUsage() {
	info := usage.NewInfo("")

	info.AddOption(OPT_HOST, "Server hostname {s-}(127.0.0.1 by default){!}", "ip/host")
	info.AddOption(OPT_PORT, "Server port {s-}(6379 by default){!}", "port")
	info.AddOption(OPT_AUTH, "Password to use when connecting to the server", "password")
	info.AddOption(OPT_TIMEOUT, "Connection timeout in seconds {s-}(3 by default){!}", "1-300")
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
