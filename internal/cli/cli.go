package cli

import (
	"os"
	"tarkov-build-optimiser/internal/helpers"
)

type Flags struct {
	PurgeCache bool
	UseCache   bool
	CacheOnly  bool
}

func constructFlags() Flags {
	return Flags{
		PurgeCache: false,
		UseCache:   false,
		CacheOnly:  false,
	}
}

func GetFlags() Flags {
	flags := constructFlags()
	if HasFlag("--purge-cache") {
		flags.PurgeCache = true
	}
	if HasFlag("--use-cache") {
		flags.UseCache = true
	}
	if HasFlag("--cache-only") {
		flags.CacheOnly = true
	}

	return flags
}

func HasFlag(flag string) bool {
	if helpers.ContainsStr(os.Args, flag) {
		return true
	}

	return false
}
