package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tetra_language/compiler/internal/format/tobj"
	"tetra_language/compiler/internal/ir"
	"tetra_language/compiler/internal/semantics"
	"tetra_language/compiler/internal/version"
)

const compilerCacheABIVersion = "linux-x64-memory-runtime-abi-v4"

func cacheDir(root, target string) string {
	return filepath.Join(root, ".tetra_cache", target)
}

func cacheKey(module, target, buildTag string, srcHash, depSigHash [32]byte) string {
	h := sha256.New()
	h.Write([]byte(module))
	h.Write([]byte{0})
	h.Write([]byte(target))
	h.Write([]byte{0})
	h.Write([]byte(buildTag))
	h.Write([]byte{0})
	h.Write([]byte(version.CompilerVersion))
	h.Write([]byte{0})
	h.Write([]byte(compilerCacheABIVersion))
	h.Write([]byte{0})
	h.Write(srcHash[:])
	h.Write(depSigHash[:])
	return hex.EncodeToString(h.Sum(nil))
}

func cachePath(root, target, buildTag, module string, srcHash, depSigHash [32]byte) string {
	modPath := moduleToCachePath(module)
	key := cacheKey(module, target, buildTag, srcHash, depSigHash)
	return filepath.Join(cacheDir(root, target), modPath, key+".tobj")
}

func moduleToCachePath(module string) string {
	if module == "" {
		return "_root"
	}
	return filepath.FromSlash(strings.ReplaceAll(module, ".", "/"))
}

func LoadCachedObject(root, target, buildTag, module string, srcHash, depSigHash [32]byte) (*tobj.Object, bool, error) {
	path := cachePath(root, target, buildTag, module, srcHash, depSigHash)
	obj, err := tobj.ReadObject(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		// Cache entries can be truncated/corrupted by interrupted writes; treat
		// parse failures as misses so incremental builds can self-heal.
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			return nil, false, err
		}
		_ = os.Remove(path)
		return nil, false, nil
	}
	if obj.Target != target || obj.Module != module {
		return nil, false, nil
	}
	if obj.SrcHash != srcHash || obj.WorldSigHash != depSigHash {
		return nil, false, nil
	}
	return obj, true, nil
}

func StoreCachedObject(root, target, buildTag string, obj *tobj.Object) error {
	if obj == nil {
		return fmt.Errorf("missing object")
	}
	path := cachePath(root, target, buildTag, obj.Module, obj.SrcHash, obj.WorldSigHash)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(dir, ".tmp-*.tobj")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tobj.WriteObject(tmpPath, obj); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func WorldSigHash(checked *semantics.CheckedProgram) [32]byte {
	var sigs []string
	for name, sig := range checked.FuncSigs {
		sigs = append(sigs, formatFuncSig(name, sig))
	}
	sort.Strings(sigs)
	h := sha256.New()
	for _, sig := range sigs {
		h.Write([]byte(sig))
		h.Write([]byte{0})
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

func BuildSigMap(checked *semantics.CheckedProgram) map[string]semantics.FuncSig {
	sigMap := make(map[string]semantics.FuncSig, len(checked.FuncSigs))
	for name, sig := range checked.FuncSigs {
		sigMap[name] = sig
	}
	return sigMap
}

func ModuleDepSigHash(module string, funcs []ir.IRFunc, sigMap map[string]semantics.FuncSig) ([32]byte, error) {
	deps := make(map[string]struct{})
	for _, fn := range funcs {
		for _, instr := range fn.Instrs {
			if instr.Kind != ir.IRCall {
				continue
			}
			targetModule := ModuleOf(instr.Name)
			if targetModule == module {
				continue
			}
			deps[instr.Name] = struct{}{}
		}
	}

	entries := make([]string, 0, len(deps))
	for name := range deps {
		sig, ok := sigMap[name]
		if !ok {
			return [32]byte{}, fmt.Errorf("missing signature for '%s'", name)
		}
		entries = append(entries, formatFuncSig(name, sig))
	}
	sort.Strings(entries)

	h := sha256.New()
	for _, entry := range entries {
		h.Write([]byte(entry))
		h.Write([]byte{0})
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out, nil
}

func BuildTypeSigMap(types map[string]*semantics.TypeInfo) (map[string]string, error) {
	sigs := make(map[string]string, len(types))
	state := make(map[string]int)
	var build func(name string) (string, error)
	build = func(name string) (string, error) {
		if sig, ok := sigs[name]; ok {
			return sig, nil
		}
		if state[name] == 1 {
			return "", fmt.Errorf("recursive type '%s'", name)
		}
		state[name] = 1
		info, ok := types[name]
		if !ok {
			return "", fmt.Errorf("missing type '%s'", name)
		}
		var sig string
		switch info.Kind {
		case semantics.TypeI32:
			sig = "i32"
		case semantics.TypeI64:
			sig = "i64"
		case semantics.TypeU8:
			sig = "u8"
		case semantics.TypeBool:
			sig = "bool"
		case semantics.TypePtr:
			sig = "ptr"
		case semantics.TypeStr:
			sig = "str"
		case semantics.TypeSlice:
			elemSig, err := build(info.ElemType)
			if err != nil {
				return "", err
			}
			sig = "[]" + elemSig
		case semantics.TypeArray:
			elemSig, err := build(info.ElemType)
			if err != nil {
				return "", err
			}
			sig = fmt.Sprintf("[%d]%s", info.ArrayLen, elemSig)
		case semantics.TypeStruct:
			parts := make([]string, 0, len(info.Fields))
			for _, field := range info.Fields {
				fieldSig, err := build(field.TypeName)
				if err != nil {
					return "", err
				}
				parts = append(parts, field.Name+":"+fieldSig)
			}
			sig = "struct{" + strings.Join(parts, ",") + "}"
		case semantics.TypeIsland:
			sig = "island"
		case semantics.TypeCap:
			sig = info.Name
		case semantics.TypeActor:
			sig = "actor"
		case semantics.TypeEnum:
			parts := make([]string, 0, len(info.EnumCases))
			for _, c := range info.EnumCases {
				parts = append(parts, c.Name+"("+strings.Join(c.PayloadTypes, ",")+")")
			}
			sig = "enum{" + strings.Join(parts, ",") + "}"
		case semantics.TypeOptional:
			elemSig, err := build(info.ElemType)
			if err != nil {
				return "", err
			}
			sig = "optional{" + elemSig + "}"
		default:
			return "", fmt.Errorf("unknown type kind")
		}
		sigs[name] = sig
		state[name] = 2
		return sig, nil
	}
	for name := range types {
		if _, err := build(name); err != nil {
			return nil, err
		}
	}
	return sigs, nil
}

func DepSigHashFromDeps(
	callees []string,
	typeDeps []string,
	sigMap map[string]semantics.FuncSig,
	typeSigMap map[string]string,
) ([32]byte, error) {
	return DepSigHashFromDepsWithInterfaceHashes(callees, typeDeps, sigMap, typeSigMap, nil)
}

func DepSigHashFromDepsWithInterfaceHashes(
	callees []string,
	typeDeps []string,
	sigMap map[string]semantics.FuncSig,
	typeSigMap map[string]string,
	interfaceHashes map[string]string,
) ([32]byte, error) {
	entries := make([]string, 0, len(callees)+len(typeDeps))
	interfaceModules := map[string]struct{}{}
	for _, name := range callees {
		sig, ok := sigMap[name]
		if !ok {
			return [32]byte{}, fmt.Errorf("missing signature for '%s'", name)
		}
		entries = append(entries, formatFuncSig(name, sig))
		if mod := ModuleOf(name); mod != "" {
			interfaceModules[mod] = struct{}{}
		}
	}
	for _, name := range typeDeps {
		sig, ok := typeSigMap[name]
		if !ok {
			return [32]byte{}, fmt.Errorf("missing type signature for '%s'", name)
		}
		entries = append(entries, formatTypeSig(name, sig))
		if mod := ModuleOf(name); mod != "" {
			interfaceModules[mod] = struct{}{}
		}
	}
	for module := range interfaceModules {
		if hash := interfaceHashes[module]; hash != "" {
			entries = append(entries, "interface:"+module+"="+hash)
		}
	}
	sort.Strings(entries)

	h := sha256.New()
	for _, entry := range entries {
		h.Write([]byte(entry))
		h.Write([]byte{0})
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out, nil
}

func ModuleOf(fullName string) string {
	idx := strings.LastIndex(fullName, ".")
	if idx == -1 {
		return ""
	}
	return fullName[:idx]
}

func formatFuncSig(name string, sig semantics.FuncSig) string {
	params := make([]string, len(sig.ParamTypes))
	for i, typ := range sig.ParamTypes {
		ownership := ""
		if i < len(sig.ParamOwnership) {
			ownership = sig.ParamOwnership[i]
		}
		if ownership != "" {
			params[i] = ownership + " " + typ
		} else {
			params[i] = typ
		}
	}
	prefix := "func"
	if sig.Async {
		prefix = "async func"
	}
	if sig.Generic {
		prefix = "generic " + prefix
	}
	throws := ""
	if sig.ThrowsType != "" {
		throws = " throws " + sig.ThrowsType
	}
	return fmt.Sprintf("%s:%s(%s)->%s%s uses %s", prefix, name, strings.Join(params, ","), sig.ReturnType, throws, strings.Join(sig.Effects, ","))
}

func formatTypeSig(name, sig string) string {
	return "type:" + name + "=" + sig
}
