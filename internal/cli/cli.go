package cli

import (
	"os"
	"tarkov-build-optimiser/internal/helpers"
)

type Flags struct {
	PurgeCache         bool
	UseCache           bool
	CacheOnly          bool
	PurgeOptimumBuilds bool
	TestRun            bool
	UseDatabaseCache   bool
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

	return flags
}
