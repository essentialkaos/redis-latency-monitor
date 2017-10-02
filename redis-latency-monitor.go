// +build linux

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
	"math"
	"net"
	"os"
	"runtime"
	"strings"
	"time"

	"pkg.re/essentialkaos/ek.v9/fmtc"
	"pkg.re/essentialkaos/ek.v9/fmtutil"
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
	VER  = "2.4.0"
	DESC = "Tiny Redis client for latency measurement"
)

const (
	LATENCY_SAMPLE_RATE int = 10
	CONNECT_SAMPLE_RATE     = 100
)

const (
	OPT_HOST       = "h:host"
	OPT_PORT       = "p:port"
	OPT_AUTH       = "a:password"
	OPT_TIMEOUT    = "t:timeout"
	OPT_INTERVAL   = "i:interval"
	OPT_CONNECT    = "c:connect"
	OPT_TIMESTAMPS = "T:timestamps"
	OPT_OUTPUT     = "o:output"
	OPT_ERROR_LOG  = "e:error-log"
	OPT_NO_COLOR   = "nc:no-color"
	OPT_HELP       = "help"
	OPT_VER        = "v:version"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// optMap is map with options
var optMap = options.Map{
	OPT_HOST:       {Value: "127.0.0.1"},
	OPT_PORT:       {Value: "6379"},
	OPT_CONNECT:    {Type: options.BOOL},
	OPT_TIMEOUT:    {Type: options.INT, Value: 3, Min: 1, Max: 300},
	OPT_AUTH:       {},
	OPT_INTERVAL:   {Type: options.INT, Value: 60, Min: 1, Max: 3600},
	OPT_TIMESTAMPS: {Type: options.BOOL},
	OPT_OUTPUT:     {},
	OPT_ERROR_LOG:  {},
	OPT_NO_COLOR:   {Type: options.BOOL},
	OPT_HELP:       {Type: options.BOOL, Alias: "u:usage"},
	OPT_VER:        {Type: options.BOOL, Alias: "ver"},
}

// pingCommand is PING command data
var pingCommand = []byte("PING\r\n")

var (
	conn         net.Conn
	host         string
	timeout      time.Duration
	outputWriter *bufio.Writer
	errorLogged  bool
)

// ////////////////////////////////////////////////////////////////////////////////// //

// main is main function
func main() {
	runtime.GOMAXPROCS(4)

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

	if options.Has(OPT_ERROR_LOG) {
		setupErrorLog()
	}

	startMeasurementProcess()
}

// setupErrorLog setup error log
func setupErrorLog() {
	err := log.Set(options.GetS(OPT_ERROR_LOG), 0644)

	if err != nil {
		printErrorAndExit(err.Error())
	}
}

// startMeasurementProcess start measurement process
func startMeasurementProcess() {
	prettyOutput := !options.Has(OPT_OUTPUT)
	interval := time.Duration(options.GetI(OPT_INTERVAL)) * time.Second

	host = options.GetS(OPT_HOST) + ":" + options.GetS(OPT_PORT)
	timeout = time.Second * time.Duration(options.GetI(OPT_TIMEOUT))

	if !options.GetB(OPT_CONNECT) {
		connectToRedis(false)
	}

	if options.Has(OPT_OUTPUT) {
		createOutputWriter()
	}

	measureLatency(interval, prettyOutput)
}

// createOutputWriter create and open file for writing data
func createOutputWriter() {
	fd, err := os.OpenFile(options.GetS(OPT_OUTPUT), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)

	if err != nil {
		printErrorAndExit(err.Error())
	}

	outputWriter = bufio.NewWriter(fd)

	go flushOutput(250 * time.Millisecond)
}

// connectToRedis connect to redis instance
func connectToRedis(reconnect bool) error {
	var err error

	conn, err = net.DialTimeout("tcp", host, timeout)

	if err != nil {
		if !reconnect {
			printErrorAndExit("Can't connect to Redis on %s", host)
		} else {
			return err
		}
	}

	if options.GetS(OPT_AUTH) != "" {
		_, err = conn.Write([]byte("AUTH " + options.GetS(OPT_AUTH) + "\r\n"))

		if err != nil {
			if !reconnect {
				printErrorAndExit("Can't send AUTH command")
			} else {
				return err
			}
		}
	}

	return nil
}

// measureLatency measure latency
func measureLatency(interval time.Duration, prettyOutput bool) {
	var (
		measurements   []float64
		count, pointer int
		t              *table.Table
		sampleRate     int
		errors         int
		buf            *bufio.Reader
	)

	if prettyOutput {
		t = createOutputTable()
	}

	connect := options.GetB(OPT_CONNECT)

	if connect {
		sampleRate = CONNECT_SAMPLE_RATE
	} else {
		sampleRate = LATENCY_SAMPLE_RATE
		buf = bufio.NewReader(conn)
	}

	measurements = createMeasurementsSlice(sampleRate)

	last := alignTime()

	for {
		time.Sleep(time.Duration(sampleRate) * time.Millisecond)

		start := time.Now()

		if connect {
			errors += makeConnection()
		} else {
			errors += execCommand(buf)
		}

		dur := float64(time.Since(start)) / float64(time.Millisecond)
		measurements[pointer] = dur

		if time.Since(last) >= interval {
			last = start

			printMeasurements(t, errors, measurements[:pointer], prettyOutput)

			if prettyOutput {
				count++

				if count == 10 {
					t.Separator()
					count = 0
				}
			}

			errors = 0
			pointer = 0
		} else {
			pointer++
		}
	}
}

// execCommand execute command and read output
func execCommand(buf *bufio.Reader) int {
	if conn == nil {
		if connectToRedis(true) != nil {
			return 1
		}
	}

	_, err := conn.Write(pingCommand)

	if err != nil {
		if options.Has(OPT_ERROR_LOG) && !errorLogged {
			log.Error(err.Error())
			errorLogged = true
		}

		conn = nil

		return 1
	}

	_, err = buf.ReadString('\n')

	if err != nil && err != io.EOF {
		if options.Has(OPT_ERROR_LOG) && !errorLogged {
			log.Error(err.Error())
			errorLogged = true
		}

		conn = nil

		return 1
	}

	errorLogged = false

	return 0
}

// makeConnection create and close connection to Redis
func makeConnection() int {
	var err error

	conn, err = net.DialTimeout("tcp", host, timeout)

	if err != nil {
		if options.Has(OPT_ERROR_LOG) && !errorLogged {
			log.Error(err.Error())
			errorLogged = true
		}

		return 1
	}

	conn.Close()

	errorLogged = false

	return 0
}

// printMeasurements calculate and print measurements
func printMeasurements(t *table.Table, errors int, measurements []float64, prettyOutput bool) {
	min, _ := stats.Min(measurements)
	max, _ := stats.Max(measurements)
	men, _ := stats.Mean(measurements)
	med, _ := stats.Median(measurements)
	mgh, _ := stats.Midhinge(measurements)
	sdv, _ := stats.StandardDeviation(measurements)
	p95, _ := stats.Percentile(measurements, 95.0)
	p99, _ := stats.Percentile(measurements, 99.0)

	if prettyOutput {
		t.Print(
			timeutil.Format(time.Now(), "%H:%M:%S.%K"),
			fmtutil.PrettyNum(len(measurements)),
			fmtutil.PrettyNum(errors),
			formatNumber(min), formatNumber(max),
			formatNumber(men), formatNumber(med),
			formatNumber(mgh), formatNumber(sdv),
			formatNumber(p95), formatNumber(p99),
		)
	} else {
		if options.GetB(OPT_TIMESTAMPS) {
			outputWriter.WriteString(
				fmt.Sprintf(
					"%d;%d;%d;%.03f;%.03f;%.03f;%.03f;%.03f;%.03f;%.03f;%.03f;\n",
					time.Now().Unix(), len(measurements), errors,
					min, max, men, med, mgh, sdv, p95, p99,
				),
			)
		} else {
			outputWriter.WriteString(
				fmt.Sprintf(
					"%s;%d;%d;%.03f;%.03f;%.03f;%.03f;%.03f;%.03f;%.03f;%.03f;\n",
					timeutil.Format(time.Now(), "%Y/%m/%d %H:%M:%S.%K"),
					len(measurements), errors,
					min, max, men, med, mgh, sdv, p95, p99,
				),
			)
		}

	}
}

// formatNumber format floating number
func formatNumber(value float64) string {
	if math.IsNaN(value) {
		return "------"
	}

	if value == 0.0 {
		return "0{s-}.001{!}"
	}

	if value > 1000.0 {
		value = math.Floor(value)
	}

	return strings.Replace(fmtutil.PrettyNum(value), ".", "{s-}.", -1) + "{!}"
}

// createOutputTable create and configure output table struct
func createOutputTable() *table.Table {
	t := table.NewTable(
		"TIME", "SAMPLES", "ERRORS", "MIN", "MAX", "MEAN",
		"MEDIAN", "STDDEV", "PERC 95", "PERC 99",
	)

	t.SetSizes(12, 8, 8, 8, 10, 8, 8, 8)

	t.SetAlignments(
		table.ALIGN_RIGHT, table.ALIGN_RIGHT, table.ALIGN_RIGHT,
		table.ALIGN_RIGHT, table.ALIGN_RIGHT, table.ALIGN_RIGHT,
		table.ALIGN_RIGHT, table.ALIGN_RIGHT,
	)

	return t
}

// alignTime block main thread until nearest interval start point
func alignTime() time.Time {
	interval := options.GetI(OPT_INTERVAL)

	for {
		now := time.Now()

		if interval >= 60 {
			if now.Second() == 0 {
				return now
			}
		} else {
			if now.Second()%interval == 0 {
				return now
			}
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// createMeasurementsSlice create float64 slice for measurements
func createMeasurementsSlice(sampleRate int) []float64 {
	size := (options.GetI(OPT_INTERVAL) * 1000) / sampleRate
	return make([]float64, size)
}

// flushOutput is function for flushing output
func flushOutput(interval time.Duration) {
	for range time.NewTicker(interval).C {
		outputWriter.Flush()
	}
}

// printErrorAndExit print error message and exit from utility
func printErrorAndExit(f string, a ...interface{}) {
	printError(f, a...)
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

	if outputWriter != nil {
		outputWriter.Flush()
	}

	os.Exit(code)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// showUsage print usage info
func showUsage() {
	info := usage.NewInfo("")

	info.AddSpoiler("Utility show PING command latency or connection latency in milliseconds (one thousandth of a second).")

	info.AddOption(OPT_HOST, "Server hostname {s-}(127.0.0.1 by default){!}", "ip/host")
	info.AddOption(OPT_PORT, "Server port {s-}(6379 by default){!}", "port")
	info.AddOption(OPT_CONNECT, "Measure connection latency instead of command latency")
	info.AddOption(OPT_AUTH, "Password to use when connecting to the server", "password")
	info.AddOption(OPT_TIMEOUT, "Connection timeout in seconds {s-}(3 by default){!}", "1-300")
	info.AddOption(OPT_INTERVAL, "Interval in seconds {s-}(60 by default){!}", "1-3600")
	info.AddOption(OPT_TIMESTAMPS, "Use unix timestamps in output")
	info.AddOption(OPT_OUTPUT, "Path to output CSV file", "file")
	info.AddOption(OPT_ERROR_LOG, "Path to log with error messages", "file")
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

	info.AddExample(
		"-h 192.168.0.123 -p 6821 -t 15",
		"Start monitoring instance on 192.168.0.123:6821 with 15 second timeout",
	)

	info.AddExample(
		"-c -i 15 -o latency.csv",
		"Start connection latency monitoring with 15 second interval and save result to CSV file",
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
