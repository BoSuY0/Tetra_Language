package compiler

import "tetra_language/compiler/internal/abisuite"

func checkX86StdoutExecutableSmoke() error {
	return abisuite.CheckX86StdoutExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32StdoutExecutableSmoke() error {
	return abisuite.CheckX32StdoutExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86StderrFDRuntimeSmoke() error {
	return abisuite.CheckX86StderrFDRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32StderrFDRuntimeSmoke() error {
	return abisuite.CheckX32StderrFDRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86AllocatorExecutableSmoke() error {
	return abisuite.CheckX86AllocatorExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86AllocatorFailureExecutableSmoke() error {
	return abisuite.CheckX86AllocatorFailureExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32AllocatorExecutableSmoke() error {
	return abisuite.CheckX32AllocatorExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32AllocatorFailureExecutableSmoke() error {
	return abisuite.CheckX32AllocatorFailureExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86RawMemoryBoundsExecutableSmoke() error {
	return abisuite.CheckX86RawMemoryBoundsExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32RawMemoryBoundsExecutableSmoke() error {
	return abisuite.CheckX32RawMemoryBoundsExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86RawPointerSlotExecutableSmoke() error {
	return abisuite.CheckX86RawPointerSlotExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32RawPointerSlotExecutableSmoke() error {
	return abisuite.CheckX32RawPointerSlotExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86RawPointerOffsetSlotExecutableSmoke() error {
	return abisuite.CheckX86RawPointerOffsetSlotExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32RawPointerOffsetSlotExecutableSmoke() error {
	return abisuite.CheckX32RawPointerOffsetSlotExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86IslandFreeExecutableSmoke() error {
	return abisuite.CheckX86IslandFreeExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32IslandFreeExecutableSmoke() error {
	return abisuite.CheckX32IslandFreeExecutableSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX86NetworkingLifecycleRuntimeSmoke() error {
	return abisuite.CheckX86NetworkingLifecycleRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}

func checkX32NetworkingLifecycleRuntimeSmoke() error {
	return abisuite.CheckX32NetworkingLifecycleRuntimeSmoke(abiSuiteRuntimeSmokeDeps())
}
