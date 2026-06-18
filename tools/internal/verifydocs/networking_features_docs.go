package verifydocs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type networkingRuntimeBoundaryDocPaths struct {
	CurrentSurface string
	Stdlib         string
	StdlibGuide    string
	CoreNet        string
	CoreNetworking string
}

type networkingRuntimeBoundaryRequirement struct {
	Name     string
	Path     string
	Required []string
}

func defaultNetworkingRuntimeBoundaryDocPaths() networkingRuntimeBoundaryDocPaths {
	return networkingRuntimeBoundaryDocPaths{
		CurrentSurface: filepath.FromSlash("docs/spec/core/current_supported_surface.md"),
		Stdlib:         filepath.FromSlash("docs/spec/standard_library/stdlib.md"),
		StdlibGuide:    filepath.FromSlash("docs/user/platform/standard_library_guide.md"),
		CoreNet:        filepath.FromSlash("lib/core/io/net.tetra"),
		CoreNetworking: filepath.FromSlash("lib/core/io/networking.tetra"),
	}
}

func networkingRuntimeBoundaryRequirements(
	paths networkingRuntimeBoundaryDocPaths,
) []networkingRuntimeBoundaryRequirement {
	return []networkingRuntimeBoundaryRequirement{
		{
			Name: "current supported surface",
			Path: paths.CurrentSurface,
			Required: []string{
				"TechEmpower-compatible web stack",
				"no production HTTP server, full HTTP header/body",
				"parser, full event-loop abstraction, io_uring path, per-core worker runtime",
				"`lib.core.net` now provides executable linux-x64 TCP socket",
				"open/bind/connect/listen/accept/read/recv/write/send/nonblocking/close helpers",
				"`SO_REUSEPORT` and `TCP_NODELAY` helpers",
				"plus epoll",
				"create/add-read/add-read-write/mod-read/mod-read-write/delete/wait-one",
				"wait-one-into readiness flag helpers",
				"`SOCK_NONBLOCK`/`SOCK_CLOEXEC`",
				"`EPOLLIN`/`EPOLLOUT`/`EPOLLERR`/`EPOLLHUP` predicates",
				"`lib.core.http` now provides",
				"executable HTTP/1.1 String and byte-buffer",
				"request-line routing, byte-buffer request-head framing",
				"response byte-buffer helpers",
				"`lib.core.json` provides executable JSON",
				"`lib.core.postgres`",
				"wire-frame byte-buffer helpers",
				"Parse/Bind/Describe/Execute/Sync",
				"RowDescription/DataRow/CommandComplete/ReadyForQuery",
				"`lib.core.net` event-loop/socket-option expansion",
			},
		},
		{
			Name: "core net module",
			Path: paths.CoreNet,
			Required: []string{
				"Stable core Linux TCP networking helpers",
				"Runtime boundary: real linux-x64 TCP socket client/server helpers",
				"socket/bind/connect/listen/accept4/read/recv/write/send/epoll/fcntl/setsockopt/close syscalls",
				"event-loop abstractions",
				"outside this current surface",
			},
		},
		{
			Name: "stdlib spec",
			Path: paths.Stdlib,
			Required: []string{
				"`lib.core.net`",
				"`lib.core.net` is a stable capability-bound Linux TCP socket client/server I/O slice",
				"open/bind/connect/listen/accept/read/recv/write/send/nonblocking/close",
				"`SO_REUSEPORT` and `TCP_NODELAY` helpers",
				"plus epoll",
				"create/add-read/add-read-write/mod-read/mod-read-write/delete/wait-one",
				"wait-one-into readiness flag helpers",
				"`SOCK_NONBLOCK`/`SOCK_CLOEXEC`",
				"`EPOLLIN`/`EPOLLOUT`/`EPOLLERR`/`EPOLLHUP` predicates",
				"Full event-loop abstractions",
				"`lib.core.networking` Runtime Boundary",
				"`lib.core.networking` remains endpoint policy only",
				"`lib.core.http`",
				"`lib.core.json`",
				"`lib.core.postgres`",
				"PostgreSQL wire-frame helper module",
				"`func write_simple_query(dst: inout []u8, query: String) -> Int`",
				("`func write_parse(dst: inout []u8, statement: String, query: " +
					"String, param_type_oids: []i32) -> Int`"),
				("`func write_bind_text_2(dst: inout []u8, portal: String, " +
					"statement: String, value0: String, value1: String) -> Int`"),
				"`func data_row_i32_at(payload: []u8, start: Int, column_index: Int) -> Int`",
				"`func command_complete_affected_rows(payload: []u8, start: Int, payload_len: Int) -> Int`",
				("HTTP/1.1 String and byte-buffer request-line routing, byte-" +
					"buffer request-head framing, and response byte-buffer serialization " +
					"helpers live in `lib.core.http`"),
				"`func route_tech_empower_bytes(request: []u8, request_len: Int) -> Int`",
				"`func request_head_len_bytes(request: []u8, request_len: Int) -> Int`",
				"not an alias for sockets",
				"does not open sockets",
			},
		},
		{
			Name: "stdlib guide",
			Path: paths.StdlibGuide,
			Required: []string{
				"Linux TCP socket client/server I/O helpers",
				"`net.socket_tcp4(io_cap)`",
				"`net.connect_tcp4_loopback(fd, port, io_cap)`",
				"`net.read(fd, buffer, start, count, io_cap)`",
				"`net.recv(fd, buffer, start, count, io_cap)`",
				"`net.send(fd, buffer, start, count, io_cap)`",
				"`net.accept_nonblocking(fd, io_cap)`",
				"`net.set_reuseport(fd, io_cap)`",
				"`net.set_tcp_nodelay(fd, io_cap)`",
				"`net.epoll_ctl_add_read_write(epfd, fd, io_cap)`",
				"`net.epoll_ctl_mod_read(epfd, fd, io_cap)`",
				"`net.epoll_ctl_mod_read_write(epfd, fd, io_cap)`",
				"`net.epoll_ctl_delete(epfd, fd, io_cap)`",
				"`net.epoll_wait_one(epfd, timeout_ms, io_cap)`",
				"`net.epoll_wait_one_into(epfd, event, timeout_ms, io_cap)`",
				"`net.epoll_event_readable(flags)`",
				"`net.epoll_event_hung_up(flags)`",
				"`lib.core.net` is a stable linux-x64 TCP socket client/server I/O slice",
				"Networking Runtime Boundary",
				"`lib.core.networking` remains endpoint policy only",
				"`lib.core.net`",
				"`lib.core.http`",
				"`lib.core.json`",
				"`lib.core.postgres`",
				"PostgreSQL wire-frame byte-buffer helpers",
				"`lib.core.postgres` is a stable executable helper surface",
				"extended-query Parse/Bind/Describe/Execute/Sync",
				"RowDescription/DataRow/CommandComplete/ReadyForQuery",
				("HTTP String and byte-buffer request-line routing, request-head " +
					"framing, and response byte-buffer helpers"),
				"`http.route_tech_empower_bytes(buffer, length)`",
				"`http.request_head_len_bytes(buffer, length)`",
				"TechEmpower-compatible web stack",
			},
		},
		{
			Name: "core networking module",
			Path: paths.CoreNetworking,
			Required: []string{
				"Runtime boundary: endpoint policy only",
				"does not perform socket, TCP, DNS, HTTP request, PostgreSQL, or database I/O",
				"Real socket open/bind/connect/listen/accept/read/recv/write/send/nonblocking/close helpers",
				"SO_REUSEPORT/TCP_NODELAY helpers",
				"epoll add/mod/delete plus wait-one",
				"fd/readiness flag capture and predicates live in",
				"`lib.core.net`",
				"`lib.core.http`",
				"`lib.core.json`",
				"`lib.core.postgres`",
			},
		},
	}
}

func verifyNetworkingRuntimeBoundaryDocs(paths networkingRuntimeBoundaryDocPaths) error {
	var errs []string
	for _, requirement := range networkingRuntimeBoundaryRequirements(paths) {
		raw, err := os.ReadFile(requirement.Path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", requirement.Path, err))
			continue
		}
		text := string(raw)
		for _, want := range requirement.Required {
			if !strings.Contains(text, want) {
				errs = append(
					errs,
					fmt.Sprintf(
						"%s: missing %q for %s networking runtime boundary",
						requirement.Path,
						want,
						requirement.Name,
					),
				)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func verifyFeatureRegistry(features []featureManifest) error {
	if len(features) == 0 {
		return fmt.Errorf("feature registry is required in generated manifest")
	}
	allowedStatus := map[string]bool{
		"current":              true,
		"experimental":         true,
		"release_candidate":    true,
		"unsupported":          true,
		"legacy_compatibility": true,
		"planned":              true,
		"post-v1":              true,
	}
	requiredStatus := map[string]bool{
		"current": false,
		"planned": false,
		"post-v1": false,
	}
	requiredIDs := map[string]string{
		"cli.core":                                "current",
		"language.flow":                           "current",
		"language.generics-mvp":                   "current",
		"language.protocol-conformance-mvp":       "current",
		"language.callable-mvp":                   "current",
		"language.callable-level1":                "current",
		"targets.wasm-artifact-preflight":         "current",
		"stdlib.experimental-mirrors":             "current",
		"language.enum-payload-match":             "current",
		"language.protocol-bound-generics-static": "current",
		"language.ownership-markers-mvp":          "current",
		"language.resource-lifetime-mvp":          "current",
		"actors.task-transfer-safety":             "current",
		"language.lifetime-ssa":                   "current",
		"safety.production-core":                  "current",
		"language.callable-level2":                "current",
		"ui.metadata-v1":                          "current",
		"wasm.runtime-execution":                  "current",
		"actors.distributed-runtime":              "current",
		"ui.native-runtime":                       "current",
		"ui.platform-runtime":                     "experimental",
		"language.full-v1-guarantees":             "planned",
		"eco.distributed-network":                 "post-v1",
		"language.full-first-class-callables":     "current",
	}
	seen := map[string]string{}
	featureByID := map[string]featureManifest{}
	var currentCount int
	for _, feature := range features {
		if feature.ID == "" {
			return fmt.Errorf("feature registry entry missing id")
		}
		if feature.Name == "" || feature.Scope == "" || feature.Stability == "" {
			return fmt.Errorf("feature %s missing name, scope, or stability", feature.ID)
		}
		if !allowedStatus[feature.Status] {
			return fmt.Errorf("feature %s has invalid status %q", feature.ID, feature.Status)
		}
		if seenStatus, ok := seen[feature.ID]; ok {
			return fmt.Errorf(
				"feature %s is duplicated with statuses %s and %s",
				feature.ID,
				seenStatus,
				feature.Status,
			)
		}
		seen[feature.ID] = feature.Status
		featureByID[feature.ID] = feature
		requiredStatus[feature.Status] = true
		if feature.Status == "current" {
			currentCount++
			if feature.Since == "" {
				return fmt.Errorf("current feature %s missing since", feature.ID)
			}
		}
		if len(feature.Docs) == 0 {
			return fmt.Errorf("feature %s must cite docs", feature.ID)
		}
		for _, doc := range feature.Docs {
			if doc == "" {
				return fmt.Errorf("feature %s has empty doc reference", feature.ID)
			}
			docPath := filepath.ToSlash(doc)
			if filepath.IsAbs(doc) || strings.Contains(docPath, "..") {
				return fmt.Errorf("feature %s has unsafe doc reference %q", feature.ID, doc)
			}
			if !strings.HasPrefix(docPath, "docs/") || !strings.HasSuffix(docPath, ".md") {
				return fmt.Errorf(
					"feature %s doc reference %q must point at docs/*.md",
					feature.ID,
					doc,
				)
			}
			if _, err := statFromRepoRoot(docPath); err != nil {
				return fmt.Errorf(
					"feature %s doc reference %q is not readable: %v",
					feature.ID,
					doc,
					err,
				)
			}
		}
	}
	if currentCount == 0 {
		return fmt.Errorf("feature registry must include current features")
	}
	for status, present := range requiredStatus {
		if !present {
			return fmt.Errorf("feature registry missing %s feature", status)
		}
	}
	for id, wantStatus := range requiredIDs {
		if gotStatus, ok := seen[id]; !ok {
			return fmt.Errorf("feature registry missing %s", id)
		} else if gotStatus != wantStatus {
			return fmt.Errorf("feature registry %s status = %s, want %s", id, gotStatus, wantStatus)
		}
	}
	if err := verifyFeatureTruthBoundaries(featureByID); err != nil {
		return err
	}
	if err := verifySurfaceBlockSystemFeatureBoundary(featureByID); err != nil {
		return err
	}
	return nil
}

func verifySurfaceBlockSystemFeatureBoundary(features map[string]featureManifest) error {
	if _, ok := features["ui.surface-core"]; !ok {
		return nil
	}
	feature, ok := features["ui.surface-block-system"]
	if !ok {
		return fmt.Errorf("feature registry missing ui.surface-block-system")
	}
	if feature.Status != "experimental" {
		return fmt.Errorf(
			("feature registry ui.surface-block-system status = %s, want " +
				"experimental with scoped P18 evidence and no production support"),
			feature.Status,
		)
	}
	haystack := feature.Scope + " " + feature.Stability
	var missing []string
	for _, want := range []string{
		"Block-first",
		"core Surface primitive",
		"recipes/compatibility",
		"tetra.surface.block-system.gate.v1",
		"block_system.memory_budget",
		"reports/surface-block/p18-budget",
		"same-commit target evidence",
		"not production support",
		"no production Block claim",
	} {
		if !strings.Contains(haystack, want) {
			missing = append(missing, want)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"feature registry ui.surface-block-system missing truth-boundary phrase(s): %s",
			strings.Join(missing, ", "),
		)
	}
	for _, doc := range []string{
		"docs/spec/core/current_supported_surface.md",
		"docs/spec/surface/surface_v1.md",
		"docs/user/surface/surface_guide.md",
		"docs/user/reference/examples_index.md",
		"docs/release/surface/surface_v1_release_contract.md",
		"docs/release/surface/surface_v1_release_notes.md",
		"docs/release/surface/surface_v1_release_audit.md",
	} {
		found := false
		for _, got := range feature.Docs {
			if got == doc {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf(
				"feature registry ui.surface-block-system missing doc reference %s",
				doc,
			)
		}
	}
	return nil
}

func verifyFeatureTruthBoundaries(features map[string]featureManifest) error {
	checks := map[string][]string{
		"language.generics-mvp": {
			"statically monomorphized",
			"no runtime generic values or dynamic dispatch",
			"generic structs",
			"future/post-v1",
		},
		"language.protocol-conformance-mvp": {
			"checked statically",
			"generic requirement signature shape",
			"no witness tables",
			"dynamic dispatch remain post-v1",
		},
		"language.ownership-markers-mvp": {
			"conservative borrow/inout/consume marker checks",
			"use-after-consume",
			"borrow escape diagnostics",
			"not a full SSA lifetime solver",
		},
		"language.resource-lifetime-mvp": {
			"conservative resource finalization checks",
			"task handles",
			"island handles",
			"double-use",
			"ambiguous provenance",
			"not a full SSA lifetime solver",
		},
		"actors.task-transfer-safety": {
			"conservative actor/task ownership transfer checks",
			"worker entrypoints",
			"use-after-transfer diagnostics",
			"conservative local MVP",
			"distributed actors",
		},
		"language.lifetime-ssa": {
			"production SSA-like local lifetime join analysis",
			"ownership consume state",
			"resource finalization state",
			"maybe-consumed diagnostics",
			"richer interprocedural lifetime proofs",
		},
		"safety.production-core": {
			"production local safety model",
			"Memory Production Core v1",
			"validate-island-proof",
			"--islands-debug sanitizer smoke",
			"island-proof-fuzz-summary",
			"leak/resource finalization evidence",
			"integrated Memory/Islands/Surface release gate",
			"memory-islands-surface-production-manifest.json",
			"artifact-hashes.json",
			"no Memory 100% claim",
		},
		"language.enum-payload-match": {
			"positional enum payload constructors",
			"match/catch/if-let",
			"exhaustive unguarded enum match/catch",
			"nested destructuring patterns",
			"guard expansion remain future/post-v1",
		},
		"language.protocol-bound-generics-static": {
			"validated statically during monomorphization",
			"same-module and cross-module impl conformance",
			"visibility diagnostics",
			"calling protocol requirements through generic bounds",
			"dynamic dispatch remain unsupported",
		},
		"ui.native-runtime": {
			"production Linux-x64 native UI runtime path",
			"native runtime widget instances",
			"click/activate events",
			"state and widget updates",
			"tetra.ui.native-runtime.v1 smoke evidence",
			"metadata-only",
			"web-only",
			"native-shell sidecar-only",
			"macOS/Windows",
		},
		"ui.platform-runtime": {
			"tetra.ui.platform-runtime.v1",
			"full-platform UI runtime promotion gate",
			"real Windows/macOS target-host reports",
			"not production until",
			"metadata-only",
			"runtime-less",
			"startup_failure",
		},
	}
	docChecks := map[string][]string{
		"language.generics-mvp": {
			"docs/spec/core/current_supported_surface.md",
			"docs/spec/flow/flow_syntax_v1.md",
			"docs/spec/flow/v1_scope.md",
		},
		"language.protocol-conformance-mvp": {
			"docs/spec/core/current_supported_surface.md",
			"docs/spec/flow/flow_syntax_v1.md",
			"docs/spec/flow/v1_scope.md",
		},
		"language.ownership-markers-mvp": {
			"docs/spec/core/current_supported_surface.md",
			"docs/spec/runtime/ownership_v1.md",
			"docs/spec/flow/v1_scope.md",
		},
		"language.resource-lifetime-mvp": {
			"docs/spec/core/current_supported_surface.md",
			"docs/spec/runtime/ownership_v1.md",
			"docs/spec/flow/v1_scope.md",
		},
		"actors.task-transfer-safety": {
			"docs/spec/core/current_supported_surface.md",
			"docs/spec/runtime/ownership_v1.md",
			"docs/spec/flow/v1_scope.md",
		},
		"language.lifetime-ssa": {
			"docs/spec/core/current_supported_surface.md",
			"docs/spec/runtime/ownership_v1.md",
			"docs/spec/flow/v1_scope.md",
		},
		"safety.production-core": {
			"docs/spec/core/current_supported_surface.md",
			"docs/spec/runtime/ownership_v1.md",
			"docs/spec/runtime/effects_capabilities_privacy_v1.md",
			"docs/spec/runtime/unsafe.md",
			"docs/spec/memory/memory_report_schema_v1.md",
			"docs/spec/memory/islands.md",
			"docs/design/memory/memory_production_core_v1.md",
			"docs/design/memory/memory_cost_model.md",
			"docs/audits/memory/islands/memory-fuzz-oracle-v1.md",
			"docs/audits/memory/production/memory-production-core-v1-final.md",
			"docs/audits/memory/production/memory-production-core-v1-artifact-map.md",
			"docs/audits/memory/production/memory-production-core-v1-nonclaims.md",
			"docs/release/surface/memory_islands_surface_scope.md",
		},
		"language.enum-payload-match": {
			"docs/spec/core/current_supported_surface.md",
			"docs/spec/flow/flow_syntax_v1.md",
			"docs/spec/flow/v0_3_scope.md",
		},
		"language.protocol-bound-generics-static": {
			"docs/spec/core/current_supported_surface.md",
			"docs/spec/flow/v0_3_scope.md",
			"docs/spec/flow/flow_syntax_v1.md",
		},
		"ui.native-runtime": {
			"docs/spec/core/current_supported_surface.md",
			"docs/spec/ui/ui_v1.md",
			"docs/user/surface/wasm_ui_guide.md",
		},
		"ui.platform-runtime": {
			"docs/spec/core/current_supported_surface.md",
			"docs/spec/ui/ui_v1.md",
			"docs/user/surface/wasm_ui_guide.md",
		},
	}
	for id, required := range checks {
		feature, ok := features[id]
		if !ok {
			return fmt.Errorf("feature registry missing %s", id)
		}
		haystack := feature.Scope + " " + feature.Stability
		for _, want := range required {
			if !strings.Contains(haystack, want) {
				return fmt.Errorf("feature registry %s missing truth-boundary phrase %q", id, want)
			}
		}
		for _, doc := range docChecks[id] {
			if !hasString(feature.Docs, doc) {
				return fmt.Errorf("feature registry %s missing doc reference %s", id, doc)
			}
		}
	}
	return nil
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func statFromRepoRoot(path string) (os.FileInfo, error) {
	if info, err := os.Stat(filepath.FromSlash(path)); err == nil {
		return info, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, filepath.FromSlash(path))
		if info, err := os.Stat(candidate); err == nil {
			return info, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return nil, os.ErrNotExist
}
