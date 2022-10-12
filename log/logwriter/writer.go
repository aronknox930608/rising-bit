package logwriter

import (
	"github.com/bitrise-io/bitrise/log"
)

// LogWriter ...
type LogWriter struct {
	logger log.Logger
}

// NewLogWriter ...
func NewLogWriter(logger log.Logger) LogWriter {
	return LogWriter{
		logger: logger,
	}
}

func (w LogWriter) Write(p []byte) (n int, err error) {
	level, message := convertColoredString(string(p))
	w.logger.LogMessage(message, level)
	return len(p), nil
}
