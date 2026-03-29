# MAMA AI Canonical Standard
Status: Canonical Draft
Version: 0.1.0
Owner: Mindburn Labs / HELM Core
Applies To: HELM commercial core, Studio, Desktop, Mission Control, governed agent runtime
Last Updated: 2026-03-27

## 0. Purpose

This document defines the canonical standard for MAMA AI inside HELM.

MAMA AI is the governed cognitive runtime and mission control surface for HELM.
It is not a separate truth plane.
It is not a second kernel.
It is not a generic chat assistant.
It is not a loose multi-agent wrapper.

MAMA AI exists to:
- translate operator intent into governed execution
- manage mission state, modes, agents, skills, memory, and evidence
- orchestrate bounded probabilistic intelligence above the deterministic HELM truth plane
- provide a single coherent control surface for human operators and autonomous execution lanes

This standard is normative.
All MAMA AI implementation work must conform to it.

---

## 1. Scope

This standard governs:

- MAMA command surface
- MAMA runtime modes
- subagent model
- skill system
- typed state model
- memory and retrieval model
- governance and proof interaction
- evaluation and benchmark lanes
- UI and control surface expectations
- canonical namespace and anti-orphan rules

This standard does not define:
- kernel internals
- specific model vendor integrations
- specific benchmark connector implementations
- product pricing or GTM copy

Those belong to separate standards.

---

## 2. Canonical positioning inside HELM

## 2.1 What MAMA AI is

MAMA AI is the executive control layer for HELM.
It is the operator-facing and runtime-facing system that:
- holds mission context
- selects modes
- manages agents
- runs skills
- requests governed effects
- explains why actions happened
- binds memory to execution
- binds evidence to decisions

## 2.2 What MAMA AI is not

MAMA AI is not:
- execution authority
- policy authority
- receipt authority
- proof authority
- final source of truth

Those remain in the HELM kernel and proof layers.

## 2.3 Rule of authority

The authority order is:

1. Kernel and policy
2. Guardian / PEP / CPI
3. ProofGraph / receipts / EvidencePack
4. MAMA mission runtime
5. UI surfaces and conversational interface

MAMA may propose, organize, compress, branch, and explain.
MAMA may not bypass or redefine execution truth.

---

## 3. Design doctrine

MAMA AI must be:

- governed
- mode-aware
- replay-native
- proof-aware
- typed-state-first
- compact on the surface and deep under the hood
- lane-aware
- composable with HELM connectors
- future-proof against stronger models

MAMA AI must not be:

- prompt spaghetti
- transcript-only memory
- command-count theater
- agent-swarm chaos
- UI-first without runtime clarity
- benchmark-specific special casing in core semantics

---

## 4. Objectives

The canonical objectives of MAMA AI are:

1. Give HELM a single operator-grade control layer for governed autonomy.
2. Turn multi-agent execution into a typed, inspectable, policy-bounded system.
3. Make memory and retrieval precise enough for long-lived agent workflows.
4. Make replay, branch, rewind, and proof first-class runtime primitives.
5. Keep the command surface small while the internal runtime remains powerful.
6. Preserve kernel truth and prevent dual-truth drift.
7. Support both normal product workflows and research / benchmark lanes without architectural corruption.

---

## 5. Non-goals

MAMA AI is not trying to be:

- a copy of Claude Code
- a copy of Cursor
- a benchmark-specific ARC agent
- a new knowledge graph product
- a new IDE
- a local model runner
- a replacement for OrgGenome, OrgPhenotype, ProofGraph, or the Guardian

MAMA AI may integrate with all of these layers and tools.
It may not replace them.

---

## 6. Canonical architecture

## 6.1 Plane mapping

MAMA spans multiple HELM planes, but does not own them.

### Plane 1 - Identity and Trust
MAMA acts under authenticated principals and mission scopes.

### Plane 2 - OrgGenome
MAMA reads mission templates, role bindings, tool permissions, ceilings, and lane definitions from OrgGenome-derived policy.

### Plane 3 - Deterministic Kernel
MAMA never executes directly.
All effects route through the kernel.

### Plane 4 - ProofGraph
MAMA consumes and presents proof.
It does not author authoritative receipts on its own.

### Plane 5 - Tools and Connectors
MAMA invokes governed tool surfaces through connector contracts.

### Plane 6 - Knowledge Plane
MAMA reads from LKS and CKS, writes to provisional memory stores, and promotes only through governance.

### Plane 7 - Surfaces
MAMA is the primary mission control and operator experience layer.

## 6.2 Canonical runtime stack

MAMA runtime is composed of:

- Mission state
- Mode machine
- Agent roster
- Skill registry
- Typed memory state
- Retrieval router
- Explanation layer
- Branch / rewind layer
- Lane policy overlay
- Proof integration layer

---

## 7. Canonical terminology

### Mission
A bounded objective with explicit success criteria, budgets, permissions, and execution context.

### Lane
A policy-constrained execution track.
Examples: product, background, research, blind-eval.

### Mode
A local runtime state that constrains behavior and transitions.
Examples: observe, plan, commit, replay.

### Agent
A specialized execution role under MAMA control.

### Skill
A reusable, governed, typed playbook or capability unit.

### Replay
A reconstructable runtime trajectory with evidence and branch points.

### Branch
A deliberate divergence in conversation state, plan state, or execution state.

### Compact
A reduction operation on live context that preserves typed state and critical evidence pointers.

### Proof
The operator-facing representation of receipts, verdicts, artifacts, and causal trace.

---

## 8. Command surface

The command surface must stay compact.
The power belongs underneath.

## 8.1 Canonical commands

### `/mission`
Shows current mission, success criteria, budgets, lane, and status.

### `/memory`
Inspects and manages memory layers, user facts, episodic memories, session summaries, and source provenance.

### `/mode`
Changes or inspects the current mode.

### `/agents`
Shows agent roster, current tasks, health, permissions, and budgets.

### `/skill`
Searches, runs, inspects, freezes, promotes, rolls back, or debugs skills.

### `/env`
Opens, inspects, or changes external execution environments or governed connectors.

### `/episode`
Creates, runs, inspects, or closes bounded execution episodes.

### `/replay`
Opens replay traces, checkpoints, failures, and divergence points.

### `/bench`
Runs or inspects benchmark suites and experimental evaluation lanes.

### `/loop`
Runs recurring checks, polling, or controlled iterative workflows.

### `/promote`
Promotes a skill, config, memory artifact, or lane-ready bundle through governance.

### `/rewind`
Rewinds runtime state to a prior checkpoint or summary boundary.

### `/branch`
Creates a new branch of mission or execution state.

### `/compact`
Compacts live context while preserving typed state and critical evidence links.

### `/permissions`
Displays and manages permission state as allowed by policy.

### `/proof`
Shows the proof chain, receipts, verdicts, and evidence artifacts for the current mission or episode.

## 8.2 Optional commands

These may exist if they are implemented cleanly and provide real leverage:

- `/model`
- `/effort`
- `/context`
- `/search`
- `/timeline`

They are optional.
They must not bloat the canonical control surface.

## 8.3 Forbidden command drift

Do not create:
- dozens of feature-fragment commands
- product-specific commands without reusable semantics
- duplicate command namespaces for the same underlying action
- benchmark-only command hierarchies if the core control plane can already express the action

---

## 9. Runtime modes

MAMA is mode-driven.
Modes are not cosmetic.
They constrain execution.

## 9.1 Canonical modes

### `observe`
Read-only inspection mode.
No mutations.
Used for state acquisition, memory hydration, and grounding.

### `explore`
Safe exploratory mode.
Used to gather information, discover affordances, and generate hypotheses.

### `plan`
Synthesis mode.
Used to generate plans, branch options, task breakdowns, and candidate strategies.

### `probe`
Controlled low-cost probing mode.
Used for bounded information-gathering actions.

### `commit`
Primary execution mode.
Only approved actions are allowed.

### `replay`
Postmortem and trajectory-inspection mode.

### `distill`
Compression and learning mode.
Used to create summaries, candidate skills, reusable tactics, and structured insights.

### `blind-eval`
Frozen overlay mode.
No hidden adaptation, no silent mutation of active stack.

## 9.2 Mode transition rules

Allowed transitions:

- `observe -> explore`
- `observe -> plan`
- `explore -> plan`
- `plan -> probe`
- `probe -> plan`
- `plan -> commit`
- `commit -> observe`
- `commit -> replay`
- `replay -> distill`
- `distill -> observe`

`blind-eval` is an overlay and may wrap `observe`, `plan`, and `commit`, but it freezes mutable runtime components according to lane policy.

## 9.3 Mode invariants

- `observe` may not mutate
- `plan` may not silently commit
- `probe` must remain bounded
- `commit` must be fully governed
- `replay` may not alter history
- `distill` may not gain execution authority automatically
- `blind-eval` may not mutate the active stack or allow silent contamination

---

## 10. Agent model

MAMA uses role-based agents.
Not arbitrary swarms.

## 10.1 Canonical agent roles

### `Explore`
Searches for context, affordances, missing data, and uncertainty reduction.

### `WorldModel`
Maintains structured interpretation of state, entities, relations, and latent variables.

### `Planner`
Builds and evaluates candidate plans.

### `Executor`
Translates approved decisions into governed effect requests.

### `Critic`
Searches for contradictions, weak plans, unsafe assumptions, regressions, or dead ends.

### `ReplayAnalyst`
Inspects past episodes, failures, successes, and divergence patterns.

### `SkillSynth`
Produces candidate skills or structured procedures from distilled evidence.

### `Governor`
Applies runtime policy overlays, budget checks, lane guards, and promotion gating.

## 10.2 Agent rules

Every agent must have:
- name
- role
- description
- allowed tools
- allowed modes
- effort profile
- budget limits
- evidence scope
- human visibility rules

Agents may be forked, but their forks must inherit type, policy, and budget constraints.

## 10.3 Multi-agent rule

MAMA may run multiple agents in parallel only when:
- the decomposition is explicit
- outputs are mergeable
- budgets permit it
- governance remains inspectable

Parallelism is a tool, not a default.

---

## 11. Skill system

Skills are core to MAMA.
They are not loose prompt files.

## 11.1 Definition

A skill is a governed, typed, reusable capability unit with:
- human-readable description
- execution instructions
- invocation policy
- tool policy
- lane policy
- promotion status
- provenance

## 11.2 Skill categories

### Reference skill
Provides reusable domain conventions or knowledge.

### Task skill
Runs a repeatable procedure or workflow.

### Analysis skill
Performs structured inspection, audit, or explanation.

### Forked skill
Runs in a subagent or isolated execution context.

## 11.3 Canonical skill frontmatter

```yaml
name: example-skill
description: Brief description of what this skill does
invocation: manual|auto
runtime: inline|fork
allowed_modes:
  - observe
  - plan
allowed_tools:
  - search.instant
  - memory.search
model_profile: planner-high
effort: high
lane_policy:
  allow:
    - product
    - research
  deny:
    - blind-eval
promotion_gate: default-skill-promotion
evidence_required: true
max_actions_per_invoke: 8
```

## 11.4 Skill lifecycle

* `draft`
* `candidate`
* `frozen`
* `promoted`
* `deprecated`
* `revoked`

No skill may gain privileged execution authority merely by existing.

## 11.5 Skill rules

Skills must:

* be versioned
* be attributable
* declare allowed modes
* declare allowed tools
* declare lane restrictions
* support rollback
* produce inspectable outputs

Skills must not:

* bypass the kernel
* silently mutate blind-eval lanes
* carry hidden policy outside registry

---

## 12. Typed state model

MAMA state must be typed.
Transcript-only state is non-canonical.

## 12.1 Required runtime states

### `MissionState`

* mission_id
* objective
* success_criteria
* lane
* policy_profile
* created_at
* status

### `ModeState`

* current_mode
* previous_mode
* entered_at
* reason
* allowed_transitions

### `AgentState`

* agent_id
* role
* health
* current_task
* active_skill
* budget_remaining
* status

### `SkillState`

* skill_id
* version
* lifecycle_status
* last_used_at
* promotion_status

### `MemoryState`

* memory_refs
* active_profile_refs
* session_summary_refs
* unresolved_conflicts
* temporal_index_refs

### `EpisodeState`

* episode_id
* environment_id
* started_at
* step_index
* status
* replay_ref

### `ReplayState`

* replay_id
* checkpoint_refs
* branch_points
* artifact_refs

### `ProofState`

* active_receipt_refs
* last_verdict_ref
* evidence_pack_refs
* proof_summary

## 12.2 State storage rule

Typed state must live in canonical runtime stores.
The chat transcript may reference state.
It may not be the only state store.

---

## 13. Memory and retrieval model

MAMA must have strong memory semantics.
Not just context stuffing.

## 13.1 Memory layers

### Working memory

Short-lived, high-relevance mission context.

### Episodic memory

Session-specific facts, actions, outcomes, and summaries.

### Profile memory

Persistent user, project, org, or system facts.

### Temporal memory

Events, dates, updates, and ordering.

### Provenance memory

Source references, proofs, and artifact links.

## 13.2 Memory rules

Memory must:

* preserve provenance
* support temporal versioning
* support updates and contradictions
* keep raw source chunk references
* distinguish provisional from promoted memory

## 13.3 Retrieval rules

Retrieval must:

* classify the query
* select the proper retriever
* prefer atomic memory over noisy raw chunks when possible
* re-inject source chunks when detail is needed
* expose why a result was selected

## 13.4 Retrieval classes

Canonical retrieval classes:

* exact_fact
* preference
* temporal
* multi_session
* knowledge_update
* policy_lookup
* artifact_lookup
* code_literal
* code_regex
* broad_context

## 13.5 Retrieval output

Every retrieval operation should return a structured bundle containing:

* query_class
* selected memories
* selected chunks
* selected files or artifacts
* explanation
* evidence refs

---

## 14. Governance and proof integration

## 14.1 Core rule

MAMA never executes directly.

MAMA creates or routes:

* requests
* candidate plans
* effect proposals
* explanation bundles

Execution authority remains external to MAMA in the kernel.

## 14.2 Proof interaction

MAMA must:

* surface proof chain
* attach proof refs to missions, episodes, and skills
* explain which receipts and verdicts matter
* make replay and evidence discoverable

MAMA must not:

* invent receipts
* override proof
* hide conflicting verdicts

## 14.3 Replay and branch semantics

Replay and branch are first-class runtime primitives.

### Replay must support:

* timeline traversal
* checkpoint inspection
* artifact linking
* failure explanation
* diff across branches

### Branch must support:

* mission branches
* plan branches
* execution branches
* replay branches

Rewind may restore:

* conversation state
* typed mission state
* branch-local runtime state

Rewind may not rewrite authoritative proof history.

---

## 15. Lanes

Lanes define constrained execution tracks.

## 15.1 Canonical lanes

### `product`

Normal governed product workflow.

### `background`

Long-running async-like local workflows within bounded runtime.

### `research`

Exploratory lane with higher flexibility, lower authority.

### `blind-eval`

Frozen, contamination-sensitive lane.

## 15.2 Lane rules

Every lane must define:

* allowed modes
* allowed skills
* allowed agents
* allowed connectors
* budget ceilings
* mutation permissions
* promotion permissions

## 15.3 Blind-eval rule

In `blind-eval`:

* no skill mutation
* no prompt mutation
* no stack mutation
* no policy mutation
* no hidden adaptation
* no silent memory promotion into execution authority

---

## 16. UI and surface requirements

MAMA should feel simple on top and rigorous underneath.

## 16.1 Required surface regions

### Mission bar

Mission identity, lane, mode, success state, budget status.

### Main conversation pane

Operator interaction, mission updates, structured explanations.

### Agent roster pane

Agents, roles, tasks, health, budgets.

### Memory pane

Profiles, episodic memory, retrieval bundles, timelines.

### Replay pane

Episodes, checkpoints, branch points, artifacts.

### Proof pane

Receipts, verdicts, EvidencePacks, causal chain.

### Skill pane

Skill discovery, active skills, promotion state, rollback.

## 16.2 UX principles

The UI must optimize for:

* legibility
* speed
* trust
* inspectability
* compactness
* progressive disclosure

The UI must avoid:

* ornamental complexity
* duplicated flows
* hidden state transitions
* proof-blind action surfaces

---

## 17. Namespaces and file ownership

## 17.1 Canonical namespace rule

MAMA-related code must live under one coherent namespace strategy.
Do not create parallel partial roots such as:

* `mama_runtime`
* `mama_ai_core`
* `agent_memory_v2`
* `mission_chat_engine`
* `bench_mama`

One canonical root per concern.
No duplicates.

## 17.2 Ownership rule

Each concern has one owner root:

* commands
* modes
* agents
* skills
* memory
* retrieval
* replay
* proof surfaces

No overlapping ownership.

---

## 18. Anti-orphan rules

These are hard rules.

## 18.1 Forbidden orphaned structures

Do not create:

* a second memory engine outside canonical memory namespace
* a separate benchmark-only mission runtime
* a prompt-only skill zoo outside skill registry
* duplicate branch / rewind systems
* UI-only state that becomes authoritative
* hidden lane mutation logic
* separate proof sidecars that do not flow through existing proof surfaces

## 18.2 Required merges

The following must remain unified:

* mission control state
* mode machine
* agent registry
* skill registry
* retrieval router
* proof surface bindings
* replay bindings

---

## 19. Conformance criteria

MAMA AI is conformant only if:

1. It does not bypass kernel truth.
2. It has a compact canonical command surface.
3. It uses typed runtime state.
4. It has role-based agents with budgets and tool policies.
5. It has governed, versioned skills.
6. It supports branch, rewind, replay, and proof as first-class constructs.
7. It supports lane isolation.
8. It uses structured memory and retrieval.
9. It introduces no orphaned runtime, memory, or proof structures.
10. It keeps UI and runtime in canonical sync.

---

## 20. Definition of done

MAMA AI is not done when it can “chat”.
It is done when:

* mission control is canonical
* modes are enforced
* agents are typed and governable
* skills are reusable and promotable
* memory is structured
* retrieval is precise
* proof is visible
* replay is inspectable
* branches are controllable
* lanes are enforceable
* no dual truth exists

---

## 21. Implementation guidance

Implementation should proceed in this order:

1. mode machine
2. typed runtime state
3. command registry
4. agent roster and role system
5. skill registry and lifecycle
6. memory and retrieval substrate
7. replay and branch system
8. proof surface integration
9. UI surface integration
10. lane hardening and conformance

Do not invert this order.
Do not start from UI polish.
Do not start from giant agent swarms.
Do not start from benchmark-specific hacks.

---

## 22. Final canonical statement

MAMA AI is the governed cognitive runtime and mission control layer of HELM.

It organizes intelligence.
It does not replace truth.

It makes agents operable.
It does not outrank the kernel.

It makes memory useful.
It does not create a second source of truth.

It makes autonomy inspectable.
It does not permit silent execution authority outside HELM’s deterministic core.

---
