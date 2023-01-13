package deploylog

import "fmt"

type errUnknownLogLevel struct {
	raw string
}

func (e *errUnknownLogLevel) Error() string {
	return fmt.Sprintf("unknown log level: %s", e.raw)
}
