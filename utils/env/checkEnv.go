package env

import (
	osSpecific "github.com/evaxio/arcella-lib/utils/env/os"
)

func IsInIDE() bool {
	return osSpecific.IsInIDE()
}
