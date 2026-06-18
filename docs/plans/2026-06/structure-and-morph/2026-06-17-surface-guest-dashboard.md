# Surface Guest Dashboard Implementation Plan

**Goal:** Add a repo-native Tetra Surface example that recreates the guest personal dashboard
empty-state page from the provided screenshot.

**Context:** The existing Surface/Morph pipeline can validate Morph-authored Block scenes and
renderer-owned command streams. This change adds a narrow visual example rather than a new
production claim.

## Task 1 - Add A Focused Smoke Test

- **Goal:** Lock the expected guest dashboard shape before implementation.
- **Files:** `tools/cmd/surface-runtime-smoke/surface_smoke_suite_test.go`.
- **Approach:** Add a test for `examples/surface/morph_flagship/surface_morph_guest_dashboard.tetra`
  that requires a Morph source, Block scene nodes for the recent-courses panel, course-overview
  panel, and both empty states.
- **Verification:** `go test ./tools/cmd/surface-runtime-smoke -run TestGuestDashboard -count=1`.
- **Done when:** The test fails before implementation because the source/scenario does not exist.

## Task 2 - Add The Surface Example

- **Goal:** Create a Morph-over-Block example with the guest dashboard structure.
- **Files:** Add `examples/surface/morph_flagship/surface_morph_guest_dashboard.tetra`.
- **Approach:** Follow existing Morph examples: use `surface`, `block`, and `morph`; create a Block
  tree with a root, page title, recent-courses panel, course-overview panel, divider, empty-state
  icons, headline, and supporting copy.
- **Verification:** Build the source through the existing Surface path used by smoke tests.
- **Done when:** The example compiles and returns success for its self-check.

## Task 3 - Wire The Smoke Scenario

- **Goal:** Make `surface-runtime-smoke` produce source-specific evidence for the guest dashboard.
- **Files:** `tools/cmd/surface-runtime-smoke/surface_smoke_scenarios.go`; possibly
  `tools/internal/surfacerender/stream.go` if source-specific render commands are needed.
- **Approach:** Add a narrow special-case source detector and scenario builder that reuses the
  existing Morph evidence machinery while exposing guest-dashboard Block graph nodes and
  accessibility metadata.
- **Verification:** The focused smoke test passes and `surface.ValidateReport` accepts the generated
  report.
- **Done when:** The report has `Block` as the core primitive and contains dashboard panels/empty
  states.

## Task 4 - Verify And Refresh Graph

- **Goal:** Produce current evidence without claiming a full release.
- **Files:** No product-gate report changes expected.
- **Approach:** Run focused Go tests and a direct `surface-runtime-smoke` command for the new
  source. Then run `graphify update .` as required after code changes.
- **Verification:** Commands exit 0.
- **Done when:** The final response reports exact commands and remaining nonclaims.
