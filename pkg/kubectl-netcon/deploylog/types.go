package deploylog

import "time"

type DeployLog struct {
	Record []*DeployLogRecord
}

type DeployLogRecord struct {
	timestamp *time.Time
	level     LogLevel
	message   string
}

type LogLevel int

const (
	LogLevelDebug = iota + 1
	LogLevelInfo
	LogLevelWarning
	LogLevelPanic
	LogLevelFatal
)
