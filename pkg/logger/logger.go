package logger

import (
	"log"
	"os"

	"golang.org/x/exp/slog"

	"github.com/yasuyuki0321/psh/pkg/aws"
	"github.com/yasuyuki0321/psh/pkg/utils"
)

var logger *slog.Logger

func init() {
	logFile, err := os.OpenFile(utils.GetHomePath("~/.psh_hisotry"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}

	ops := slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger = slog.New(slog.NewJSONHandler(logFile, &ops))
	slog.SetDefault(logger)
}

func LogCommandExecution(target aws.InstanceInfo, command string, err error) {
	if err != nil {
		logger.Info(
			"Failed executing SSH command",
			"IP", target.IP,
			"Name", target.Name,
			"Command", command,
			"Error", err.Error(),
		)
	} else {
		logger.Info(
			"Successfully executed SSH command",
			"IP", target.IP,
			"Name", target.Name,
			"Command", command,
		)
	}
}
