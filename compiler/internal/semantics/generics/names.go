package generics

import (
	"fmt"
	"strconv"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

const ClosureBindingPrefix = "\x00generic-closure:"

type CanonicalTypeFunc func(string) (string, bool)

func MangleName(base string, order []string, subst map[string]string) string {
	var parts []string
	for _, tp := range order {
		parts = append(parts, tp+"_"+SanitizeType(subst[tp]))
	}
	return base + "__" + strings.Join(parts, "__")
}

func SanitizeType(tname string) string {
	if tname == "" {
		return "unknown"
	}
	var b strings.Builder
	for _, r := range tname {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if r == '_' {
			b.WriteString("__")
			continue
		}
		b.WriteString("_")
		b.WriteString(strconv.FormatInt(int64(r), 16))
		b.WriteString("_")
	}
	return b.String()
}

func UnsanitizeType(tname string) (string, error) {
	var b strings.Builder
	for i := 0; i < len(tname); {
		if tname[i] != '_' {
			b.WriteByte(tname[i])
			i++
			continue
		}
		if i+1 < len(tname) && tname[i+1] == '_' {
			b.WriteByte('_')
			i += 2
			continue
		}
		j := strings.IndexByte(tname[i+1:], '_')
		if j < 0 {
			return "", fmt.Errorf("malformed sanitized generic type %q", tname)
		}
		j += i + 1
		if j >= len(tname) {
			return "", fmt.Errorf("malformed sanitized generic type %q", tname)
		}
		code, err := strconv.ParseInt(tname[i+1:j], 16, 32)
		if err != nil {
			return "", fmt.Errorf("malformed sanitized generic type %q", tname)
		}
		b.WriteRune(rune(code))
		i = j + 1
	}
	return b.String(), nil
}

func TypeName(ref frontend.TypeRef, canonical CanonicalTypeFunc) string {
	switch ref.Kind {
	case frontend.TypeRefSlice:
		if ref.Elem == nil {
			return "[]"
		}
		return "[]" + TypeName(*ref.Elem, canonical)
	case frontend.TypeRefArray:
		if ref.Elem == nil {
			return fmt.Sprintf("[%d]", ref.Len)
		}
		return fmt.Sprintf("[%d]%s", ref.Len, TypeName(*ref.Elem, canonical))
	case frontend.TypeRefOptional:
		if ref.Elem == nil {
			return "?"
		}
		return TypeName(*ref.Elem, canonical) + "?"
	case frontend.TypeRefFunction:
		params := make([]string, 0, len(ref.Params))
		for i, param := range ref.Params {
			formatted := TypeName(param, canonical)
			if i < len(ref.ParamOwnership) && ref.ParamOwnership[i] != "" {
				formatted = ref.ParamOwnership[i] + " " + formatted
			}
			params = append(params, formatted)
		}
		ret := "?"
		if ref.Return != nil {
			ret = TypeName(*ref.Return, canonical)
		}
		out := "fn(" + strings.Join(params, ",") + ")->" + ret
		if ref.Throws != nil {
			out += " throws " + TypeName(*ref.Throws, canonical)
		}
		if len(ref.Uses) > 0 {
			out += " uses " + strings.Join(ref.Uses, ",")
		}
		return out
	default:
		if canonical != nil {
			if canonicalName, ok := canonical(ref.Name); ok {
				return canonicalName
			}
		}
		if len(ref.TypeArgs) > 0 {
			args := make([]string, 0, len(ref.TypeArgs))
			for _, arg := range ref.TypeArgs {
				args = append(args, TypeName(arg, canonical))
			}
			return ref.Name + "<" + strings.Join(args, ",") + ">"
		}
		return ref.Name
	}
}

func ClosureBindingKey(name string) string {
	return ClosureBindingPrefix + name
}
