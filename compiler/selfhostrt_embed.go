package compiler

import _ "embed"

// Embedded self-host runtime sources.
//
// These are compiled into TOBJ objects on demand and linked when actors are used.

//go:embed selfhostrt/actors_sysv.tetra
var embeddedActorsSysV []byte

//go:embed selfhostrt/actors_win64.tetra
var embeddedActorsWin64 []byte
