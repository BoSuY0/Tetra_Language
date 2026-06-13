package surface

import (
	"fmt"
	"strings"
)

type BlockAssetManifestReport struct {
	Schema        string             `json:"schema"`
	Source        string             `json:"source"`
	Quality       string             `json:"quality"`
	HashAlgorithm string             `json:"hash_algorithm"`
	ManifestHash  string             `json:"manifest_hash"`
	LocalOnly     bool               `json:"local_only"`
	FontCount     int                `json:"font_count"`
	IconCount     int                `json:"icon_count"`
	ImageCount    int                `json:"image_count"`
	EmbeddedCount int                `json:"embedded_count"`
	RemoteCount   int                `json:"remote_count"`
	Assets        []BlockAssetReport `json:"assets"`
}

type BlockAssetReport struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Path     string `json:"path"`
	Embedded bool   `json:"embedded"`
	Local    bool   `json:"local"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Family   string `json:"family,omitempty"`
	CacheKey string `json:"cache_key"`
}

type BlockAssetCacheReport struct {
	ID            string `json:"id"`
	Strategy      string `json:"strategy"`
	BudgetBytes   int    `json:"budget_bytes"`
	UsedBytes     int    `json:"used_bytes"`
	EntryCount    int    `json:"entry_count"`
	MaxEntries    int    `json:"max_entries"`
	RepeatedLoads int    `json:"repeated_loads"`
	Eviction      string `json:"eviction"`
	Bounded       bool   `json:"bounded"`
}

type BlockAssetDiagnosticReport struct {
	Order       int    `json:"order"`
	AssetID     string `json:"asset_id"`
	Kind        string `json:"kind"`
	Code        string `json:"code"`
	Message     string `json:"message"`
	FallbackID  string `json:"fallback_id,omitempty"`
	RejectedURL string `json:"rejected_url,omitempty"`
	Pass        bool   `json:"pass"`
}

type BlockAssetRenderCommandReport struct {
	Order    int        `json:"order"`
	Command  string     `json:"command"`
	AssetID  string     `json:"asset_id"`
	BlockID  int        `json:"block_id"`
	Rect     RectReport `json:"rect"`
	Tint     string     `json:"tint,omitempty"`
	Scale    int        `json:"scale,omitempty"`
	Quality  string     `json:"quality"`
	Checksum string     `json:"checksum"`
}

func validateBlockAssetEvidence(report Report) []string {
	if !hasBlockAssetEvidence(report) {
		return nil
	}

	var issues []string
	if report.BlockAssetQualityLevel != "deterministic-local-block-assets-v1" {
		issues = append(issues, fmt.Sprintf("block_asset_quality_level is %q, want deterministic-local-block-assets-v1", report.BlockAssetQualityLevel))
	}
	if report.BlockAssetNetworkFetchAllowed {
		issues = append(issues, "block asset network fetch must be disabled")
	}

	assetIDs := map[string]BlockAssetReport{}
	if report.BlockAssetManifest == nil {
		issues = append(issues, "block_asset_manifest evidence is required")
	} else {
		manifest := report.BlockAssetManifest
		if manifest.Schema != "tetra.surface.block-assets.v1" {
			issues = append(issues, fmt.Sprintf("block_asset_manifest schema is %q, want tetra.surface.block-assets.v1", manifest.Schema))
		}
		if manifest.Quality != "deterministic-local-block-assets-v1" {
			issues = append(issues, fmt.Sprintf("block_asset_manifest quality is %q, want deterministic-local-block-assets-v1", manifest.Quality))
		}
		if manifest.HashAlgorithm != "sha256" {
			issues = append(issues, fmt.Sprintf("block_asset_manifest hash_algorithm is %q, want sha256", manifest.HashAlgorithm))
		}
		if !validSHA256Digest(manifest.ManifestHash) {
			issues = append(issues, "block_asset_manifest manifest_hash must be sha256 evidence")
		}
		if strings.TrimSpace(manifest.Source) == "" || normalizeEvidencePath(manifest.Source) != normalizeEvidencePath(report.Source) {
			issues = append(issues, "block_asset_manifest source must match report source")
		}
		if !manifest.LocalOnly || manifest.RemoteCount != 0 {
			issues = append(issues, "block_asset_manifest must be local-only with remote_count 0")
		}
		if manifest.FontCount <= 0 || manifest.IconCount <= 0 || manifest.ImageCount <= 0 {
			issues = append(issues, "block_asset_manifest requires font, icon, and image counts")
		}
		if manifest.EmbeddedCount <= 0 {
			issues = append(issues, "block_asset_manifest requires embedded/local sample asset evidence")
		}
		if len(manifest.Assets) < 3 {
			issues = append(issues, "block_asset_manifest assets require font/icon/image asset hashes")
		}
		kindCounts := map[string]int{}
		for i, asset := range manifest.Assets {
			id := strings.TrimSpace(asset.ID)
			kind := normalizeStateToken(asset.Kind)
			if id == "" {
				issues = append(issues, fmt.Sprintf("block_asset_manifest assets[%d] id is required", i))
			} else if _, exists := assetIDs[id]; exists {
				issues = append(issues, fmt.Sprintf("block_asset_manifest duplicate asset id %q", id))
			}
			assetIDs[id] = asset
			if !validBlockAssetKind(kind) {
				issues = append(issues, fmt.Sprintf("block_asset_manifest asset %q kind is %q, want font, icon, or image", id, asset.Kind))
			}
			kindCounts[kind]++
			if strings.TrimSpace(asset.Path) == "" {
				issues = append(issues, fmt.Sprintf("block_asset_manifest asset %q path is required", id))
			}
			if isNetworkAssetPath(asset.Path) {
				issues = append(issues, fmt.Sprintf("block_asset_manifest asset %q uses network path %q", id, asset.Path))
			}
			if !asset.Local && !asset.Embedded {
				issues = append(issues, fmt.Sprintf("block_asset_manifest asset %q must be local or embedded", id))
			}
			if !validSHA256Digest(asset.SHA256) {
				issues = append(issues, fmt.Sprintf("block_asset_manifest asset %q sha256 must be present", id))
			}
			if asset.Size <= 0 {
				issues = append(issues, fmt.Sprintf("block_asset_manifest asset %q size must be positive", id))
			}
			if strings.TrimSpace(asset.CacheKey) == "" {
				issues = append(issues, fmt.Sprintf("block_asset_manifest asset %q cache_key is required", id))
			}
			if kind == "font" && strings.TrimSpace(asset.Family) == "" {
				issues = append(issues, fmt.Sprintf("block_asset_manifest font asset %q family is required", id))
			}
			if (kind == "icon" || kind == "image") && (asset.Width <= 0 || asset.Height <= 0) {
				issues = append(issues, fmt.Sprintf("block_asset_manifest %s asset %q width/height must be positive", kind, id))
			}
		}
		if kindCounts["font"] < manifest.FontCount || kindCounts["icon"] < manifest.IconCount || kindCounts["image"] < manifest.ImageCount {
			issues = append(issues, "block_asset_manifest counts must be backed by matching asset entries")
		}
	}

	cache := report.BlockAssetCache
	if strings.TrimSpace(cache.ID) == "" {
		issues = append(issues, "block_asset_cache evidence is required")
	}
	if normalizeStateToken(cache.Strategy) != "bounded_lru" {
		issues = append(issues, fmt.Sprintf("block_asset_cache strategy is %q, want bounded-lru", cache.Strategy))
	}
	if !cache.Bounded {
		issues = append(issues, "block_asset_cache must be bounded")
	}
	if cache.BudgetBytes <= 0 || cache.BudgetBytes > 1<<20 {
		issues = append(issues, fmt.Sprintf("block_asset_cache budget_bytes = %d, want 1..1048576", cache.BudgetBytes))
	}
	if cache.UsedBytes < 0 || (cache.BudgetBytes > 0 && cache.UsedBytes > cache.BudgetBytes) {
		issues = append(issues, "block_asset_cache used_bytes must be within budget")
	}
	if cache.MaxEntries <= 0 || cache.EntryCount < 0 || cache.EntryCount > cache.MaxEntries {
		issues = append(issues, "block_asset_cache entry_count must be within max_entries")
	}
	if cache.RepeatedLoads <= cache.EntryCount {
		issues = append(issues, "block_asset_cache repeated_loads must exceed entry_count to prove reuse")
	}
	if strings.TrimSpace(cache.Eviction) == "" {
		issues = append(issues, "block_asset_cache eviction policy is required")
	}

	if len(report.BlockAssetDiagnostics) == 0 {
		issues = append(issues, "block_asset_diagnostics evidence is required")
	}
	lastDiagnosticOrder := 0
	hasMissingDiagnostic := false
	hasNetworkDiagnostic := false
	for i, diagnostic := range report.BlockAssetDiagnostics {
		if diagnostic.Order <= lastDiagnosticOrder {
			issues = append(issues, fmt.Sprintf("block_asset_diagnostics order %d is not strictly greater than previous order %d", diagnostic.Order, lastDiagnosticOrder))
		}
		lastDiagnosticOrder = diagnostic.Order
		if strings.TrimSpace(diagnostic.AssetID) == "" || strings.TrimSpace(diagnostic.Kind) == "" || strings.TrimSpace(diagnostic.Code) == "" || strings.TrimSpace(diagnostic.Message) == "" {
			issues = append(issues, fmt.Sprintf("block_asset_diagnostics[%d] asset_id, kind, code, and message are required", i))
		}
		if !diagnostic.Pass {
			issues = append(issues, fmt.Sprintf("block_asset_diagnostics[%d] pass must be true", i))
		}
		code := normalizeStateToken(diagnostic.Code)
		if code == "missing_asset_fallback" {
			hasMissingDiagnostic = true
			if strings.TrimSpace(diagnostic.FallbackID) == "" {
				issues = append(issues, "block_asset_diagnostics missing asset fallback requires fallback_id")
			}
		}
		if code == "network_asset_rejected" {
			hasNetworkDiagnostic = true
			if !isNetworkAssetPath(diagnostic.RejectedURL) {
				issues = append(issues, "block_asset_diagnostics network rejection requires rejected_url")
			}
		}
	}
	if !hasMissingDiagnostic {
		issues = append(issues, "block_asset_diagnostics require missing asset fallback diagnostic")
	}
	if !hasNetworkDiagnostic {
		issues = append(issues, "block_asset_diagnostics require network asset rejection diagnostic")
	}

	if len(report.BlockAssetRenderCommands) == 0 {
		issues = append(issues, "block_asset_render_commands evidence is required")
	}
	lastCommandOrder := 0
	commands := map[string]bool{}
	for i, command := range report.BlockAssetRenderCommands {
		if command.Order <= lastCommandOrder {
			issues = append(issues, fmt.Sprintf("block_asset_render_commands order %d is not strictly greater than previous order %d", command.Order, lastCommandOrder))
		}
		lastCommandOrder = command.Order
		name := normalizeStateToken(command.Command)
		commands[name] = true
		if strings.TrimSpace(command.AssetID) == "" {
			issues = append(issues, fmt.Sprintf("block_asset_render_commands[%d] asset_id is required", i))
		}
		if command.BlockID <= 0 {
			issues = append(issues, fmt.Sprintf("block_asset_render_commands[%d] block_id must be positive", i))
		}
		if command.Rect.W <= 0 || command.Rect.H <= 0 {
			issues = append(issues, fmt.Sprintf("block_asset_render_commands[%d] rect dimensions must be positive", i))
		}
		if strings.TrimSpace(command.Quality) == "" {
			issues = append(issues, fmt.Sprintf("block_asset_render_commands[%d] quality is required", i))
		}
		if !validChecksumLike(command.Checksum) {
			issues = append(issues, fmt.Sprintf("block_asset_render_commands[%d] checksum must be sha256 evidence", i))
		}
		if name == "tint_icon" && strings.TrimSpace(command.Tint) == "" {
			issues = append(issues, "block_asset_render_commands tint_icon requires tint evidence")
		}
		if name == "scale_image" && command.Scale < 2 {
			issues = append(issues, "block_asset_render_commands scale_image requires scale evidence")
		}
	}
	for _, required := range []string{"load_font", "tint_icon", "scale_image", "fallback_missing"} {
		if !commands[required] {
			issues = append(issues, fmt.Sprintf("block_asset_render_commands require %s command", required))
		}
	}

	if !hasEventTargetKind(report.Events, "IconBlock", "mouse_up") {
		issues = append(issues, "block asset evidence requires IconBlock tint trigger event")
	}
	for _, requirement := range []struct {
		component string
		field     string
	}{
		{"IconBlock", "tint"},
		{"ImageBlock", "scale"},
		{"MissingAssetBlock", "fallback"},
	} {
		if !hasTransition(report.StateTransitions, requirement.component, requirement.field) {
			issues = append(issues, fmt.Sprintf("block asset evidence requires %s %s state transition", requirement.component, requirement.field))
		}
	}
	if len(report.Frames) < 2 || strings.TrimSpace(report.Frames[0].Checksum) == "" || report.Frames[0].Checksum == report.Frames[1].Checksum {
		issues = append(issues, "block asset frame checksum evidence must show asset-driven visual change")
	}
	for _, required := range []string{
		"block asset deterministic manifest hashes",
		"block asset local embedded only",
		"block asset bounded cache",
		"block asset icon tint evidence",
		"block asset image scale evidence",
		"block asset missing fallback diagnostic",
		"block asset network url rejected",
	} {
		if !caseNameContains(report.Cases, required) {
			issues = append(issues, fmt.Sprintf("block asset report requires %s evidence", required))
		}
	}
	return issues
}

func hasBlockAssetEvidence(report Report) bool {
	return report.BlockAssetManifest != nil ||
		strings.TrimSpace(report.BlockAssetQualityLevel) != "" ||
		report.BlockAssetNetworkFetchAllowed ||
		strings.TrimSpace(report.BlockAssetCache.ID) != "" ||
		len(report.BlockAssetDiagnostics) > 0 ||
		len(report.BlockAssetRenderCommands) > 0
}

func validBlockAssetKind(kind string) bool {
	return kind == "font" || kind == "icon" || kind == "image"
}

func isNetworkAssetPath(path string) bool {
	value := strings.ToLower(strings.TrimSpace(path))
	return strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")
}
