# MPC12-S2 Report And Runtime Claims Audit Result

Status: integrated.

Kierkegaard completed the read-only report/runtime-claims audit. Key accepted
findings:

- No source claimed distributed or full production actor-runtime zero-copy.
- `compiler/reports.go` did emit `zero_copy_move` rows for local actor-transfer
  reports without a machine-readable `claim_level`, which became MPC-12 RED
  coverage.
- `actorsafety` and `actorsrt` audits already reject distributed zero-copy and
  runtime-behavior-change claims; MPC-12 keeps those non-claims intact.
- Docs already scoped actor zero-copy to local typed mailbox evidence, but
  needed explicit MPC-12 wording for `evidence_only` report rows and
  production-runtime non-validation.

Commands were read-only `rg`, `sed`, and `nl` inspections across
`compiler/reports.go`, `compiler/internal/actorsafety`,
`compiler/internal/actorsrt`, `compiler/internal/parallelrt`, validators, and
docs.
