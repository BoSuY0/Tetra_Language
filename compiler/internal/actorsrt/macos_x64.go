package actorsrt

import (
	"tetra_language/compiler/internal/backend/x64abi"
	"tetra_language/compiler/internal/format/tobj"
)

func BuildMacOSX64(entries []string) (*tobj.Object, error) {
	abi := x64abi.MacSysV()
	const macMapPrivateAnon = 0x1002
	return buildSysVUnixX64(entries, abi.SysMmap, macMapPrivateAnon)
}
