package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2024 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/essentialkaos/ek/v12/fmtc"
	"github.com/essentialkaos/ek/v12/fmtutil"
	"github.com/essentialkaos/ek/v12/fmtutil/table"
	"github.com/essentialkaos/ek/v12/log"
	"github.com/essentialkaos/ek/v12/mathutil"
	"github.com/essentialkaos/ek/v12/options"
	"github.com/essentialkaos/ek/v12/strutil"
	"github.com/essentialkaos/ek/v12/support"
	"github.com/essentialkaos/ek/v12/support/deps"
	"github.com/essentialkaos/ek/v12/terminal"
	"github.com/essentialkaos/ek/v12/terminal/tty"
	"github.com/essentialkaos/ek/v12/timeutil"
	"github.com/essentialkaos/ek/v12/usage"
	"github.com/essentialkaos/ek/v12/usage/completion/bash"
	"github.com/essentialkaos/ek/v12/usage/completion/fish"
	"github.com/essentialkaos/ek/v12/usage/completion/zsh"
	"github.com/essentialkaos/ek/v12/usage/man"
	"github.com/essentialkaos/ek/v12/usage/update"

	"github.com/essentialkaos/redis-latency-monitor/stats"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// App info
const (
	APP  = "Redis Latency Monitor"
	VER  = "3.2.3"
	DESC = "Tiny Redis client for latency measurement"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Main constants
const (
	LATENCY_SAMPLE_RATE int = 10
	CONNECT_SAMPLE_RATE     = 100
)

// Options
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

	OPT_VERB_VER     = "vv:verbose-version"
	OPT_COMPLETION   = "completion"
	OPT_GENERATE_MAN = "generate-man"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// optMap is map with options
var optMap = options.Map{
	OPT_HOST:       {Type: options.MIXED, Value: "127.0.0.1"},
	OPT_PORT:       {Value: "6379"},
	OPT_CONNECT:    {Type: options.BOOL},
	OPT_TIMEOUT:    {Type: options.INT, Value: 3, Min: 1, Max: 300},
	OPT_AUTH:       {},
	OPT_INTERVAL:   {Type: options.INT, Value: 60, Min: 1, Max: 3600},
	OPT_TIMESTAMPS: {Type: options.BOOL},
	OPT_OUTPUT:     {},
	OPT_ERROR_LOG:  {},
	OPT_NO_COLOR:   {Type: options.BOOL},
	OPT_HELP:       {Type: options.BOOL},
	OPT_VER:        {Type: options.MIXED},

	OPT_VERB_VER:     {Type: options.BOOL},
	OPT_COMPLETION:   {},
	OPT_GENERATE_MAN: {Type: options.BOOL},
}

// colorTagApp contains color tag for app name
var colorTagApp string

// colorTagVer contains color tag for app version
var colorTagVer string

// pingCommand is PING command data
var pingCommand = []byte("PING\r\n")

// conn is connection to Redis
var conn net.Conn

// host is Redis host
var host string

// timeout is connection timeout
var timeout time.Duration

// outputWriter is buffered output writer
var outputWriter *bufio.Writer

// errorLogged is error logging flag
var errorLogged bool

// ////////////////////////////////////////////////////////////////////////////////// //

// Run is main application function
func Run(gitRev string, gomod []byte) {
	preConfigureUI()

	runtime.GOMAXPROCS(4)

	_, errs := options.Parse(optMap)

	if !errs.IsEmpty() {
		terminal.Error("Options parsing errors:")
		terminal.Error(errs.String())
		os.Exit(1)
	}

	configureUI()

	switch {
	case options.Has(OPT_COMPLETION):
		os.Exit(printCompletion())
	case options.Has(OPT_GENERATE_MAN):
		printMan()
		os.Exit(0)
	case options.GetB(OPT_VER):
		genAbout(gitRev).Print(options.GetS(OPT_VER))
		os.Exit(0)
	case options.GetB(OPT_VERB_VER):
		support.Collect(APP, VER).
			WithRevision(gitRev).
			WithDeps(deps.Extract(gomod)).
			WithApps(getRedisVersionInfo()).
			Print()
		os.Exit(0)
	case options.GetB(OPT_HELP), options.GetS(OPT_HOST) == "true":
		genUsage().Print()
		os.Exit(0)
	}

	if options.Has(OPT_ERROR_LOG) {
		setupErrorLog()
	}

	startMeasurementProcess()
}

// preConfigureUI preconfigures UI based on information about user terminal
func preConfigureUI() {
	if !tty.IsTTY() {
		fmtc.DisableColors = true
	}

	switch {
	case fmtc.IsTrueColorSupported():
		colorTagApp, colorTagVer = "{*}{#DC382C}", "{#A32422}"
	case fmtc.Is256ColorsSupported():
		colorTagApp, colorTagVer = "{*}{#160}", "{#124}"
	default:
		colorTagApp, colorTagVer = "{r*}", "{r}"
	}
}

// configureUI configures user interface
func configureUI() {
	if options.GetB(OPT_NO_COLOR) {
		fmtc.DisableColors = true
	}
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
		measurements   stats.Data
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

		dur := uint64(time.Since(start) / time.Microsecond)
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
func printMeasurements(t *table.Table, errors int, measurements stats.Data, prettyOutput bool) {
	measurements.Sort()

	min := stats.Min(measurements)
	max := stats.Max(measurements)
	men := stats.Mean(measurements)
	sdv := stats.StandardDeviation(measurements)
	p95 := stats.Percentile(measurements, 95.0)
	p99 := stats.Percentile(measurements, 99.0)

	if prettyOutput {
		t.Print(
			timeutil.Format(time.Now(), "%H:%M:%S{s-}.%K{!}"),
			fmtutil.PrettyNum(len(measurements)),
			fmtutil.PrettyNum(errors),
			formatNumber(min), formatNumber(max),
			formatNumber(men), formatNumber(sdv),
			formatNumber(p95), formatNumber(p99),
		)
	} else {
		if options.GetB(OPT_TIMESTAMPS) {
			outputWriter.WriteString(
				fmt.Sprintf(
					"%d;%d;%d;%.03f;%.03f;%.03f;%.03f;%.03f;%.03f;\n",
					time.Now().Unix(), len(measurements), errors,
					usToMs(min), usToMs(max), usToMs(men),
					usToMs(sdv), usToMs(p95), usToMs(p99),
				),
			)
		} else {
			outputWriter.WriteString(
				fmt.Sprintf(
					"%s;%d;%d;%.03f;%.03f;%.03f;%.03f;%.03f;%.03f;\n",
					timeutil.Format(time.Now(), "%Y/%m/%d %H:%M:%S.%K"),
					len(measurements), errors,
					usToMs(min), usToMs(max), usToMs(men),
					usToMs(sdv), usToMs(p95), usToMs(p99),
				),
			)
		}

	}
}

// formatNumber format floating number
func formatNumber(value uint64) string {
	if value == 0 {
		return "{s-}------{!}"
	}

	fv := float64(value) / 1000.0

	switch {
	case fv > 1000.0:
		fv = mathutil.Round(fv, 0)
	case fv > 10:
		fv = mathutil.Round(fv, 1)
	case fv > 1:
		fv = mathutil.Round(fv, 2)
	}

	switch {
	case fv >= 100.0:
		return "{r}" + fmtutil.PrettyNum(fv) + "{!}"
	case fv >= 10.0:
		return "{y}" + fmtutil.PrettyNum(fv) + "{!}"
	default:
		return strings.Replace(fmtutil.PrettyNum(fv), ".", "{s}.", -1) + "{!}"
	}
}

// usToMs convert us in uint64 to ms in float64
func usToMs(us uint64) float64 {
	return float64(us) / 1000.0
}

// createOutputTable create and configure output table struct
func createOutputTable() *table.Table {
	t := table.NewTable(
		"TIME", "SAMPLES", "ERRORS", "MIN", "MAX",
		"MEAN", "STDDEV", "PERC 95", "PERC 99",
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
func createMeasurementsSlice(sampleRate int) []uint64 {
	size := (options.GetI(OPT_INTERVAL) * 1000) / sampleRate
	return make(stats.Data, size)
}

// flushOutput is function for flushing output
func flushOutput(interval time.Duration) {
	for range time.NewTicker(interval).C {
		outputWriter.Flush()
	}
}

// printErrorAndExit print error message and exit from utility
func printErrorAndExit(f string, a ...interface{}) {
	terminal.Error(f, a...)
	shutdown(1)
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

// getRedisVersionInfo returns info about Redis version
func getRedisVersionInfo() support.App {
	cmd := exec.Command("redis-server", "--version")
	output, err := cmd.Output()

	if err != nil {
		return support.App{"Redis", ""}
	}

	ver := strutil.ReadField(string(output), 2, false, ' ')
	ver = strings.TrimLeft(ver, "v=")

	return support.App{"Redis", ver}
}

// printCompletion prints completion for given shell
func printCompletion() int {
	info := genUsage()

	switch options.GetS(OPT_COMPLETION) {
	case "bash":
		fmt.Print(bash.Generate(info, "redis-latency-monitor"))
	case "fish":
		fmt.Print(fish.Generate(info, "redis-latency-monitor"))
	case "zsh":
		fmt.Print(zsh.Generate(info, optMap, "redis-latency-monitor"))
	default:
		return 1
	}

	return 0
}

// printMan prints man page
func printMan() {
	fmt.Println(man.Generate(genUsage(), genAbout("")))
}

// genUsage generates usage info
func genUsage() *usage.Info {
	info := usage.NewInfo()

	info.AppNameColorTag = colorTagApp

	info.AddSpoiler("{&}Utility shows PING command latency or connection latency in milliseconds (one thousandth\nof a second).{!}")

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

	return info
}

// genAbout generates info about version
func genAbout(gitRev string) *usage.About {
	about := &usage.About{
		App:     APP,
		Version: VER,
		Desc:    DESC,
		Year:    2006,
		Owner:   "ESSENTIAL KAOS",
		License: "Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>",

		AppNameColorTag: colorTagApp,
		VersionColorTag: colorTagVer,
		DescSeparator:   "{s}—{!}",
	}

	if gitRev != "" {
		about.Build = "git:" + gitRev
		about.UpdateChecker = usage.UpdateChecker{
			"essentialkaos/redis-latency-monitor",
			update.GitHubChecker,
		}
	}

	return about
}
