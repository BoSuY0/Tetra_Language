package islandkernel

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestIslandKernelRequiredDecisionQuestions(t *testing.T) {
	liveRef := MemoryRef{
		BaseID:     "buf",
		IslandID:   "island:a",
		Epoch:      7,
		Provenance: ProvenanceBorrowedView,
		Bounds:     Bounds{Known: true, InBounds: true},
		AliasState: AliasUniqueLocal,
	}
	liveToken := Token{IslandID: "island:a", Epoch: 7, OwnerID: "fn:main"}
	staleToken := Token{IslandID: "island:a", Epoch: 8, OwnerID: "fn:main"}
	boundsProof := Proof{
		ID:            "proof:bounds:a",
		Kind:          ProofBounds,
		SubjectBaseID: "buf",
		IslandID:      "island:a",
		Epoch:         7,
		Operation:     OperationIndexLoad,
		Verified:      true,
	}
	storageProof := Proof{
		ID:            "proof:storage:a",
		Kind:          ProofStorage,
		SubjectBaseID: "alloc:a",
		IslandID:      "island:a",
		Epoch:         7,
		Operation:     OperationExplicitIslandStorage,
		Verified:      true,
	}

	tests := []struct {
		name string
		got  Result
		want Decision
		code string
	}{
		{
			name: "borrow same island live epoch accepts",
			got:  CanBorrow(BorrowRequest{Ref: liveRef, Token: liveToken}),
			want: Accept,
			code: "borrow.live_epoch",
		},
		{
			name: "borrow stale epoch rejects",
			got:  CanBorrow(BorrowRequest{Ref: liveRef, Token: staleToken}),
			want: Reject,
			code: "borrow.stale_epoch",
		},
		{
			name: "return borrowed local rejects",
			got:  CanReturn(EscapeRequest{Ref: liveRef}),
			want: Reject,
			code: "escape.return_borrow",
		},
		{
			name: "store global borrowed ref rejects",
			got:  CanStoreGlobal(EscapeRequest{Ref: liveRef}),
			want: Reject,
			code: "escape.global_borrow",
		},
		{
			name: "capture borrowed ref conservatively rejects",
			got:  CanCaptureClosure(EscapeRequest{Ref: liveRef}),
			want: Reject,
			code: "escape.closure_borrow",
		},
		{
			name: "actor borrowed boundary rejects",
			got:  CanSendToActor(BoundaryRequest{Ref: liveRef, Transfer: TransferBorrowedView}),
			want: Reject,
			code: "boundary.actor_borrow",
		},
		{
			name: "task borrowed boundary rejects",
			got:  CanSendToTask(BoundaryRequest{Ref: liveRef, Transfer: TransferBorrowedView}),
			want: Reject,
			code: "boundary.task_borrow",
		},
		{
			name: "owned island move accepts and consumes source token",
			got:  CanMoveIsland(TokenRequest{Token: liveToken}),
			want: Accept,
			code: "token.move_consumes_source",
		},
		{
			name: "free with live borrows rejects",
			got:  CanFreeIsland(TokenRequest{Token: liveToken, LiveBorrows: 1}),
			want: Reject,
			code: "token.free_live_borrows",
		},
		{
			name: "reset advances epoch",
			got:  CanResetIsland(TokenRequest{Token: liveToken}),
			want: Accept,
			code: "token.reset_epoch_advanced",
		},
		{
			name: "noalias across distinct proven islands accepts narrowly",
			got: CanClaimNoAlias(NoAliasRequest{
				Left: liveRef,
				Right: MemoryRef{
					BaseID:     "other",
					IslandID:   "island:b",
					Epoch:      3,
					Provenance: ProvenanceOwned,
					AliasState: AliasUniqueLocal,
				},
				Proof: Proof{
					ID:            "proof:noalias",
					Kind:          ProofNoAlias,
					SubjectBaseID: "buf",
					IslandID:      "island:a",
					Epoch:         7,
					Operation:     OperationNoAlias,
					Verified:      true,
				},
			}),
			want: Accept,
			code: "noalias.distinct_proven_islands",
		},
		{
			name: "noalias with external unsafe rejects",
			got: CanClaimNoAlias(NoAliasRequest{
				Left: liveRef,
				Right: MemoryRef{
					BaseID:      "raw",
					IslandID:    ExternalUnsafeIsland,
					Epoch:       1,
					Provenance:  ProvenanceUnsafeUnknown,
					UnsafeClass: UnsafeUnknown,
				},
			}),
			want: Reject,
			code: "noalias.unsafe_external",
		},
		{
			name: "bounds elimination requires verified proof",
			got: CanEliminateBoundsCheck(
				ProofRequest{Ref: liveRef, Proof: boundsProof, Operation: OperationIndexLoad},
			),
			want: Accept,
			code: "bounds.proof_verified",
		},
		{
			name: "bounds elimination missing proof rejects",
			got: CanEliminateBoundsCheck(
				ProofRequest{Ref: liveRef, Operation: OperationIndexLoad},
			),
			want: Reject,
			code: "bounds.missing_proof",
		},
		{
			name: "explicit island lowering accepts no escape with storage proof",
			got: CanLowerAsExplicitIsland(StorageRequest{
				Ref:             liveRef,
				PlannedStorage:  StorageExplicitIsland,
				ActualStorage:   StorageExplicitIsland,
				Proof:           storageProof,
				EscapesLifetime: false,
			}),
			want: Accept,
			code: "storage.explicit_island_trusted",
		},
		{
			name: "explicit island lowering rejects escape",
			got: CanLowerAsExplicitIsland(StorageRequest{
				Ref:             liveRef,
				PlannedStorage:  StorageExplicitIsland,
				ActualStorage:   StorageExplicitIsland,
				Proof:           storageProof,
				EscapesLifetime: true,
			}),
			want: Reject,
			code: "storage.explicit_island_escape",
		},
		{
			name: "unsafe unknown promotion rejects",
			got: CanPromoteUnsafeRoot(
				UnsafeRequest{
					Ref: MemoryRef{
						BaseID:      "raw",
						IslandID:    ExternalUnsafeIsland,
						Epoch:       1,
						Provenance:  ProvenanceUnsafeUnknown,
						UnsafeClass: UnsafeUnknown,
					},
				},
			),
			want: Reject,
			code: "unsafe.unknown_promotion",
		},
		{
			name: "trusted storage rejects heap fallback promotion",
			got: CanTrustStorage(StorageRequest{
				Ref:            liveRef,
				PlannedStorage: StorageExplicitIsland,
				ActualStorage:  StorageHeap,
				Proof:          storageProof,
			}),
			want: Reject,
			code: "storage.heap_fallback_not_trusted",
		},
		{
			name: "erase runtime check requires verified proof",
			got: CanEraseRuntimeCheck(ProofRequest{
				Ref:       liveRef,
				Proof:     boundsProof,
				Operation: OperationIndexLoad,
			}),
			want: Accept,
			code: "runtime_check.erase_verified",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requireDecision(t, test.got, test.want, test.code)
		})
	}
}

func TestIslandKernelDecisionMetadata(t *testing.T) {
	token := Token{IslandID: "island:a", Epoch: 7, OwnerID: "fn:main"}
	move := CanMoveIsland(TokenRequest{Token: token})
	requireDecision(t, move, Accept, "token.move_consumes_source")
	if !move.ConsumesToken {
		t.Fatalf("CanMoveIsland() = %+v, want ConsumesToken", move)
	}

	free := CanFreeIsland(TokenRequest{Token: token})
	requireDecision(t, free, Accept, "token.free_consumes_source")
	if !free.ConsumesToken {
		t.Fatalf("CanFreeIsland() = %+v, want ConsumesToken", free)
	}

	reset := CanResetIsland(TokenRequest{Token: token})
	requireDecision(t, reset, Accept, "token.reset_epoch_advanced")
	if !reset.ConsumesToken {
		t.Fatalf("CanResetIsland() = %+v, want ConsumesToken", reset)
	}
	if reset.NextEpoch != 8 {
		t.Fatalf("CanResetIsland() = %+v, want NextEpoch 8", reset)
	}
}

func TestIslandKernelRejectsExternalUnsafeTrustedStoragePromotion(t *testing.T) {
	externalRef := MemoryRef{
		BaseID:      "raw",
		IslandID:    ExternalUnsafeIsland,
		Epoch:       1,
		Provenance:  ProvenanceUnsafeUnknown,
		UnsafeClass: UnsafeUnknown,
	}
	storageProof := Proof{
		ID:            "proof:storage:external",
		Kind:          ProofStorage,
		SubjectBaseID: "raw",
		IslandID:      ExternalUnsafeIsland,
		Epoch:         1,
		Operation:     OperationExplicitIslandStorage,
		Verified:      true,
	}

	lowerAsIsland := CanLowerAsExplicitIsland(StorageRequest{
		Ref:            externalRef,
		PlannedStorage: StorageExplicitIsland,
		ActualStorage:  StorageExplicitIsland,
		Proof:          storageProof,
	})
	requireDecision(t, lowerAsIsland, Reject, "storage.unsafe_external")

	trustedStorage := CanTrustStorage(StorageRequest{
		Ref:            externalRef,
		PlannedStorage: StorageExplicitIsland,
		ActualStorage:  StorageExplicitIsland,
		Proof:          storageProof,
	})
	requireDecision(t, trustedStorage, Reject, "storage.unsafe_external")
}

func TestIslandKernelDangerousDecisionRouteCoverage(t *testing.T) {
	routes := DangerousDecisionRoutes()
	if err := ValidateDangerousDecisionRoutes(
		routes,
		fileExistsFromRepoRoot(t),
		fileContainsTokenFromRepoRoot(t),
	); err != nil {
		t.Fatalf("ValidateDangerousDecisionRoutes: %v", err)
	}
	t.Logf("validated %d IslandKernel dangerous decision routes", len(routes))

	seen := map[string]DangerousDecisionRoute{}
	for _, route := range routes {
		if _, exists := seen[route.Decision]; exists {
			t.Fatalf("duplicate route for decision %q", route.Decision)
		}
		seen[route.Decision] = route
	}

	for _, decision := range RequiredDangerousDecisions() {
		route, exists := seen[decision]
		if !exists {
			t.Fatalf("missing route for required dangerous decision %q", decision)
		}
		if !slices.Contains(route.EvidenceTokens, decision) && route.KernelFunction != decision {
			t.Fatalf("route %q does not name its kernel decision in evidence: %+v", decision, route)
		}
	}
}

func fileExistsFromRepoRoot(t *testing.T) func(string) bool {
	t.Helper()
	root := repoRootFromIslandKernelTest(t)
	return func(path string) bool {
		if strings.TrimSpace(path) == "" || filepath.IsAbs(path) {
			return false
		}
		info, err := os.Stat(filepath.Join(root, filepath.Clean(path)))
		return err == nil && !info.IsDir()
	}
}

func fileContainsTokenFromRepoRoot(t *testing.T) func(string, string) bool {
	t.Helper()
	root := repoRootFromIslandKernelTest(t)
	return func(path string, token string) bool {
		if strings.TrimSpace(token) == "" {
			return false
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.Clean(path)))
		return err == nil && strings.Contains(string(data), token)
	}
}

func repoRootFromIslandKernelTest(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

func requireDecision(t *testing.T, got Result, want Decision, code string) {
	t.Helper()
	if got.Decision != want {
		t.Fatalf("decision = %+v, want %s", got, want)
	}
	if got.Reason.Code != code {
		t.Fatalf("reason = %+v, want code %q", got.Reason, code)
	}
	if got.Reason.Message == "" {
		t.Fatalf("reason = %+v, want reviewable message", got.Reason)
	}
}
