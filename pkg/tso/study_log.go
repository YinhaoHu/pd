package tso

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/fatih/color"
)

const (
	traceDirPath    = "user-local/pd-logs"
	traceWriterName = "trace.log"
)

var traceWriter *os.File

func init() {
	os.MkdirAll(traceDirPath, os.ModePerm)

	var err error
	traceWriter, err = os.Create(path.Join(traceDirPath, traceWriterName))
	if err != nil {
		panic(err)
	}
}

func tracef(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000000")
	prefix := fmt.Sprintf("%s %v", timestamp, message)
	fmt.Fprintf(traceWriter, "%s\n", prefix)
}

func tracefInStdout(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	yellow := color.New(color.FgYellow).SprintFunc()
	timestamp := time.Now().Format("2006-01-02 15:04:05.000000")
	prefix := fmt.Sprintf("%s %v %v", timestamp, yellow("Trace"), message)
	fmt.Printf("%s\n", prefix)
}
