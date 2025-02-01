package logutil

import (
	"fmt"
	"log"
	"path"
	"runtime"
	"sync"
)

var (
	logOccurenceMap      = make(map[string]int)
	logOccurenceMapMutex = sync.Mutex{}
)

func LogWithLimitation(ident string, limit int, format string, args ...any) {
	logOccurenceMapMutex.Lock()
	defer logOccurenceMapMutex.Unlock()

	if _, ok := logOccurenceMap[ident]; !ok {
		logOccurenceMap[ident] = 0
	}
	if logOccurenceMap[ident] < limit {
		logOccurenceMap[ident]++
		logf(2, format, args...)
	}
}

func Logf(format string, args ...any) {
	logf(2, format, args...)
}

func logf(skipCaller int, format string, args ...any) {
	_, file, line, _ := runtime.Caller(skipCaller)
	file = path.Base(file)
	log.Printf("%v:%v %v", file, line, fmt.Sprintf(format, args...))
}
