package deploylog

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
)

type DeployLogPrinter interface {
	Print(log *DeployLog) error
	SetLevel(level LogLevel)
}

type PrettyDeployLogPrinter struct {
	writer   io.Writer
	level    LogLevel
	location *time.Location
}

func NewPrettyDeployLogPrinter(writer io.Writer) DeployLogPrinter {
	return &PrettyDeployLogPrinter{
		writer:   writer,
		level:    LogLevelInfo,
		location: time.UTC,
	}
}

func NewPrettyDeployLogPrinterWithLocation(
	writer io.Writer,
	location *time.Location,
) DeployLogPrinter {
	return &PrettyDeployLogPrinter{
		writer:   writer,
		level:    LogLevelInfo,
		location: location,
	}
}

func (p *PrettyDeployLogPrinter) SetLevel(level LogLevel) {
	p.level = level
}

func (p *PrettyDeployLogPrinter) Print(log *DeployLog) error {
	timestampColor := color.New(color.FgBlack).SprintFunc()
	messageColor := color.New().SprintFunc()

	for _, line := range log.Record {
		if line.level < p.level {
			continue
		}

		timestamp := line.timestamp.In(p.location).Format(time.RFC3339)
		level := p.formatLogLevel(line.level)
		message := strings.ReplaceAll(line.message, "\n", "\n"+strings.Repeat(" ", 33))

		fmt.Fprintf(p.writer, "%s  %s %s\n",
			timestampColor(timestamp),
			level,
			messageColor(message),
		)
	}
	return nil
}

func (p *PrettyDeployLogPrinter) formatLogLevel(level LogLevel) string {
	switch level {
	case LogLevelDebug:
		return color.BlueString("DEBUG")
	case LogLevelInfo:
		return color.GreenString(" INFO")
	case LogLevelWarning:
		return color.YellowString(" WARN")
	case LogLevelError:
		return color.RedString("ERROR")
	case LogLevelPanic:
		return color.RedString("PANIC")
	case LogLevelFatal:
		return color.HiRedString("FATAL")
	}

	return "     "

}
