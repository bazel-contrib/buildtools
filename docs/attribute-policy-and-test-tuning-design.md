# Design: Attribute-Policy Linting & Empirical Test-Tuning

Status: **Draft for review**
Owner: (tbd)
Last updated: 2026-07-01

---

## 1. Summary

We want two related capabilities for Bazel `BUILD` files:

1. **Static attribute-policy linting** in `buildifier` — enforce declarative rules about
   attribute values, e.g. forbid `timeout = "eternal"` unless the target is on an
   approved allow-list, forbid `exclusive` in a test's `tags`, cap `shard_count` at 50,
   and similar attribute/rule-kind constraints. Purely static, config-driven, no runtime data.

2. **Empirical test-tuning** in a **new, separate tool** (`testpolicy`) — read
   historical test-execution stats from a metrics warehouse, compute recommended
   `timeout`, `flaky`, and (in theory) `shard_count` values, and **emit the `buildozer`
   commands** that would repair the repo. `buildifier` never touches the warehouse.

   `timeout`, `flaky`, and `shard_count` are the three test attributes meant to reflect
   **how the test currently executes**, not how it was designed. `shard_count` tuning is
   feasible only if telemetry can separate fixture/SUT setup time from assertion time;
   otherwise the tool cannot tell whether more shards would shorten the critical path.

These are split deliberately: `buildifier` stays a fast, hermetic, offline static
linter; all data-dependent analysis lives in a tool that queries a warehouse and
produces buildozer commands. **The tool's job ends when those commands are produced** —
running them, opening PRs, and routing to reviewers are downstream concerns left to
whatever CI/automation invokes the tool.

### Decisions locked in (from design review)

| Question | Decision |
|---|---|
| Source of empirical data | Metrics DB / warehouse (queried by the new tool) |
| How empirical recommendations are applied | Tool emits `buildozer` commands (its terminal deliverable); applying/PRs are downstream. `buildifier` stays purely static |
| Where policy config lives | Extend the existing `.buildifier.json` config |

### Motivating use cases

| Policy | Encoding | Context |
|---|---|---|
| No `timeout = "eternal"` without approval | `forbidValues` + `allowlist` | Initial ask |
| No `exclusive` in test `tags` | `forbidListItems` | Initial ask |
| No `local = True` on tests | `forbidValues: ["True"]` | Hermetic-test policy |
| No `execution_requirements["no-cache"] = "1"` | `forbidDictEntries` | Cache-bypass policy |
| `shard_count` ≤ 50 on tests | `maxValue: 50` | [figma/bazel#12](https://github.com/figma/bazel/pull/12) removed Bazel's hardcoded 50-shard cap; repos that want to keep that limit (or any other bound) can enforce it in buildifier instead of forking Bazel |

---

## 2. Background: how `buildifier` warnings work today

(Reference for implementing agents. File paths are current as of this draft.)

- **Warnings are pure functions** over one parsed file. Three signatures /
  registries in `warn/warn.go`:
  - `FileWarningMap: map[string]func(f *build.File) []*LinterFinding`
  - `MultiFileWarningMap: map[string]func(f *build.File, fileReader *FileReader) []*LinterFinding`
  - `RuleWarningMap: map[string]func(call *build.CallExpr, pkg string) *LinterFinding`
- **Findings** are created with `makeLinterFinding(node, message, ...LinterReplacement)`
  (autofix optional). Nodes carry positions via `.Span()`.
- **Rule/attribute access** (`build/rule.go`):
  - `f.Rules(kind string) []*Rule` (`kind == ""` → all)
  - `rule.Kind() string`, `rule.Name() string`, `rule.ExplicitName() string`
  - `rule.Attr(key) Expr`, `rule.AttrString(key) string`, `rule.AttrStrings(key) []string`
  - `rule.AttrDefn(key) *AssignExpr`
- **Config-derived globals** already exist: the `tables` package holds process-global
  overrides loaded from config (`tables.ParseAndUpdateJSONDefinitions`, applied in
  `buildifier/config/config.go` `Validate()` around lines 214–231). We mirror this
  pattern for policy config.
- **Config file**: `.buildifier.json` → `buildifier/config.Config` (struct at
  `buildifier/config/config.go:93`). Located via `-config` flag,
  `BUILDIFIER_CONFIG` env var, or workspace root. `Validate()` applies side effects
  (like loading tables). Lint entry point is `buildifier/utils/utils.go:126`
  `Lint(...)` → `warn.FileWarnings(...)`.
- **Suppression**: `# buildifier: disable=<warning-name>` comments already work for
  every registered warning; the new warning gets this for free.
- **Label utilities** (`labels/labels.go`): `labels.Parse(target) Label`,
  `labels.Equal(l1, l2, pkg) bool`; `Label{Repository, Package, Target}`.
- **Tests**: `warn/*_test.go` use `checkFindingsAndFix(t, categories, input, output,
  expected, scope)` and scope constants (`scopeBuild`, `scopeBzl`, ...) from
  `warn/warn_test.go`.

---

## 3. Workstream A — `attr-policy` warning (buildifier)

### 3.1 Goal

A **single generalized** warning, `attr-policy`, driven entirely by config. It covers
the initial asks (eternal-timeout allow-list, no `exclusive` on tests) and any future
"attribute constrained on rule-kind unless allow-listed" rule **without new Go code** —
including boolean literals (`local = True`), dict attributes
(`execution_requirements = {"no-cache": "1"}`), and numeric bounds (`shard_count` ≤ 50).

Rationale for one generalized warning vs. many hardcoded ones: buildifier warnings are
individually toggleable by name, but the *policy content* here is site-specific and
belongs in config, not in the binary. One warning + rich config keeps the binary
generic and lets each repo express its own policy.

### 3.2 Config schema (extends `.buildifier.json`)

`.buildifier.json` is parsed with standard `encoding/json` (no comments). Examples
below are valid JSON you can copy into config.

```json
{
  "attrPolicy": {
    "rules": [
      {
        "name": "no-eternal-timeout",
        "ruleKinds": ["*_test"],
        "attr": "timeout",
        "forbidValues": ["eternal"],
        "allowlist": ["//slow/...", "//foo:big_test"],
        "message": "'eternal' timeout requires approval; add the target to the attrPolicy allowlist."
      },
      {
        "name": "no-exclusive-tests",
        "ruleKinds": ["*_test"],
        "attr": "tags",
        "forbidListItems": ["exclusive"],
        "allowlist": []
      },
      {
        "name": "no-local-tests",
        "ruleKinds": ["*_test"],
        "attr": "local",
        "forbidValues": ["True"],
        "message": "Tests must not set local = True; use a hermetic test instead."
      },
      {
        "name": "no-no-cache",
        "attr": "execution_requirements",
        "forbidDictEntries": {
          "no-cache": "1"
        },
        "message": "Do not set execution_requirements['no-cache'] = '1'."
      },
      {
        "name": "max-shard-count",
        "ruleKinds": ["*_test"],
        "attr": "shard_count",
        "maxValue": 50,
        "allowlist": ["//huge_suite:..."],
        "message": "shard_count must not exceed 50; add the target to the allowlist for larger suites."
      }
    ]
  }
}
```

**Per-rule fields:**

| Field | Type | Meaning |
|---|---|---|
| `name` | string (required) | Stable identifier, included in the finding message. Must be unique. |
| `ruleKinds` | []string | Globs matched against `rule.Kind()`. Empty/absent ⇒ matches all kinds. |
| `attr` | string (required) | Attribute to inspect. |
| `forbidValues` | []string | Scalar attr must not equal any of these (see scalar matching below). |
| `requireValues` | []string | If attr present, must equal one of these. (If also want "must be present", see `required`.) |
| `forbidListItems` | []string | List attr must not contain any of these items. |
| `requireListItems` | []string | List attr must contain all of these items. |
| `allowListItems` | []string | Every list item must match an allowed value. Entries ending in `*` are prefix patterns; all others are exact matches. |
| `forbidDictEntries` | object (string→string) | Dict attr must not contain any of these key→value pairs. |
| `requireDictEntries` | object (string→string) | Dict attr must contain all of these key→value pairs. |
| `forbidDictKeys` | []string | Dict attr must not contain any of these keys (value ignored). |
| `minValue` | int | Numeric attr must be ≥ this (inclusive). At least one of `minValue` / `maxValue` required for the numeric family. |
| `maxValue` | int | Numeric attr must be ≤ this (inclusive). |
| `required` | bool | Attr must be present at all. |
| `allowlist` | []string | Target patterns exempt from this rule (see grammar below). |
| `suppressible` | bool (default `true`) | Whether `# buildifier: disable=attr-policy` silences this rule. `false` = a "hard" rule enforced authoritatively by CI regardless of in-file comments (see §6). |
| `message` | string | Custom message. If absent, a default is synthesized from the constraint. |

**Scalar matching (`forbidValues` / `requireValues`).** Covers string attributes
(`timeout = "eternal"`) and boolean/identifier literals (`local = True`,
`flaky = False`). At check time, read the attribute with `rule.AttrString` first,
then fall back to `rule.AttrLiteral` (which returns `True`/`False` for boolean
literals and identifier names for unquoted tokens). Compare the normalized string
form. An absent attribute does not match `forbidValues`; it only fails
`requireValues` when the attribute is present but wrong.

**Dict matching (`forbidDictEntries` / `requireDictEntries` / `forbidDictKeys`).**
Covers `string_dict` / `label_dict` attributes such as `execution_requirements`.
The attribute must be a `*build.DictExpr`; look up keys with `edit.DictionaryGet`.
Dict values are normalized the same way as scalars (string literal or
`AttrLiteral` on the value node). Example: `execution_requirements = {"no-cache":
"1"}` is flagged by the `no-no-cache` rule above because the dict contains key
`no-cache` with value `"1"`. `forbidDictKeys` is a weaker variant that flags the
presence of a key regardless of its value (useful when any non-default value is
undesirable but the exact value varies).

**Numeric matching (`minValue` / `maxValue`).** Covers integer attributes such as
`shard_count`. Read the attribute with `rule.AttrLiteral` (a `*build.LiteralExpr`
whose `Token` is the decimal representation) and parse with `strconv.Atoi`. An absent
attribute does not violate `maxValue` / `minValue` — only an explicitly set value
outside the range is flagged. This is the mechanism repos use to re-impose limits
Bazel no longer enforces globally (e.g. the former 50-shard cap removed in
[figma/bazel#12](https://github.com/figma/bazel/pull/12)).

Exactly one *constraint family* (`forbidValues`/`requireValues` **or**
`forbidListItems`/`requireListItems` **or**
`forbidDictEntries`/`requireDictEntries`/`forbidDictKeys` **or**
`minValue`/`maxValue`) should be set per rule; `required` is orthogonal. Validation
enforces this (see 3.5).

**Allow-list pattern grammar.** Entries are Bazel-style target patterns (a subset of
what users already know from `bazel build`):

| Pattern | Matches |
|---|---|
| `//pkg:name` or `//pkg` | exactly the target `//pkg:name` (`//pkg` ⇒ `//pkg:pkg`) |
| `//pkg:all` (also `//pkg:*`) | any target *directly* in package `pkg` |
| `//pkg/...` | any target in package `pkg` or any package beneath it (recursive) |
| `//...` | every target |

There is **no off-the-shelf matcher** for this in the repo — `labels.Equal` is
exact-only, and buildozer's `/...` handling is for filesystem BUILD discovery, not
in-memory label matching. So the warning ships a small matcher (`allowlistMatch`) that
parses each entry once at config-compile time and, for a given target
`//{f.Pkg}:{rule.Name()}`, tests: exact via `labels.Equal`; `:all`/`:*` via package
equality; `/...` via `pkg == P || strings.HasPrefix(pkg, P+"/")`.

**Semantics:**
- Empty/absent `attrPolicy` ⇒ warning is a no-op. Safe to enable in `--warnings=all`.
- A target matches a policy rule if its `Kind()` matches any `ruleKinds` glob **and**
  its label matches **no** entry in `allowlist`.
- Target label computed as `//{f.Pkg}:{rule.Name()}` and tested with `allowlistMatch`
  per the grammar above (not a bare `labels.Equal`, which can't express patterns).
- Finding is anchored on the offending node. Default: the attribute expression
  (`rule.Attr(attr)`). Exceptions for precise IDE highlighting:
  - **List items:** anchor on the matching string literal inside the list (e.g. the
    `"exclusive"` entry in `tags`), not the whole list.
  - **Dict entries:** anchor on the offending value node when possible.
  - **Missing required attr:** anchor on `rule.Call`.

### 3.3 Go types

New file `buildifier/config/attrpolicy.go` (or inline in `config.go`) — keep JSON tags:

```go
type AttrPolicy struct {
    Rules []AttrPolicyRule `json:"rules,omitempty"`
}

type AttrPolicyRule struct {
    Name               string            `json:"name"`
    RuleKinds          []string          `json:"ruleKinds,omitempty"`
    Attr               string            `json:"attr"`
    ForbidValues       []string          `json:"forbidValues,omitempty"`
    RequireValues      []string          `json:"requireValues,omitempty"`
    ForbidListItems    []string          `json:"forbidListItems,omitempty"`
    RequireListItems   []string          `json:"requireListItems,omitempty"`
    ForbidDictEntries  map[string]string `json:"forbidDictEntries,omitempty"`
    RequireDictEntries map[string]string `json:"requireDictEntries,omitempty"`
    ForbidDictKeys     []string          `json:"forbidDictKeys,omitempty"`
    MinValue           *int              `json:"minValue,omitempty"` // pointer: absent vs set-to-zero
    MaxValue           *int              `json:"maxValue,omitempty"`
    Required           bool              `json:"required,omitempty"`
    Allowlist          []string          `json:"allowlist,omitempty"`
    Suppressible       *bool             `json:"suppressible,omitempty"` // pointer so absent ⇒ default true
    Message            string            `json:"message,omitempty"`
}
```

Add to `Config` struct (`buildifier/config/config.go:93`):

```go
AttrPolicy *AttrPolicy `json:"attrPolicy,omitempty"`
```

### 3.4 Wiring config → warning (mirror the `tables` pattern)

The warning function has signature `func(f *build.File) []*LinterFinding` and cannot
take config directly (would churn 100+ warning signatures). Instead use a package-level
global in `warn`, set once during config application — exactly how `tables` works.

- In `warn/warn_attr_policy.go`, define:
  ```go
  // AttrPolicyConfig is process-global policy, set from buildifier config before linting.
  var AttrPolicyConfig []AttrPolicyRuleCompiled // compiled form: kind globs + allow-list patterns precompiled

  func SetAttrPolicy(rules []AttrPolicyRuleCompiled) { AttrPolicyConfig = rules }
  ```
  To avoid an import cycle (`warn` must not import `buildifier/config`), define the
  *compiled* policy type in the `warn` package and have the config layer translate
  `config.AttrPolicy` → `[]warn.AttrPolicyRuleCompiled` and call `warn.SetAttrPolicy`.
- Apply in `buildifier/config/config.go` `Validate()` (next to the tables block,
  ~line 231): if `c.AttrPolicy != nil`, compile and call `warn.SetAttrPolicy(...)`.
  Return a validation error on malformed rules.

> **Import-cycle note for agents:** confirm direction. `buildifier/config` may import
> `warn`; `warn` must not import `buildifier/config`. If `warn` already imports config
> anywhere, keep the compiled type in `warn` regardless. This is the one wiring risk —
> verify before writing code.

### 3.5 Validation (`buildifier/config/validation.go`)

On load, reject:
- duplicate `name`s,
- empty `name` or empty `attr`,
- a rule with no constraint set,
- a rule mixing scalar, list, dict, and numeric constraint families,
- a numeric rule with neither `minValue` nor `maxValue` set, or with `minValue` >
  `maxValue` when both are set,
- malformed globs in `ruleKinds` / malformed target patterns in `allowlist` (must
  match the §3.2 grammar; reject a bare `...`, repository-qualified entries, etc.).

Emit clear errors (`fmt.Errorf("attrPolicy rule %q: ...", name)`).

### 3.6 Warning implementation sketch (`warn/warn_attr_policy.go`)

```go
func attrPolicyWarning(f *build.File) []*LinterFinding {
    if f.Type != build.TypeBuild || len(AttrPolicyConfig) == 0 {
        return nil
    }
    var findings []*LinterFinding
    for _, rule := range f.Rules("") {
        kind := rule.Kind()
        label := labels.Label{Package: f.Pkg, Target: rule.Name()}.Format()
        for _, p := range AttrPolicyConfig {
            if !p.matchesKind(kind) || p.allowed(label, f.Pkg) {
                continue
            }
            if fnd := p.check(rule, f); fnd != nil {
                findings = append(findings, fnd)
            }
        }
    }
    return findings
}
```

- `p.allowed` runs the precompiled `allowlistMatch` (§3.2 grammar) over the target
  label — exact / `:all` / `/...` — not a bare `labels.Equal`.
- `p.check` handles the constraint families:
  - **Scalars:** `attrScalarString(rule, attr)` → `rule.AttrString`, else
    `rule.AttrLiteral`; compare against `forbidValues` / `requireValues`.
  - **Lists:** `rule.AttrStrings`; for `forbidListItems`, anchor on the matching
    list element node (not the whole list).
  - **Dicts:** `rule.Attr(attr)` as `*build.DictExpr`; `edit.DictionaryGet` per
    key; normalize values like scalars. `forbidDictKeys` flags any listed key
    that is present.
  - **Numerics:** `strconv.Atoi(rule.AttrLiteral(attr))`; flag if value `< minValue`
    or `> maxValue` (bounds inclusive; absent bound ignored).
  Returns `makeLinterFinding(node, msg)` anchored on the offending attr node (or a
  specific dict entry's value node when possible).
- Scope: BUILD files only (targets live there). Return `nil` for other file types.
- **Autofix: none initially.** We cannot infer a correct scalar value. The one
  mechanically-safe fix (remove a forbidden list item / forbidden tag) is a good
  fast-follow via `LinterReplacement`; leave a `// TODO(attr-policy): autofix remove-item`.

### 3.7 Registration, docs, tests

- Register in `warn/warn.go`: `FileWarningMap["attr-policy"] = attrPolicyWarning`.
- Decide default-on vs. opt-in: **default-on** (included in `DefaultWarnings`). When no
  `attrPolicy` config is present, the warning enforces Bazel's default `shard_count`
  constraint (`maxValue: 50` on `*_test` rules). Custom rules replace the default
  entirely once `attrPolicy.rules` is non-empty.
- Docs: add an entry to `WARNINGS.md` and `warn/docs/warnings.textproto` describing the
  warning **and** the `attrPolicy` config block (with the example rules, including
  boolean, dict, and numeric constraints).
- Add a `-config=example` sample entry so `buildifier -config=example` prints an
  `attrPolicy` stub (see `config.Example()`).
- Tests `warn/warn_attr_policy_test.go`:
  - Set `warn.SetAttrPolicy(...)` in the test, then use `checkFindings`/
    `checkFindingsAndFix` with `scopeBuild`.
  - Cases: forbidden scalar value flagged; allow-listed target exempt; recursive
    `//pkg/...` allow-list; forbidden list item in `tags`; `ruleKinds` glob match &
    non-match; missing-when-`required`; `# buildifier: disable=attr-policy` suppression;
    forbidden boolean literal (`local = True`); forbidden dict entry
    (`execution_requirements = {"no-cache": "1"}`); `forbidDictKeys` on a dict key;
    `shard_count` above `maxValue` flagged, within range OK, absent OK; allow-listed
    high-shard target exempt; empty config applies built-in defaults (`shard_count` ≤ 50,
    deprecated `licenses`, deprecated `output_licenses` on binary-producing rules).
  - Config tests in `buildifier/config/config_test.go`: parse the sample JSON;
    validation rejects malformed rules.

### 3.8 Acceptance criteria (Workstream A)

- `buildifier --lint=warn --warnings=attr-policy BUILD` flags eternal timeout on
  non-allow-listed test targets, `exclusive` tags on tests, and `shard_count > 50`,
  given the sample config.
- Zero findings when `attrPolicy` is absent.
- All new + existing `warn` and `config` tests pass; `WARNINGS.md` regeneration (if
  applicable) is consistent.

---

## 4. Workstream B — `testpolicy` tool (empirical tuning)

A **new binary** beside `buildifier`/`buildozer`. It never runs inside buildifier and
never blocks pre-commit.

### 4.1 Layout

```
testpolicy/
  main.go            // CLI: window, filters, report output
  source/            // warehouse adapter
    source.go        //   interface + data model
    bigquery.go      //   concrete impl (behind a build tag / flag)
  analyze/
    timeout.go       //   timeout recommender
    flaky.go         //   flakiness scoring + flaky recommender
    shard.go         //   (future) shard_count recommender — needs setup vs assertion timing
  emit/
    buildozer.go     //   emit buildozer command script (terminal deliverable)
  report/            //   human-readable + machine (JSON) report
```

### 4.2 Warehouse adapter interface (data-source-agnostic)

```go
// source/source.go
type TargetStats struct {
    Label            string
    Runs             int
    DurationP50      time.Duration
    DurationP95      time.Duration
    DurationMax      time.Duration
    DeclaredTimeout  string        // bucket keyword (short|moderate|long|eternal) as observed, if available; may be empty (then derived from `size`, see §4.3)
    TimeoutFailures  int           // failures attributed to hitting the timeout
    // Per-attempt outcomes for flakiness math:
    Attempts         int           // total attempts observed
    PassByAttempt    []float64     // PassByAttempt[k] = P(pass by attempt k+1), empirical
    // Per-shard timing (optional; needed for shard_count recommender):
    ShardCount       int           // declared shard_count, if any
    ShardDurationP95 []time.Duration // per-shard P95, when available
    SetupDurationP95 time.Duration // fixture/SUT setup time P95, when separable from assertions
}

type Source interface {
    // Query returns stats for targets matching the filter over [since, until].
    Query(ctx context.Context, since, until time.Time, filter Filter) ([]TargetStats, error)
}
```

Keeping this interface is what makes the "Metrics DB / warehouse" decision concrete
while letting the query backend be swapped/tested with a fake.

### 4.3 Timeout recommender (`analyze/timeout.go`)

**Bucket → seconds is a configured input, not a constant.** The `timeout` attribute is a
*bucket keyword* (`short`/`moderate`/`long`/`eternal`); the number of seconds each
bucket allows defaults to `60 / 300 / 900 / 3600` but a repo can override it with
`--test_timeout=short,moderate,long,eternal` (usually in `.bazelrc`). The recommender
cannot resolve a bucket to seconds — nor pick a bucket for an observed duration —
without the repo's actual mapping. So:

- The tool takes a **`timeoutBuckets` config** (`map[string]int` keyword→seconds),
  defaulting to Bazel's `60/300/900/3600`. Populate it either by parsing the repo's
  `.bazelrc` for `--test_timeout`, or via an explicit flag/config value. Surface the
  effective mapping in the report so recommendations are auditable.
- **Timeout may be implicit.** If a target sets no `timeout`, Bazel derives the bucket
  from `size` (`small→short`, `medium→moderate`, `large→long`, `enormous→eternal`), then
  resolves seconds via the same `timeoutBuckets` map. The recommender must apply this
  fallback when `DeclaredTimeout` is empty, and decide whether to write a `timeout` attr
  or adjust `size` (recommend setting `timeout` explicitly to avoid perturbing other
  `size`-driven behavior like resource reservations).
- Recommend the **smallest bucket** whose configured seconds ≥ `DurationP95 *
  safetyFactor` (default `safetyFactor = 1.5`, configurable).
- **Bump up** if either: observed runtime is within `X%` (default 20%) of the current
  bucket's configured seconds, **or** `TimeoutFailures > 0`.
- **Never recommend `eternal`** unless the target is on the eternal allow-list (shared
  with Workstream A's config so policy stays single-sourced). If data says a target
  needs > `long` and isn't allow-listed, emit a **report finding for a human**, not an
  auto-edit.
- Output per target: `{current, recommended, reason}`.

### 4.4 Flakiness scoring (`analyze/flaky.go`)

Answers "does a single retry likely pass, or does it need multiple?".

- Let `a = P(a single attempt passes | not chronically broken)`, estimated from history
  (prefer empirical `PassByAttempt` when present; else independence model).
- Under independence: `P(pass within N) = 1 - (1-a)^N`. To reach target pass-rate `T`:

  ```
  N ≥ ceil( log(1 - T) / log(1 - a) )
  ```

**Boundary cases** (handle before applying the formula):

| Condition | Handling |
|---|---|
| `a = 1` (never fails) | `N = 1`; recommend `flaky = 0` / leave unset |
| `a = 0` or `a` below a minimum threshold (e.g. 0.05) | Classify as chronically broken; do **not** recommend retries |
| `a` very close to 1 (e.g. `a ≥ 0.999`) | Treat as `a = 1` to avoid `log(0)` |

- `N` is used internally to distinguish "a single retry almost always recovers it"
  from "retries rarely help", which drives the `flaky` recommendation:
  | `N` | Meaning | Recommendation |
  |---|---|---|
  | ≤ 1 | not flaky | `flaky = 0` (or leave unset) |
  | 2–3 | retries reliably recover it | `flaky = N - 1` (extra retries beyond the first attempt) |
  | > 3 | retries rarely recover it | report for owner; more retries won't reliably help |
  | very low `a` | chronically broken | **do not** mask with retries; report for owner |

- **Semantics (with integer `flaky`):** When `--flaky_test_attempts=default`, Bazel runs
  `1 + flaky` total attempts for flaky targets (`flaky = 0` ⇒ one attempt). The tool
  writes the integer directly: `buildozer 'set flaky 2' //pkg:target` for three total
  attempts. This is implemented in [figma/bazel#13](https://github.com/figma/bazel/pull/13);
  upstream tracking in [bazelbuild/bazel#30108](https://github.com/bazelbuild/bazel/issues/30108).
  When CI sets `--flaky_test_attempts=N` (bare integer), that flag overrides the attribute
  for all targets — the tool should surface that in its report so owners know BUILD-file
  edits may have no effect.

### 4.5 Shard-count recommender (`analyze/shard.go`) — future

`shard_count` is the third execution-reflecting test attribute alongside `timeout` and
`flaky`: all three describe **how the test currently executes**, not how it was designed.

Tuning `shard_count` is feasible **only if** telemetry can separate fixture/SUT setup time
from assertion (test-case) time. Without that split, the tool cannot tell whether more
shards would shorten the critical path or just duplicate expensive setup across shards.

When per-shard data is available:

- Recommend raising `shard_count` when per-shard assertion time is high and setup is
  amortizable (e.g. many cases per shard, setup ≪ case time).
- Recommend lowering `shard_count` when shards are mostly idle or setup dominates.
- Respect `attr-policy` caps (e.g. `shard_count` ≤ 50) from Workstream A.

Output per target: `{current, recommended, reason}`; emit `buildozer 'set shard_count N'`.

### 4.6 Emit (`emit/`)

The tool's **terminal output** is a `buildozer` command script that would repair the
repo, printed to stdout/file, e.g.:
```
buildozer 'set timeout "short"' //pkg:target
buildozer 'set flaky 2'         //pkg:other
buildozer 'set shard_count 4'   //pkg:sharded
```
- Also emit a machine-readable JSON report (recommendations + reasons + effective
  `timeoutBuckets`) for auditability.
- The tool **does not run buildozer, commit, or open PRs.** Executing the script,
  batching edits, opening PRs, and routing to reviewers are downstream responsibilities
  of whatever CI/automation calls `testpolicy`. Keeping the boundary here makes the tool
  trivially testable (assert on emitted commands) and reusable by any apply/review flow.
- **Skip stale targets:** warehouse rows may refer to deleted or renamed targets.
  Before emitting a `buildozer` line, verify the label still exists in the workspace
  (e.g. index targets while walking BUILD files, or a fast `bazel query` check). Report
  skipped labels in the JSON output instead of emitting commands that would fail.

### 4.7 Safety / guardrails

- Require a minimum `Runs` sample size before recommending (default e.g. 20); otherwise
  report "insufficient data".
- **Lowering timeouts is a normal, first-class recommendation** — not gated behind a
  flag. Bazel's default `size` is `medium` ⇒ `moderate` (300s), so many genuinely fast
  tests sit at `moderate` unnecessarily; dropping them to `short` speeds failure
  detection and scheduling. Safety comes from the recommender itself, not from
  suppressing downgrades: the `safetyFactor` headroom and the bump-up-on-`TimeoutFailures`
  / near-limit rules (§4.3) already keep a downgrade from cutting it too close, and the
  minimum-sample-size guard avoids acting on noise.
- Respect the eternal allow-list from `.buildifier.json` so the two systems agree.
- Dry-run is the default mode.
- Do not emit `buildozer` commands for labels absent from the workspace (see §4.6).

### 4.8 Acceptance criteria (Workstream B)

- With a fake `Source`, `testpolicy` produces correct `timeout`, `flaky`, and (when
  fixture data includes per-shard timing) `shard_count` recommendations and a valid
  buildozer command script for a fixture dataset.
- No warehouse credentials required for tests (fake source).
- Output is deterministic (stable ordering) so the emitted script can be asserted on.

---

## 5. Interaction between the two workstreams

- The **eternal-timeout allow-list lives once** in `.buildifier.json` (`attrPolicy`).
  Both `buildifier` (enforce) and `testpolicy` (never recommend eternal off-list) read
  it. Consider a tiny shared loader package so the schema isn't duplicated.
- `buildifier` enforces the *invariants*; `testpolicy` proposes the *values*. A
  `testpolicy`-emitted edit must itself satisfy `attr-policy` — i.e. the tool won't emit
  a buildozer command that buildifier would then reject.

---

## 6. Suppression & enforcement

Every buildifier warning is silenceable with `# buildifier: disable=<category>` (also
`# buildozer: disable=`). Suppression is centralized: `runWarningsFunction` drops any
finding for which `DisabledWarning(f, line, category)` is true (`warn/warn.go:281`),
keyed only on the category string. There is **no per-warning "cannot be suppressed"
flag** today — so out of the box a user can write:

```python
# buildifier: disable=attr-policy
my_test(name = "x", timeout = "eternal")
```

…and the policy evaporates locally. We handle this deliberately rather than fighting it.

**Reframe:** a `disable=` comment is not a silent back door — it lives in the BUILD
file, shows up in the diff, in `git blame`, and in code review. The governance question
is not "can it be bypassed?" (it can) but "is the bypass visible and attributable?"
(yes). That points to a two-tier model.

### 6.1 Two-tier enforcement

1. **buildifier `attr-policy` — fast, local, suppressible.** Normal buildifier UX:
   dev-time feedback, silenceable with a visible comment. Advisory; not a gate.
2. **Authoritative CI gate — non-bypassable.** A CI job (naturally hosted in the
   `testpolicy` tool, which already loads the policy config) re-evaluates the *same*
   `.buildifier.json` policy against the tree and, for rules with `suppressible: false`,
   **ignores `disable=` comments entirely**. In-file suppression can't defeat it because
   the gate doesn't consult the file's comments for hard rules. This is where
   "eternal timeout requires the allow-list, full stop" actually lives.

Result: the **sanctioned** escape hatch is the allow-list (a reviewed edit to
`.buildifier.json`, ideally CODEOWNER-guarded); the **unsanctioned** one (a `disable=`
comment on a hard rule) is ignored by the gate and surfaced as debt.

### 6.2 `suppressible` config field

Per-rule `suppressible` (§3.2, default `true`) tunes this:
- `true` — soft rule; local `disable=attr-policy` silences it (advisory policy).
- `false` — hard rule; the CI gate enforces it regardless of comments. buildifier
  locally still shows (and, unless we take the core change in §6.4, still lets users
  *locally* suppress) it, but the merge gate is authoritative.

### 6.3 Suppression audit

A cheap job/report that inventories every `disable=attr-policy` (and, if we add
sub-scoping, `attr-policy=<rule>`) comment across the repo, so escape hatches are
**visible and burn-down-able** instead of accumulating silently. Pairs well with
CODEOWNERS on `.buildifier.json` and on BUILD files so both allow-list edits and
suppressions land on a policy owner.

### 6.4 Optional core change (open question)

If we want the *local* linter — not just CI — to refuse suppression for hard rules,
that's a small but real change to buildifier's contract: add a `NonSuppressible bool`
to `LinterFinding` and have `runWarningsFunction` skip the `DisabledWarning` check when
it's set; `attr-policy` sets it per-finding from the rule's `suppressible` flag. Clean
and gives per-rule granularity, **but** it breaks the long-standing invariant that every
warning is silenceable, so it needs buy-in before upstreaming. Recommendation: keep the
CI gate as the real enforcement regardless, and treat this as a nice-to-have. See §8.

---

## 7. Phasing & work breakdown (for agent hand-off)

Each task is independently ownable; dependencies noted. "AC" = acceptance criteria above.

### Phase 1 — buildifier `attr-policy` (Workstream A)
- **A1. Config types + parsing** — add `AttrPolicy`/`AttrPolicyRule` to
  `buildifier/config`, JSON round-trip test. *(no deps)*
- **A2. Validation** — `buildifier/config/validation.go` rules from §3.5. *(dep: A1)*
- **A3. Warning + compiled policy type + global** — `warn/warn_attr_policy.go`,
  glob & allow-list matching, register in `warn/warn.go`. *(no deps; can stub config)*
- **A4. Config→warn wiring** — compile in `Validate()`, resolve import-cycle question
  (§3.4 note). *(dep: A1, A3)*
- **A5. Tests** — `warn/warn_attr_policy_test.go` cases from §3.7. *(dep: A3/A4)*
- **A6. Docs** — `WARNINGS.md`, `warn/docs/warnings.textproto`, `-config=example`
  stub. *(dep: A1, A3)*

### Phase 2 — `testpolicy` skeleton + timeout recommender (Workstream B)
- **B1. Tool scaffold + CLI + fake source** — `testpolicy/main.go`, `source` interface,
  in-memory fake. *(no deps)*
- **B2. Timeout recommender** — §4.3 + tests on fixtures. *(dep: B1)*
- **B3. buildozer emitter + report** — §4.6 dry-run path. *(dep: B1)*

### Phase 3 — flakiness + real data source
- **B4. Flakiness scoring** — §4.4 + tests. *(dep: B1)*
- **B5. Warehouse (BigQuery/DB) source impl** — behind flag. *(dep: B1)*

### Phase 4 — shard_count tuning (future)
- **B6. Shard-count recommender** — §4.5; requires per-shard telemetry with setup vs
  assertion split. *(dep: B1, B5)*

### Phase 1b — enforcement (Workstream A, in parallel with Phase 2)
- **A7. Authoritative CI gate** — evaluate policy ignoring `disable=` for
  `suppressible: false` rules (§6.1/§6.2); non-zero exit on violation. *(dep: A1–A4)*
- **A8. Suppression audit** — inventory `disable=attr-policy` comments across the
  repo as a report (§6.3). *(dep: A3)*

---

## 8. Open questions

1. **Import direction** between `warn` and `buildifier/config` — confirm no cycle
   (§3.4). If one exists, the compiled-policy-type-in-`warn` approach resolves it.
2. ~~Should `attr-policy` be **default-on** (config-gated no-op) or opt-in via
   `--warnings`?~~ **Resolved:** default-on; with no `attrPolicy` config, enforces
   Bazel's `shard_count` ≤ 50 constraint on test rules.
3. **`NonSuppressible` core change (§6.4)** — do we extend buildifier so hard rules
   can't be silenced by `disable=` locally, accepting a break to the "every warning is
   suppressible" contract? Or rely solely on the CI gate? (Leaning CI-gate-only for the
   first cut.)
4. Warehouse **schema/columns** available for `TargetStats` — especially whether
   per-attempt outcomes (`PassByAttempt`) exist, or only aggregate pass/fail. This
   changes flakiness estimation fidelity.
5. **`timeoutBuckets` sourcing (§4.3)** — parse `.bazelrc` for `--test_timeout`, or
   require it as explicit tool config? Note `--test_timeout` can differ per bazelrc
   `--config`/platform, so "the" mapping may be ambiguous; do we pin one config, or
   analyze per-config?
6. Timeout **safety factor** and bucket cutoffs — defaults proposed; confirm with SRE/CI.
7. Do we want the shared eternal-allow-list loader as its own small package now, or
   duplicate-read for Phase 1 and refactor later?
