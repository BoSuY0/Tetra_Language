package linker

func emitEntryStubSysVLinuxX86() ([]byte, int) {
	stub := []byte{
		0xE8, 0, 0, 0, 0, // call main
		0x89, 0xC3, // mov ebx,eax
		0xB8, 0x01, 0, 0, 0, // mov eax,1 (exit)
		0xCD, 0x80, // int 0x80
	}
	return stub, 1
}
