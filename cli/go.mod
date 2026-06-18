module tetra_language/cli

go 1.20

require (
	tetra_language/compiler v0.0.0
	tetra_language/tools v0.0.0
)

replace tetra_language/compiler => ../compiler

replace tetra_language/tools => ../tools
