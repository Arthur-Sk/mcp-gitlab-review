package logger

import (
	"fmt"
	"os"

	"github.com/thessem/zap-prettyconsole"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New() (*zap.Logger, error) {
	encoderConfig := prettyconsole.NewEncoderConfig()
	encoder := prettyconsole.NewEncoder(encoderConfig)

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stderr),
		zap.DebugLevel,
	)

	logger := zap.New(core)
	if logger == nil {
		return nil, fmt.Errorf("failed to create logger")
	}

	return logger, nil
}
