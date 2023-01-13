package deploylog

import (
	"bytes"
	"time"

	"github.com/kr/logfmt"
)

type DeployLogParser struct{}

func (p *DeployLogParser) tryParseLogLine(data []byte) (*DeployLogRecord, error) {
	var line struct {
		Time  string
		Level string
		Msg   string
	}

	if err := logfmt.Unmarshal(data, &line); err != nil {
		return nil, err
	}

	timestamp, err := time.Parse(time.RFC3339, line.Time)
	if err != nil {
		return nil, err
	}

	return &DeployLogRecord{
		timestamp: &timestamp,
		level:     p.parseLogLevel(line.Level),
		message:   line.Msg,
	}, nil
}

func (p *DeployLogParser) Parse(data []byte) (*DeployLog, error) {
	lines := bytes.Split(data, []byte("\n"))

	log := &DeployLog{}
	for _, line := range lines {
		record, err := p.tryParseLogLine(line)
		if err != nil {
			record = &DeployLogRecord{
				timestamp: nil,
				level:     LogLevelUnknown,
				message:   string(line),
			}
		}

		log.Record = append(log.Record, record)
	}

	return log, nil
}

func (p *DeployLogParser) parseLogLevel(level string) LogLevel {
	switch level {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warning":
		return LogLevelWarning
	case "error":
		return LogLevelError
	case "panic":
		return LogLevelPanic
	case "fatal":
		return LogLevelFatal
	}
	return LogLevelUnknown
}
