package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"runtime/trace"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	pd "github.com/tikv/pd/client"
	"github.com/tikv/pd/client/pkg/caller"
	"go.uber.org/zap"

	plog "github.com/pingcap/log"

	"github.com/spf13/cobra"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

const (
	histogramDirPath = "user-local/histograms"
	slowThreshold    = 30 * time.Millisecond
)

var (
	nItersFlag   string
	nClientsFlag string
	pdAddrsFlag  string
	nWorkersFlag int

	pdAddrs  []string
	nWorkers int
)

var rootCmd = &cobra.Command{
	Use:   "testapp",
	Short: "Test app to run iterations and generate histograms",
	Run:   run,
}

func init() {
	rootCmd.Flags().StringVar(&nItersFlag, "nIters", "", "Comma-separated values for number of iterations (e.g., 10,20,30)")
	if err := rootCmd.MarkFlagRequired("nIters"); err != nil {
		log.Fatalf("Failed to mark nIters as required: %v", err)
	}

	rootCmd.Flags().StringVar(&nClientsFlag, "nClients", "", "Number of PD clients to use")
	if err := rootCmd.MarkFlagRequired("nClients"); err != nil {
		log.Fatalf("Failed to mark nClients as required: %v", err)
	}

	rootCmd.Flags().StringVar(&pdAddrsFlag, "pdAddrs", "", "Comma-separated PD addresses")
	if err := rootCmd.MarkFlagRequired("pdAddrs"); err != nil {
		log.Fatalf("Failed to mark pdAddrs as required: %v", err)
	}

	rootCmd.Flags().IntVar(&nWorkersFlag, "nWorkers", 1, "Number of workers to use for getting TS")
	if err := rootCmd.MarkFlagRequired("nWorkers"); err != nil {
		log.Fatalf("Failed to mark nWorkers as required: %v", err)
	}
}

func main() {
	plog.ReplaceGlobals(zap.NewNop(), &plog.ZapProperties{})

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	// Parse nIters from the flag
	nIters := parseNIntegersFlag(nItersFlag)
	nClients := parseNIntegersFlag(nClientsFlag)
	pdAddrs = strings.Split(pdAddrsFlag, ",")
	nWorkers = nWorkersFlag

	os.MkdirAll(histogramDirPath, os.ModePerm)

	logf("Running benchmark with nIters=%v nClients=%v", nIters, nClients)
	for i, nIter := range nIters {
		for j, nClient := range nClients {
			runBench(fmt.Sprintf("test-%v-%v", i, j), nIter, nClient, path.Join(histogramDirPath, fmt.Sprintf("histogram_%d_%v.png", nIter, nClient)))
		}
	}
}

func parseNIntegersFlag(flag string) []int {
	var nIters []int
	for _, val := range strings.Split(flag, ",") {
		nIter, err := strconv.Atoi(val)
		if err != nil {
			log.Fatalf("Invalid value for nIters: %s", val)
		}
		nIters = append(nIters, nIter)
	}
	return nIters
}

func runBench(ident string, nIter int, nClients int, filename string) {
	// Create the PD clients
	pdClients := make([]pd.Client, nClients)
	for i := 0; i < nClients; i++ {
		pdClient, err := pd.NewClient(caller.TestComponent, pdAddrs, pd.SecurityOption{})
		if err != nil {
			log.Fatalf("Failed to create PD client: %v", err)
		}
		pdClients[i] = pdClient
	}

	var (
		wg        = sync.WaitGroup{}
		durations = make([]float64, nIter)
		tasks     = make(chan int, nIter)
		slowCount = int32(0)
	)
	defer startTrace()()

	// create workers for getting TS
	for i := 0; i < nWorkers; i++ {
		wg.Add(1)
		go func() {
			for task := range tasks {
				startTime := time.Now()
				if _, _, err := pdClients[task%nClients].GetTS(context.TODO()); err != nil {
					log.Fatalf("Failed to get TS: %v", err)
				}
				endTime := time.Now()

				if endTime.After(startTime.Add(slowThreshold)) {
					atomic.AddInt32(&slowCount, 1)
				}
				durations[task] = float64(endTime.Sub(startTime).Milliseconds())
			}
			wg.Done()
		}()
	}

	// Start the test
	start := time.Now()
	for i := 0; i < nIter; i++ {
		tasks <- i
	}
	close(tasks)
	wg.Wait()
	duration := time.Since(start)

	// Sort the durations
	sort.Float64s(durations)
	genHistogram(durations, filename)

	logf("%s: nIter=%v  nWorkers=%v  nClients=%v  slowCount=%v  duration=%v",
		ident, humanize.Comma(int64(nIter)), humanize.Comma(int64(slowCount)),
		nClients, humanize.Comma(int64(slowCount)), duration)
}

func genHistogram(durations []float64, filename string) {
	// Create and configure the plot
	p := plot.New()
	p.Title.Text = "Duration Histogram"
	p.X.Label.Text = "Milliseconds"
	p.Y.Label.Text = "Frequency"

	// Create a histogram with defined bins
	hist, err := plotter.NewHist(plotter.Values(durations), 10)
	if err != nil {
		log.Fatalf("Failed to create histogram: %v", err)
	}
	hist.Normalize(1)

	p.Add(hist)

	// Save the plot to a PNG file
	if err := p.Save(6*vg.Inch, 6*vg.Inch, filename); err != nil {
		log.Fatalf("Failed to save plot: %v", err)
	}
}

func logf(format string, args ...interface{}) {
	log.Printf("%v %v", color.YellowString("LOG"), fmt.Sprintf(format, args...))
}

func startTrace() func() {
	f, err := os.Create("trace.out")
	if err != nil {
		panic(err)
	}

	if err := trace.Start(f); err != nil {
		panic(err)
	}

	return func() {
		trace.Stop()
		f.Close()
	}
}
