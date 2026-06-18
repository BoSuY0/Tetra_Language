# Gate Contract v1

Status: minimal release validation contract schema.

`tetra.gate-contract.v1` is the machine-readable source of truth for a release gate. It lists what
the gate runs, which reports must exist, which validators check those reports, which claims the
reports support, which nonclaims remain explicitly out of scope, and which artifacts CI must
preserve.

The Go model and validator live in `tools/internal/gatecontract`.

## Required Top-Level Fields

- `schema`: must be `tetra.gate-contract.v1`.
- `id`: stable contract identifier.
- `title`: human-readable gate title.
- `scope`: product or release slice covered by this contract.
- `producer`: script, tool, or workflow responsible for producing the evidence.
- `entrypoint`: local command or script users and CI should invoke for this gate.
- `fresh_report_dir_policy`: named policy for report directory freshness, such as requiring a new or
  empty directory before evidence is written.
- `host_preconditions`: host assumptions that must hold before the gate runs.
- `steps`: ordered gate steps.
- `required_reports`: reports that must be produced and validated.
- `validators`: validator commands available to steps and reports.
- `artifact_hashes`: artifact hash policy for report and CI evidence.
- `claims`: positive claims that can be made only when referenced reports pass.
- `nonclaims`: explicit statements the gate does not prove.
- `ci_artifacts`: artifacts CI must retain or upload for this gate.

## Steps

Each `steps` entry requires:

- `id`: unique step identifier.
- `kind`: step category, for example `go-test`, `go-run`, or `shell`.
- `command`: command string used by the producer or runner.
- `working_dir`: working directory for the command.
- `required`: whether failure blocks the gate.
- `report_outputs`: report paths expected from the step.
- `validator_refs`: validator IDs used by the step.
- `host_preconditions`: extra host assumptions for this step.
- `blocked_status_policy`: how a blocked step is represented in gate status.

Step IDs must be unique. Every `validator_refs` entry must point at an existing validator ID.

## Required Reports

Each `required_reports` entry requires:

- `path`: unique report path.
- `schema`: report schema expected at that path.
- `validator`: validator ID that checks the report.
- `same_commit_required`: whether the report must match the current commit.
- `artifact_hash_required`: whether the report must be included in artifact hash evidence.
- `claim_refs`: claim IDs supported by the report.

Report paths must be unique. The report validator must exist. Every `claim_refs` entry must point at
an existing claim ID.

When `artifact_hashes.enabled` or `artifact_hashes.required` is true, every required report must set
`artifact_hash_required` to true. This prevents a release from claiming hash-covered evidence while
an individual required report quietly opts out.

## Validators, Claims, Nonclaims, And CI Artifacts

Validators require unique `id` values plus `kind` and `command`. They are the bridge from the
contract to Go validators or other local validation commands.

Claims require unique `id` values and a `statement`. A release document should only make a claim
when the contract has a passing required report that references that claim.

Nonclaims require `id` and `statement`. They keep known limitations visible next to positive claims,
so docs and handoff notes do not accidentally promote local evidence into a broader production
guarantee.

CI artifacts record paths that the workflow must retain. The contract validator checks their basic
shape; later runners can use the same list to upload or hash the exact evidence set.

## Drift Prevention

The contract prevents drift by making one file name the relationship between:

- scripts and entrypoints that produce evidence;
- validators that check report schemas;
- report paths emitted by gate steps;
- artifact hash policy applied to those reports;
- docs claims supported by validated reports;
- nonclaims that must remain visible in release notes;
- CI artifacts that preserve the evidence.

Strict JSON decoding rejects unknown fields, and validation rejects missing required fields,
duplicate step IDs, duplicate report paths, duplicate validator IDs, missing validator references,
missing claim references, and required reports that opt out of hashes while contract-level hashes
are enabled.
