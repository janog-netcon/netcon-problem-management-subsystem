package deploylog

import (
	"bytes"
	"time"

	"github.com/go-logfmt/logfmt"
)

type DeployLogParser struct{}

func (p *DeployLogParser) Parse(data []byte) (*DeployLog, error) {
	reader := bytes.NewReader(data)

	decoder := logfmt.NewDecoder(reader)

	log := &DeployLog{}
	for decoder.ScanRecord() {
		line := &DeployLogRecord{}
		for decoder.ScanKeyval() {
			key, value := string(decoder.Key()), string(decoder.Value())
			switch key {
			case "time":
				timestamp, err := time.Parse(time.RFC3339, value)
				if err != nil {
					return nil, &errInvalidTimestampFormat{raw: value}
				}

				line.timestamp = &timestamp
			case "level":
				level, err := p.parseLogLevel(value)
				if err != nil {
					return nil, err
				}
				line.level = level
			case "msg":
				line.message = value
			default:
				return nil, &errUnexpectedField{key: key}
			}
		}
		log.Record = append(log.Record, line)
	}

	return log, nil
}

func (p *DeployLogParser) parseLogLevel(level string) (LogLevel, error) {
	switch level {
	case "debug":
		return LogLevelDebug, nil
	case "info":
		return LogLevelInfo, nil
	case "warning":
		return LogLevelWarning, nil
	case "panic":
		return LogLevelPanic, nil
	case "fatal":
		return LogLevelFatal, nil
	}
	return 0, &errUnknownLogLevel{raw: level}
}
