//go:build tools
// +build tools

// Placeholder so that go mod tidy doesn't
// remove the genqlient tool from go.mod

package tarkovdev

import _ "github.com/Khan/genqlient/generate"
