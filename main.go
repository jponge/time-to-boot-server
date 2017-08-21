/*
 * Copyright (c) 2017 Julien Ponge
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package main

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/fatih/color"
	"github.com/montanaflynn/stats"
	"github.com/urfave/cli"
)

func boot(command string, args ...string) (*exec.Cmd, error) {
	cmd := exec.Command(command, args...)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

func tryConnectingWithTCP(target string) (bool, func()) {
	conn, err := net.Dial("tcp", target)
	if err == nil {
		return true, func() {
			conn.Close()
		}
	}
	return false, nil
}

func tryConnectingWithHTTPGet(target string) (bool, func()) {
	resp, err := http.Get(target)
	if err == nil && resp.StatusCode == 200 {
		ioutil.ReadAll(resp.Body)
		return true, func() {
			resp.Body.Close()
		}
	}
	return false, nil
}

func connectionFunctionFor(mode string) func(string) (bool, func()) {
	if mode == "tcp-connect" {
		return tryConnectingWithTCP
	} else if mode == "http-get" {
		return tryConnectingWithHTTPGet
	}
	log.Fatal("Unknow mode: ", mode)
	return nil
}

func measure(mode string, target string, command string, args ...string) time.Duration {
	connectionFunction := connectionFunctionFor(mode)
	start := time.Now()
	cmd, err := boot(command, args...)
	if err != nil {
		log.Fatal(err)
	}
	for {
		if status, houseKeeper := connectionFunction(target); status == true {
			duration := time.Since(start)
			houseKeeper()
			cmd.Process.Kill()
			cmd.Process.Wait()
			return duration
		}
	}
}

func benchmark(mode string, dryRuns int, runs int, pauseBetweenRuns time.Duration, target string, command string, args ...string) {

	color.Cyan("Dry runs")
	for i := 0; i < dryRuns; i++ {
		duration := measure(mode, target, command, args...)
		color.Cyan("  - %s", duration)
		time.Sleep(pauseBetweenRuns)
	}

	durations := make([]float64, runs)
	color.Green("Runs")
	for i := 0; i < runs; i++ {
		duration := measure(mode, target, command, args...)
		durations[i] = float64(duration.Nanoseconds())
		color.Green("  - %s", duration)
		time.Sleep(pauseBetweenRuns)
	}

	min, _ := stats.Min(durations)
	color.Yellow("Min: %s", float64ToDuration(min))

	max, _ := stats.Max(durations)
	color.Yellow("Max: %s", float64ToDuration(max))

	med, _ := stats.Median(durations)
	dev, _ := stats.StandardDeviation(durations)
	color.Yellow("Median: %s (std dev %s)", float64ToDuration(med), float64ToDuration(dev))

	outliers, _ := stats.QuartileOutliers(durations)
	color.Yellow("Ouliers:")
	color.Yellow("  - mild: %s", float64DataToDurations(outliers.Mild))
	color.Yellow("  - extreme: %s", float64DataToDurations(outliers.Extreme))

	percentiles := []float64{75.0, 80.0, 85.0, 90.0, 95.0, 97.5, 98.0, 99.0, 99.9, 100.0}
	color.Yellow("Percentiles:")
	for i := range percentiles {
		r, _ := stats.Percentile(durations, percentiles[i])
		color.Yellow("  - %f%%: %s", percentiles[i], float64ToDuration(r))
	}
}

func float64ToDuration(f float64) time.Duration {
	return time.Duration(int64(f))
}

func float64DataToDurations(data stats.Float64Data) []time.Duration {
	durations := make([]time.Duration, len(data))
	for i := range data {
		durations[i] = float64ToDuration(data[i])
	}
	return durations
}

func main() {
	app := cli.NewApp()

	app.Name = "time-to-boot-server"
	app.Usage = "Measure the time to boot a server and make a first connection"
	app.Version = "0.1"
	app.ArgsUsage = "executable application arguments\n   (tip: use -- to pass flags to the executable, as in --executable python -- -m SimpleHTTPServer 8080)"

	var mode string
	var dryRuns int
	var runs int
	var pauseDuration int
	var target string
	var executable string

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "mode",
			Usage:       "mode for connecting in: http-get, tcp-connect",
			Value:       "http-get",
			Destination: &mode,
		},
		cli.IntFlag{
			Name:        "dry-runs",
			Usage:       "number of dry runs",
			Value:       2,
			Destination: &dryRuns,
		},
		cli.IntFlag{
			Name:        "runs",
			Usage:       "number of runs",
			Value:       20,
			Destination: &runs,
		},
		cli.IntFlag{
			Name:        "pause",
			Usage:       "pause duration (in seconds) between runs",
			Value:       10,
			Destination: &pauseDuration,
		},
		cli.StringFlag{
			Name:        "target",
			Usage:       "connection target",
			Value:       "http://localhost:8080/",
			Destination: &target,
		},
		cli.StringFlag{
			Name:        "executable",
			Usage:       "executable to run",
			Value:       "",
			Destination: &executable,
		},
	}

	app.Action = func(c *cli.Context) error {
		if len(executable) == 0 {
			log.Fatal("An executable must be specified")
		}
		benchmark(mode, dryRuns, runs, time.Duration(pauseDuration)*time.Second, target, executable, c.Args()...)
		return nil
	}

	app.Run(os.Args)
}
