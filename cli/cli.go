package cli

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2025 ESSENTIAL KAOS                          //
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

	"github.com/essentialkaos/ek/v13/errors"
	"github.com/essentialkaos/ek/v13/fmtc"
	"github.com/essentialkaos/ek/v13/fmtutil"
	"github.com/essentialkaos/ek/v13/fmtutil/table"
	"github.com/essentialkaos/ek/v13/log"
	"github.com/essentialkaos/ek/v13/mathutil"
	"github.com/essentialkaos/ek/v13/options"
	"github.com/essentialkaos/ek/v13/signal"
	"github.com/essentialkaos/ek/v13/strutil"
	"github.com/essentialkaos/ek/v13/support"
	"github.com/essentialkaos/ek/v13/support/deps"
	"github.com/essentialkaos/ek/v13/terminal"
	"github.com/essentialkaos/ek/v13/terminal/tty"
	"github.com/essentialkaos/ek/v13/timeutil"
	"github.com/essentialkaos/ek/v13/usage"
	"github.com/essentialkaos/ek/v13/usage/completion/bash"
	"github.com/essentialkaos/ek/v13/usage/completion/fish"
	"github.com/essentialkaos/ek/v13/usage/completion/zsh"
	"github.com/essentialkaos/ek/v13/usage/man"
	"github.com/essentialkaos/ek/v13/usage/update"

	"github.com/essentialkaos/redis-latency-monitor/stats"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// App info
const (
	APP  = "Redis Latency Monitor"
	VER  = "3.3.0"
	DESC = "Tiny Valkey/Redis client for latency measurement"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Main constants
const (
	LATENCY_SAMPLE_RATE int = 10
	CONNECT_SAMPLE_RATE int = 100
)

// Options
const (
	OPT_HOST       = "h:host"
	OPT_PORT       = "p:port"
	OPT_AUTH       = "a:password"
	OPT_TIMEOUT    = "t:timeout"
	OPT_INTERVAL   = "i:interval"
	OPT_CONNECT    = "C:connect"
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

// conn is connection to server
var conn net.Conn

// host is server host
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
		terminal.Error(errs.Error(" - "))
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
			WithApps(getServerVersionInfo()).
			Print()
		os.Exit(0)
	case options.GetB(OPT_HELP), options.GetS(OPT_HOST) == "true":
		genUsage().Print()
		os.Exit(0)
	}

	err := errors.Chain(
		setupErrorLog,
		createOutputWriter,
		setupSignalHandlers,
		startMeasurementProcess,
	)

	if err != nil {
		terminal.Error(err.Error())
		os.Exit(1)
	}
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

// setupErrorLog setups error log
func setupErrorLog() error {
	if !options.Has(OPT_ERROR_LOG) {
		return nil
	}

	err := log.Set(options.GetS(OPT_ERROR_LOG), 0644)

	if err != nil {
		return fmt.Errorf("Can't setup error log: %w", err)
	}

	return nil
}

// createOutputWriter creates and opens file for writing data
func createOutputWriter() error {
	if !options.Has(OPT_OUTPUT) {
		return nil
	}

	fd, err := os.OpenFile(options.GetS(OPT_OUTPUT), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)

	if err != nil {
		return fmt.Errorf("Can't open output file: %w", err)
	}

	outputWriter = bufio.NewWriter(fd)

	go flushOutput(250 * time.Millisecond)

	return nil
}

// setupSignalHandlers setups signals handler
func setupSignalHandlers() error {
	signal.Handlers{
		signal.INT:  signalHandler,
		signal.TERM: signalHandler,
		signal.QUIT: signalHandler,
	}.TrackAsync()

	return nil
}

// startMeasurementProcess starts measurement process
func startMeasurementProcess() error {
	prettyOutput := !options.Has(OPT_OUTPUT)
	interval := time.Duration(options.GetI(OPT_INTERVAL)) * time.Second

	host = options.GetS(OPT_HOST) + ":" + options.GetS(OPT_PORT)
	timeout = time.Second * time.Duration(options.GetI(OPT_TIMEOUT))

	if !options.GetB(OPT_CONNECT) {
		err := connectToServer()

		if err != nil {
			return err
		}
	}

	measureLatency(interval, prettyOutput)

	return nil
}

// connectToServer connects to server
func connectToServer() error {
	var err error

	conn, err = net.DialTimeout("tcp", host, timeout)

	if err != nil {
		return fmt.Errorf("Can't connect to server on %s: %w", host, err)
	}

	if options.GetS(OPT_AUTH) != "" {
		_, err = fmt.Fprintf(conn, "AUTH %s\r\n", options.GetS(OPT_AUTH))

		if err != nil {
			return fmt.Errorf("Can't send AUTH command: %w", err)
		}
	}

	return nil
}

// measureLatency measures latency
func measureLatency(interval time.Duration, prettyOutput bool) {
	var measurements stats.Data
	var count, pointer, sampleRate, errorNum int
	var t *table.Table
	var buf *bufio.Reader

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

	for range time.NewTicker(time.Duration(sampleRate) * time.Millisecond).C {
		start := time.Now()

		if connect {
			errorNum += makeConnection()
		} else {
			errorNum += execCommand(buf)
		}

		dur := uint64(time.Since(start) / time.Microsecond)
		measurements[pointer] = dur

		if time.Since(last) >= interval {
			last = start

			printMeasurements(t, errorNum, measurements[:pointer], prettyOutput)

			if prettyOutput {
				count++

				if count == 10 {
					t.Separator()
					count = 0
				}
			}

			errorNum, pointer = 0, 0
		} else {
			pointer++
		}
	}
}

// execCommand executes command and reads the output
func execCommand(buf *bufio.Reader) int {
	if conn == nil {
		if connectToServer() != nil {
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

// makeConnection creates and closes connection to server to check connect latency
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

// printMeasurements calculates and prints measurements
func printMeasurements(t *table.Table, errorNum int, measurements stats.Data, prettyOutput bool) {
	measurements.Sort()

	min := stats.Min(measurements)
	max := stats.Max(measurements)
	men := stats.Mean(measurements)
	sdv := stats.StandardDeviation(measurements)
	p95 := stats.Percentile(measurements, 95.0)
	p99 := stats.Percentile(measurements, 99.0)
	errs := fmtutil.PrettyNum(errorNum)

	if errorNum > 0 {
		errs = "{r}" + errs + "{!}"
	}

	if prettyOutput {
		t.Print(
			timeutil.Format(time.Now(), "%H{s}:{!}%M{s}:{!}%S{s-}.%K{!}"),
			fmtutil.PrettyNum(len(measurements)), errs,
			formatNumber(min), formatNumber(max),
			formatNumber(men), formatNumber(sdv),
			formatNumber(p95), formatNumber(p99),
		)
	} else {
		if options.GetB(OPT_TIMESTAMPS) {
			fmt.Fprintf(outputWriter,
				"%d;%d;%d;%.03f;%.03f;%.03f;%.03f;%.03f;%.03f;\n",
				time.Now().Unix(), len(measurements), errorNum,
				usToMs(min), usToMs(max), usToMs(men),
				usToMs(sdv), usToMs(p95), usToMs(p99),
			)
		} else {
			fmt.Fprintf(outputWriter,
				"%s;%d;%d;%.03f;%.03f;%.03f;%.03f;%.03f;%.03f;\n",
				timeutil.Format(time.Now(), "%Y/%m/%d %H:%M:%S.%K"),
				len(measurements), errorNum,
				usToMs(min), usToMs(max), usToMs(men),
				usToMs(sdv), usToMs(p95), usToMs(p99),
			)
		}

	}
}

// formatNumber formats floating number
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

// usToMs converts us in uint64 to ms in float64
func usToMs(us uint64) float64 {
	return float64(us) / 1000.0
}

// createOutputTable creates and configures output table struct
func createOutputTable() *table.Table {
	t := table.NewTable(
		"TIME", "SAMPLES", "ERRORS", "MIN", "MAX",
		"MEAN", "STDDEV", "PERC 95", "PERC 99",
	)

	t.SetSizes(12, 8, 8, 8, 10, 8, 8, 8)
	t.Width = 110

	t.SetAlignments(
		table.ALIGN_RIGHT, table.ALIGN_RIGHT, table.ALIGN_RIGHT,
		table.ALIGN_RIGHT, table.ALIGN_RIGHT, table.ALIGN_RIGHT,
		table.ALIGN_RIGHT, table.ALIGN_RIGHT,
	)

	return t
}

// alignTime blocks main thread until start of the minute
func alignTime() time.Time {
	var pause time.Duration

	if options.GetI(OPT_INTERVAL) >= 60 {
		pause = time.Minute
	} else {
		pause = time.Duration(options.GetI(OPT_INTERVAL)) * time.Second
	}

	waitDur := time.Until(time.Now().Truncate(pause).Add(pause))

	time.Sleep(waitDur)

	return time.Now()
}

// createMeasurementsSlice creates float64 slice for measurements
func createMeasurementsSlice(sampleRate int) []uint64 {
	size := (options.GetI(OPT_INTERVAL) * 1000) / sampleRate
	return make(stats.Data, size+sampleRate)
}

// flushOutput is function for flushing output
func flushOutput(interval time.Duration) {
	for range time.NewTicker(interval).C {
		outputWriter.Flush()
	}
}

// signalHandler is signal handler
func signalHandler() {
	if conn != nil {
		conn.Close()
	}

	if outputWriter != nil {
		outputWriter.Flush()
	}

	os.Exit(1)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// getServerVersionInfo returns info about Redis version
func getServerVersionInfo() support.App {
	var cmd *exec.Cmd
	var name, ver string

	switch {
	case hasApp("valkey-server"):
		name = "Valkey"
		cmd = exec.Command("valkey-server", "--version")
	case hasApp("redis-server"):
		name = "Redis"
		cmd = exec.Command("redis-server", "--version")
	default:
		return support.App{}
	}

	output, err := cmd.Output()

	if err != nil {
		return support.App{}
	}

	switch name {
	case "Redis":
		ver = strutil.ReadField(string(output), 2, false, ' ')
	case "Valkey":
		ver = strutil.ReadField(string(output), 1, false, ' ')
	}

	ver = strings.TrimLeft(ver, "v=")

	return support.App{name, ver}
}

// hasApp returns true if given app is installed on the system
func hasApp(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
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
