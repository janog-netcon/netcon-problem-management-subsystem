package deploylog

import "fmt"

type errInvalidTimestampFormat struct {
	raw string
}

func (e *errInvalidTimestampFormat) Error() string {
	return fmt.Sprintf("invalid timestamp format: %s", e.raw)
}

type errUnexpectedField struct {
	key string
}

func (e *errUnexpectedField) Error() string {
	return fmt.Sprintf("unexpected field: %s", e.key)
}

type errUnknownLogLevel struct {
	raw string
}

func (e *errUnknownLogLevel) Error() string {
	return fmt.Sprintf("unknown log level: %s", e.raw)
}
