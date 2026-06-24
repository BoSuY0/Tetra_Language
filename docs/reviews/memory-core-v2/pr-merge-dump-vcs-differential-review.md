# PR Merge Dump/VCS Differential Review

reviewed_head

`48b4b45e03ef356ef4bfc65748700e6ac1eb5064`

reviewed_merge_commit

`df54bc2ff05f2a14c097705cebce56bc0c651d7f`

branch_tree

`429e926e5983f8a9ea642f41df04f91300362934`

merge_tree

`2ce63da17af71cbc9eb025d9f1a199284320b76f`

non_dump_trees_equivalent

`true`. `git diff --name-status` excluding `dumps/**` was empty.

dump_files

`10` tracked files were added under `dumps/` in the PR merge tree:
`tetra_language_dump_20260622_105404Z_part_001.md` through
`tetra_language_dump_20260622_105404Z_part_010.md`.

dump_total_bytes

`47,376,491`

single_parent_result

Single-parent commit shape is rejected as causal. `case-a` exact and package
`test_all` runs passed without dumps; `case-b` exact, package, and production
baseline runs passed with the dump tree.

merge_parent_result

Merge-parent shape is rejected as causal. `case-c` used the feature tree with
two parents and passed exact, package, and production baseline runs. `case-d`
used the actual PR merge tree and also passed exact, package, and production
baseline runs in a clean env.

dump_absent_result

Dump absence did not reproduce the failure. `case-a` and `case-c` exact and
package `test_all` runs passed.

dump_present_result

Dump presence did not reproduce the failure. `case-b` and `case-d` exact,
package `test_all`, and production baseline runs passed.

buildvcs_on_result

Pass in all four matrix cases for the exact primary test.

buildvcs_off_result

Pass in all four matrix cases for the exact primary test.

causal_dimension

`test_state_ambient_env_inheritance`

Rejected dimensions:
- `dump_tree_content`
- `merge_commit_shape_or_vcs_metadata`
- `go_vcs_metadata`
- `github_pr_environment`

root_cause

The failure is caused by test harness state, not by the PR merge tree. The
`test_all` fake-repo helpers inherited ambient process env from `os.Environ()`,
including test-only fake controls read by the fake `go` script. On the actual
PR merge tree, clean env and PR-like `CI/GITHUB_*` env passed, while ambient
`TETRA_FAKE_SKIP_UNSAFE_PROMOTION_LIST=1` plus
`TETRA_FAKE_SKIP_RAM_CONTRACT_LIST=1` reproduced both observed CI failures.

minimal_fix

Filter inherited fake test controls in `runTestAll`, `runTestAllSplit`, and
`runTestAllFromWorkingDir`, while preserving explicit test env overrides.

regression_test

Add a fake-repo quick test with ambient `TETRA_FAKE_*` controls set in the
parent test process and assert the normal quick run passes. Keep existing
explicit-env negative tests as proof that intentional fake failure controls are
still honored.

verdict

TEST_STATE_CAUSE_CONFIRMED
