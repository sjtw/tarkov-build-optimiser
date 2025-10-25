package cli

import (
	"os"
	"strings"
	"tarkov-build-optimiser/internal/helpers"

	"github.com/rs/zerolog"
)

type Flags struct {
	PurgeCache         bool
	UseCache           bool
	CacheOnly          bool
	PurgeOptimumBuilds bool
	TestRun            bool
	UseDatabaseCache   bool
	LogLevel           string
}

func GetFlags() Flags {
	flags := Flags{}
	if helpers.ContainsStr(os.Args, "--purge-cache") {
		flags.PurgeCache = true
	}
	if helpers.ContainsStr(os.Args, "--use-cache") {
		flags.UseCache = true
	}
	if helpers.ContainsStr(os.Args, "--cache-only") {
		flags.CacheOnly = true
	}
	if helpers.ContainsStr(os.Args, "--purge-optimum-builds") {
		flags.PurgeOptimumBuilds = true
	}
	if helpers.ContainsStr(os.Args, "--test-run") {
		// makes evaluator only use a few weapons for testing
		flags.TestRun = true
	}
	if helpers.ContainsStr(os.Args, "--use-database-cache") {
		flags.UseDatabaseCache = true
	}

	// Parse log level from --log-level flag
	flags.LogLevel = parseLogLevel()

	return flags
}

// parseLogLevel extracts the log level from command line arguments
// Supports: --log-level=debug, --log-level=info, --log-level=warn, --log-level=error
// Defaults to "info" if not specified or invalid
func parseLogLevel() string {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "--log-level=") {
			level := strings.TrimPrefix(arg, "--log-level=")
			// Validate the level
			switch strings.ToLower(level) {
			case "trace", "debug", "info", "warn", "error", "fatal", "panic":
				return strings.ToLower(level)
			}
		}
	}
	return "info" // default
}

// SetLogLevel configures zerolog with the specified log level
func SetLogLevel(level string) {
	switch strings.ToLower(level) {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel) // fallback to info
	}
}
