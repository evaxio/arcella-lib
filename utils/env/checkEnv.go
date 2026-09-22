package env

import (
	osSpecific "axgit.vixiv.ru/snake/arcella-lib/utils/env/os"
)

func IsInIDE() bool {
	return osSpecific.IsInIDE()
}
