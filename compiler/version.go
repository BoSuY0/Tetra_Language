package compiler

import "tetra_language/compiler/internal/version"

func Version() string {
	return version.CompilerVersion
}
