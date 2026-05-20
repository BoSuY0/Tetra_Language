package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"tetra_language/compiler"
)

type ecoPackageMetadata struct {
	Schema           string                   `json:"schema"`
	Compression      string                   `json:"compression"`
	MTimeUnix        int64                    `json:"mtime_unix"`
	Reproducible     bool                     `json:"reproducible,omitempty"`
	BuildInputsSHA   string                   `json:"build_inputs_sha256,omitempty"`
	ManifestSchema   string                   `json:"manifest_schema,omitempty"`
	PermissionsModel string                   `json:"permissions_model,omitempty"`
	FileCount        int                      `json:"file_count"`
	Files            []ecoPackageMetadataFile `json:"files"`
}

type ecoPackageMetadataFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

func runEcoPack(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra eco pack [--project] [-o PATH] [Capsule.t4]")
		return 0
	}
	capsulePath, outPath, project, err := parseEcoPackArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	manifest, err := parseCapsule(capsulePath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if outPath == "" {
		outPath = manifest.Name + compiler.TodexFragmentExtension
	}
	if project {
		if err := packCapsuleProject(manifest.Path, outPath); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	} else if err := packCapsule(manifest.Path, outPath); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Packed: %s\n", outPath)
	return 0
}

func runEcoUnpack(args []string, stdout io.Writer, stderr io.Writer) int {
	if isHelpArgs(args) {
		fmt.Fprintln(stdout, "usage: tetra eco unpack PACKAGE [-C DIR]")
		return 0
	}
	pkgPath, outDir, err := parseEcoUnpackArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if outDir == "" {
		outDir = "."
	}
	if err := unpackCapsule(pkgPath, outDir); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "Unpacked: %s\n", outDir)
	return 0
}

func packageMetadataFingerprint(files []ecoPackageMetadataFile) string {
	var b strings.Builder
	for _, file := range files {
		b.WriteString(file.Path)
		b.WriteByte('|')
		b.WriteString(file.SHA256)
		b.WriteByte('|')
		b.WriteString(strconv.FormatInt(file.Size, 10))
		b.WriteByte('\n')
	}
	return b.String()
}

func parseEcoPackArgs(args []string) (capsulePath string, outPath string, project bool, err error) {
	capsulePath = defaultCapsulePath()
	sawCapsulePath := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project":
			project = true
		case "-o", "--out":
			i++
			if i >= len(args) {
				return "", "", false, fmt.Errorf("%s requires a value", args[i-1])
			}
			outPath = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", false, fmt.Errorf("unknown option %s", args[i])
			}
			if sawCapsulePath {
				return "", "", false, fmt.Errorf("eco pack accepts one capsule path")
			}
			sawCapsulePath = true
			capsulePath = args[i]
		}
	}
	return capsulePath, outPath, project, nil
}

func parseEcoUnpackArgs(args []string) (pkgPath string, outDir string, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-C", "--dir":
			i++
			if i >= len(args) {
				return "", "", fmt.Errorf("%s requires a value", args[i-1])
			}
			outDir = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", fmt.Errorf("unknown option %s", args[i])
			}
			if pkgPath != "" {
				return "", "", fmt.Errorf("eco unpack accepts one package path")
			}
			pkgPath = args[i]
		}
	}
	if pkgPath == "" {
		return "", "", fmt.Errorf("eco unpack requires a package path")
	}
	return pkgPath, outDir, nil
}

func prepareEcoFileInsideRootNoSymlink(root string, slashRel string, label string) (string, error) {
	if slashRel == "" {
		return "", fmt.Errorf("%s path is required", label)
	}
	if strings.Contains(slashRel, "\\") {
		return "", fmt.Errorf("unsafe %s path %s", label, slashRel)
	}
	relPath := filepath.FromSlash(slashRel)
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("unsafe %s path %s", label, slashRel)
	}
	cleanRel := filepath.Clean(relPath)
	if cleanRel == "." || cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe %s path %s", label, slashRel)
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if err := ensureEcoDirectoryNoSymlink(rootAbs, label+" root"); err != nil {
		return "", err
	}
	parentRel := filepath.Dir(cleanRel)
	current := rootAbs
	if parentRel != "." {
		for _, part := range strings.Split(parentRel, string(os.PathSeparator)) {
			if part == "" || part == "." {
				continue
			}
			current = filepath.Join(current, part)
			info, err := os.Lstat(current)
			if err != nil {
				if !os.IsNotExist(err) {
					return "", err
				}
				if err := os.Mkdir(current, 0o755); err != nil && !os.IsExist(err) {
					return "", err
				}
				info, err = os.Lstat(current)
				if err != nil {
					return "", err
				}
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("%s path crosses symlink %s", label, current)
			}
			if !info.IsDir() {
				return "", fmt.Errorf("%s parent is not a directory: %s", label, current)
			}
		}
	}
	outPath := filepath.Join(rootAbs, cleanRel)
	relCheck, err := filepath.Rel(rootAbs, outPath)
	if err != nil {
		return "", err
	}
	if relCheck == "." || relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe %s path %s", label, slashRel)
	}
	if info, err := os.Lstat(outPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%s path is a symlink: %s", label, outPath)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return outPath, nil
}

func ensureEcoDirectoryNoSymlink(path string, label string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
		info, err = os.Lstat(path)
		if err != nil {
			return err
		}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink: %s", label, path)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory: %s", label, path)
	}
	return nil
}

func writeEcoFileInsideRootNoSymlink(root string, slashRel string, raw []byte, perm os.FileMode, label string) (string, error) {
	path, err := prepareEcoFileInsideRootNoSymlink(root, slashRel, label)
	if err != nil {
		return "", err
	}
	out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return "", err
	}
	if _, err := out.Write(raw); err != nil {
		_ = out.Close()
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	return path, nil
}

func writeEcoJSONFileInsideRootNoSymlink(root string, slashRel string, v interface{}, label string) (string, error) {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	raw = append(raw, '\n')
	return writeEcoFileInsideRootNoSymlink(root, slashRel, raw, 0o644, label)
}

func packCapsule(capsulePath string, outPath string) error {
	return packFiles(filepath.Dir(capsulePath), []string{filepath.Base(capsulePath)}, outPath)
}

func packCapsuleProject(capsulePath string, outPath string) error {
	root := filepath.Dir(capsulePath)
	var relPaths []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".tetra_cache" || name == "tetra_cache" {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if filepath.Clean(path) == filepath.Clean(outPath) || strings.HasSuffix(rel, ".todex") || strings.HasSuffix(rel, compiler.TodexFragmentExtension) {
			return nil
		}
		relPaths = append(relPaths, rel)
		return nil
	}); err != nil {
		return err
	}
	sort.Strings(relPaths)
	return packFiles(root, relPaths, outPath)
}

func packFiles(root string, relPaths []string, outPath string) (err error) {
	zeroTime := time.Unix(0, 0).UTC()
	cleanRelPaths := append([]string(nil), relPaths...)
	sort.Strings(cleanRelPaths)
	for _, rel := range cleanRelPaths {
		if filepath.ToSlash(rel) == ecoPackageMetadataPath {
			return fmt.Errorf("project already contains reserved file %s", ecoPackageMetadataPath)
		}
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); err == nil {
			err = closeErr
		}
	}()
	gz := gzip.NewWriter(out)
	gz.ModTime = zeroTime
	gz.OS = 255
	defer func() {
		if closeErr := gz.Close(); err == nil {
			err = closeErr
		}
	}()
	tw := tar.NewWriter(gz)
	defer func() {
		if closeErr := tw.Close(); err == nil {
			err = closeErr
		}
	}()
	metadata := ecoPackageMetadata{
		Schema:           ecoPackageSchemaV1,
		Compression:      "gzip",
		MTimeUnix:        0,
		Reproducible:     true,
		ManifestSchema:   capsuleManifestSchemaV1,
		PermissionsModel: ecoPermissionsModelV1,
		Files:            make([]ecoPackageMetadataFile, 0, len(cleanRelPaths)),
	}
	for _, rel := range cleanRelPaths {
		if rel == "" || strings.HasPrefix(filepath.Clean(rel), "..") || filepath.IsAbs(rel) {
			return fmt.Errorf("unsafe archive path %q", rel)
		}
		path := filepath.Join(root, rel)
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		info, err := in.Stat()
		if err != nil {
			_ = in.Close()
			return err
		}
		header := &tar.Header{
			Name:       filepath.ToSlash(rel),
			Mode:       0o644,
			Size:       info.Size(),
			Format:     tar.FormatPAX,
			ModTime:    zeroTime,
			AccessTime: zeroTime,
			ChangeTime: zeroTime,
			Uid:        0,
			Gid:        0,
			Uname:      "",
			Gname:      "",
		}
		if err := tw.WriteHeader(header); err != nil {
			_ = in.Close()
			return err
		}
		hash := sha256.New()
		if _, err := io.Copy(tw, io.TeeReader(in, hash)); err != nil {
			_ = in.Close()
			return err
		}
		if err := in.Close(); err != nil {
			return err
		}
		metadata.Files = append(metadata.Files, ecoPackageMetadataFile{
			Path:   filepath.ToSlash(rel),
			SHA256: "sha256:" + hex.EncodeToString(hash.Sum(nil)),
			Size:   info.Size(),
		})
	}
	metadata.FileCount = len(metadata.Files)
	sum := sha256.Sum256([]byte(packageMetadataFingerprint(metadata.Files)))
	metadata.BuildInputsSHA = "sha256:" + hex.EncodeToString(sum[:])
	rawMetadata, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	rawMetadata = append(rawMetadata, '\n')
	if err := tw.WriteHeader(&tar.Header{
		Name:       ecoPackageMetadataPath,
		Mode:       0o644,
		Size:       int64(len(rawMetadata)),
		Format:     tar.FormatPAX,
		ModTime:    zeroTime,
		AccessTime: zeroTime,
		ChangeTime: zeroTime,
		Uid:        0,
		Gid:        0,
		Uname:      "",
		Gname:      "",
	}); err != nil {
		return err
	}
	if _, err := tw.Write(rawMetadata); err != nil {
		return err
	}
	return nil
}

func unpackCapsule(pkgPath string, outDir string) error {
	in, err := os.Open(pkgPath)
	if err != nil {
		return err
	}
	defer in.Close()
	gz, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	entries := map[string][]byte{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return fmt.Errorf("unsupported archive entry type for %s", header.Name)
		}
		name := filepath.Clean(header.Name)
		if name == "." || strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			return fmt.Errorf("unsafe archive path %q", header.Name)
		}
		normalizedName := filepath.ToSlash(name)
		if normalizedName != header.Name {
			return fmt.Errorf("archive path %q is not normalized", header.Name)
		}
		if _, exists := entries[normalizedName]; exists {
			return fmt.Errorf("duplicate archive path %q", header.Name)
		}
		raw, err := io.ReadAll(tr)
		if err != nil {
			return err
		}
		if header.Size >= 0 && int64(len(raw)) != header.Size {
			return fmt.Errorf("archive size mismatch for %s", header.Name)
		}
		entries[normalizedName] = raw
	}
	if err := validateEcoPackageEntries(entries); err != nil {
		return err
	}
	var names []string
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		outPath, err := prepareEcoFileInsideRootNoSymlink(outDir, name, "unpack output")
		if err != nil {
			return err
		}
		out, err := os.OpenFile(outPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		if _, err := out.Write(entries[name]); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
	}
	return nil
}

func validateEcoPackageEntries(entries map[string][]byte) error {
	rawMetadata, ok := entries[ecoPackageMetadataPath]
	if !ok {
		return fmt.Errorf("missing %s", ecoPackageMetadataPath)
	}
	decoder := json.NewDecoder(bytes.NewReader(rawMetadata))
	decoder.DisallowUnknownFields()
	var metadata ecoPackageMetadata
	if err := decoder.Decode(&metadata); err != nil {
		return fmt.Errorf("invalid %s: %w", ecoPackageMetadataPath, err)
	}
	if metadata.Schema != ecoPackageSchemaV1 {
		return fmt.Errorf("unsupported package metadata schema %q", metadata.Schema)
	}
	if metadata.Compression != "gzip" {
		return fmt.Errorf("package metadata compression must be gzip")
	}
	if metadata.MTimeUnix != 0 {
		return fmt.Errorf("package metadata mtime_unix must be 0")
	}
	if metadata.ManifestSchema != "" && metadata.ManifestSchema != capsuleManifestSchemaV1 {
		return fmt.Errorf("unsupported package metadata manifest_schema %q", metadata.ManifestSchema)
	}
	if metadata.PermissionsModel != "" && metadata.PermissionsModel != ecoPermissionsModelV1 {
		return fmt.Errorf("unsupported package metadata permissions_model %q", metadata.PermissionsModel)
	}
	if metadata.FileCount != len(metadata.Files) {
		return fmt.Errorf("package metadata file_count mismatch: expected %d, got %d", len(metadata.Files), metadata.FileCount)
	}
	if metadata.FileCount <= 0 {
		return fmt.Errorf("package metadata file_count must be positive")
	}
	declared := map[string]struct{}{ecoPackageMetadataPath: {}}
	lastPath := ""
	for _, file := range metadata.Files {
		if file.Path == "" {
			return fmt.Errorf("package metadata has empty path")
		}
		cleanPath := filepath.Clean(file.Path)
		if cleanPath == "." || strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
			return fmt.Errorf("package metadata has unsafe path %s", file.Path)
		}
		normalizedPath := filepath.ToSlash(cleanPath)
		if normalizedPath != file.Path {
			return fmt.Errorf("package metadata path %s is not normalized", file.Path)
		}
		if normalizedPath == ecoPackageMetadataPath {
			return fmt.Errorf("package metadata must not self-reference %s", ecoPackageMetadataPath)
		}
		if normalizedPath <= lastPath {
			return fmt.Errorf("package metadata files must be strictly sorted by path")
		}
		lastPath = normalizedPath
		if _, exists := declared[normalizedPath]; exists {
			return fmt.Errorf("package metadata has duplicate file path %s", normalizedPath)
		}
		raw, ok := entries[normalizedPath]
		if !ok {
			return fmt.Errorf("package metadata references missing file %s", normalizedPath)
		}
		if int64(len(raw)) != file.Size {
			return fmt.Errorf("package metadata size mismatch for %s", normalizedPath)
		}
		hashHex, err := ecoPublishPackageHashHex(file.SHA256)
		if err != nil {
			return fmt.Errorf("package metadata %s: %w", normalizedPath, err)
		}
		sum := sha256.Sum256(raw)
		if hex.EncodeToString(sum[:]) != hashHex {
			return fmt.Errorf("package metadata hash mismatch for %s", normalizedPath)
		}
		declared[normalizedPath] = struct{}{}
	}
	if _, ok := declared[compiler.CapsuleFileName]; !ok {
		if _, legacyOK := declared[compiler.LegacyCapsuleFileName]; !legacyOK {
			return fmt.Errorf("package metadata missing %s entry", compiler.CapsuleFileName)
		}
	}
	for name := range entries {
		if _, ok := declared[name]; !ok {
			return fmt.Errorf("archive contains undeclared file %s", name)
		}
	}
	if metadata.BuildInputsSHA != "" {
		hashHex, err := ecoPublishPackageHashHex(metadata.BuildInputsSHA)
		if err != nil {
			return fmt.Errorf("package metadata build_inputs_sha256: %w", err)
		}
		sum := sha256.Sum256([]byte(packageMetadataFingerprint(metadata.Files)))
		if hex.EncodeToString(sum[:]) != hashHex {
			return fmt.Errorf("package metadata build_inputs_sha256 mismatch")
		}
	}
	return nil
}
