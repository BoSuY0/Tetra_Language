package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"tetra_language/compiler/internal/actorsrt"
	"tetra_language/compiler/internal/netrt"
	"tetra_language/compiler/internal/pgrt"
	"tetra_language/compiler/internal/runtimeabi"
)

const (
	securityReviewGateV1Schema    = "tetra.security.review_gate.v1"
	securityReviewGateV1ScopeP240 = "p24.0_security_review_gate"

	p24SecurityReviewUnsafeWitnessID     = "unsafe_api_surface"
	p24SecurityReviewCapabilityWitnessID = "capability_surface"
	p24SecurityReviewAllocatorWitnessID  = "memory_allocator_surface"
	p24SecurityReviewNetworkWitnessID    = "network_runtime_surface"
	p24SecurityReviewActorWitnessID      = "actor_runtime_surface"
	p24SecurityReviewDBWitnessID         = "db_protocol_surface"
	p24SecurityReviewEcoWitnessID        = "package_eco_surface"
	p24SecurityReviewBuildWitnessID      = "build_script_surface"
	p24SecurityReviewSupplyWitnessID     = "supply_chain_surface"
	p24SecurityReviewArtifactsWitnessID  = "security_review_artifacts"
)

type SecurityReviewGateV1ID string

const (
	SecurityReviewUnsafeAPISurface  SecurityReviewGateV1ID = "unsafe_api_surface"
	SecurityReviewCapabilitySurface SecurityReviewGateV1ID = "capability_surface"
	SecurityReviewMemoryAllocator   SecurityReviewGateV1ID = "memory_allocator_surface"
	SecurityReviewNetworkRuntime    SecurityReviewGateV1ID = "network_runtime_surface"
	SecurityReviewActorRuntime      SecurityReviewGateV1ID = "actor_runtime_surface"
	SecurityReviewDBProtocol        SecurityReviewGateV1ID = "db_protocol_surface"
	SecurityReviewPackageEcoSystem  SecurityReviewGateV1ID = "package_eco_system"
	SecurityReviewBuildScripts      SecurityReviewGateV1ID = "build_scripts"
	SecurityReviewSupplyChain       SecurityReviewGateV1ID = "supply_chain"
	SecurityReviewArtifactSet       SecurityReviewGateV1ID = "security_review_artifacts"
)

type SecurityReviewGateV1Report struct {
	SchemaVersion string                        `json:"schema_version"`
	Scope         string                        `json:"scope"`
	Rows          []SecurityReviewGateV1Row     `json:"rows"`
	Witnesses     []SecurityReviewGateV1Witness `json:"witnesses"`
	Artifacts     []SecurityReviewArtifact      `json:"artifacts"`
	NonClaims     []string                      `json:"non_claims"`

	UnsafeAPISurfaceReviewed      bool `json:"unsafe_api_surface_reviewed"`
	CapabilitySurfaceReviewed     bool `json:"capability_surface_reviewed"`
	MemoryAllocatorReviewed       bool `json:"memory_allocator_reviewed"`
	NetworkRuntimeReviewed        bool `json:"network_runtime_reviewed"`
	ActorRuntimeReviewed          bool `json:"actor_runtime_reviewed"`
	DBProtocolReviewed            bool `json:"db_protocol_reviewed"`
	PackageEcoSystemReviewed      bool `json:"package_eco_system_reviewed"`
	BuildScriptsReviewed          bool `json:"build_scripts_reviewed"`
	SupplyChainReviewed           bool `json:"supply_chain_reviewed"`
	SecurityReviewArtifactPresent bool `json:"security_review_artifact_present"`
	ThreatModelArtifactPresent    bool `json:"threat_model_artifact_present"`
	UnsafeSurfaceMapPresent       bool `json:"unsafe_surface_map_present"`
	CapabilitySurfaceMapPresent   bool `json:"capability_surface_map_present"`
	SecurityCertifiedClaimed      bool `json:"security_certified_claimed"`
	ExternalPenTestClaimed        bool `json:"external_pen_test_claimed"`
	CVEFreeClaimed                bool `json:"cve_free_claimed"`
	ReleaseSignoffClaimed         bool `json:"release_signoff_claimed"`
	RuntimeBehaviorChanged        bool `json:"runtime_behavior_changed"`
	SafeSemanticsChanged          bool `json:"safe_semantics_changed"`
	PerformanceClaimed            bool `json:"performance_claimed"`
}

type SecurityReviewGateV1Row struct {
	ID         SecurityReviewGateV1ID `json:"id"`
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	Evidence   []string               `json:"evidence"`
	Tests      []string               `json:"tests"`
	Boundaries []string               `json:"boundaries"`
	WitnessIDs []string               `json:"witness_ids"`
}

type SecurityReviewArtifact struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Present bool   `json:"present"`
}

type SecurityReviewGateV1Witness struct {
	ID                              string   `json:"id"`
	Kind                            string   `json:"kind"`
	Paths                           []string `json:"paths,omitempty"`
	UnsafeAPISurfaceReviewed        bool     `json:"unsafe_api_surface_reviewed,omitempty"`
	CapabilitySurfaceReviewed       bool     `json:"capability_surface_reviewed,omitempty"`
	MemoryAllocatorReviewed         bool     `json:"memory_allocator_reviewed,omitempty"`
	RuntimeAllocationContracts      int      `json:"runtime_allocation_contracts,omitempty"`
	RawPointerBoundsMetadataVersion string   `json:"raw_pointer_bounds_metadata_version,omitempty"`
	NetworkRuntimeReviewed          bool     `json:"network_runtime_reviewed,omitempty"`
	IOReactorRows                   int      `json:"io_reactor_rows,omitempty"`
	ActorRuntimeReviewed            bool     `json:"actor_runtime_reviewed,omitempty"`
	ActorBoundaryRows               int      `json:"actor_boundary_rows,omitempty"`
	DBProtocolReviewed              bool     `json:"db_protocol_reviewed,omitempty"`
	ProductionPostgresRows          int      `json:"production_postgres_rows,omitempty"`
	PackageEcoSystemReviewed        bool     `json:"package_eco_system_reviewed,omitempty"`
	EcoValidatorPaths               int      `json:"eco_validator_paths,omitempty"`
	BuildScriptsReviewed            bool     `json:"build_scripts_reviewed,omitempty"`
	ReleaseSecurityScripts          int      `json:"release_security_scripts,omitempty"`
	SupplyChainReviewed             bool     `json:"supply_chain_reviewed,omitempty"`
	SupplyChainEvidencePaths        int      `json:"supply_chain_evidence_paths,omitempty"`
	SecurityReviewArtifactPresent   bool     `json:"security_review_artifact_present,omitempty"`
	ThreatModelArtifactPresent      bool     `json:"threat_model_artifact_present,omitempty"`
	UnsafeSurfaceMapPresent         bool     `json:"unsafe_surface_map_present,omitempty"`
	CapabilitySurfaceMapPresent     bool     `json:"capability_surface_map_present,omitempty"`
}

func BuildP24SecurityReviewGateV1Report() (SecurityReviewGateV1Report, error) {
	unsafeWitness := buildP24UnsafeWitness()
	capabilityWitness := buildP24CapabilityWitness()
	allocatorWitness, err := buildP24AllocatorWitness()
	if err != nil {
		return SecurityReviewGateV1Report{}, err
	}
	networkWitness, err := buildP24NetworkWitness()
	if err != nil {
		return SecurityReviewGateV1Report{}, err
	}
	actorWitness, err := buildP24ActorWitness()
	if err != nil {
		return SecurityReviewGateV1Report{}, err
	}
	dbWitness, err := buildP24DBWitness()
	if err != nil {
		return SecurityReviewGateV1Report{}, err
	}
	ecoWitness := buildP24EcoWitness()
	buildWitness := buildP24BuildScriptsWitness()
	supplyWitness := buildP24SupplyChainWitness()
	artifacts := p24SecurityReviewArtifacts()
	artifactWitness := buildP24ArtifactsWitness(artifacts)

	report := SecurityReviewGateV1Report{
		SchemaVersion: securityReviewGateV1Schema,
		Scope:         securityReviewGateV1ScopeP240,
		Witnesses: []SecurityReviewGateV1Witness{
			unsafeWitness,
			capabilityWitness,
			allocatorWitness,
			networkWitness,
			actorWitness,
			dbWitness,
			ecoWitness,
			buildWitness,
			supplyWitness,
			artifactWitness,
		},
		Artifacts: artifacts,
		Rows: []SecurityReviewGateV1Row{
			p24SecurityReviewGateRow(SecurityReviewUnsafeAPISurface, "Unsafe API surface", "reviewed_current_surface",
				[]string{
					"docs/spec/unsafe.md records unsafe-only builtins including core.cap_mem, core.cap_io, core.alloc_bytes, pointer arithmetic, load/store, MMIO, symbol address, context switch, and island operations.",
					"Unsafe APIs remain gated by explicit unsafe syntax and capability/effect requirements where applicable.",
				},
				[]string{
					"go test ./compiler -run 'P24SecurityReviewGate' -count=1",
					"go test ./compiler/... -run 'Unsafe|Capability|Effect|MMIO|Mem' -count=1",
					"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
				},
				[]string{
					"this is an inventory and policy review, not a proof that all unsafe callers are memory safe",
					"unsafe syntax remains required for unsafe-only builtins",
				},
				[]string{p24SecurityReviewUnsafeWitnessID}),
			p24SecurityReviewGateRow(SecurityReviewCapabilitySurface, "Capability surface", "reviewed_current_surface",
				[]string{
					"docs/spec/capabilities.md and docs/spec/effects_capabilities_privacy_v1.md define cap.mem, cap.io, uses propagation, and attenuation checks.",
					"uses declarations remain audit metadata and do not manufacture capability tokens.",
				},
				[]string{
					"go test ./compiler/... -run 'Capability|Effect|Uses|Capsule' -count=1",
					"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
				},
				[]string{
					"cap.mem is permission, not provenance, lifetime, bounds, alias, or sendability proof",
					"privacy consent tokens are separate from cap.mem and cap.io",
				},
				[]string{p24SecurityReviewCapabilityWitnessID}),
			p24SecurityReviewGateRow(SecurityReviewMemoryAllocator, "Memory allocator surface", "reviewed_runtime_contracts",
				[]string{
					"runtimeabi.RuntimeAllocationContracts validates core.alloc_bytes, slice builders, islands, regions, guard behavior, failure behavior, debug instrumentation, and report hooks.",
					"runtimeabi.RuntimeRawPointerBoundsABI exposes raw-pointer-bounds-v1 metadata for allocation roots, derived offsets, external unknown pointers, and rejected impossible ptr_add cases.",
				},
				[]string{
					"go test ./compiler/internal/runtimeabi -run 'Allocation|Region|SmallHeap|RawPointer' -count=1",
					"go test ./compiler -run 'P24SecurityReviewGate' -count=1",
				},
				[]string{
					"allocator contracts are runtime ABI evidence, not a formal memory-safety proof",
					"external unknown raw pointers remain bounded as unknown rather than promoted to verified allocation roots",
				},
				[]string{p24SecurityReviewAllocatorWitnessID}),
			p24SecurityReviewGateRow(SecurityReviewNetworkRuntime, "Network runtime surface", "reviewed_runtime_boundary",
				[]string{
					"netrt.IOReactorCoverage validates Linux epoll, readiness polling, nonblocking accept/read/write, I/O task wakeups, timer, cancellation, backpressure, HTTP smoke, DB smoke, and stress evidence.",
					"Linux epoll is current narrow evidence; cross-platform parity and io_uring remain non-claims.",
				},
				[]string{
					"go test ./compiler/internal/netrt -run 'IOReactor|Poller|Readiness|Backpressure' -count=1",
					"go test ./compiler -run 'P24SecurityReviewGate' -count=1",
				},
				[]string{
					"network runtime review is bounded to current netrt evidence and does not claim full production web-stack security",
					"kqueue, IOCP, WASI/web event adapters, and io_uring remain documented boundaries",
				},
				[]string{p24SecurityReviewNetworkWitnessID}),
			p24SecurityReviewGateRow(SecurityReviewActorRuntime, "Actor runtime surface", "reviewed_runtime_boundary",
				[]string{
					"actorsrt.ActorRuntimeProductionBoundaryAudit records current actor runtime limits, scheduler prototype features, production acceptance requirements, and full-claim blockers.",
					"Current evidence records message pool limits and explicitly states scheduler prototype evidence is not a production multi-threaded actor scheduler.",
				},
				[]string{
					"go test ./compiler/internal/actorsrt ./compiler/internal/parallelrt -run 'ActorRuntime|ProductionBoundary|SchedulerModel' -count=1",
					"go test ./compiler -run 'P24SecurityReviewGate' -count=1",
				},
				[]string{
					"actor runtime review does not promote distributed actor support or production broker deployment",
					"message pool exhaustion/reclamation and full race-safety proof remain blockers for a production actor runtime claim",
				},
				[]string{p24SecurityReviewActorWitnessID}),
			p24SecurityReviewGateRow(SecurityReviewDBProtocol, "DB protocol surface", "reviewed_protocol_boundary",
				[]string{
					"pgrt.ProductionPostgresCoverage validates SCRAM-SHA-256 startup, prepared statements, binary protocol, pooling backpressure, borrowed row decode, endpoint workloads, and benchmark honesty rows.",
					"compiler/internal/pgrt/wire.go rejects malformed frames with ErrMalformedFrame and oversized payloads with ErrFrameTooLarge; pool.go returns ErrPoolExhausted instead of opening past maxOpen.",
				},
				[]string{
					"go test ./compiler/internal/pgrt -run 'ProductionPostgres|SCRAM|Frame|Pool' -count=1",
					"go test ./compiler -run 'P24SecurityReviewGate' -count=1",
				},
				[]string{
					"DB protocol review is local PostgreSQL wire-protocol compatibility evidence, not TLS, channel binding, or external production database deployment evidence",
					"official TechEmpower and production database benchmark claims remain forbidden by the pgrt coverage validator",
				},
				[]string{p24SecurityReviewDBWitnessID}),
			p24SecurityReviewGateRow(SecurityReviewPackageEcoSystem, "Package/Eco system surface", "reviewed_local_supply_surface",
				[]string{
					"docs/spec/eco_publishing_v1.md defines tetra.eco.publish.v1, Tetra.lock hash semantics, permission escalation checks, artifact hashes, trust snapshots, materialization metadata, and reproducible packaging basics.",
					"tools/cmd/validate-eco-lock, validate-eco-publish, validate-eco-vault, validate-eco-mirror, and validate-eco-unpack reject schema drift, unsafe paths, hash mismatches, unknown fields, and tampered package content.",
				},
				[]string{
					"go test ./cli/... ./tools/... -run 'Eco|Permission|Capsule|Trust' -count=1",
					"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
				},
				[]string{
					"Eco/Todex trust is local metadata and validator evidence, not a global package trust network",
					"proof-carrying capsules and distributed EcoNet remain outside the current claim",
				},
				[]string{p24SecurityReviewEcoWitnessID}),
			p24SecurityReviewGateRow(SecurityReviewBuildScripts, "Build and release script surface", "reviewed_release_validator_boundary",
				[]string{
					"scripts/release/v1_0/security-review.sh checks current_release_version, reviewed commit, Decision, Evidence Commands, Artifact Hashes, and Residual Risks for release signoff files.",
					"tools/scriptstest/security_review_test.go rejects template signoffs and stale review metadata for the release security-review script family.",
				},
				[]string{
					"go test ./tools/scriptstest -run 'SecurityReview' -count=1",
					"bash scripts/release/v1_0/security-review.sh --help",
				},
				[]string{
					"P24.0 security-review.md is an audit artifact and does not count as a release signoff file",
					"release signoff still requires the release script validator over a release report directory",
				},
				[]string{p24SecurityReviewBuildWitnessID}),
			p24SecurityReviewGateRow(SecurityReviewSupplyChain, "Supply-chain surface", "reviewed_local_hash_boundary",
				[]string{
					"go.sum pins Go module checksums for this repository and Eco validators require sha256 metadata for locks, packages, trust snapshot files, mirrors, vault objects, and unpacked package content.",
					"docs/spec/eco_publishing_v1.md records trust snapshot and local artifact hash boundaries; no network trust claim is made for remote registries or global package identity.",
				},
				[]string{
					"go test ./tools/cmd/validate-eco-lock ./tools/cmd/validate-eco-publish ./tools/cmd/validate-eco-vault ./tools/cmd/validate-eco-mirror ./tools/cmd/validate-eco-unpack -count=1",
					"go test ./compiler -run 'P24SecurityReviewGate' -count=1",
				},
				[]string{
					"supply-chain evidence is local lock/hash/metadata validation, not SLSA certification or external registry trust",
					"remote fetch/mirror paths must validate package bytes and metadata before writing local store files",
				},
				[]string{p24SecurityReviewSupplyWitnessID}),
			p24SecurityReviewGateRow(SecurityReviewArtifactSet, "Security review artifact set", "required_artifacts_present",
				[]string{
					"docs/audits/security-review.md summarizes the P24.0 review with evidence, residual risks, and commands.",
					"docs/audits/threat-model.md records assets, trust boundaries, attacker capabilities, abuse paths, mitigations, assumptions, and open questions.",
					"docs/audits/unsafe-surface-map.md maps unsafe builtins, required syntax/effects/capabilities, owners, tests, and residual risks.",
					"docs/audits/capability-surface-map.md maps cap.io, cap.mem, privacy consent, capsule attenuation, permission metadata, and local Eco trust boundaries.",
				},
				[]string{
					"go test ./compiler -run 'P24SecurityReviewGate' -count=1",
					"go run ./tools/cmd/verify-docs --manifest docs/generated/manifest.json",
				},
				[]string{
					"artifacts are current-branch review artifacts, not external audit reports or release signoff",
					"future promotion must update artifact hashes in the release report directory",
				},
				[]string{p24SecurityReviewArtifactsWitnessID}),
		},
		NonClaims: []string{
			"security certification is not claimed",
			"external penetration test is not claimed",
			"CVE-free status is not claimed",
			"release security signoff is not claimed",
			"runtime behavior does not change",
			"safe-program semantics do not change",
			"no performance claim is made",
		},
		UnsafeAPISurfaceReviewed:      unsafeWitness.UnsafeAPISurfaceReviewed,
		CapabilitySurfaceReviewed:     capabilityWitness.CapabilitySurfaceReviewed,
		MemoryAllocatorReviewed:       allocatorWitness.MemoryAllocatorReviewed,
		NetworkRuntimeReviewed:        networkWitness.NetworkRuntimeReviewed,
		ActorRuntimeReviewed:          actorWitness.ActorRuntimeReviewed,
		DBProtocolReviewed:            dbWitness.DBProtocolReviewed,
		PackageEcoSystemReviewed:      ecoWitness.PackageEcoSystemReviewed,
		BuildScriptsReviewed:          buildWitness.BuildScriptsReviewed,
		SupplyChainReviewed:           supplyWitness.SupplyChainReviewed,
		SecurityReviewArtifactPresent: artifactWitness.SecurityReviewArtifactPresent,
		ThreatModelArtifactPresent:    artifactWitness.ThreatModelArtifactPresent,
		UnsafeSurfaceMapPresent:       artifactWitness.UnsafeSurfaceMapPresent,
		CapabilitySurfaceMapPresent:   artifactWitness.CapabilitySurfaceMapPresent,
		SecurityCertifiedClaimed:      false,
		ExternalPenTestClaimed:        false,
		CVEFreeClaimed:                false,
		ReleaseSignoffClaimed:         false,
		RuntimeBehaviorChanged:        false,
		SafeSemanticsChanged:          false,
		PerformanceClaimed:            false,
	}
	if err := ValidateP24SecurityReviewGateV1Report(report); err != nil {
		return SecurityReviewGateV1Report{}, err
	}
	return report, nil
}

func ValidateP24SecurityReviewGateV1Report(report SecurityReviewGateV1Report) error {
	if report.SchemaVersion != securityReviewGateV1Schema {
		return fmt.Errorf("security review gate v1: schema_version is %q", report.SchemaVersion)
	}
	if report.Scope != securityReviewGateV1ScopeP240 {
		return fmt.Errorf("security review gate v1: scope is %q", report.Scope)
	}
	if report.SecurityCertifiedClaimed {
		return fmt.Errorf("security review gate v1: security certification claim is forbidden")
	}
	if report.ExternalPenTestClaimed {
		return fmt.Errorf("security review gate v1: external penetration test claim is forbidden")
	}
	if report.CVEFreeClaimed {
		return fmt.Errorf("security review gate v1: CVE-free claim is forbidden")
	}
	if report.ReleaseSignoffClaimed {
		return fmt.Errorf("security review gate v1: release signoff claim is forbidden")
	}
	if report.RuntimeBehaviorChanged {
		return fmt.Errorf("security review gate v1: runtime behavior change claim is forbidden")
	}
	if report.SafeSemanticsChanged {
		return fmt.Errorf("security review gate v1: safe semantics change claim is forbidden")
	}
	if report.PerformanceClaimed {
		return fmt.Errorf("security review gate v1: performance claim is forbidden")
	}
	if !report.UnsafeAPISurfaceReviewed {
		return fmt.Errorf("security review gate v1: unsafe API surface review missing")
	}
	if !report.CapabilitySurfaceReviewed {
		return fmt.Errorf("security review gate v1: capability surface review missing")
	}
	if !report.MemoryAllocatorReviewed {
		return fmt.Errorf("security review gate v1: memory allocator review missing")
	}
	if !report.NetworkRuntimeReviewed {
		return fmt.Errorf("security review gate v1: network runtime review missing")
	}
	if !report.ActorRuntimeReviewed {
		return fmt.Errorf("security review gate v1: actor runtime review missing")
	}
	if !report.DBProtocolReviewed {
		return fmt.Errorf("security review gate v1: DB protocol review missing")
	}
	if !report.PackageEcoSystemReviewed {
		return fmt.Errorf("security review gate v1: package/Eco system review missing")
	}
	if !report.BuildScriptsReviewed {
		return fmt.Errorf("security review gate v1: build scripts review missing")
	}
	if !report.SupplyChainReviewed {
		return fmt.Errorf("security review gate v1: supply chain review missing")
	}
	if err := p24SecurityReviewValidateArtifacts(report); err != nil {
		return err
	}
	for _, want := range []string{
		"security certification is not claimed",
		"external penetration test is not claimed",
		"CVE-free status is not claimed",
		"release security signoff is not claimed",
		"runtime behavior does not change",
		"safe-program semantics do not change",
		"no performance claim is made",
	} {
		if !p24SecurityReviewHasString(report.NonClaims, want) {
			return fmt.Errorf("security review gate v1: missing non-claim %q", want)
		}
	}
	if err := p24SecurityReviewValidateRowsAndWitnesses(report.Rows, report.Witnesses); err != nil {
		return err
	}
	return nil
}

func buildP24UnsafeWitness() SecurityReviewGateV1Witness {
	paths := []string{
		"docs/spec/unsafe.md",
		"examples/flow_unsafe_cap_mem_smoke.tetra",
		"lib/core/capability.tetra",
	}
	return SecurityReviewGateV1Witness{
		ID:                       p24SecurityReviewUnsafeWitnessID,
		Kind:                     "unsafe_api_surface",
		Paths:                    paths,
		UnsafeAPISurfaceReviewed: p24AllRepoPathsExist(paths),
	}
}

func buildP24CapabilityWitness() SecurityReviewGateV1Witness {
	paths := []string{
		"docs/spec/capabilities.md",
		"docs/spec/effects_capabilities_privacy_v1.md",
		"examples/core_capability_smoke.tetra",
		"lib/core/capability.tetra",
	}
	return SecurityReviewGateV1Witness{
		ID:                        p24SecurityReviewCapabilityWitnessID,
		Kind:                      "capability_surface",
		Paths:                     paths,
		CapabilitySurfaceReviewed: p24AllRepoPathsExist(paths),
	}
}

func buildP24AllocatorWitness() (SecurityReviewGateV1Witness, error) {
	contracts := runtimeabi.RuntimeAllocationContracts()
	for _, contract := range contracts {
		if err := runtimeabi.ValidateRuntimeAllocationContract(contract); err != nil {
			return SecurityReviewGateV1Witness{}, err
		}
	}
	rawBounds := runtimeabi.RuntimeRawPointerBoundsABI()
	return SecurityReviewGateV1Witness{
		ID:                              p24SecurityReviewAllocatorWitnessID,
		Kind:                            "memory_allocator_surface",
		Paths:                           []string{"compiler/internal/runtimeabi/allocation_contract.go", "compiler/internal/runtimeabi/raw_pointer_bounds.go"},
		MemoryAllocatorReviewed:         len(contracts) >= 5 && rawBounds.MetadataVersion == "raw-pointer-bounds-v1",
		RuntimeAllocationContracts:      len(contracts),
		RawPointerBoundsMetadataVersion: rawBounds.MetadataVersion,
	}, nil
}

func buildP24NetworkWitness() (SecurityReviewGateV1Witness, error) {
	report, err := netrt.IOReactorCoverage()
	if err != nil {
		return SecurityReviewGateV1Witness{}, err
	}
	if err := netrt.ValidateIOReactorCoverage(report); err != nil {
		return SecurityReviewGateV1Witness{}, err
	}
	return SecurityReviewGateV1Witness{
		ID:                     p24SecurityReviewNetworkWitnessID,
		Kind:                   "network_runtime_surface",
		Paths:                  []string{"compiler/internal/netrt/io_reactor_coverage.go", "compiler/internal/netrt/netrt_linux.go"},
		NetworkRuntimeReviewed: len(report.Rows) >= 10 && !report.FullProductionWebStackClaimed && !report.CrossPlatformParityClaimed,
		IOReactorRows:          len(report.Rows),
	}, nil
}

func buildP24ActorWitness() (SecurityReviewGateV1Witness, error) {
	report, err := actorsrt.ActorRuntimeProductionBoundaryAudit()
	if err != nil {
		return SecurityReviewGateV1Witness{}, err
	}
	if err := actorsrt.ValidateActorRuntimeProductionBoundaryAudit(report); err != nil {
		return SecurityReviewGateV1Witness{}, err
	}
	return SecurityReviewGateV1Witness{
		ID:                   p24SecurityReviewActorWitnessID,
		Kind:                 "actor_runtime_surface",
		Paths:                []string{"compiler/internal/actorsrt/production_boundary.go", "docs/spec/actors.md"},
		ActorRuntimeReviewed: len(report.Rows) >= 4 && !report.FullProductionClaimed,
		ActorBoundaryRows:    len(report.Rows),
	}, nil
}

func buildP24DBWitness() (SecurityReviewGateV1Witness, error) {
	report, err := pgrt.ProductionPostgresCoverage()
	if err != nil {
		return SecurityReviewGateV1Witness{}, err
	}
	if err := pgrt.ValidateProductionPostgresCoverage(report); err != nil {
		return SecurityReviewGateV1Witness{}, err
	}
	return SecurityReviewGateV1Witness{
		ID:                     p24SecurityReviewDBWitnessID,
		Kind:                   "db_protocol_surface",
		Paths:                  []string{"compiler/internal/pgrt/production_postgres_coverage.go", "compiler/internal/pgrt/wire.go", "compiler/internal/pgrt/scram.go", "compiler/internal/pgrt/pool.go"},
		DBProtocolReviewed:     len(report.Rows) >= 8 && !report.ExternalProductionDatabaseClaimed && !report.FullSourceLevelDriverClaimed,
		ProductionPostgresRows: len(report.Rows),
	}, nil
}

func buildP24EcoWitness() SecurityReviewGateV1Witness {
	paths := []string{
		"docs/spec/eco_publishing_v1.md",
		"cli/cmd/tetra/eco_publish.go",
		"cli/cmd/tetra/eco_seed.go",
		"tools/cmd/validate-eco-lock/main.go",
		"tools/cmd/validate-eco-publish/main.go",
		"tools/cmd/validate-eco-vault/main.go",
		"tools/cmd/validate-eco-mirror/main.go",
		"tools/cmd/validate-eco-unpack/main.go",
	}
	return SecurityReviewGateV1Witness{
		ID:                       p24SecurityReviewEcoWitnessID,
		Kind:                     "package_eco_surface",
		Paths:                    paths,
		PackageEcoSystemReviewed: p24AllRepoPathsExist(paths),
		EcoValidatorPaths:        len(paths) - 3,
	}
}

func buildP24BuildScriptsWitness() SecurityReviewGateV1Witness {
	paths := []string{
		"scripts/release/v1_0/security-review.sh",
		"scripts/release/v0_4_0/security-review.sh",
		"scripts/release/v0_3_0/security-review.sh",
		"tools/scriptstest/security_review_test.go",
	}
	return SecurityReviewGateV1Witness{
		ID:                     p24SecurityReviewBuildWitnessID,
		Kind:                   "build_script_surface",
		Paths:                  paths,
		BuildScriptsReviewed:   p24AllRepoPathsExist(paths),
		ReleaseSecurityScripts: 3,
	}
}

func buildP24SupplyChainWitness() SecurityReviewGateV1Witness {
	paths := []string{
		"go.sum",
		"docs/spec/eco_publishing_v1.md",
		"tools/cmd/validate-eco-lock/main.go",
		"tools/cmd/validate-eco-publish/main.go",
		"tools/cmd/validate-eco-vault/main.go",
	}
	return SecurityReviewGateV1Witness{
		ID:                       p24SecurityReviewSupplyWitnessID,
		Kind:                     "supply_chain_surface",
		Paths:                    paths,
		SupplyChainReviewed:      p24AllRepoPathsExist(paths),
		SupplyChainEvidencePaths: len(paths),
	}
}

func buildP24ArtifactsWitness(artifacts []SecurityReviewArtifact) SecurityReviewGateV1Witness {
	witness := SecurityReviewGateV1Witness{
		ID:    p24SecurityReviewArtifactsWitnessID,
		Kind:  "security_review_artifacts",
		Paths: make([]string, 0, len(artifacts)),
	}
	for _, artifact := range artifacts {
		witness.Paths = append(witness.Paths, artifact.Path)
		switch artifact.Path {
		case "docs/audits/security-review.md":
			witness.SecurityReviewArtifactPresent = artifact.Present
		case "docs/audits/threat-model.md":
			witness.ThreatModelArtifactPresent = artifact.Present
		case "docs/audits/unsafe-surface-map.md":
			witness.UnsafeSurfaceMapPresent = artifact.Present
		case "docs/audits/capability-surface-map.md":
			witness.CapabilitySurfaceMapPresent = artifact.Present
		}
	}
	return witness
}

func p24SecurityReviewValidateRowsAndWitnesses(rows []SecurityReviewGateV1Row, witnesses []SecurityReviewGateV1Witness) error {
	byWitness := map[string]SecurityReviewGateV1Witness{}
	for _, witness := range witnesses {
		if strings.TrimSpace(witness.ID) == "" || strings.TrimSpace(witness.Kind) == "" {
			return fmt.Errorf("security review gate v1: witness missing id or kind")
		}
		if _, exists := byWitness[witness.ID]; exists {
			return fmt.Errorf("security review gate v1: duplicate witness %q", witness.ID)
		}
		byWitness[witness.ID] = witness
	}
	expected := map[SecurityReviewGateV1ID]bool{}
	for _, id := range p24SecurityReviewGateV1IDs() {
		expected[id] = true
	}
	seen := map[SecurityReviewGateV1ID]bool{}
	for _, row := range rows {
		if !expected[row.ID] {
			return fmt.Errorf("security review gate v1: unexpected row %q", row.ID)
		}
		if seen[row.ID] {
			return fmt.Errorf("security review gate v1: duplicate row %q", row.ID)
		}
		seen[row.ID] = true
		if strings.TrimSpace(row.Name) == "" || strings.TrimSpace(row.Status) == "" {
			return fmt.Errorf("security review gate v1: row %q missing name or status", row.ID)
		}
		if len(row.Evidence) == 0 || len(row.Tests) == 0 || len(row.Boundaries) == 0 || len(row.WitnessIDs) == 0 {
			return fmt.Errorf("security review gate v1: row %q missing evidence, tests, boundaries, or witness ids", row.ID)
		}
		for _, text := range append(append(append([]string{}, row.Evidence...), row.Tests...), row.Boundaries...) {
			if p24SecurityReviewIsPlaceholder(text) {
				return fmt.Errorf("security review gate v1: row %q has placeholder evidence", row.ID)
			}
		}
		for _, id := range row.WitnessIDs {
			if _, ok := byWitness[id]; !ok {
				return fmt.Errorf("security review gate v1: row %q references missing witness %q", row.ID, id)
			}
		}
	}
	for _, id := range p24SecurityReviewGateV1IDs() {
		if !seen[id] {
			return fmt.Errorf("security review gate v1: missing row %q", id)
		}
	}
	if !byWitness[p24SecurityReviewUnsafeWitnessID].UnsafeAPISurfaceReviewed {
		return fmt.Errorf("security review gate v1: unsafe API witness incomplete")
	}
	if !byWitness[p24SecurityReviewCapabilityWitnessID].CapabilitySurfaceReviewed {
		return fmt.Errorf("security review gate v1: capability witness incomplete")
	}
	allocator := byWitness[p24SecurityReviewAllocatorWitnessID]
	if !allocator.MemoryAllocatorReviewed || allocator.RuntimeAllocationContracts < 5 || allocator.RawPointerBoundsMetadataVersion != "raw-pointer-bounds-v1" {
		return fmt.Errorf("security review gate v1: memory allocator witness incomplete")
	}
	network := byWitness[p24SecurityReviewNetworkWitnessID]
	if !network.NetworkRuntimeReviewed || network.IOReactorRows < 10 {
		return fmt.Errorf("security review gate v1: network runtime witness incomplete")
	}
	actor := byWitness[p24SecurityReviewActorWitnessID]
	if !actor.ActorRuntimeReviewed || actor.ActorBoundaryRows < 4 {
		return fmt.Errorf("security review gate v1: actor runtime witness incomplete")
	}
	db := byWitness[p24SecurityReviewDBWitnessID]
	if !db.DBProtocolReviewed || db.ProductionPostgresRows < 8 {
		return fmt.Errorf("security review gate v1: DB protocol witness incomplete")
	}
	eco := byWitness[p24SecurityReviewEcoWitnessID]
	if !eco.PackageEcoSystemReviewed || eco.EcoValidatorPaths < 5 {
		return fmt.Errorf("security review gate v1: package/Eco witness incomplete")
	}
	build := byWitness[p24SecurityReviewBuildWitnessID]
	if !build.BuildScriptsReviewed || build.ReleaseSecurityScripts < 3 {
		return fmt.Errorf("security review gate v1: build scripts witness incomplete")
	}
	supply := byWitness[p24SecurityReviewSupplyWitnessID]
	if !supply.SupplyChainReviewed || supply.SupplyChainEvidencePaths < 5 {
		return fmt.Errorf("security review gate v1: supply chain witness incomplete")
	}
	artifacts := byWitness[p24SecurityReviewArtifactsWitnessID]
	if !artifacts.SecurityReviewArtifactPresent || !artifacts.ThreatModelArtifactPresent || !artifacts.UnsafeSurfaceMapPresent || !artifacts.CapabilitySurfaceMapPresent {
		return fmt.Errorf("security review gate v1: security review artifacts witness incomplete")
	}
	return nil
}

func p24SecurityReviewValidateArtifacts(report SecurityReviewGateV1Report) error {
	if !report.SecurityReviewArtifactPresent {
		return fmt.Errorf("security review gate v1: docs/audits/security-review.md artifact missing")
	}
	if !report.ThreatModelArtifactPresent {
		return fmt.Errorf("security review gate v1: docs/audits/threat-model.md artifact missing")
	}
	if !report.UnsafeSurfaceMapPresent {
		return fmt.Errorf("security review gate v1: docs/audits/unsafe-surface-map.md artifact missing")
	}
	if !report.CapabilitySurfaceMapPresent {
		return fmt.Errorf("security review gate v1: docs/audits/capability-surface-map.md artifact missing")
	}
	present := map[string]bool{}
	for _, artifact := range report.Artifacts {
		if strings.TrimSpace(artifact.Kind) == "" || strings.TrimSpace(artifact.Path) == "" {
			return fmt.Errorf("security review gate v1: artifact missing kind or path")
		}
		present[artifact.Path] = artifact.Present
	}
	for _, path := range []string{
		"docs/audits/security-review.md",
		"docs/audits/threat-model.md",
		"docs/audits/unsafe-surface-map.md",
		"docs/audits/capability-surface-map.md",
	} {
		if !present[path] {
			return fmt.Errorf("security review gate v1: required artifact %s missing", path)
		}
	}
	return nil
}

func p24SecurityReviewGateV1IDs() []SecurityReviewGateV1ID {
	return []SecurityReviewGateV1ID{
		SecurityReviewUnsafeAPISurface,
		SecurityReviewCapabilitySurface,
		SecurityReviewMemoryAllocator,
		SecurityReviewNetworkRuntime,
		SecurityReviewActorRuntime,
		SecurityReviewDBProtocol,
		SecurityReviewPackageEcoSystem,
		SecurityReviewBuildScripts,
		SecurityReviewSupplyChain,
		SecurityReviewArtifactSet,
	}
}

func p24SecurityReviewGateRow(id SecurityReviewGateV1ID, name, status string, evidence, tests, boundaries, witnessIDs []string) SecurityReviewGateV1Row {
	return SecurityReviewGateV1Row{
		ID:         id,
		Name:       name,
		Status:     status,
		Evidence:   evidence,
		Tests:      tests,
		Boundaries: boundaries,
		WitnessIDs: witnessIDs,
	}
}

func p24SecurityReviewArtifacts() []SecurityReviewArtifact {
	return []SecurityReviewArtifact{
		p24SecurityReviewArtifact("security_review", "docs/audits/security-review.md"),
		p24SecurityReviewArtifact("threat_model", "docs/audits/threat-model.md"),
		p24SecurityReviewArtifact("unsafe_surface_map", "docs/audits/unsafe-surface-map.md"),
		p24SecurityReviewArtifact("capability_surface_map", "docs/audits/capability-surface-map.md"),
	}
}

func p24SecurityReviewArtifact(kind string, rel string) SecurityReviewArtifact {
	_, err := os.Stat(p24RepoPath(rel))
	return SecurityReviewArtifact{
		Kind:    kind,
		Path:    rel,
		Present: err == nil,
	}
}

func p24AllRepoPathsExist(paths []string) bool {
	for _, path := range paths {
		if _, err := os.Stat(p24RepoPath(path)); err != nil {
			return false
		}
	}
	return true
}

func p24RepoPath(rel string) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return filepath.FromSlash(rel)
	}
	return filepath.Join(filepath.Dir(filepath.Dir(file)), filepath.FromSlash(rel))
}

func p24SecurityReviewHasString(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}

func p24SecurityReviewIsPlaceholder(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "" ||
		lower == "todo" ||
		lower == "tbd" ||
		strings.Contains(lower, "placeholder")
}
