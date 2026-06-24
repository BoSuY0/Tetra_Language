package resources

import (
	"fmt"
	"strconv"
	"strings"
)

type Path string

func Root(name string) (Path, error) {
	components, err := parsePathComponents(name, false)
	if err != nil {
		return "", err
	}
	if len(components) != 1 {
		return "", fmt.Errorf("ownership path root %q must be a single segment", name)
	}
	return Path(name), nil
}

func ParsePath(raw string) (Path, error) {
	if _, err := parsePathComponents(raw, false); err != nil {
		return "", err
	}
	return Path(raw), nil
}

func (p Path) String() string {
	return string(p)
}

func (p Path) Field(name string) Path {
	if p == "" {
		return Path(name)
	}
	return Path(string(p) + "." + name)
}

func (p Path) EnumPayload(ordinal int32, index int) Path {
	return p.Field(fmt.Sprintf("$case%d", ordinal)).Field(fmt.Sprintf("payload%d", index))
}

func (p Path) Element() Path {
	return p.Field("$elem")
}

func (p Path) Parent() (Path, bool) {
	components, err := parsePathComponents(string(p), true)
	if err != nil || len(components) <= 1 {
		return "", false
	}
	return pathFromComponents(components[:len(components)-1]), true
}

func (p Path) IsAncestorOf(other Path) bool {
	left, right, ok := parsedPathPair(p, other)
	if !ok || len(left) >= len(right) {
		return false
	}
	return componentsPrefix(left, right)
}

func (p Path) IsDescendantOf(other Path) bool {
	return other.IsAncestorOf(p)
}

func (p Path) Aliases(other Path) bool {
	if p == other {
		return true
	}
	return p.IsAncestorOf(other) || p.IsDescendantOf(other)
}

func (p Path) RelativeTo(root Path) (Path, bool) {
	if root == "" {
		return p, true
	}
	pathComponents, rootComponents, ok := parsedPathPair(p, root)
	if !ok || len(rootComponents) > len(pathComponents) ||
		!componentsPrefix(rootComponents, pathComponents) {
		return "", false
	}
	if len(rootComponents) == len(pathComponents) {
		return "", true
	}
	return pathFromComponents(pathComponents[len(rootComponents):]), true
}

func JoinPath(root Path, leaf Path) Path {
	if root == "" {
		return leaf
	}
	if leaf == "" {
		return root
	}
	return Path(string(root) + "." + string(leaf))
}

func parsedPathPair(left Path, right Path) ([]string, []string, bool) {
	leftComponents, err := parsePathComponents(string(left), true)
	if err != nil {
		return nil, nil, false
	}
	rightComponents, err := parsePathComponents(string(right), true)
	if err != nil {
		return nil, nil, false
	}
	return leftComponents, rightComponents, true
}

func componentsPrefix(prefix []string, path []string) bool {
	if len(prefix) == 0 || len(prefix) > len(path) {
		return false
	}
	for i := range prefix {
		if !pathComponentsMatch(prefix[i], path[i]) {
			return false
		}
	}
	return true
}

func pathComponentsMatch(left string, right string) bool {
	return left == right || left == "[_]" || right == "[_]"
}

func pathFromComponents(components []string) Path {
	return Path(strings.Join(components, "."))
}

func parsePathComponents(raw string, allowSyntheticRoot bool) ([]string, error) {
	if raw == "" {
		return nil, fmt.Errorf("ownership path is empty")
	}
	parts := splitPathParts(raw)
	components := make([]string, 0, len(parts))
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if part == "" {
			return nil, fmt.Errorf("ownership path %q contains an empty segment", raw)
		}
		if i == 0 && strings.HasPrefix(part, "$") && !allowSyntheticRoot {
			return nil, fmt.Errorf("ownership path root %q must not be synthetic", raw)
		}
		switch {
		case part == "$elem":
			components = append(components, part)
		case strings.HasPrefix(part, "$case"):
			ordinal, ok := parseSyntheticIndex(part, "$case")
			if !ok {
				return nil, fmt.Errorf("ownership path %q has malformed enum case segment %q", raw, part)
			}
			if i+1 >= len(parts) {
				return nil, fmt.Errorf("ownership path %q enum case segment lacks payload", raw)
			}
			payload := parts[i+1]
			if _, ok := parseSyntheticIndex(payload, "payload"); !ok {
				return nil, fmt.Errorf("ownership path %q has malformed enum payload segment %q", raw, payload)
			}
			components = append(components, fmt.Sprintf("$case%d.%s", ordinal, payload))
			i++
		case strings.HasPrefix(part, "$"):
			return nil, fmt.Errorf("ownership path %q has malformed synthetic segment %q", raw, part)
		default:
			components = append(components, part)
		}
	}
	return components, nil
}

func splitPathParts(raw string) []string {
	parts := make([]string, 0, 4)
	start := 0
	depth := 0
	for i := 0; i <= len(raw); i++ {
		if i == len(raw) || (raw[i] == '.' && depth == 0) {
			parts = append(parts, raw[start:i])
			start = i + 1
			continue
		}
		switch raw[i] {
		case '[':
			depth++
		case ']':
			if depth > 0 {
				depth--
			}
		}
	}
	return parts
}

func parseSyntheticIndex(segment string, prefix string) (int, bool) {
	if !strings.HasPrefix(segment, prefix) {
		return 0, false
	}
	raw := strings.TrimPrefix(segment, prefix)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, false
	}
	return value, true
}
