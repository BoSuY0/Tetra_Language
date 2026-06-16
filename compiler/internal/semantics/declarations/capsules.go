package declarations

import (
	"fmt"
	"strings"

	"tetra_language/compiler/internal/frontend"
)

func ValidateCapsuleDecls(file *frontend.FileAST) error {
	if file == nil {
		return nil
	}
	for _, capsule := range file.Capsules {
		if capsule == nil {
			continue
		}
		seen := make(map[string]struct{}, len(capsule.Entries))
		for _, entry := range capsule.Entries {
			if _, exists := seen[entry.Key]; exists {
				return fmt.Errorf("%s: duplicate capsule metadata key '%s'", frontend.FormatPos(entry.At), entry.Key)
			}
			seen[entry.Key] = struct{}{}
			if !IsCapsuleMetadataKey(entry.Key) {
				return fmt.Errorf("%s: invalid capsule metadata key '%s'", frontend.FormatPos(entry.At), entry.Key)
			}
			if !IsCapsuleMetadataLiteral(entry.Value) {
				return fmt.Errorf("%s: capsule metadata value for key '%s' must be a literal (string/number/bool)", frontend.FormatPos(entry.At), entry.Key)
			}
		}
	}
	return nil
}

func IsCapsuleMetadataLiteral(expr frontend.Expr) bool {
	switch expr.(type) {
	case *frontend.StringLitExpr, *frontend.NumberExpr, *frontend.BoolLitExpr:
		return true
	default:
		return false
	}
}

func IsCapsuleMetadataKey(key string) bool {
	if key == "" {
		return false
	}
	parts := strings.Split(key, ".")
	for _, part := range parts {
		if !IsCapsuleKeySegment(part) {
			return false
		}
	}
	return true
}

func IsCapsuleKeySegment(seg string) bool {
	if seg == "" {
		return false
	}
	for i, r := range seg {
		switch {
		case i == 0 && r >= 'a' && r <= 'z':
			continue
		case i > 0 && r >= 'a' && r <= 'z':
			continue
		case i > 0 && r >= '0' && r <= '9':
			continue
		case i > 0 && r == '_':
			continue
		default:
			return false
		}
	}
	return true
}
