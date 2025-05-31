package utils

import (
	"fmt"
	"path"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Taken from https://github.com/sirupsen/logrus/issues/63#issuecomment-236052137

type LogrusSourceContextHook struct{}

func (hook LogrusSourceContextHook) Levels() []log.Level {
	return log.AllLevels
}

func (hook LogrusSourceContextHook) Fire(entry *log.Entry) error {
	pc := make([]uintptr, 3, 3)
	cnt := runtime.Callers(6, pc)

	var traces []string
	skipping := true
	for i := 0; i < cnt; i++ {
		fu := runtime.FuncForPC(pc[i] - 1)
		if skipping {
			name := fu.Name()
			if !strings.Contains(name, "sirupsen/logrus") {
				entry.Data["func"] = path.Base(name)
				skipping = false
			}
		}
		if !skipping {
			file, line := fu.FileLine(pc[i] - 1)
			traces = append(traces, fmt.Sprintf("%s:%d", path.Base(file), line))
		}
	}
	if len(traces) > 0 {
		entry.Data["trace"] = traces
	}
	return nil
}
