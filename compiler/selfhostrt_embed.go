package compiler

import _ "embed"

// Embedded self-host runtime sources.
//
// These are compiled into TOBJ objects on demand and linked when actors are used.

//go:embed selfhostrt/actors_poc_sysv.tetra
var embeddedActorsPocSysV []byte

//go:embed selfhostrt/actors_poc_win64.tetra
var embeddedActorsPocWin64 []byte
