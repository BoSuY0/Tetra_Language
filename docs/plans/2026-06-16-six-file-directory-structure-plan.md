# План структури з правилом 6 файлів

**Мета:** зробити проект читабельним: у кожній людській, підтримуваній
директорії має бути не більше 6 активних source/script файлів.

**Контекст:** репозиторій уже рухається до domain-owned директорій, але частина
структури досі занадто пласка. Найбільші поточні проблемні місця:

- `examples/microservices`: 313 active source/script files.
- `examples`: 164 active source/script files.
- `compiler`: 119 active source/script files.
- `tools/validators/surface`: 73 active source/script files.
- `cli/cmd/tetra`: 73 active source/script files.
- `tools/scriptstest`: 68 active source/script files.
- `compiler/tests/semantics`: 55 active source/script files.
- `compiler/internal/semantics`: 54 active source/script files.
- `compiler/internal/lower`: 39 active source/script files.
- `lib/core`: 31 active source/script files.

## Правило бюджету директорії

1. В одній директорії може бути максимум 6 активних source/script файлів.
2. Активні source/script файли: `.go`, `.tetra`, `.sh`, `.mjs`, `.js`,
   `.ts`.
3. Конфіги й навігаційні файли не рахуються в цей бюджет:
   `README.md`, `AGENTS.md`, `go.mod`, `go.sum`, `package.json`, lock files,
   schemas, generated manifests, and indexes.
4. Тести рахуються в той самий бюджет, якщо вони лежать поруч із кодом.
   Якщо тестів багато, вони мають жити в окремій гілці `tests/`, `testkit/`,
   `fixtures/` або `testdata/`.
5. Згенеровані логи, кеші й машинні артефакти не є джерелами структури.
   Вони мають жити в `generated/`, `reports/`, `.cache/` або `.tetra_cache/`,
   а не змішуватись із підтримуваним кодом.
6. Батьківська директорія зазвичай має бути фасадом або роутером. Реальна
   реалізація має лежати в малих доменних директоріях.

## Цільова форма

Це ідеальна цільова форма, а не прямий move в один commit. Назви можна уточнювати
під час міграції, але правило бюджету має лишитись незмінним.

```text
Tetra_Language/
├── compiler/
│   ├── go.mod
│   ├── api.go
│   ├── compiler.go
│   ├── diagnostics.go
│   ├── format.go
│   ├── manifest.go
│   ├── version.go
│   ├── facade/
│   │   ├── lsp.go
│   │   ├── reports.go
│   │   ├── targets.go
│   │   ├── runtime.go
│   │   ├── features.go
│   │   └── compatibility.go
│   ├── target/
│   ├── actorwire/
│   ├── memoryvocab/
│   ├── selfhostrt/
│   ├── internal/
│   │   ├── frontend/
│   │   │   ├── ast/
│   │   │   ├── lexer/
│   │   │   ├── parser/
│   │   │   ├── recovery/
│   │   │   ├── diagnostics/
│   │   │   ├── fixtures/
│   │   │   └── tests/
│   │   ├── semantics/
│   │   │   ├── builtins/
│   │   │   ├── checker/
│   │   │   │   ├── core.go
│   │   │   │   ├── declarations.go
│   │   │   │   ├── expressions.go
│   │   │   │   ├── statements.go
│   │   │   │   ├── resources.go
│   │   │   │   └── policies.go
│   │   │   ├── expressions/
│   │   │   ├── functions/
│   │   │   ├── generics/
│   │   │   ├── regions/
│   │   │   ├── actors/
│   │   │   ├── memory/
│   │   │   ├── representation/
│   │   │   ├── manifest/
│   │   │   ├── diagnostics/
│   │   │   └── tests/
│   │   ├── lower/
│   │   │   ├── core/
│   │   │   ├── callables/
│   │   │   ├── constructors/
│   │   │   ├── expressions/
│   │   │   ├── statements/
│   │   │   ├── lvalues/
│   │   │   ├── tasks/
│   │   │   ├── rangeproof/
│   │   │   ├── atomic/
│   │   │   ├── ui/
│   │   │   ├── verify/
│   │   │   └── tests/
│   │   ├── ir/
│   │   │   ├── plir/
│   │   │   ├── ssair/
│   │   │   ├── validation/
│   │   │   ├── allocation/
│   │   │   ├── proofs/
│   │   │   └── tests/
│   │   ├── backend/
│   │   │   ├── shared/
│   │   │   ├── x64abi/
│   │   │   ├── x64core/
│   │   │   ├── x64obj/
│   │   │   ├── wasm32_web/
│   │   │   ├── wasm32_wasi/
│   │   │   └── native_shell/
│   │   ├── build/
│   │   │   ├── api/
│   │   │   ├── plan/
│   │   │   ├── runtime/
│   │   │   ├── native/
│   │   │   ├── wasm/
│   │   │   ├── link/
│   │   │   └── reports/
│   │   ├── runtime/
│   │   │   ├── actors/
│   │   │   ├── http/
│   │   │   ├── json/
│   │   │   ├── net/
│   │   │   ├── pg/
│   │   │   ├── web/
│   │   │   └── stdlib/
│   │   ├── gates/
│   │   │   ├── abi/
│   │   │   ├── atomic/
│   │   │   ├── fuzz/
│   │   │   ├── formal/
│   │   │   ├── security/
│   │   │   └── selfhost/
│   │   └── testkit/
│   ├── tests/
│   │   ├── backend/
│   │   ├── callables/
│   │   ├── frontend/
│   │   ├── lowering/
│   │   ├── ownership/
│   │   ├── runtime/
│   │   ├── safety/
│   │   └── semantics/
│   └── testdata/
│       ├── backend/
│       ├── callables/
│       ├── frontend/
│       ├── lowering/
│       ├── ownership/
│       ├── safety/
│       └── semantics/
│
├── cli/
│   ├── cmd/
│   │   └── tetra/
│   │       ├── main.go
│   │       ├── root.go
│   │       └── wire.go
│   ├── internal/
│   │   ├── commands/
│   │   │   ├── build/
│   │   │   ├── check/
│   │   │   ├── clean/
│   │   │   ├── doctor/
│   │   │   ├── eco/
│   │   │   ├── fmt/
│   │   │   ├── interface/
│   │   │   ├── lsp/
│   │   │   ├── metadata/
│   │   │   ├── newapp/
│   │   │   ├── project/
│   │   │   ├── run/
│   │   │   ├── smoke/
│   │   │   ├── surface/
│   │   │   ├── testcmd/
│   │   │   └── workspace/
│   │   ├── actornet/
│   │   ├── config/
│   │   ├── diagnostics/
│   │   └── sourcefiles/
│   ├── testkit/
│   └── tests/
│       ├── commands/
│       ├── eco/
│       ├── lsp/
│       ├── smoke/
│       ├── surface/
│       └── workspace/
│
├── tools/
│   ├── cmd/
│   │   ├── validate/
│   │   │   ├── directory-budget/
│   │   │   ├── memory/
│   │   │   ├── ownership/
│   │   │   ├── docs/
│   │   │   ├── release/
│   │   │   └── workspace/
│   │   ├── smoke/
│   │   │   ├── compiler/
│   │   │   ├── memory/
│   │   │   ├── surface/
│   │   │   ├── runtime/
│   │   │   └── wasm/
│   │   ├── report/
│   │   │   ├── checklist/
│   │   │   ├── project/
│   │   │   ├── benchmark/
│   │   │   ├── release/
│   │   │   └── docs/
│   │   └── dev/
│   │       ├── benchmark/
│   │       ├── dump/
│   │       └── fixtures/
│   ├── validators/
│   │   ├── common/
│   │   ├── surface/
│   │   │   ├── app/
│   │   │   ├── artifacts/
│   │   │   ├── block/
│   │   │   ├── browser/
│   │   │   ├── i18n/
│   │   │   ├── inspector/
│   │   │   ├── motion/
│   │   │   ├── performance/
│   │   │   ├── release/
│   │   │   ├── render/
│   │   │   ├── report/
│   │   │   ├── security/
│   │   │   ├── tokens/
│   │   │   ├── visual/
│   │   │   └── widgets/
│   │   ├── memory/
│   │   ├── actor/
│   │   ├── release/
│   │   ├── wasm/
│   │   └── workspace/
│   ├── release/
│   │   ├── gates/
│   │   ├── security/
│   │   ├── smoke/
│   │   └── evidence/
│   ├── internal/
│   │   ├── artifacts/
│   │   ├── cache/
│   │   ├── reports/
│   │   ├── render/
│   │   └── telemetry/
│   ├── testkit/
│   └── scriptstest/
│       ├── ci/
│       ├── release/
│       │   ├── v0_1/
│       │   ├── v0_2/
│       │   ├── v0_3/
│       │   ├── v0_4/
│       │   ├── v1_0/
│       │   └── current/
│       ├── security/
│       ├── surface/
│       ├── workspace/
│       └── fixtures/
│
├── lib/
│   ├── core/
│   │   ├── base/
│   │   │   ├── math.tetra
│   │   │   ├── time.tetra
│   │   │   ├── strings.tetra
│   │   │   ├── testing.tetra
│   │   │   └── capability.tetra
│   │   ├── collections/
│   │   ├── memory/
│   │   ├── async/
│   │   ├── io/
│   │   ├── net/
│   │   ├── data/
│   │   │   ├── json.tetra
│   │   │   ├── serialization.tetra
│   │   │   ├── crypto.tetra
│   │   │   └── postgres.tetra
│   │   └── text/
│   ├── block/
│   │   ├── accessibility/
│   │   ├── component/
│   │   ├── draw/
│   │   ├── style/
│   │   ├── widgets/
│   │   └── surface/
│   ├── morph/
│   │   ├── core/
│   │   ├── app/
│   │   ├── shell/
│   │   ├── render/
│   │   └── demos/
│   └── experimental/
│       ├── actor/
│       ├── backend/
│       ├── memory/
│       ├── surface/
│       └── wasm/
│
├── examples/
│   ├── smoke/
│   │   ├── core/
│   │   ├── ownership/
│   │   ├── tasks/
│   │   ├── ui/
│   │   └── wasm/
│   ├── projects/
│   │   ├── dogfood_actor_task/
│   │   ├── dogfood_cli/
│   │   ├── dogfood_wasi/
│   │   ├── dogfood_web_ui/
│   │   ├── eco_dogfood/
│   │   ├── hello_t4/
│   │   └── tetra_control_center/
│   ├── microservices/
│   │   ├── compiler/
│   │   │   ├── actor/
│   │   │   ├── async/
│   │   │   ├── callables/
│   │   │   ├── generics/
│   │   │   ├── memory/
│   │   │   ├── optional/
│   │   │   ├── protocols/
│   │   │   ├── resources/
│   │   │   └── tasks/
│   │   ├── backend/
│   │   │   ├── capsule/
│   │   │   ├── modular_web/
│   │   │   └── services/
│   │   └── parallel/
│   ├── benchmarks/
│   │   ├── compiler/
│   │   ├── runtime/
│   │   ├── memory/
│   │   └── surface/
│   └── surface/
│       ├── reference/
│       ├── product_slice/
│       ├── morph/
│       └── toon/
│
└── docs/
    ├── architecture/
    │   ├── index.md
    │   ├── compiler/
    │   ├── cli/
    │   ├── tools/
    │   ├── runtime/
    │   ├── surface/
    │   └── structure/
    ├── spec/
    │   ├── language/
    │   ├── compiler/
    │   ├── runtime/
    │   ├── surface/
    │   ├── memory/
    │   └── release/
    ├── plans/
    │   ├── active/
    │   ├── completed/
    │   │   ├── compiler/
    │   │   ├── surface/
    │   │   ├── actor/
    │   │   ├── memory/
    │   │   ├── release/
    │   │   └── benchmarks/
    │   ├── prompts/
    │   └── archive/
    ├── release/
    │   ├── current/
    │   ├── v0_1/
    │   ├── v0_2/
    │   ├── v0_3/
    │   ├── v0_4/
    │   └── v1_0/
    ├── user/
    │   ├── guides/
    │   ├── cookbook/
    │   ├── examples/
    │   └── troubleshooting/
    ├── audits/
    │   ├── compiler/
    │   ├── memory/
    │   ├── surface/
    │   ├── actor/
    │   └── release/
    ├── generated/
    │   ├── v1_0/
    │   └── release/
    └── assets/
```

## План міграції

### Task 1: Додати validator бюджету директорій

**Goal:** зробити правило `<=6` вимірюваним до переміщення файлів.

**Files:**
- Add `tools/cmd/validate/directory-budget/`.
- Add focused tests under `tools/scriptstest/structure/` or
  `tools/scriptstest/workspace/`.

**Approach:**
- Рахувати тільки активні source/script extensions.
- Виключити кеші, generated artifacts, lock files і navigation docs.
- Показувати кожну директорію-порушника з кількістю файлів і списком файлів.
- Дозволити тимчасовий baseline, щоб зменшувати борг поступово.

**Verification:**
- `go test -buildvcs=false ./tools/... -count=1`
- Run the validator and confirm it reports the known hotspots.

**Done when:** CI може впасти, якщо нова директорія перевищує 6 активних файлів.

### Task 2: Спочатку розділити docs

**Goal:** прибрати найбільш шумні planning/audit директорії без зміни поведінки
коду.

**Files:**
- Move `docs/plans/*` into `active/`, `completed/<domain>/`, `prompts/`, and
  `archive/`.
- Move `docs/audits/*`, `docs/spec/*`, and `docs/release/*` into topic
  directories.
- Update links from `README.md`, `docs/architecture/*`, release checklists, and
  any validator that reads docs paths.

**Approach:**
- Залишити `docs/plans/index.md` як точку входу.
- Рухати максимум один docs domain за один PR.
- По можливості зберігати історичні назви файлів.

**Verification:**
- `go test -buildvcs=false ./tools/... -run Test.*Docs -count=1`
- `go test -buildvcs=false ./tools/... -count=1`
- Run docs/report validators that reference `docs/plans`, `docs/spec`, or
  `docs/release`.

**Done when:** жодна docs leaf-директорія не має більше 6 підтримуваних Markdown
файлів.

### Task 3: Розділити examples і microservices

**Goal:** зробити examples читабельними за use case, а не одним великим списком.

**Files:**
- Move root `examples/*.tetra` into `examples/smoke/*`,
  `examples/benchmarks/*`, or `examples/surface/*`.
- Move `examples/microservices/*` into
  `compiler/{actor,async,callables,generics,memory,optional,protocols,resources,tasks}`,
  `backend/*`, or `parallel/*`.
- Update smoke registries, docs, and script tests that enumerate example paths.

**Approach:**
- Залишити compatibility index, якщо path discovery цього потребує.
- Використовувати категорії замість довгих prefix-назв файлів.
- Не рухати generated `.tetra_cache` outputs.

**Verification:**
- `go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1`
- Run any existing smoke command that enumerates `examples/`.
- Run the directory-budget validator.

**Done when:** `examples/` і `examples/microservices/` не мають прямих активних
source файлів, крім indexes або launchers.

### Task 4: Розділити CLI command package

**Goal:** зробити `cli/cmd/tetra` тонким entrypoint для команди.

**Files:**
- Keep only `main.go`, `root.go`, and `wire.go` in `cli/cmd/tetra`.
- Move command implementation to `cli/internal/commands/<command>/`.
- Move command tests to `cli/tests/commands/<command>/` or command-local
  internal test directories.
- Keep shared helpers in `cli/testkit`.

**Approach:**
- Мігрувати по одній command family: `doctor`, `fmt`, `build`, `check`,
  `eco`, `lsp`, `surface`, `workspace`, then smoke/test commands.
- Зробити так, щоб root command wiring імпортував internal command packages.
- Не відкривати command internals як public API.

**Verification:**
- `go test -buildvcs=false ./cli/... -count=1`
- CLI smoke tests for `tetra check`, `tetra build`, `tetra run`,
  `tetra fmt`, `tetra lsp`, `tetra eco`, and workspace commands.
- Run the directory-budget validator.

**Done when:** `cli/cmd/tetra` є тільки binary entrypoint і має максимум 6
активних source файлів.

### Task 5: Розділити tools validators і script tests

**Goal:** зробити validators і release tests доменними, а не пласкими.

**Files:**
- Split `tools/validators/surface` by concern: `app`, `block`, `render`,
  `report`, `release`, `security`, `tokens`, `visual`, and `widgets`.
- Split `tools/scriptstest` into `ci`, `release/<version>`, `security`,
  `surface`, `workspace`, and `fixtures`.
- Move shared helper code to `tools/testkit` or `tools/internal/*`.

**Approach:**
- Виносити shared helpers тільки після того, як хоча б три callers підтвердять
  реальну межу.
- Тримати validator package APIs малими; re-export через surface aggregator
  робити тільки якщо existing commands потребують стабільних imports.

**Verification:**
- `go test -buildvcs=false ./tools/... -count=1`
- Release/surface validator smoke commands.
- Run the directory-budget validator.

**Done when:** `tools/validators/surface` і `tools/scriptstest` більше не є
пласкими директоріями.

### Task 6: Розділити compiler facade і internal domains

**Goal:** зробити `compiler/` справжнім facade і перенести реалізацію в internal
domain packages.

**Files:**
- Keep only public facade files in `compiler/`.
- Move ABI, atomic, formal, fuzz, security, compatibility, runtime hardening,
  and self-hosting gates into `compiler/internal/gates/*`.
- Split `compiler/internal/semantics` into domain packages.
- Split `compiler/internal/lower` into domain packages.
- Move broad tests into `compiler/tests/<domain>/` and helpers into
  `compiler/internal/testkit`.

**Approach:**
- Почати з tests і gates, бо там межі найзрозуміліші.
- Потім розділити `semantics` і `lower` за реальними dependency boundaries.
- Якщо white-box test потребує unexported symbols, спочатку винести production
  code в менший subpackage, а потім тримати тест поруч із ним.

**Verification:**
- `go test -buildvcs=false ./compiler/... -count=1`
- `go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1`
- Run the directory-budget validator.
- Run `graphify update .` after code moves.

**Done when:** `compiler/`, `compiler/internal/semantics`,
`compiler/internal/lower` і compiler test directories дотримуються бюджету
6 файлів.

### Task 7: Розділити standard library domains

**Goal:** зробити `lib/core` читабельним і не тримати UI/surface APIs у базовому
stdlib bucket.

**Files:**
- Move base APIs into `lib/core/base`.
- Move collections, async, memory, IO, net, data, and text into separate
  directories.
- Move block UI APIs into `lib/block`.
- Move morph/surface app APIs into `lib/morph`.
- Keep experimental APIs under `lib/experimental/<domain>`.

**Approach:**
- Додати temporary import aliases, якщо Tetra module resolution потребує
  стабільних шляхів.
- Оновлювати examples і docs в тому самому slice, де рухається stdlib module.

**Verification:**
- Run compiler tests that compile stdlib examples.
- Run example smoke tests.
- Run the directory-budget validator.

**Done when:** `lib/core` є малим index/facade, а не bucket на 31 файл.

## Критерії прийняття

- Кожна людська source/script директорія в `compiler`, `cli`, `tools`, `lib`,
  `examples` і `docs` має максимум 6 активних файлів.
- Parent directories є facades, indexes або routers.
- Tests, fixtures, generated artifacts і reports відділені від implementation.
- Усі moved imports, docs links, smoke registries і release gates оновлені.
- Broad verification проходить:

```sh
GOTELEMETRY=off \
GOCACHE="$(pwd)/.cache/go-build-six-file-structure" \
GOTMPDIR="$(pwd)/.cache/go-tmp-six-file-structure" \
go test -buildvcs=false ./compiler/... ./cli/... ./tools/... -count=1
```

- Фінальна міграція запускає `graphify update .`, щоб architecture graph був
  актуальним.

## Чернетка перевірки бюджету

Ця команда показує поточні порушення бюджету active source/script файлів:

```sh
find compiler cli tools lib examples docs \
  \( -path '*/.cache/*' -o -path '*/.tetra_cache/*' -o -path '*/node_modules/*' \) -prune -o \
  -type f \( -name '*.go' -o -name '*.tetra' -o -name '*.sh' -o -name '*.mjs' -o -name '*.js' -o -name '*.ts' \) -print |
awk '{p=$0; sub("/[^/]+$","",p); c[p]++} END{for(p in c) if(c[p]>6) print c[p], p}' |
sort -nr
```

## Відкриті рішення

Потрібно остаточно вирішити:

- Чи має `*_test.go` рахуватись у тому самому budget, чи tests завжди мають
  їхати в mirrored `tests/<domain>/` directories?
- Чи docs мають мати таке саме hard `<=6` правило для Markdown files, чи для
  docs достатньо м'якшого topic-index rule?
- Чи validator має стартувати як advisory з baseline, чи одразу падати тільки
  на нових порушеннях?

Рекомендовані відповіді:

- Рахувати tests у бюджет, якщо вони не лежать в окремій test branch.
- Застосувати правило і до docs, але дозволити generated docs під
  `docs/generated`.
- Почати з baseline, а потім зменшувати борг по одному domain за раз.
