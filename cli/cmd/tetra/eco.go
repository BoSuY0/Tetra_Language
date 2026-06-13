package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"tetra_language/compiler"
	ctarget "tetra_language/compiler/target"
	"tetra_language/internal/outputformat"
)

const (
	capsuleManifestSchemaV1  = "tetra.capsule.v1"
	ecoPermissionsModelV1    = "tetra.eco.permissions.v1"
	ecoLockSchemaV1          = "tetra.eco.lock.v1"
	ecoSeedSchemaV1          = "tetra.eco.seed.v1"
	ecoNeedMapSchemaV1       = "tetra.eco.needmap.v1"
	ecoTrustSnapshotSchemaV1 = "tetra.eco.trust-snapshot.v1"
	ecoMaterializerSchemaV1  = "tetra.eco.materialization.v1"
	ecoPublishSchemaV1       = "tetra.eco.publish.v1"
	ecoPublishSchemaV1Beta   = "tetra.eco.publish.v1beta"
	ecoMirrorSchemaV1        = "tetra.eco.mirror.v1"
	ecoPackageSchemaV1       = "tetra.eco.package.v1"
	ecoPackageMetadataPath   = "tetra.package.json"
)

func runEco(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: tetra eco <verify|artifacts|pack|unpack|vault|seed|needmap|trust|materialize|publish|download|tetrahub> [options]")
		return 2
	}
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra eco <verify|artifacts|pack|unpack|vault|seed|needmap|trust|materialize|publish|download|tetrahub> [options]")
		fmt.Fprintln(stdout, "lock generation/validation is available through workflows such as: tetra eco verify --lock <path> <Capsule.t4>")
		return 0
	}
	switch args[0] {
	case "verify":
		return runEcoVerify(args[1:], stdout, stderr)
	case "artifacts":
		return runEcoArtifacts(args[1:], stdout, stderr)
	case "pack":
		return runEcoPack(args[1:], stdout, stderr)
	case "unpack":
		return runEcoUnpack(args[1:], stdout, stderr)
	case "vault":
		return runEcoVault(args[1:], stdout, stderr)
	case "seed":
		return runEcoSeed(args[1:], stdout, stderr)
	case "needmap":
		return runEcoNeedMap(args[1:], stdout, stderr)
	case "trust":
		return runEcoTrust(args[1:], stdout, stderr)
	case "materialize":
		return runEcoMaterialize(args[1:], stdout, stderr)
	case "publish":
		return runEcoPublish(args[1:], stdout, stderr)
	case "download":
		return runEcoDownload(args[1:], stdout, stderr)
	case "tetrahub":
		return runEcoTetraHub(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown eco command %q\n", args[0])
		return 2
	}
}

func runEcoVerify(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("eco verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	target := fs.String("target", "", "validate capsule target compatibility")
	lockPath := fs.String("lock", "", "write dependency lock/provenance JSON")
	lockFormat := fs.String("lock-format", outputformat.JSON, "lock output format: json, toon, or both")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	manifests, err := parseCapsuleGraphArgs(fs.Args())
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := validateCapsuleGraph(manifests, *target); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if *lockPath != "" {
		if _, err := writeEcoLockFormatted(*lockPath, *lockFormat, manifests); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if len(manifests) == 1 {
		manifest := manifests[0]
		fmt.Fprintf(stdout, "Capsule OK: %s %s (%s)\n", manifest.Name, manifest.Version, manifest.ID)
		return 0
	}
	fmt.Fprintf(stdout, "Capsule graph OK: %d capsules\n", len(manifests))
	return 0
}

func writeJSONFile(path string, v interface{}) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func writeEcoStructuredFile(path string, format string, v interface{}) ([]string, error) {
	return outputformat.WriteStructuredFiles(path, format, v)
}

func decodeEcoStructured(raw []byte, format string, out interface{}) error {
	return outputformat.DecodeStructured(raw, format, out)
}

func decodeEcoStructuredStrict(raw []byte, format string, out interface{}) error {
	return outputformat.DecodeStructuredStrict(raw, format, out)
}

func defaultCapsulePath() string {
	if fileExists(compiler.CapsuleFileName) {
		return compiler.CapsuleFileName
	}
	if fileExists(compiler.LegacyCapsuleFileName) {
		return compiler.LegacyCapsuleFileName
	}
	return compiler.CapsuleFileName
}

func findCapsulePath(dir string) (string, error) {
	for _, name := range []string{compiler.CapsuleFileName, compiler.LegacyCapsuleFileName} {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return "", fmt.Errorf("missing %s (or legacy %s)", compiler.CapsuleFileName, compiler.LegacyCapsuleFileName)
}

func dedupeStrings(values []string) []string {
	if len(values) <= 1 {
		return values
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func appendUniqueString(values []string, value string) []string {
	if containsString(values, value) {
		return values
	}
	return append(values, value)
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func isSupportedCapsuleTarget(target string) bool {
	for _, triple := range ctarget.SupportedTriples() {
		if triple == target {
			return true
		}
	}
	for _, triple := range ctarget.BuildOnlyTriples() {
		if triple == target {
			return true
		}
	}
	return false
}
