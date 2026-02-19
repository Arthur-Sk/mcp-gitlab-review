package logger

import (
	"fmt"

	"github.com/thessem/zap-prettyconsole"
	"go.uber.org/zap"
)

func New() (*zap.Logger, error) {
	logger := prettyconsole.NewLogger(zap.DebugLevel)

	if logger == nil {
		return nil, fmt.Errorf("failed to create logger")
	}

	return logger, nil
}
