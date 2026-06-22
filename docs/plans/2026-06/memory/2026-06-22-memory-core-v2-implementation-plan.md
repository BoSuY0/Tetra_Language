# Tetra Memory Core v2 — строгий план реалізації

**Тип документа:** execution plan для агента-імплементатора
**Мова виконання та звітів:** українська; identifiers, commands, paths і code symbols залишати англійською
**Базовий snapshot:** dump `2026-06-22T10:54:05Z`, Git HEAD `8ccb413f0be7b2c83715ca20e57f4e2c5036217a`
**Ціль:** реалізувати єдине виконуване ядро істини про пам’ять: `semantics -> PLIR -> MemoryFactGraph -> allocation plan -> lowering -> optimizer -> validation -> runtime domains/backend -> reports`
**Головний принцип:** reports нічого не вирішують; вони лише проєктують рішення, які вже були використані normal build pipeline.

---

## 1. Місія

Результатом має бути **Memory Core v2 — FactGraph-driven Memory Domains**:

```text
Checked semantics
      ↓
PLIR with compiler-owned memory evidence
      ↓
Canonical MemoryFactGraph + immutable Snapshot
      ↓
Fact-driven AllocationPlan
      ↓
Plan-driven Lowering + actual lowering evidence
      ↓
Proof-driven Optimizer + explicit invalidation delta
      ↓
Validation + runtime MemoryDomain/MemoryBackend evidence
      ↓
Reports as read-only projections
```

`island` є першим повністю завершеним `MemoryDomain`: identity, owner, parent, lifetime, epoch, budget, requested/reserved/committed/current/peak/released bytes, transfer, reset, close, stale-reference rejection.

Необхідно усунути такі класи дефектів:

1. `MemoryFactGraph` створюється лише для `--emit-memory-report` і після реального lowering.
2. PLIR, allocation plan і lowering виконуються повторно в normal build, module build і report path.
3. Існують дубльовані `from_plir_*` реалізації та паралельні memory policy layers.
4. `allocplan` частково повторно виводить escape/domain facts через локальні scans і string heuristics.
5. `ActualLoweringStorage` прогнозується planner-ом замість фіксації фактичного lowering.
6. `MemoryBackend` і `MemoryDomain` переважно описують contracts/reports, але не утворюють єдиний lifecycle/accounting substrate.
7. `islandkernel` має багато `validated_equivalent` routes замість прямої участі у небезпечних рішеннях.
8. optimizer не споживає canonical memory proofs та повторно виводить alias/bounds/storage умови.
9. `memorymodel/mini.go`, `memoryvocab`, `ramcontract`, reports і validators мають shadow policy або дубльовану vocabulary.
10. Report flags можуть запускати окрему compilation path, а це допускає decision drift.

---

## 2. Непорушні архітектурні рішення

Агент **не має права змінювати** ці рішення без окремої письмової поправки до цього плану, яку затвердив parent controller до редагування коду.

### 2.1. Канонічне джерело істини

Канонічним compiler-side джерелом істини залишається package:

```text
compiler/internal/memoryfacts
```

Він володіє:

- typed memory vocabulary;
- `Fact`, `Graph`, `Snapshot`, proof queries і deltas;
- invariant validation;
- deterministic digest;
- report projection `tetra.memory-report.v1`.

Він **не імпортує**:

```text
plir
allocplan
lower
opt
validation
runtimeabi
semantics
```

Після завершення core package має імпортувати тільки standard library.

### 2.2. Adapter packages

Перетворення між stages живуть лише в окремих adapters:

```text
compiler/internal/memoryfacts/fromplir
compiler/internal/memoryfacts/fromallocplan
compiler/internal/memoryfacts/fromlowering
compiler/internal/memoryfacts/fromvalidation
compiler/internal/memoryfacts/fromoptimizer
```

Adapters можуть імпортувати stage packages. Core package не може імпортувати adapters.

### 2.3. Normal build завжди має canonical memory state

Для executable build canonical memory state створюється незалежно від:

```text
--explain
--emit-plir
--emit-proof
--emit-alloc-report
--emit-bounds-report
--emit-memory-report
--emit-ram-contract-report
```

Report flags не можуть змінювати PLIR, allocation decisions, lowering або optimization.

Винятки:

- `InterfaceOnly` build не виконує allocation/lowering; він повинен записати explicit `not_applicable` evidence.
- Повний cache hit може не виконувати lowering, але мусить перевірити cache-attested memory plan/lowering digests. Старий cache entry без цих digests вважається miss.

### 2.4. Одне PLIR, один plan, не більше одного lowering

Для одного build invocation:

- `plir.FromCheckedProgram` — рівно один раз;
- canonical allocation planning — рівно один раз;
- lowering — нуль разів при валідному full cache hit без report/optimization requirements, інакше рівно один раз для всього program;
- module workers отримують function slices із уже lowered program; вони не будують PLIR/plan повторно.

### 2.5. Planner не визначає actual lowering

`allocplan` визначає лише `PlannedStorage`.

До застосування `lower.LoweringEvidence`:

```text
ActualLoweringStorage = UnknownConservative
LoweringStatus = pending
```

Actual storage записується лише з evidence, створеного в тій branch lowering code, яка реально емітувала IR.

### 2.6. Optimizer не виводить memory truth заново

Memory-sensitive rewrite виконується лише за наявності canonical verified proof. Без proof rewrite пропускається із deterministic conservative reason code.

Optimizer повертає explicit delta:

- preserved proofs;
- consumed proofs;
- invalidated facts/proofs;
- new derived facts;
- decision rows із proof IDs.

### 2.7. Islands не є окремою паралельною моделлю

`islandkernel` використовує canonical `memoryfacts` types і є policy evaluator для island-specific dangerous decisions. Після реалізації applicable routes не можуть залишатися `RouteValidatedEquivalent`.

### 2.8. Compatibility schemas зберігаються

Не ламати без окремої versioned migration:

```text
tetra.memory-report.v1
existing allocation report schemas
existing RAM contract schemas
existing runtime allocation contract schemas
```

Нові поля додаються як optional. Старі JSON field names не перейменовуються в межах цього плану.

### 2.9. Ніяких benchmark-specific shortcuts

Заборонено:

- recognizer для конкретного test source;
- hard-coded `17/17` result;
- report-only fake evidence;
- validator weakening;
- Go-only backend model, представлений як real runtime backend;
- synthetic RSS/heap measurements, позначені як measured;
- broad `Memory 100%`, all-target parity або universal zero-heap claims.

---

## 3. Умови запуску та isolation

### 3.1. Execution activation

Цей план активується лише invocation-ом, який прямо каже виконувати:

```text
Tetra Memory Core v2 implementation plan
```

Поточний unrelated `GOAL.md` не змішувати з цією роботою.

### 3.2. Base commit guard

План розроблено для:

```bash
BASE_MEMORY_CORE_V2=8ccb413f0be7b2c83715ca20e57f4e2c5036217a
```

На старті:

```bash
git rev-parse --show-toplevel
git status --short --branch
git cat-file -e "$BASE_MEMORY_CORE_V2^{commit}"
git rev-parse HEAD
```

Якщо execution HEAD не дорівнює base commit:

1. Не редагувати код.
2. Створити `reports/stabilization/memory-core-v2/head-drift.md`.
3. Для кожного critical symbol із розділу 4 записати `present`, `renamed`, `deleted` або `behavior_changed`.
4. Parent controller або переносить execution у worktree від base commit, або вносить path/symbol amendment у цей plan.
5. Саб-агентам заборонено самостійно port-ити архітектуру на невідомий HEAD.

### 3.3. Worktree

Завжди використовувати окремий worktree; не stash/reset існуючу роботу:

```bash
git worktree add ../Tetra_Language-memory-core-v2 \
  -b feat/memory-core-v2 \
  "$BASE_MEMORY_CORE_V2"
cd ../Tetra_Language-memory-core-v2
```

Якщо branch або directory уже існує — зупинитися та зафіксувати blocker; не видаляти їх автоматично.

### 3.4. Workflow state

Створити:

```text
.workflow/memory-core-v2/GOAL.md
.workflow/memory-core-v2/CONTROL.md
.workflow/memory-core-v2/PLAN.md
.workflow/memory-core-v2/ATTEMPTS.md
.workflow/memory-core-v2/NOTES.md
```

Скопіювати цей документ у canonical repo path:

```text
docs/plans/2026-06/memory/2026-06-22-memory-core-v2-implementation-plan.md
```

Parent controller є єдиним writer для workflow files.

### 3.5. Cache discipline

Для Go evidence:

```bash
export GOTELEMETRY=off
export GOCACHE="$PWD/.cache/go-build-memory-core-v2"
export GOTMPDIR="$PWD/.cache/go-tmp-memory-core-v2"
mkdir -p "$GOCACHE" "$GOTMPDIR"
```

Заборонено використовувати `/tmp` як `GOCACHE`.

Після final evidence:

```bash
GOCACHE="$PWD/.cache/go-build-memory-core-v2" go clean -cache
rm -rf "$GOTMPDIR"
```

---

## 4. Critical symbols, які мають існувати до початку

Baseline audit має перевірити ці exact paths/symbols:

```text
compiler/compiler_facade.go
  BuildFileWithStatsOpt
  planNativeModuleBuild
  compileNativeModulePlan

compiler/compiler_reports.go
  emitExplainReports

compiler/internal/lower/lower_core.go
  LowerWithOptions
  LowerModuleWithOptions
  lowerCheckedFuncWithOptions

compiler/internal/allocplan/plan.go
  FromPLIRWithOptions
  planAllocation
  chooseStorage
  actualLoweringStorage

compiler/internal/allocplan/verify.go
  VerifyPlan

compiler/internal/memoryfacts/facts.go
  Fact

compiler/internal/memoryfacts/graph.go
  Graph
  AddFact
  DeriveFact
  InvalidateFact
  MarkValidated
  Validate

compiler/internal/memoryfacts/fromplir/from_plir.go
  FromPLIRAndAllocPlan

compiler/internal/memoryfacts/from_validation.go
  AddBoundsProofFacts

compiler/internal/memoryfacts/from_plir_allocplan.go
compiler/internal/memoryfacts/from_plir_borrow.go
compiler/internal/memoryfacts/from_plir_copy.go
compiler/internal/memoryfacts/from_plir_summary.go
compiler/internal/memoryfacts/from_plir_unsafe.go

compiler/internal/memorymodel/mini.go
  Evaluate

compiler/memoryvocab/vocab.go

compiler/internal/islandkernel/kernel.go
  CanBorrow
  CanReturn
  CanStoreGlobal
  CanCaptureClosure
  CanSendToActor
  CanSendToTask
  CanMoveIsland
  CanFreeIsland
  CanResetIsland
  CanClaimNoAlias
  CanEliminateBoundsCheck
  CanLowerAsExplicitIsland
  CanPromoteUnsafeRoot
  CanTrustStorage
  CanEraseRuntimeCheck

compiler/internal/islandkernel/coverage.go
  DangerousDecisionRoutes

compiler/internal/runtimeabi/memory_backend.go
  MemoryBackendContract
  MemoryBackendAllocationEvidence
  RuntimeMemoryBackendContract

compiler/internal/runtimeabi/memory_domain.go
  MemoryDomain
  ValidateMemoryDomain

compiler/internal/opt/opt_core.go
  Pass
  Manager
  Options
  RegisteredPasses
  RunWithOptions

compiler/internal/ramcontract/from_allocplan.go
  BuildReportFromAllocPlan
```

Baseline commands:

```bash
rg -n "func (BuildFileWithStatsOpt|planNativeModuleBuild|compileNativeModulePlan|emitExplainReports)" compiler
rg -n "func (LowerWithOptions|LowerModuleWithOptions|FromPLIRWithOptions|FromPLIRAndAllocPlan)" compiler
rg -n "RouteValidatedEquivalent|looksActorSend|looksTaskBoundary|actualLoweringStorage" compiler/internal
rg -n "compiler/internal/memoryfacts|compiler/memoryvocab|compiler/internal/memorymodel" compiler --glob '*.go'
rg -n "BuildReportFromAllocPlan|MemoryDomain" compiler/internal/ramcontract compiler/internal/runtimeabi
```

---

## 5. Target package dependency graph

Після `MCV2-T07` import direction має бути такою:

```text
memoryfacts                 -> stdlib only
islandkernel                -> memoryfacts
plir                        -> semantics (+ optional memoryfacts typed aliases only)
fromplir                    -> plir, memoryfacts
allocplan                   -> plir, memoryfacts, islandkernel, runtimeabi
fromallocplan               -> plir, allocplan, memoryfacts
lower                       -> semantics, allocplan, ir, runtimeabi
fromlowering                -> lower, memoryfacts
validation                  -> ir, plir, allocplan
fromvalidation              -> validation, memoryfacts
opt                         -> ir, lower verifier, validation, memoryfacts
fromoptimizer               -> opt, memoryfacts
memorypipeline              -> semantics, plir, fromplir, allocplan,
                               fromallocplan, lower evidence adapters,
                               validation adapters, optimizer adapters
ramcontract                 -> memoryfacts Snapshot, allocplan Plan,
                               runtimeabi domain snapshots
compiler                    -> memorypipeline, lower, opt, validation,
                               projections, codegen/cache/link
```

Заборонені edges:

```text
memoryfacts -> allocplan
memoryfacts -> plir
memoryfacts -> lower
memoryfacts -> validation
lower -> memorypipeline
opt -> memorypipeline
reports -> semantics decision APIs
ramcontract -> independent storage/domain classification
```

Додати architecture test, який парсить imports і падає при забороненому edge.

---

## 6. Обов’язкові target APIs

Наведені names і responsibilities є обов’язковими. Агент може змінити лише formatting або unexported helper names.

### 6.1. `memoryfacts` typed vocabulary

Додати до `compiler/internal/memoryfacts`:

```go
type Claim string
type ProofKind string

type ValueKey struct {
    FunctionID string
    ValueID    string
}

type AllocationKey struct {
    FunctionID       string
    AllocationSiteID string
}

type ProofKey struct {
    FunctionID string
    ProofID    string
}

const (
    ProofBounds       ProofKind = "bounds"
    ProofNoAlias      ProofKind = "noalias"
    ProofStorage      ProofKind = "storage"
    ProofNoEscape     ProofKind = "no_escape"
    ProofRegionAlive  ProofKind = "region_alive"
    ProofBorrow       ProofKind = "borrow"
    ProofDomainMove   ProofKind = "domain_move"
)
```

`Fact.Claim` змінити з `string` на `Claim`; JSON representation лишається string.

До `Fact` додати optional fields:

```go
LifetimeBirth string `json:"lifetime_birth,omitempty"`
LifetimeDeath string `json:"lifetime_death,omitempty"`
LifetimeOwner string `json:"lifetime_owner,omitempty"`
DecisionCode string `json:"decision_code,omitempty"`
```

До `SourceStage` додати:

```go
StageOptimization SourceStage = "optimization"
```

### 6.2. Immutable snapshot

```go
type Snapshot struct { /* unexported immutable indexes */ }

func (g *Graph) Snapshot() (Snapshot, error)
func (s Snapshot) ProgramID() string
func (s Snapshot) Facts() []Fact
func (s Snapshot) Fact(id FactID) (Fact, bool)
func (s Snapshot) FactsForValue(key ValueKey) []Fact
func (s Snapshot) FactsForAllocation(key AllocationKey) []Fact
func (s Snapshot) ResolveAllocation(key ValueKey) (AllocationEvidence, error)
func (s Snapshot) ResolveProof(query ProofQuery) (ProofEvidence, bool)
func (s Snapshot) Digest() string
```

`Facts()` і query methods повертають defensive copies.

```go
type AllocationEvidence struct {
    FunctionID       string
    ValueID          string
    AllocationSiteID string
    SiteID           string
    SourceSpan       string
    TypeName         string
    ProvenanceClass  ProvenanceClass
    UnsafeClass      UnsafeClass
    BorrowState      BorrowState
    EscapeState      EscapeState
    AliasState       AliasState
    RegionID         string
    IslandID         string
    Epoch            int
    OwnerID          string
    LifetimeBirth    string
    LifetimeDeath    string
    LifetimeOwner    string
    NoEscapeProofID  string
    StorageProofID   string
    SourceFactIDs    []FactID
}

type ProofQuery struct {
    FunctionID   string
    ProofID      string
    Kind         ProofKind
    SubjectBaseID string
    Operation    string
    IslandID     string
    Epoch        int
}

type ProofEvidence struct {
    FactID        FactID
    ProofID       string
    Kind          ProofKind
    SubjectBaseID string
    Operation     string
    IslandID      string
    Epoch         int
    ValidatorName string
    SourceStage   SourceStage
}
```

`ResolveProof` повертає `true` лише якщо proof:

- має exact kind/subject/operation/island/epoch match для непорожніх query fields;
- `ValidationState == ValidationPass`;
- не invalidated;
- не походить з `unsafe_unknown` для optimization-authorizing kinds;
- має compiler-owned `ValidatorName`.

### 6.3. Stage delta

```go
type Invalidation struct {
    FactID FactID
    Reason string
}

type ArtifactAttachment struct {
    FactID     FactID
    ArtifactID string
}

type ValidationMark struct {
    FactID        FactID
    ValidatorName string
}

type Delta struct {
    Stage        SourceStage
    Add          []Fact
    Invalidate   []Invalidation
    Attach       []ArtifactAttachment
    Validate     []ValidationMark
}

func (g *Graph) Apply(delta Delta) error
func (g *Graph) AdvanceTo(stage SourceStage) error
func (g *Graph) CurrentStage() SourceStage
```

Stage order:

```text
semantics < unsafe_gateway_lowering < plir < allocplan < lowering < optimization < validation
```

Не дозволяти додавати fact із stage, старішим за `CurrentStage`.

### 6.4. Builders/adapters

```go
// compiler/internal/memoryfacts/fromplir
func Build(programID string, prog *plir.Program) (*memoryfacts.Graph, error)

// compiler/internal/memoryfacts/fromallocplan
func Delta(
    snapshot memoryfacts.Snapshot,
    prog *plir.Program,
    plan *allocplan.Plan,
) (memoryfacts.Delta, error)

// compiler/internal/memoryfacts/fromlowering
func Delta(
    snapshot memoryfacts.Snapshot,
    evidence lower.LoweringEvidence,
) (memoryfacts.Delta, error)

// compiler/internal/memoryfacts/fromvalidation
func BoundsDelta(
    snapshot memoryfacts.Snapshot,
    report validation.ProofReport,
) (memoryfacts.Delta, error)

// compiler/internal/memoryfacts/fromoptimizer
func Delta(
    snapshot memoryfacts.Snapshot,
    report opt.Report,
) (memoryfacts.Delta, error)
```

### 6.5. Fact-driven allocation planner

```go
type Input struct {
    Program *plir.Program
    Facts   memoryfacts.Snapshot
}

func Build(input Input, opt Options) (*Plan, error)
func VerifyPlanned(plan *Plan) error
func VerifyLowered(plan *Plan) error
```

До `allocplan.Allocation` додати optional fields:

```go
SourceFactIDs []memoryfacts.FactID `json:"source_fact_ids,omitempty"`
ProofIDs      []string             `json:"proof_ids,omitempty"`
DecisionCode  string               `json:"decision_code,omitempty"`
PlanDigest    string               `json:"plan_digest,omitempty"`
```

`Build` зобов’язаний:

1. Для кожного `plir.ValueAllocIntent` викликати `Facts.ResolveAllocation`.
2. Падати при missing або ambiguous evidence.
3. Не сканувати operation names для actor/task escape.
4. Не використовувати `looksActorSend`, `looksTaskBoundary` або їхні renamed аналоги.
5. Не встановлювати predicted actual storage.
6. Зберігати source fact IDs і decision code.

### 6.6. Plan-driven lowering

```go
type AllocationLoweringEvidence struct {
    FunctionID       string
    Module           string
    AllocationID     string
    ValueID          string
    SiteID           string
    PlannedStorage   allocplan.StorageClass
    ActualStorage    allocplan.StorageClass
    ArtifactID       string
    IRInstructionFrom int
    IRInstructionTo   int
    RuntimePath      runtimeabi.AllocationRuntimePath
    BackendClass     runtimeabi.MemoryBackendClass
    DomainID         string
    ProofIDs         []string
    DecisionCode     string
}

type LoweringEvidence struct {
    Allocations []AllocationLoweringEvidence
}

type ProgramResult struct {
    Program  *ir.IRProgram
    Evidence LoweringEvidence
    // module index is unexported
}

func LowerPlannedProgram(
    checked *semantics.CheckedProgram,
    plan *allocplan.Plan,
    opt Options,
) (*ProgramResult, error)

func (result *ProgramResult) ModuleFuncs(module string) ([]ir.IRFunc, error)
func (result *ProgramResult) ModuleLoweringDigest(module string) (string, error)
```

`ModuleFuncs` повертає defensive copy, придатну для worker-local optimization/codegen.

### 6.7. Canonical build state

```go
type Phase string

const (
    PhasePLIR       Phase = "plir"
    PhasePlanned    Phase = "planned"
    PhaseLowered    Phase = "lowered"
    PhaseOptimized  Phase = "optimized"
    PhaseValidated  Phase = "validated"
    PhaseFinal      Phase = "final"
)

type Options struct {
    Target    string
    AllocPlan allocplan.Options
}

type State struct {
    ProgramID string
    Target    string
    PLIR      *plir.Program
    Graph     *memoryfacts.Graph
    Plan      *allocplan.Plan
    Phase     Phase
}

func Build(
    checked *semantics.CheckedProgram,
    opt Options,
) (*State, error)

func (s *State) ApplyLowering(result *lower.ProgramResult) error
func (s *State) ApplyOptimization(report opt.Report) error
func (s *State) ApplyValidation(report validation.ProofReport) error
func (s *State) AttachCachedModuleEvidence(module string, evidence CacheEvidence) error
func (s *State) Snapshot() (memoryfacts.Snapshot, error)
func (s *State) ModulePlanDigest(module string) (string, error)
func (s *State) FinalDigest() (string, error)
func (s *State) Finalize() error
```

`Build` exact order:

1. `plir.FromCheckedProgram`.
2. `plir.VerifyProgram`.
3. deterministic `ProgramID` from target, relevant allocation options і canonical PLIR digest.
4. `fromplir.Build`.
5. graph validation/snapshot.
6. `allocplan.Build`.
7. `allocplan.VerifyPlanned`.
8. `fromallocplan.Delta` + `Graph.Apply`.
9. stage advance to `allocplan`.
10. state phase `PhasePlanned`.

### 6.8. Domain ledger

До `MemoryDomain` додати optional fields:

```go
State            MemoryDomainState `json:"state,omitempty"`
Epoch            uint64            `json:"epoch,omitempty"`
DecommittedBytes int64             `json:"decommitted_bytes,omitempty"`
```

```go
type MemoryDomainState string

const (
    DomainStateActive MemoryDomainState = "active"
    DomainStateClosed MemoryDomainState = "closed"
)

type MemoryDomainEventKind string

const (
    DomainEventRequest  MemoryDomainEventKind = "request"
    DomainEventReserve  MemoryDomainEventKind = "reserve"
    DomainEventCommit   MemoryDomainEventKind = "commit"
    DomainEventAllocate MemoryDomainEventKind = "allocate"
    DomainEventFree     MemoryDomainEventKind = "free"
    DomainEventDecommit MemoryDomainEventKind = "decommit"
    DomainEventRelease  MemoryDomainEventKind = "release"
    DomainEventTrim     MemoryDomainEventKind = "trim"
    DomainEventReset    MemoryDomainEventKind = "reset"
    DomainEventClose    MemoryDomainEventKind = "close"
    DomainEventCopy     MemoryDomainEventKind = "copy"
    DomainEventMove     MemoryDomainEventKind = "move"
)

type MemoryDomainEvent struct {
    Kind             MemoryDomainEventKind
    DomainID         string
    DestinationID    string
    Bytes            int64
    ReservationBytes int64
    CommitBytes      int64
    ReasonCode       string
}

type MemoryDomainLedger struct { /* mutex + private maps */ }

func NewMemoryDomainLedger(process MemoryDomain) (*MemoryDomainLedger, error)
func (l *MemoryDomainLedger) Register(domain MemoryDomain) error
func (l *MemoryDomainLedger) Apply(event MemoryDomainEvent) error
func (l *MemoryDomainLedger) Snapshot() []MemoryDomain
func (l *MemoryDomainLedger) Validate() error
```

Accounting semantics:

- `RequestedBytes`: cumulative payload requests.
- `ReservedBytes`: current reserved address-space bytes.
- `CommittedBytes`: current committed bytes.
- `CurrentBytes`: current live payload bytes.
- `PeakBytes`: maximum observed `CurrentBytes`.
- `ReleasedBytes`: cumulative released reservation bytes.
- `DecommittedBytes`: cumulative decommitted bytes.
- `CopyCount`/`BytesCopied`: cumulative destination-side copy accounting.

Invariants:

```text
0 <= CurrentBytes <= CommittedBytes <= ReservedBytes
PeakBytes >= CurrentBytes
BudgetBytes == 0 || CurrentBytes <= BudgetBytes
all non-process parents exist
process root has no parent
parent graph has no cycles
events on closed domain are rejected
free <= CurrentBytes
decommit <= CommittedBytes - CurrentBytes
release <= ReservedBytes - CommittedBytes
```

`Reset` for island:

- reject closed domain;
- `CurrentBytes = 0`;
- increment `Epoch` exactly once;
- preserve domain identity;
- backend decommit/release accounting is recorded by separate events, not fabricated by reset.

`Close`:

- requires `CurrentBytes == 0`, `CommittedBytes == 0`, `ReservedBytes == 0`;
- changes state to `closed`;
- is idempotency-rejecting: second close returns deterministic error.

`Move`:

- source and destination active;
- source has enough current bytes;
- source current decreases, destination current increases;
- no copy counters change.

`Copy`:

- source current unchanged;
- destination request/allocate accounting increases;
- destination `CopyCount++`, `BytesCopied += bytes`.

### 6.9. Runtime backend contract

```go
type MemoryBackendOperationSupport struct {
    Operation         MemoryBackendOperation `json:"operation"`
    Supported         bool                   `json:"supported"`
    Method            string                 `json:"method,omitempty"`
    UnsupportedReason string                 `json:"unsupported_reason,omitempty"`
}
```

`MemoryBackendContract` має використовувати support rows замість припущення, що всі targets реалізують усі operations.

Support matrix:

| Target | reserve | commit | decommit | release | trim | footprint |
|---|---:|---:|---:|---:|---:|---:|
| `linux-x64` | yes | yes | yes | yes | yes | yes |
| `wasm32-wasi` | yes, combined with grow | yes, combined with grow | no | no | no | no host RSS |
| `wasm32-web` | yes, combined with grow | yes, combined with grow | no | no | no | no host RSS |
| other | no until adapter exists | no | no | no | no | no |

Linux behavior:

- reserve: anonymous address-space reservation;
- commit: make reserved pages readable/writable;
- decommit: discard pages and make them unavailable while preserving reservation;
- release: release reservation;
- trim: allocator/backend trim, may report zero reclaimed bytes;
- footprint: real process measurement with method named in evidence.

WASM unsupported operations must return explicit unsupported evidence; never no-op success.

### 6.10. Optimizer context

```go
type MemoryContext struct {
    Facts memoryfacts.Snapshot
}

type PassContext struct {
    Program   *ir.IRProgram
    Memory    MemoryContext
    Decisions *[]PassDecision
    Delta     *memoryfacts.Delta
}
```

Змінити pass execution contract:

```go
type Pass struct {
    // existing metadata remains
    RequiredMemoryProofs   []memoryfacts.ProofKind
    PreservedMemoryProofs  []memoryfacts.ProofKind
    InvalidatedMemoryProofs []memoryfacts.ProofKind
    Run func(*PassContext) error
}
```

`opt.Options`:

```go
MemoryFacts memoryfacts.Snapshot
```

`PassDecision` додати:

```go
ProofIDs    []string `json:"proof_ids,omitempty"`
DecisionCode string  `json:"decision_code,omitempty"`
```

---

## 7. Саб-агент policy

### 7.1. Write-enabled subagents

Для delegated editing дозволено лише:

```text
agent_type=worker
model=gpt-5.5
reasoning_effort=xhigh
fork_context=true
```

Позначення в цьому плані: **`gpt-5.5-xhigh worker`**.

Не замінювати модель або reasoning level. Якщо exact worker недоступний, delegated task має status `BLOCKED`; parent controller не видає його іншій write-моделі.

### 7.2. Shared files

Лише parent controller редагує:

```text
compiler/compiler_facade.go
compiler/compiler_reports.go
compiler/compiler_build_runtime.go
compiler/internal/buildapi/options.go
.workflow/memory-core-v2/**
docs/plans/2026-06/memory/2026-06-22-memory-core-v2-implementation-plan.md
scripts/release/v0_4_0/gate.sh
```

### 7.3. Merge discipline

Для кожного subagent task:

1. Parent фіксує clean scope diff перед spawn.
2. Worker отримує exact allowed paths і forbidden paths.
3. Worker не змінює workflow/status files.
4. Worker запускає тільки named focused tests.
5. Parent перевіряє `git diff -- <allowed scope>`.
6. Parent запускає review і focused tests повторно.
7. Наступний task не починається, поки review findings не закриті.
8. Один task — один commit після acceptance.

### 7.4. Worker final report format

Кожен worker повертає:

```text
Status: DONE | PARTIAL | BLOCKED
Files changed:
Behavior implemented:
Tests run:
Evidence:
Residual risks:
Forbidden-scope confirmation:
```

`DONE` worker-а означає лише task-level `LOCAL` або `INTEGRATION`, не final project completion.

---

## 8. Execution DAG

```text
MCV2-T00
  ↓
MCV2-T01 → MCV2-T02 → MCV2-T03
  ↓
MCV2-T04 → MCV2-T05 → MCV2-T06 → MCV2-T07
                              ↓
                 ┌────────────┼────────────┐
                 ↓            ↓            ↓
             MCV2-T08     MCV2-T09-12  MCV2-T13
                 └────────────┼────────────┘
                              ↓
                         MCV2-T14
                              ↓
                         MCV2-T15
                              ↓
                         MCV2-T16
```

`MCV2-T09-12` і `MCV2-T13` можуть виконуватися паралельно лише після green `MCV2-T07`, тому що їхні write scopes не перетинаються. Parent інтегрує їх послідовно.

---

# 9. Детальні tasks

## MCV2-T00 — Baseline, goal isolation і drift freeze

**Owner:** parent controller
**Completion target:** `LOCAL`
**Code edits:** заборонені

### Дії

1. Виконати rules із розділу 3.
2. Прочитати:
   - `AGENTS.md`;
   - active `GOAL.md`;
   - `CONTROL.md`, якщо існує;
   - `graphify-out/GRAPH_REPORT.md`, якщо існує у worktree;
   - цей canonical plan.
3. Записати:
   - base HEAD;
   - branch;
   - clean/dirty status;
   - Go version;
   - OS/arch;
   - наявність `graphify`;
   - exact critical symbol inventory.
4. Створити baseline report:

```text
reports/stabilization/memory-core-v2/mcv2-t00-baseline.md
```

5. Створити workflow files.
6. У `ATTEMPTS.md` додати attempt `T00-A1` із commands/results.
7. У `GOAL.md` додати acceptance checklist `MCV2-T00`…`MCV2-T16` без позначення виконання наступних tasks.

### Baseline tests

```bash
git diff --check
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false \
  ./compiler/internal/memoryfacts/... \
  ./compiler/internal/allocplan/... \
  ./compiler/internal/lower/... \
  ./compiler/internal/runtimeabi/... \
  ./compiler/internal/islandkernel/... \
  ./compiler/internal/memorymodel/... \
  ./compiler/internal/ramcontract/... \
  ./compiler/internal/opt/... \
  -count=1
```

Окремо записати всі baseline failures. Не виправляти їх у T00.

### Acceptance

- execution base зафіксовано;
- critical symbols mapped;
- baseline command output збережено;
- unrelated dirty work не змінено;
- active memory goal isolated;
- status не вище `LOCAL`.

---

## MCV2-T01 — Canonical typed vocabulary та pure `memoryfacts` core

**Owner:** `gpt-5.5-xhigh worker SA-MEMFACTS-1`
**Depends on:** T00
**Completion target:** `LOCAL`

### Allowed write scope

```text
compiler/internal/memoryfacts/facts.go
compiler/internal/memoryfacts/graph.go
compiler/internal/memoryfacts/validate.go
compiler/internal/memoryfacts/report.go
compiler/internal/memoryfacts/*_test.go
compiler/internal/memoryfacts_test/**
compiler/internal/memoryfacts/vocabulary.go          (new)
compiler/internal/memoryfacts/vocabulary_test.go     (new)
compiler/memoryvocab/vocab.go
compiler/memoryvocab/vocab_test.go
```

### Forbidden scope

```text
compiler/internal/memoryfacts/fromplir/**
compiler/internal/allocplan/**
compiler/internal/lower/**
compiler/compiler_*.go
```

### Exact work

1. Перенести constants, lists і policy functions із `compiler/memoryvocab/vocab.go` у `compiler/internal/memoryfacts/vocabulary.go`.
2. Зробити APIs typed там, де inputs/outputs відповідають `Claim`, `SourceStage`, `ProvenanceClass`, `UnsafeClass`, `AliasState`, `StorageClass`, `ClaimLevel`, `ValidatorStatus`, `CostClass`.
3. Додати `Claim`, `ProofKind`, keys і new `Fact` fields із розділу 6.1.
4. Додати `StageOptimization`.
5. Оновити `memoryfacts/validate.go` та `report.go`, щоб вони не імпортували `compiler/memoryvocab`.
6. Тимчасово перетворити `compiler/memoryvocab` на compatibility re-export package:
   - constants мають делегувати canonical values;
   - policy functions мають викликати `memoryfacts`;
   - заборонено залишати duplicate arrays/maps/rules.
7. `memoryfacts` core після цього task не має non-stdlib imports.
8. Зберегти existing JSON strings і report schema.
9. Додати parity test: кожен compatibility value/function дає той самий результат, що canonical implementation.
10. Додати test, який перевіряє known vocabulary uniqueness: без duplicate claims/stages/classes.

### Focused validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false \
  ./compiler/internal/memoryfacts/... \
  ./compiler/internal/memoryfacts_test/... \
  ./compiler/memoryvocab/... \
  -count=1

rg -n 'compiler/memoryvocab' compiler/internal/memoryfacts --glob '*.go'
```

Другий command має повернути zero matches.

### Acceptance

- `memoryfacts` core import graph = stdlib only;
- compatibility package не містить власної policy logic;
- `tetra.memory-report.v1` tests green;
- no JSON vocabulary change;
- no edits outside allowed scope.

### Worker prompt

```text
Implement MCV2-T01 exactly. Do not touch adapters, allocplan, lower, compiler facade or reports pipeline. Move vocabulary/policy into memoryfacts, keep memoryvocab as a delegating compatibility layer only, preserve all serialized strings, add typed Claim/ProofKind and required Fact fields, and prove parity with tests. Return task-level status only.
```

---

## MCV2-T02 — Видалення duplicate builders і розділення stage adapters

**Owner:** `gpt-5.5-xhigh worker SA-MEMFACTS-2`
**Depends on:** T01
**Completion target:** `INTEGRATION`

### Allowed write scope

```text
compiler/internal/memoryfacts/fromplir/**
compiler/internal/memoryfacts/fromallocplan/**       (new)
compiler/internal/memoryfacts/fromvalidation/**      (new)
compiler/internal/memoryfacts/from_plir_allocplan.go
compiler/internal/memoryfacts/from_plir_borrow.go
compiler/internal/memoryfacts/from_plir_copy.go
compiler/internal/memoryfacts/from_plir_summary.go
compiler/internal/memoryfacts/from_plir_unsafe.go
compiler/internal/memoryfacts/from_validation.go
compiler/internal/memoryfacts_test/**
```

### Exact work

1. Перед видаленням duplicate root builders додати equivalence tests:
   - побудувати graph старим root path;
   - побудувати graph current `fromplir` path;
   - порівняти sorted full facts.
2. Змінити `fromplir.FromPLIRAndAllocPlan` на `fromplir.Build(programID, prog)`:
   - він додає тільки PLIR/semantic facts;
   - не імпортує `allocplan`;
   - не встановлює `StoragePlan` або `ActualLoweringStorage`.
3. Створити `fromallocplan.Delta(snapshot, prog, plan)` і перенести туди allocation plan projection.
4. Перенести `AddBoundsProofFacts` та rejection helper у `fromvalidation.BoundsDelta`.
5. Заборонити dot imports; використовувати explicit `memoryfacts.` names.
6. Перенести/оновити tests так, щоб tests напряму перевіряли кожен stage:
   - graph after PLIR;
   - graph after allocplan delta;
   - graph after validation delta.
7. Після green equivalence test видалити root duplicate files:

```text
compiler/internal/memoryfacts/from_plir_allocplan.go
compiler/internal/memoryfacts/from_plir_borrow.go
compiler/internal/memoryfacts/from_plir_copy.go
compiler/internal/memoryfacts/from_plir_summary.go
compiler/internal/memoryfacts/from_plir_unsafe.go
compiler/internal/memoryfacts/from_validation.go
```

8. Видалити equivalence harness, який залежить від deleted implementation; залишити expected-fact regression tests.
9. Не змінювати compiler call sites у цьому task.

### Focused validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false \
  ./compiler/internal/memoryfacts/... \
  ./compiler/internal/memoryfacts_test/... \
  -count=1

find compiler/internal/memoryfacts -maxdepth 1 -name 'from_plir_*.go' -print
rg -n '^\s*\.\s+"tetra_language/compiler/internal/memoryfacts"' compiler/internal/memoryfacts --glob '*.go'
rg -n 'compiler/internal/allocplan' compiler/internal/memoryfacts/fromplir --glob '*.go'
```

Усі три inventory commands мають дати zero forbidden matches/files.

### Acceptance

- одна PLIR builder implementation;
- PLIR builder не залежить від allocplan;
- allocation і validation adapters окремі;
- root duplicates видалені;
- tests зберігають існуючі fact semantics.

---

## MCV2-T03 — Immutable Snapshot, indexes, proof resolution і stage delta

**Owner:** `gpt-5.5-xhigh worker SA-MEMFACTS-3`
**Depends on:** T02
**Completion target:** `INTEGRATION`

### Allowed write scope

```text
compiler/internal/memoryfacts/**
compiler/internal/memoryfacts_test/**
```

### Exact work

1. Реалізувати APIs із розділів 6.2 і 6.3.
2. `Snapshot` створює indexes:
   - by fact ID;
   - by `ValueKey`;
   - by `AllocationKey`;
   - by `ProofKey`;
   - by parent fact.
3. Snapshot не посилається на mutable slices/maps graph-а.
4. `ResolveAllocation`:
   - фільтрує факти exact function/value;
   - об’єднує лише consistent fields;
   - повертає error на conflicting owner/island/epoch/unsafe/escape;
   - вимагає allocation site для alloc intent;
   - повертає sorted unique source fact IDs.
5. `ResolveProof` реалізувати за rules із 6.2.
6. `Graph.Apply` має бути atomic:
   - validate entire delta on cloned state;
   - commit лише якщо всі operations valid;
   - при error original graph не змінюється.
7. `AdvanceTo` enforce stage order.
8. `Snapshot.Digest()`:
   - SHA-256;
   - facts sorted by full deterministic key;
   - `DerivedFactIDs` sorted;
   - no map iteration nondeterminism;
   - same logical graph in different insertion order produces same digest.
9. Додати concurrency test: багато goroutines читають один Snapshot; race-free under `go test -race`.
10. Додати negative tests:
    - stale epoch proof;
    - mismatched subject;
    - invalidated proof;
    - unsafe_unknown optimization proof;
    - non-atomic failed delta;
    - stage regression;
    - conflicting allocation evidence.
11. Додати import architecture test для forbidden core dependencies.

### Focused validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false \
  ./compiler/internal/memoryfacts/... \
  ./compiler/internal/memoryfacts_test/... \
  -count=1

GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -race -buildvcs=false \
  ./compiler/internal/memoryfacts/... \
  -run 'Snapshot|Delta|Proof|Digest' \
  -count=1
```

### Acceptance

- canonical immutable read model exists;
- proof query is typed and fail-closed;
- delta application atomic;
- digest deterministic;
- core dependency gate green.

---

## MCV2-T04 — `memorypipeline.State` до lowering

**Owner:** `gpt-5.5-xhigh worker SA-PIPELINE-1`
**Depends on:** T03
**Completion target:** `INTEGRATION`

### Allowed write scope

```text
compiler/internal/memorypipeline/**                 (new)
compiler/internal/plir/**                           (digest helper only if required)
compiler/internal/memoryfacts/fromplir/**
compiler/internal/memoryfacts/fromallocplan/**
```

### Forbidden scope

```text
compiler/internal/allocplan/**
compiler/internal/lower/**
compiler/compiler_*.go
```

### Exact work

1. Створити package `compiler/internal/memorypipeline` і API із 6.7.
2. У цьому task реалізувати лише `Build`, `Snapshot`, `ModulePlanDigest`, phase guards до `PhasePlanned`.
3. `ProgramID` format:

```text
program:sha256:<64 lowercase hex>
```

4. Canonical digest input:
   - target;
   - allocation options;
   - normalized PLIR.
5. Normalize PLIR copy before hashing:
   - functions by `(Module, Name)`;
   - values by `ID`;
   - operations by `ID`;
   - facts by `(Kind, ValueID, Block, Source)`;
   - blocks by `ID`;
   - proof guards/uses/terms by `ID` then operation fields;
   - range facts by stable full tuple.
6. Не mut-ити original PLIR під час normalization.
7. `ModulePlanDigest(module)` format:

```text
memory-plan:sha256:<64 lowercase hex>
```

Digest включає:

- program schema version;
- target/options;
- function PLIR rows цього module;
- allocation rows цього module;
- source fact IDs.

8. Додати tests permutation invariance і option sensitivity.
9. Тимчасово використати current planner API через narrow adapter тільки якщо T05 ще не integrated; позначити helper `legacyBuildAllocationPlanForT04` і видалити його в T05. Він не може потрапити у compiler production call sites.

### Focused validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false \
  ./compiler/internal/memorypipeline/... \
  ./compiler/internal/plir/... \
  ./compiler/internal/memoryfacts/... \
  -count=1
```

### Acceptance

- canonical pre-lowering State будується deterministically;
- report flags відсутні в `memorypipeline.Options`;
- state після Build має `PhasePlanned`;
- no compiler facade edits;
- temporary legacy helper явно isolated.

---

## MCV2-T05 — Fact-driven `allocplan`

**Owner:** `gpt-5.5-xhigh worker SA-ALLOCPLAN`
**Depends on:** T04
**Completion target:** `INTEGRATION`

### Allowed write scope

```text
compiler/internal/allocplan/**
compiler/internal/memorypipeline/**
compiler/internal/memoryfacts/fromallocplan/**
compiler/internal/islandkernel/**                    (planner-facing typed API only)
```

### Exact work

1. Реалізувати `allocplan.Input`, `Build`, `VerifyPlanned`, `VerifyLowered`.
2. Перевести `memorypipeline.Build` на `allocplan.Build`; видалити `legacyBuildAllocationPlanForT04`.
3. `FromPLIR` і `FromPLIRWithOptions`:
   - migrate всі repo callers/tests;
   - після zero callers видалити functions;
   - не залишати compatibility wrapper, який знову класифікує без graph.
4. Escape classification отримувати лише з `AllocationEvidence`.
5. Видалити:
   - `looksActorSend`;
   - `looksTaskBoundary`;
   - operation-name based boundary detection;
   - local duplicate escape mapping, якщо canonical evidence уже має state.
6. Якщо PLIR і graph evidence не узгоджені — error; не обирати conservative path silently.
7. Planning rules:
   - unsafe/external roots не отримують trusted storage;
   - escaped values не отримують stack/region/island/task/actor trusted storage;
   - compiler-owned no-escape proof потрібен для trusted placement, де current verifier цього вимагає;
   - explicit island planning вимагає island identity/epoch/owner evidence;
   - `TaskRegion`/`ActorMoveRegion` вимагають відповідний move proof; без нього heap/conservative із exact reason code;
   - heap fallback завжди має non-empty `HeapReasonCodes`.
8. Planner встановлює:

```text
Storage = PlannedStorage       // compatibility field
ActualLoweringStorage = UnknownConservative
LoweringStatus = pending
```

9. Видалити `actualLoweringStorage` predictor.
10. `VerifyPlanned` не вимагає actual evidence, але вимагає pending state.
11. `VerifyLowered` вимагає actual evidence, artifact і consistent plan/lowering.
12. Додати `SourceFactIDs`, `ProofIDs`, `DecisionCode`, `PlanDigest`.
13. Domain assignment:
   - explicit island -> island domain;
   - proven task move -> task domain;
   - proven actor move -> actor domain;
   - external -> external domain;
   - otherwise process domain;
   - request domain лише за typed request owner evidence; не за string match.
14. Не змінювати report schema field names.

### Required tests

- missing allocation fact -> error;
- conflicting graph/PLIR evidence -> error;
- actor/task names у source text без typed escape не впливають на plan;
- typed actor/task escape впливає;
- actual storage remains unknown before lowering;
- unsafe_unknown cannot authorize trusted storage;
- stale island epoch rejects trusted island plan;
- every heap allocation has reason code;
- deterministic plan digest.

### Focused validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false \
  ./compiler/internal/allocplan/... \
  ./compiler/internal/memorypipeline/... \
  ./compiler/internal/memoryfacts/... \
  ./compiler/internal/islandkernel/... \
  -count=1

rg -n 'FromPLIRWithOptions|FromPLIR\(' compiler --glob '*.go'
rg -n 'looksActorSend|looksTaskBoundary|actualLoweringStorage' compiler/internal/allocplan --glob '*.go'
```

Обидва inventory commands мають повернути zero matches.

### Acceptance

- planner consumes canonical Snapshot;
- no string boundary heuristics;
- no predicted actual lowering;
- planned/final verification розділені;
- all callers migrated.

---

## MCV2-T06 — Plan-driven lowering і actual lowering evidence

**Owner:** `gpt-5.5-xhigh worker SA-LOWERING`
**Depends on:** T05
**Completion target:** `INTEGRATION`

### Allowed write scope

```text
compiler/internal/lower/**
compiler/internal/memoryfacts/fromlowering/**        (new)
compiler/internal/ir/**                              (тільки evidence metadata, якщо необхідно)
```

### Forbidden scope

```text
compiler/internal/allocplan/**
compiler/internal/memorypipeline/**
compiler/compiler_*.go
```

### Exact work

1. Реалізувати `LowerPlannedProgram`, `ProgramResult`, `LoweringEvidence`, `ModuleFuncs`, `ModuleLoweringDigest` із 6.6.
2. `LowerPlannedProgram`:
   - перевіряє non-nil plan;
   - викликає `allocplan.VerifyPlanned`;
   - не створює PLIR;
   - не створює allocation plan;
   - будує allocation lookup із переданого plan;
   - lower-ить усі checked functions і generated wrappers один раз.
3. Module index формувати під час lowering, не відновлювати через string prefixes.
4. Кожна allocation branch має записувати evidence в момент emission:
   - eliminated/register;
   - stack;
   - function temp region;
   - explicit island;
   - task region;
   - actor move region;
   - small heap/heap;
   - large mmap;
   - external;
   - conservative fallback.
5. Evidence `ArtifactID` format:

```text
ir:<function>:<first-instruction>:<last-instruction>:<allocation-id>
```

Для eliminated allocation:

```text
ir:<function>:eliminated:<allocation-id>
```

6. Один allocation має рівно один evidence row.
7. Якщо emitted branch не відповідає `PlannedStorage`:
   - actual storage записується чесно;
   - `DecisionCode` пояснює fallback;
   - trusted claim не ставиться;
   - `VerifyLowered` пізніше вирішує, чи fallback допустимий.
8. `ModuleFuncs` повертає deep-enough copy, щоб parallel optimizer/codegen не mut-или shared IR.
9. `ModuleLoweringDigest` включає sorted evidence rows і normalized IR instructions module-а.
10. Створити `fromlowering.Delta`, який:
    - прикріплює actual storage;
    - додає artifact IDs;
    - не позначає trusted storage validated до validation stage.
11. Видалити internal `LowerWithOptions`/`LowerModuleWithOptions` після migration of internal tests. Public compiler facade migration виконує parent у T07; до цього дозволений compile break лише в working branch між T06 і T07, але commit T06 має бути buildable. Для цього додати deprecated wrappers, які **вимагають explicit plan parameter**; wrapper, що сам будує PLIR/plan, заборонений.

### Required tests

- one evidence row per allocation;
- actual branch records exact storage;
- fallback records exact reason;
- module slicing preserves generated wrappers;
- module copies do not mutate shared result;
- deterministic lowering digest;
- no PLIR/allocplan construction imports/calls in lower.

### Focused validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false \
  ./compiler/internal/lower/... \
  ./compiler/internal/memoryfacts/fromlowering/... \
  -count=1

rg -n 'plir\.FromCheckedProgram|allocplan\.(FromPLIR|Build)' compiler/internal/lower --glob '*.go'
```

Останній command має повернути zero matches.

### Acceptance

- lower consumes exact plan;
- actual storage is emitted-path evidence;
- program/module lowering no longer rebuilds plan;
- evidence deterministic і complete.

---

## MCV2-T07 — Normal build integration, cache attestation і report-path removal

**Owner:** parent controller
**Depends on:** T06
**Completion target:** `END_TO_END`

### Primary write scope

```text
compiler/compiler_facade.go
compiler/compiler_reports.go
compiler/compiler_build_runtime.go
compiler/internal/buildplan/**
compiler/internal/cache/**
compiler/internal/tobj/** або object metadata owner package
compiler/internal/memorypipeline/**
compiler/compiler_external_test.go
compiler/compiler_suite_test.go
```

### Exact work

#### A. Build orchestration

1. Після `semantics.CheckWorldOpt` і target validations створити:

```go
memoryState, err := memorypipeline.Build(checked, ...)
```

2. Передати `memoryState` у `planNativeModuleBuild` і `compileNativeModulePlan`.
3. Видалити з `compileNativeModulePlan`:
   - local `plir.FromCheckedProgram`;
   - local allocation planning;
   - `allocationSummaryProgram` built through old path;
   - `lower.LowerModuleWithOptions` worker calls.
4. Якщо `len(plan.ToCompile) > 0`, виконати `lower.LowerPlannedProgram` рівно один раз до worker pool.
5. Застосувати `memoryState.ApplyLowering`.
6. Виконати `validation.ValidateAllocationLowering` на canonical plan/result.
7. Виконати `validation.CheckBoundsProofsWithPLIR` на canonical IR/PLIR.
8. Застосувати validation delta.
9. Worker бере `result.ModuleFuncs(job.Module)` і виконує target validation/codegen.
10. Усі workers використовують immutable/copy IR slices.

#### B. Cache attestation

До module object metadata додати:

```text
memory_plan_schema
memory_plan_digest
memory_lowering_schema
memory_lowering_digest
```

Exact schema values:

```text
tetra.memory-plan.v2
tetra.memory-lowering.v2
```

Rules:

1. Cache lookup отримує expected module plan digest.
2. Cache entry без fields — miss.
3. Digest mismatch — miss, не error.
4. Після codegen object metadata отримує current plan/lowering digests.
5. Full cache hit без report/optimizer requirement:
   - lowering не запускається;
   - кожен hit attested;
   - state отримує cached module evidence;
   - finalization допускає cache-attested lowering stage.
6. Full cache hit із report, proof, bounds або memory report requirement:
   - lowering запускається один раз для actual report projection;
   - cache objects не переписуються без source rebuild.
7. Memory schema version включити у cache compatibility/version, щоб old entries не вважались compatible.

#### C. Report path

Змінити `emitExplainReports` так, щоб він отримував prepared canonical state/result/proof report.

У `emitExplainReports` заборонені calls:

```text
plir.FromCheckedProgram
allocplan.Build / FromPLIR*
lower.Lower*
fromplir.Build / FromPLIRAndAllocPlan
validation decision construction, окрім report validation
```

Функція лише:

- перевіряє required artifact presence;
- формує projections;
- валідовує schemas;
- записує files;
- виконує enforcement із canonical data.

#### D. Public facade

`compiler.Lower(checked)` і `compiler.BuildPLIR(checked)`:

- `BuildPLIR` може будувати PLIR напряму як explicit API;
- `Lower` має створити `memorypipeline.State` і викликати `LowerPlannedProgram`;
- не викликати removed lower wrappers.

#### E. Profiler evidence

Compiler phase report має показувати:

- one PLIR construction;
- one allocation planning;
- zero/one whole-program lowering;
- zero module-local re-planning;
- cache-attested path окремою reason row.

### Required end-to-end tests

1. **Report flag parity**:
   - build same source без reports;
   - build із `EmitMemoryReport`, `EmitAllocReport`, `EmitRAMContractReport`;
   - compare module plan digest, lowering digest і executable behavior;
   - вони однакові.
2. **Single construction count**:
   - compiler phase report підтверджує counts.
3. **Cache path**:
   - first build miss;
   - second build full hit;
   - second build не lower-ить;
   - corrupt plan digest causes miss.
4. **Report on cache hit**:
   - canonical report створюється;
   - lowering рівно один раз;
   - object cache metadata не підмінює actual report evidence.
5. **No-report normal build graph**:
   - test hook/state digest підтверджує, що graph/state існував.
6. **Allocation mismatch**:
   - intentionally mismatched test plan is rejected before codegen.

### Focused validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false \
  ./compiler/... \
  -run 'MemoryCore|MemoryReport|AllocReport|RAMContract|Cache|CompilerPhase|AllocationLowering' \
  -count=1

rg -n 'plir\.FromCheckedProgram|allocplan\.(Build|FromPLIR)|lower\.Lower' \
  compiler/compiler_reports.go

rg -n 'plir\.FromCheckedProgram|allocplan\.(Build|FromPLIR)|LowerModuleWithOptions' \
  compiler/compiler_facade.go
```

Report file command має zero matches. Facade matches дозволені лише в explicit public `BuildPLIR`/`Lower` adapters, не в module/report build path; parent записує exact reviewed lines у evidence.

### Acceptance

- executable normal build використовує canonical state;
- no duplicate module planning/lowering;
- cache attests memory decisions;
- report flags не впливають на decisions;
- report path projection-only;
- focused compiler tests green.

---

## MCV2-T08 — Memory/alloc/RAM reports як projections

**Owner:** `gpt-5.5-xhigh worker SA-REPORTS`
**Depends on:** T07
**Completion target:** `INTEGRATION`

### Allowed write scope

```text
compiler/internal/memoryfacts/report.go
compiler/internal/memoryfacts/**/*report*test*.go
compiler/internal/allocplan/report.go
compiler/internal/allocplan/**/*report*test*.go
compiler/internal/ramcontract/**
compiler/internal/runtimeabi/memory_domain.go          (type alias/support only)
```

### Exact work

1. `memoryfacts.BuildReportFromGraph` лишається єдиним builder для `tetra.memory-report.v1`.
2. Memory report не має викликати planner/lower/validation.
3. Allocation report використовує final canonical `Plan` після lowering evidence.
4. RAM contract API замінити:

```go
type Input struct {
    Facts   memoryfacts.Snapshot
    Plan    *allocplan.Plan
    Domains []runtimeabi.MemoryDomain
}

func BuildReport(input Input, target, gitHead, producer string) Report
```

5. Видалити `BuildReportFromAllocPlan` після migration callers.
6. `ramcontract` не має own storage/domain classification rules.
7. Якщо `ramcontract/types.go` має duplicate `MemoryDomain`, замінити на alias або explicit serialization DTO, заповнюваний лише з `runtimeabi.MemoryDomain`; не залишати decision logic.
8. Preserve schemas; optional fields allowed:
   - memory graph digest;
   - plan digest;
   - lowering digest;
   - evidence source `normal_build`/`cache_attested`.
9. Reports reject:
   - actual storage without lowering artifact;
   - trusted storage with heap fallback;
   - measured footprint without measured method;
   - unsupported backend with byte counts;
   - domains with invalid ledger invariants.
10. Add deterministic projection tests.
11. Add test: mutate report JSON cannot mutate graph/plan.

### Focused validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false \
  ./compiler/internal/memoryfacts/... \
  ./compiler/internal/allocplan/... \
  ./compiler/internal/ramcontract/... \
  -count=1

rg -n 'BuildReportFromAllocPlan' . --glob '*.go'
rg -n 'func .*classif|looksActor|looksTask|chooseStorage' compiler/internal/ramcontract --glob '*.go'
```

Zero matches required.

### Acceptance

- RAM contract consumes canonical input;
- no report package decides placement/domain/proof;
- schemas remain valid;
- deterministic outputs.

---

## MCV2-T09 — Executable `MemoryDomainLedger`

**Owner:** `gpt-5.5-xhigh worker SA-RUNTIME-DOMAINS`
**Depends on:** T07
**Completion target:** `INTEGRATION`

### Allowed write scope

```text
compiler/internal/runtimeabi/memory_domain.go
compiler/internal/runtimeabi/memory_domain_ledger.go       (new)
compiler/internal/runtimeabi/memory_domain_ledger_test.go  (new)
compiler/internal/runtimeabi/runtimeabi_test/**
```

### Exact work

1. Реалізувати types/API/invariants із 6.8.
2. Додати constructors:

```go
TaskMemoryDomain(...)
ActorMemoryDomain(...)
RequestMemoryDomain(...)
```

3. `DefaultProcessMemoryDomain`:
   - root ID `domain:process`;
   - no parent;
   - `State=active`;
   - counters починаються з zero; requested/reserved не підставляти з budget.
4. Existing constructors не можуть ставити `BudgetBytes=requested` за замовчуванням. Budget — explicit input.
5. `ValidateMemoryDomain` enforce single-domain invariants.
6. `MemoryDomainLedger.Validate` enforce graph/lifecycle invariants.
7. Ledger thread-safe; snapshot sorted by `DomainID`.
8. Error messages мають deterministic prefixes:

```text
memory domain ledger: missing parent
memory domain ledger: cycle
memory domain ledger: budget exceeded
memory domain ledger: domain closed
memory domain ledger: accounting invariant
```

9. Add exhaustive state transition tests, включно з all invalid events.
10. Run race test.

### Focused validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false ./compiler/internal/runtimeabi/... \
  -run 'MemoryDomain|Ledger' -count=1

GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -race -buildvcs=false ./compiler/internal/runtimeabi/... \
  -run 'MemoryDomain|Ledger' -count=1
```

### Acceptance

- parent DAG validated;
- lifecycle/accounting executable;
- transfer semantics deterministic;
- no negative/counter inconsistencies accepted.

---

## MCV2-T10 — Real MemoryBackend operations і runtime wiring

**Owner:** `gpt-5.5-xhigh worker SA-RUNTIME-BACKEND`
**Depends on:** T09
**Completion target:** `END_TO_END` для `linux-x64`; explicit unsupported для WASM

### Allowed write scope

```text
compiler/internal/runtimeabi/memory_backend.go
compiler/internal/runtimeabi/memory_backend_test.go
compiler/internal/runtimeabi/runtimeabi_test/**
compiler/internal/runtimeabi/smallheap/**
compiler/internal/backend/**                     (only located runtime backend files)
__rt/**                                          (only located allocation/backend files)
internal/**                                      (only located runtime backend files)
```

Worker має спочатку повернути parent-у exact located runtime files; parent додає їх у allowed scope перед edit. Якщо allocation symbols не знайдені однозначно — `BLOCKED`.

### Required discovery

```bash
rg -n 'core\.alloc_bytes|island_new|island_make|region.*alloc|mmap|munmap|madvise|memory\.grow' \
  __rt compiler internal --glob '*.{go,s,S,asm,tetra}'
```

### Exact work

1. Додати operation support rows і support matrix із 6.9.
2. `ValidateMemoryBackendContract`:
   - кожна operation має рівно один support row;
   - supported row requires method;
   - unsupported row requires reason;
   - Linux всі required supported;
   - WASM unsupported rows не вимагають byte evidence.
3. Визначити runtime ABI symbols у `runtimeabi` constants:

```text
__tetra_memory_reserve_v1
__tetra_memory_commit_v1
__tetra_memory_decommit_v1
__tetra_memory_release_v1
__tetra_memory_trim_v1
__tetra_memory_footprint_v1
```

4. Реалізувати Linux-x64 adapters у реальному runtime path, не лише в Go test model.
5. Existing heap/region/island allocators мають викликати backend operations або спільний adapter, а не паралельний direct OS path.
6. Backend operation має записувати event у domain ledger/telemetry hook.
7. `smallheap`, region та island path не повинні double-count reserve/commit.
8. WASM:
   - reserve/commit mapping до linear-memory growth може бути combined;
   - decommit/release/trim/host footprint повертають unsupported;
   - no-op success заборонений.
9. Footprint evidence:
   - Linux method must match actual implementation;
   - current/peak values measured from real process source;
   - errors produce blocked evidence, not zero measured values.
10. Add runtime smoke program/test that exercises:
    - heap allocation/free;
    - explicit island allocate/reset/free;
    - region lifecycle;
    - footprint before/after;
    - ledger snapshot.
11. Add negative runtime tests:
    - over-release;
    - double free/close;
    - operation after closed domain;
    - unsupported WASM operation.
12. Do not claim performance/RSS reduction; only operation correctness and evidence class.

### Focused validation

Worker records exact commands based on located runtime harness. Minimum Go checks:

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false ./compiler/internal/runtimeabi/... -count=1
```

Linux end-to-end smoke is mandatory; a Go-only unit test is insufficient.

### Acceptance

- emitted Linux-x64 runtime executes backend operations;
- domain accounting receives real events;
- WASM unsupported operations fail explicitly;
- no duplicate direct OS allocation path remains for covered allocators;
- evidence class is truthful.

---

## MCV2-T11 — Islands як перший повний domain і direct kernel routes

**Owner:** `gpt-5.5-xhigh worker SA-ISLANDS`
**Depends on:** T09; може йти паралельно з T10
**Completion target:** `END_TO_END`

### Allowed write scope

```text
compiler/internal/islandkernel/**
compiler/internal/semantics/semantics_memory_resources.go
compiler/internal/semantics/**/*island*test*.go
compiler/internal/actorsafety/ownership_transfer.go
compiler/internal/actorsafety/**/*test*.go
```

Cross-package changes у `allocplan`, `lower`, `validation`, `opt` виконує parent у integration step цього task після worker patch.

### Exact work

1. Refactor `islandkernel` request types to use canonical `memoryfacts` classes/proofs. Не дублювати provenance/unsafe/storage constants.
2. Додати planner-facing decision:

```go
func CanPlanExplicitIsland(req StoragePlanRequest) Result
```

Він перевіряє provenance, escape, island ID, epoch, owner і storage proof prerequisites без actual lowering.

3. `CanLowerAsExplicitIsland` лишається post-lowering check із planned/actual/artifact proof.
4. Усі 15 required decisions мають direct production caller:

| Decision | Required production owner |
|---|---|
| `CanBorrow` | semantics borrow binding |
| `CanReturn` | semantics return escape validation |
| `CanStoreGlobal` | semantics/global escape validation |
| `CanCaptureClosure` | callable capture classification |
| `CanSendToActor` | actor transfer validation |
| `CanSendToTask` | task transfer validation |
| `CanMoveIsland` | ownership transfer path |
| `CanFreeIsland` | island free validation/runtime lifecycle |
| `CanResetIsland` | island reset validation/runtime lifecycle |
| `CanClaimNoAlias` | proof/optimizer authorization |
| `CanEliminateBoundsCheck` | bounds proof authorization |
| `CanPlanExplicitIsland` | allocation planner |
| `CanLowerAsExplicitIsland` | lowering validation |
| `CanPromoteUnsafeRoot` | unsafe gateway validation |
| `CanTrustStorage` | final allocation lowering validation |
| `CanEraseRuntimeCheck` | optimizer/runtime-check removal |

5. Оновити `RequiredDangerousDecisions` для 16 entries через split plan/lower decision.
6. Для кожної applicable route `Strategy=RouteThroughIslandKernel`.
7. `RouteValidatedEquivalent` після task має zero uses. `RouteNotApplicable` дозволений лише для target-specific runtime operation з explicit reason; у required 16 routes він не допускається.
8. Кожен caller зберігає `Reason.Code` у corresponding fact/allocation/optimizer decision.
9. Island domain lifecycle:
   - create active domain epoch 0;
   - borrow proof binds exact island+epoch;
   - reset only with zero live borrows, increments epoch;
   - stale old-epoch refs rejected;
   - move consumes source token і changes owner atomically;
   - free requires zero live borrows і zero current bytes after release;
   - use/reset/free after close rejected.
10. Unsafe/external memory не може авторизувати island trusted storage/noalias.
11. Add test matrix for every decision with accept/reject/conservative branches.
12. Update coverage validator to inspect actual call tokens, not just file presence.

### Parent integration step

Після worker patch parent робить bounded edits:

- `allocplan`: call `CanPlanExplicitIsland`;
- `lower`/`validation`: call `CanLowerAsExplicitIsland`/`CanTrustStorage`;
- `opt`: call noalias/bounds/runtime-check decisions through proof adapter;
- runtime ledger: apply island lifecycle events.

### Focused validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false \
  ./compiler/internal/islandkernel/... \
  ./compiler/internal/semantics/... \
  ./compiler/internal/actorsafety/... \
  ./compiler/internal/allocplan/... \
  ./compiler/internal/lower/... \
  -run 'Island|Borrow|Escape|OwnershipTransfer|AllocationLowering' \
  -count=1

rg -n 'RouteValidatedEquivalent' compiler/internal/islandkernel
```

Zero matches required.

### Acceptance

- 16/16 dangerous decisions route directly;
- island lifecycle tied to domain ledger and epoch;
- no stale/unsafe trusted promotion;
- coverage test proves production calls.

---

## MCV2-T12 — Task/actor/request domains і transfer semantics

**Owner:** parent controller
**Depends on:** T09, T11
**Completion target:** `END_TO_END`

### Exact work

1. Extend `fromplir` evidence for typed domain owner/boundary fields where semantics already knows them.
2. Do not derive owner/domain from source text or function name.
3. Allocation/domain mapping:

```text
local/no escape          -> process or scoped region domain
explicit island          -> island domain
owned move to task       -> task domain
owned move to actor      -> actor domain
serialized copy          -> destination task/actor domain + copy counters
borrowed task/actor send -> reject
external pointer         -> external domain
request-owned allocation -> request domain only with typed request owner evidence
```

4. `TaskRegion` і `ActorMoveRegion` trusted placement requires:
   - owned/moved transfer;
   - validated move proof;
   - source token/value consumed;
   - no live borrowed alias crossing boundary;
   - destination domain active;
   - actual lowering evidence.
5. Без proof — heap/conservative, exact reason:

```text
domain.task_move_unproven
domain.actor_move_unproven
domain.request_owner_unproven
```

6. Wire `actorsafety/ownership_transfer.go` result into canonical facts and ledger move/copy events.
7. Cancellation/task completion closes or releases task domain only after tracked resources current bytes reach zero.
8. Actor domain lifetime persists with actor; message-level copies не змінюють actor domain identity.
9. Add end-to-end tests:
   - owned task move;
   - borrowed task rejection;
   - owned actor move;
   - borrowed actor rejection;
   - serialized copy accounting;
   - cancellation cleanup;
   - request domain proven/unproven;
   - source use after move rejection.

### Focused validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false \
  ./compiler/internal/actorsafety/... \
  ./compiler/internal/allocplan/... \
  ./compiler/internal/runtimeabi/... \
  ./compiler/internal/memoryfacts/... \
  ./compiler/... \
  -run 'Task.*Memory|Actor.*Memory|Request.*Memory|OwnershipTransfer|Cancellation' \
  -count=1
```

### Acceptance

- actor/task/request domains derive from typed evidence;
- borrowed crossings rejected;
- move/copy accounting correct;
- trusted boundary storage proof-carrying.

---

## MCV2-T13 — Proof-driven optimizer і production wiring

**Owner:** `gpt-5.5-xhigh worker SA-OPTIMIZER`
**Depends on:** T07; може йти паралельно з T09-T12
**Completion target:** `END_TO_END` для `ReleaseOptimize`

### Allowed write scope

```text
compiler/internal/opt/**
compiler/internal/memoryfacts/fromoptimizer/**       (new)
compiler/internal/validation/**                      (optimizer proof validation only)
compiler/internal/differential/**                    (tests/harness only)
```

Parent owns compiler facade wiring after worker patch.

### Exact work

1. Реалізувати `MemoryContext`, `PassContext`, pass metadata і `opt.Options.MemoryFacts` із 6.10.
2. Migrate all registered pass runners to `func(*PassContext) error`.
3. Existing non-memory rewrites мають працювати без memory proof, але зберігати/invalidate site mappings explicit metadata.
4. Memory-sensitive categories:

| Rewrite category | Required proof/action |
|---|---|
| bounds-check removal | exact verified `ProofBounds` + island epoch match when applicable |
| noalias-based rewrite | exact verified `ProofNoAlias` + `islandkernel.CanClaimNoAlias` |
| trusted allocation sinking | `ProofNoEscape` + `ProofStorage`; reject unsafe/external |
| scalar replacement of memory object | no-escape, no observable alias, exact lifetime proof |
| LICM of memory access | noalias + region alive + no write invalidation |
| inlining across memory boundary | preserve or explicitly invalidate callee/caller proofs |
| runtime-check erasure | exact proof + `CanEraseRuntimeCheck` |

5. Для missing/mismatched proof:
   - skip rewrite;
   - append `PassDecision` із `DecisionCode`;
   - не повертати compile error, якщо conservative IR valid.
6. Для invalidated proof:
   - append `memoryfacts.Invalidation`;
   - reason includes pass name і transformation site.
7. Pass metadata must declare required/preserved/invalidated proof kinds.
8. Manager validates:
   - pass cannot claim preserve+invalidate same kind;
   - every performed memory rewrite records proof ID;
   - every invalidating rewrite records delta;
   - report decision points to existing proof in input snapshot.
9. `Report` includes memory delta and snapshot digest before/after application metadata.
10. `fromoptimizer.Delta` converts report to canonical graph delta.
11. Freeze new optimization kinds: do not add new passes until existing six registered passes comply.
12. Add negative tests for stale/missing/mismatched/unsafe proofs.
13. Add differential tests optimized vs unoptimized for memory fixtures.

### Parent production wiring

У canonical build path:

1. Після `ApplyLowering`, якщо `ReleaseOptimize`:
   - get state snapshot;
   - run `opt.NewManager().RunWithOptions` with `RegisteredPasses` and snapshot;
   - apply optimizer delta;
   - run translation validation;
   - rerun bounds/allocation validation on optimized IR;
   - module codegen uses optimized result.
2. Без `ReleaseOptimize`:
   - no optimizer passes;
   - state phase advances without fabricated optimizer facts.
3. Reports consume same optimizer report.

### Focused validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false \
  ./compiler/internal/opt/... \
  ./compiler/internal/differential/... \
  ./compiler/internal/validation/... \
  ./compiler/internal/memoryfacts/fromoptimizer/... \
  -run 'Memory|Proof|NoAlias|Bounds|LICM|Mem2Reg|Scalar|Allocation|Inline' \
  -count=1
```

### Acceptance

- optimizer consumes canonical proofs;
- missing proof = conservative skip;
- every memory rewrite has proof IDs;
- invalidation explicit;
- `ReleaseOptimize` production path validated.

---

## MCV2-T14 — Видалення shadow models

**Owner:** `gpt-5.5-xhigh worker SA-LEGACY-REMOVAL`
**Depends on:** T08, T11, T12, T13
**Completion target:** `INTEGRATION`

### Allowed write scope

```text
compiler/internal/memorymodel/**
compiler/internal/memoryfacts_test/**
compiler/internal/memorypipeline/**
compiler/memoryvocab/**
all Go files importing memorymodel or memoryvocab, після exact parent-approved inventory
```

### Exact work

#### A. `memorymodel/mini.go`

1. Inventory every `Outcome` constant і every test case.
2. Create parity manifest:

```text
reports/stabilization/memory-core-v2/memorymodel-parity.md
```

Для кожного outcome:

```text
old outcome
old test name
new real-pipeline test name
canonical fact/decision code expected
status
```

3. Port every scenario to tests that exercise the real path:

```text
semantics/PLIR -> memorypipeline -> allocplan -> lowering/validation/optimizer as applicable
```

4. Tests не можуть викликати `memorymodel.Evaluate`.
5. Після 100% outcome parity видалити package `compiler/internal/memorymodel`.
6. Якщо хоча б один outcome не має real syntax/runtime path, task status `PARTIAL`; package не видаляти і назвати exact gap.

#### B. `memoryvocab`

1. Migrate all imports to `memoryfacts` typed vocabulary.
2. `rg` має показати zero imports.
3. Видалити compatibility package.
4. Заборонено копіювати constants в інший package.

#### C. Other shadow rules

1. `ramcontract` не має classification policy.
2. `islandkernel` не має duplicate string enums.
3. `allocplan` не має duplicate canonical claim/proof policy.
4. Add architecture test scanning forbidden policy function names outside `memoryfacts`/`islandkernel`.

### Focused validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false ./compiler/... \
  -run 'MemoryIdeal|MemoryCore|Borrow|Inout|Unsafe|Bounds|Storage|Protocol|Async|FFI' \
  -count=1

rg -n 'compiler/internal/memorymodel|memorymodel\.' . --glob '*.go'
rg -n 'compiler/memoryvocab|memoryvocab\.' . --glob '*.go'
```

Zero matches required for `DONE`.

### Acceptance

- every mini-model outcome represented by real-pipeline test;
- `memorymodel` deleted;
- `memoryvocab` deleted;
- no new shadow model introduced.

---

## MCV2-T15 — Evidence, gates, docs і claim hardening

**Owner:** `gpt-5.5-xhigh worker SA-EVIDENCE` for scoped files; parent integrates release gate
**Depends on:** T14
**Completion target:** `END_TO_END`

### Allowed worker scope

```text
docs/spec/**memory**
docs/user/**memory**
docs/release/**memory**
reports/stabilization/memory-core-v2/**
tools/cmd/validate-memory-core-v2/**              (new)
tools/validators/**memory**
scripts/release/memory/memory-core-v2-gate.sh     (new)
```

Parent alone edits `scripts/release/v0_4_0/gate.sh`.

### Exact work

1. Додати spec:

```text
docs/spec/memory/memory_core_v2.md
```

Він описує:

- canonical graph/state;
- phase order;
- proof lifecycle;
- storage planning vs actual lowering;
- domain accounting;
- island epoch/lifecycle;
- backend target support matrix;
- optimizer proof requirements;
- explicit nonclaims.

2. Додати evidence schema/report:

```text
tetra.memory-core-v2.evidence.v1
```

Required report fields:

```text
schema
git_head
target
program_id
memory_graph_digest
module_plan_digests
module_lowering_digests
normal_build_state_built
report_flag_decision_parity
cache_attestation_checked
island_routes_total
island_routes_direct
memorymodel_outcomes_total
memorymodel_outcomes_real_pipeline
backend_operation_support
optimizer_memory_rewrites
optimizer_rewrites_with_proof_ids
negative_guards
nonclaims
final_signoff
```

3. Validator rejects:
   - missing digest;
   - report-only state;
   - route count mismatch;
   - proofless optimizer rewrite;
   - unsupported backend marked supported;
   - mini-model parity incomplete;
   - broad claims;
   - `final_signoff=true` при будь-якому failed requirement.
4. Add positive and negative fixtures.
5. Add gate:

```text
scripts/release/memory/memory-core-v2-gate.sh
```

Gate exact order:

1. focused canonical packages;
2. compiler integration tests;
3. Linux backend/domain/island runtime smoke;
4. optimizer proof tests;
5. report validators;
6. claim scanner;
7. evidence report validation.

6. Parent wires this gate into `scripts/release/v0_4_0/gate.sh` як required memory subgate, але не змінює unrelated release requirements.
7. Evidence/doc duplication cleanup:
   - build checksum inventory of memory audit files;
   - byte-identical duplicates можуть бути видалені тільки якщо no inbound links або links migrated;
   - historical reports з різними results не переписувати;
   - позначити superseded reports через canonical index;
   - status correction лишається authoritative, доки new same-commit evidence не пройде gate.
8. Додати canonical index:

```text
docs/audits/memory/README.md
```

9. Update README/current supported surface лише narrow claims:
   - canonical memory state in normal build;
   - island direct-route/domain lifecycle evidence;
   - Linux backend operation evidence;
   - proof-driven optimizer evidence;
   - no universal memory safety/performance/zero-heap/all-target claim.
10. Run `graphify update .` після implementation changes і перед final evidence.

### Focused validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false \
  ./tools/cmd/validate-memory-core-v2/... \
  ./tools/validators/... \
  -run 'MemoryCoreV2|Memory|Claim' \
  -count=1

bash scripts/release/memory/memory-core-v2-gate.sh
```

### Acceptance

- evidence schema/validator/gate exist;
- release gate consumes memory gate;
- docs match implementation;
- duplicate historical evidence handled without rewriting history;
- unsupported claims rejected.

---

## MCV2-T16 — Final integration, current-main port і final verdict

**Owner:** parent controller
**Depends on:** T15
**Completion target:** `FINAL`

### 16.1. Pre-final source audit

```bash
git status --short --branch
git diff --check
rg -n 'FromPLIRAndAllocPlan|FromPLIRWithOptions|looksActorSend|looksTaskBoundary|actualLoweringStorage' compiler --glob '*.go'
rg -n 'RouteValidatedEquivalent' compiler/internal/islandkernel --glob '*.go'
rg -n 'compiler/internal/memorymodel|compiler/memoryvocab' . --glob '*.go'
rg -n 'plir\.FromCheckedProgram|allocplan\.(Build|FromPLIR)|lower\.Lower' compiler/compiler_reports.go
```

Усі forbidden scans мають zero matches.

### 16.2. Full validation

```bash
GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false ./compiler/... -count=1

GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false ./cli/... ./tools/... -count=1

GOTELEMETRY=off GOCACHE="$GOCACHE" GOTMPDIR="$GOTMPDIR" \
  go test -buildvcs=false ./... -count=1

bash scripts/release/memory/memory-core-v2-gate.sh
bash scripts/release/v0_4_0/gate.sh
graphify update .
git diff --check
```

Якщо repo-wide test має unrelated baseline failure, final status не може бути `DONE`, доки failure не класифіковано доказово як pre-existing і affected surfaces все одно fully green. За repo rules unknown gap = `PARTIAL` або `BLOCKED`.

### 16.3. Runtime matrix

Mandatory:

```text
linux-x64 executable build/run
linux-x64 island lifecycle smoke
linux-x64 backend operation smoke
linux-x64 report parity build
wasm32-wasi contract validation with explicit unsupported operations
wasm32-web contract validation with explicit unsupported operations
```

WASM runtime execution не потрібне для operations, позначених unsupported; validator correctness потрібна.

### 16.4. Current-main integration

Після green base branch:

1. Record current `origin/main` HEAD.
2. Rebase або merge у fresh integration worktree.
3. Не вирішувати semantic conflicts шляхом вибору `ours/theirs` для memory files.
4. Для conflict у critical architecture symbol:
   - stop;
   - update `head-drift.md`;
   - status `BLOCKED` до plan amendment.
5. Повторити full validation на integrated HEAD.

### 16.5. Final evidence artifacts

Обов’язкові:

```text
reports/stabilization/memory-core-v2/mcv2-t00-baseline.md
reports/stabilization/memory-core-v2/head-drift.md                  (якщо був drift)
reports/stabilization/memory-core-v2/memorymodel-parity.md
reports/stabilization/memory-core-v2/memory-core-v2-evidence.json
reports/stabilization/memory-core-v2/memory-core-v2-final.md
reports/stabilization/memory-core-v2/test-commands.txt
reports/stabilization/memory-core-v2/runtime-linux-x64.json
reports/stabilization/memory-core-v2/report-flag-parity.json
reports/stabilization/memory-core-v2/cache-attestation.json
```

### 16.6. Final status rule

`DONE` лише якщо:

- усі T00-T16 accepted;
- all required gates pass на integrated HEAD;
- no shadow model/import remains;
- no report-only decision path remains;
- 16/16 island routes direct;
- Linux backend smoke real;
- optimizer memory rewrites proof-carrying;
- memorymodel parity complete;
- no known blocker.

`PARTIAL` якщо implementation substantial, але залишився exact unverified/unsupported/incomplete slice.

`BLOCKED` якщо architecture cannot be integrated, required runtime path unavailable, exact subagent model unavailable для delegated task, або critical conflict не має plan-defined resolution.

---

## 10. Required regression matrix

### 10.1. Graph and facts

- insertion-order digest invariance;
- snapshot immutability;
- conflicting fact rejection;
- invalid stage regression rejection;
- atomic delta rollback;
- parent/derived consistency;
- unsafe_unknown optimization authorization rejection;
- stale proof rejection.

### 10.2. Allocation planning

- no escape -> eligible trusted storage only with proof;
- return/global/call unknown/closure/aggregate -> conservative/heap;
- actor/task typed boundary;
- names containing `actor`/`task` without facts do nothing;
- explicit island requires exact epoch/owner;
- external/unsafe root never trusted;
- heap reason codes complete;
- actual unknown until lowering.

### 10.3. Lowering

- plan branch matches emitted IR;
- fallback truthfully recorded;
- one evidence row per allocation;
- artifact range valid;
- module index includes wrappers;
- no duplicate plan construction;
- actual storage validator rejects fabricated row.

### 10.4. Compiler pipeline/cache

- report/no-report plan parity;
- report/no-report lowering parity;
- one PLIR and one plan;
- zero/one lower count;
- cache digest hit;
- old cache miss;
- corrupted digest miss;
- report on cache hit uses real lowering;
- executable behavior unchanged.

### 10.5. Domains/backend

- parent missing/cycle;
- budget overflow;
- reserve/commit/allocate/free/decommit/release ordering;
- closed domain events;
- move/copy semantics;
- island reset epoch;
- task/actor ownership transfer;
- Linux backend real operations;
- WASM unsupported operations explicit;
- footprint evidence class truthful.

### 10.6. Island kernel

For each direct decision:

- positive case;
- negative case;
- conservative case where defined;
- exact reason code;
- production route coverage.

Mandatory negative cases:

- borrow with consumed token;
- island mismatch;
- stale epoch;
- borrowed return/global/closure/task/actor escape;
- move/free/reset consumed token;
- free/reset with live borrow;
- unsafe/external noalias;
- proof mismatch;
- escaped explicit island storage;
- heap fallback trusted claim;
- unsafe root safe promotion;
- runtime check erasure without proof.

### 10.7. Optimizer

- each registered pass metadata valid;
- proofless memory rewrite skipped;
- stale proof skipped;
- unsafe proof skipped;
- proof IDs recorded;
- invalidations applied;
- bounds proof preserved/mapped or check restored;
- differential optimized/unoptimized equality;
- release build uses optimized canonical IR.

### 10.8. Reports and claims

- schema v1 compatibility;
- projection determinism;
- graph digest included;
- report mutation cannot affect state;
- unsupported target cannot show measured bytes;
- no final signoff on incomplete evidence;
- claim scanner rejects `Memory 100%`, universal zero heap, all-target RSS parity.

---

## 11. Заборонені shortcuts — final scan

Final gate має падати при виявленні:

```text
new package that reimplements memory scenario evaluation
new string heuristic for actor/task/request/domain classification
planner-written actual lowering
report-triggered graph construction
report-triggered allocation planning
optimizer memory rewrite without proof ID
RouteValidatedEquivalent for required island route
WASM unsupported operation returning success/no-op
measured footprint with synthetic zero values
hard-coded benchmark/source recognizer
compatibility wrapper that rebuilds PLIR/plan inside lower
```

Recommended scanner patterns:

```bash
rg -n 'looksActor|looksTask|looksRequest|strings\.Contains\(.*actor|strings\.Contains\(.*task' compiler/internal
rg -n 'ActualLoweringStorage\s*=' compiler/internal/allocplan
rg -n 'FromCheckedProgram|allocplan\.(Build|FromPLIR)|LowerPlannedProgram' compiler/compiler_reports.go
rg -n 'RouteValidatedEquivalent' compiler/internal/islandkernel
rg -n 'OutcomeValidBorrowLocal|func Evaluate\(s Scenario\)' compiler
```

Кожен non-zero result має бути reviewed і documented; forbidden production use блокує `DONE`.

---

## 12. Commit sequence

Не squash до final integration. Recommended commits:

```text
memory-core-v2: freeze baseline and workflow
memory-core-v2: canonicalize memory facts vocabulary
memory-core-v2: split memory fact adapters
memory-core-v2: add immutable snapshots and proof deltas
memory-core-v2: add canonical memory pipeline state
memory-core-v2: make allocation planning fact driven
memory-core-v2: make lowering plan driven with evidence
memory-core-v2: wire normal build and cache attestations
memory-core-v2: make reports read-only projections
memory-core-v2: implement memory domain ledger
memory-core-v2: wire executable backend operations
memory-core-v2: route island decisions through kernel
memory-core-v2: wire task actor request domains
memory-core-v2: make optimizer proof driven
memory-core-v2: remove shadow memory models
memory-core-v2: add evidence validator and release gate
memory-core-v2: finalize integrated evidence
```

---

## 13. Parent controller phase checklist

Перед кожним phase transition:

1. Re-read canonical plan, `GOAL.md`, `CONTROL.md`.
2. Inspect current diff.
3. Confirm previous task acceptance.
4. Update `ATTEMPTS.md` і `GOAL.md` progress.
5. Confirm no open review findings.
6. Confirm next worker write scope disjoint.
7. Spawn only exact allowed worker.
8. Re-run focused tests after integration.
9. Commit accepted task.
10. Never mark project `DONE` from worker-local green tests.

Якщо та сама fix strategy fails двічі:

- stop that strategy;
- record both attempts;
- write exact blocker;
- do not try a third variant without new evidence or plan amendment.

---

## 14. Final handoff template

```markdown
Status: DONE | PARTIAL | BLOCKED

Completed:
- ...

Scope covered:
- canonical memoryfacts graph/snapshot
- allocation planning
- lowering
- compiler normal build/cache
- reports/RAM contract
- domains/backend
- islands
- task/actor/request transfer
- optimizer
- legacy model removal
- validators/gates/docs

Validation:
- `<exact command>` — PASS/FAIL
- ...

Evidence:
- `path`
- commit/HEAD
- graph/plan/lowering digests
- runtime reports

Not verified / risks:
- exact residual gap, or `none known`

Direct-solution statement:
- Confirm whether the delivered path solves the requested architecture directly.
- Name any remaining wrapper, fallback, mock, unsupported target, or provisional path.
```

---

## 15. Final Definition of Done

Memory Core v2 вважається завершеним лише тоді, коли одночасно істинні всі твердження:

1. Normal executable build створює canonical memory state без report flags.
2. PLIR і allocation plan будуються один раз.
3. Lowering виконується не більше одного разу та споживає exact plan.
4. Actual lowering походить із emitted branch evidence.
5. Cache hits перевіряють plan/lowering attestations.
6. Reports є projections і не запускають decision pipeline.
7. `memoryfacts` core pure та typed.
8. Duplicate root/fromplir builders видалені.
9. `memoryvocab` і `memorymodel` shadow layers видалені після parity.
10. `allocplan` не використовує actor/task/request string heuristics.
11. `MemoryDomainLedger` enforce lifecycle, DAG, budgets і accounting.
12. Linux-x64 backend operations виконуються real runtime path.
13. WASM unsupported operations позначені чесно.
14. Island domain має owner, parent, epoch, reset/free/move semantics.
15. 16/16 dangerous island decisions мають direct kernel routes.
16. Task/actor/request domains і move/copy semantics proof-carrying.
17. Optimizer memory rewrites мають exact canonical proof IDs.
18. Missing/stale/unsafe proof дає conservative skip або rejection.
19. Existing report schemas лишаються compatible.
20. Full compiler/tool/release gates проходять на integrated HEAD.
21. Немає broad unsupported memory/performance claims.
22. Final status відповідає repo completion rules.
