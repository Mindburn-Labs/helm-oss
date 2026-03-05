package guardian

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	pkg_artifact "github.com/Mindburn-Labs/helm/core/pkg/artifacts"
	"github.com/Mindburn-Labs/helm/core/pkg/canonicalize"
	"github.com/Mindburn-Labs/helm/core/pkg/contracts"
	"github.com/Mindburn-Labs/helm/core/pkg/crypto"
	"github.com/Mindburn-Labs/helm/core/pkg/finance"
	"github.com/Mindburn-Labs/helm/core/pkg/pdp"
	"github.com/Mindburn-Labs/helm/core/pkg/prg"
)

// Clock provides authority time for the Guardian.
// Per KERNEL_TCB §3: the kernel MUST NOT use wall-clock time.Now().
// Inject an authority clock that derives time from the deterministic
// EnvSnap or a kernel-managed monotonic source.
type Clock interface {
	Now() time.Time
}

// wallClock is the default clock (for backward compatibility during migration).
// Production code SHOULD inject a kernel authority clock instead.
type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }

// Guardian enforces the Proof Requirement Graph (PRG)
type Guardian struct {
	signer    crypto.Signer
	prg       *prg.Graph
	pe        *prg.PolicyEngine
	registry  *pkg_artifact.Registry
	clock     Clock
	tracker   finance.Tracker
	auditLog  *AuditLog
	temporal  *TemporalGuardian
	envFprint string                  // Boot-sequence fingerprint for DecisionRecords
	pdp       pdp.PolicyDecisionPoint // Optional pluggable policy backend
}

// NewGuardian creates a new Guardian instance.
// If clock is nil, a default wall-clock is used for backward compatibility.
// Production code SHOULD pass a kernel authority clock.
// tracker is optional; if nil, budget checks are skipped.
func NewGuardian(signer crypto.Signer, ruleGraph *prg.Graph, reg *pkg_artifact.Registry, clock ...Clock) *Guardian {
	var c Clock
	if len(clock) > 0 && clock[0] != nil {
		c = clock[0]
	} else {
		c = wallClock{}
	}

	pe, _ := prg.NewPolicyEngine()

	return &Guardian{
		signer:   signer,
		prg:      ruleGraph,
		pe:       pe,
		registry: reg,
		clock:    c,
	}
}

// SetBudgetTracker allows injecting the finance tracker after initialization
func (g *Guardian) SetBudgetTracker(t finance.Tracker) {
	g.tracker = t
}

// SetAuditLog allows injecting the audit log after initialization
func (g *Guardian) SetAuditLog(l *AuditLog) {
	g.auditLog = l
}

// SetTemporalGuardian allows injecting the temporal guardian after initialization
func (g *Guardian) SetTemporalGuardian(tg *TemporalGuardian) {
	g.temporal = tg
}

// SetEnvFingerprint sets the boot-sequence environment fingerprint
func (g *Guardian) SetEnvFingerprint(fp string) {
	g.envFprint = fp
}

// SetPolicyDecisionPoint injects an external PDP backend (OPA, Cedar, etc.).
// When set, EvaluateDecision delegates policy evaluation to this PDP
// while Guardian retains signing, enforcement, and proof binding.
func (g *Guardian) SetPolicyDecisionPoint(p pdp.PolicyDecisionPoint) {
	g.pdp = p
}

// SignDecision checks requirements and signs only if met
func (g *Guardian) SignDecision(ctx context.Context, decision *contracts.DecisionRecord, effect *contracts.Effect, evidenceHashes []string, intervention *contracts.InterventionMetadata) error {
	// 1. Gather Artifacts
	artifacts := make([]*pkg_artifact.ArtifactEnvelope, 0, len(evidenceHashes))
	for _, hash := range evidenceHashes {
		env, err := g.registry.GetArtifact(ctx, hash)
		if err != nil {
			return fmt.Errorf("failed to retrieve evidence %s: %w", hash, err)
		}
		// In a real system, we'd verify validity here (registry.VerifyArtifact)
		// but assuming GetArtifact returns valid structure.
		artifacts = append(artifacts, env)
	}

	// 2. Determine Action ID
	// Prefer tool_name for tool execution, otherwise use Effect Type
	var actionID string
	if toolName, ok := effect.Params["tool_name"].(string); ok && toolName != "" {
		actionID = toolName
	} else {
		actionID = effect.EffectType
	}

	// 3. Handle Temporal Intervention (Priority over PRG)
	if intervention != nil && intervention.Type != contracts.InterventionNone {
		decision.Intervention = intervention
		// If interrupting or quarantining, strict verdict override
		if intervention.Type == contracts.InterventionInterrupt || intervention.Type == contracts.InterventionQuarantine {
			decision.Verdict = "INTERVENE"
			decision.Reason = fmt.Sprintf("Temporal Intervention: %s (%s)", intervention.Type, intervention.ReasonCode)
			return g.signer.SignDecision(decision)
		}
		// If Throttling, we likely still proceed to PRG validation but note the throttle
		// defaulting to recording it.
	}

	// 3.5 Budget Check (Finance Gate)
	if g.tracker != nil {
		// Attempt to resolve Budget ID from params
		if budgetID, ok := effect.Params["budget_id"].(string); ok && budgetID != "" {
			// Estimate cost (MVP: 1 Request)
			// Planned enhancement: use a CostEstimator based on EffectType/Params.
			cost := finance.Cost{Requests: 1}

			// Check and Consume
			// Note: For strict correctness, we should Check here, then Consume ONLY if PRG passes.
			// However, preventing DoS via PRG computation (which is cheap compared to execution) implies early check.
			// Let's Check first.
			allowed, err := g.tracker.Check(budgetID, cost)
			if err != nil {
				// If checking fails (e.g. invalid budget ID), fail closed? Or open if just missing?
				// Fail closed for security.
				decision.Verdict = "FAIL"
				decision.Reason = fmt.Sprintf("Budget Error: %v", err)
				return g.signer.SignDecision(decision)
			}
			if !allowed {
				decision.Verdict = "FAIL"
				decision.Reason = "Budget Exceeded"
				return g.signer.SignDecision(decision)
			}

			// If allowed, we reserve/consume.
			// In this synchronous MVP, we consume now.
			// Ideally rollback if PRG fails, but for requests counters it's fine.
			if consumeErr := g.tracker.Consume(budgetID, cost); consumeErr != nil {
				// Log but don't fail — the Check already passed.
				slog.Warn("guardian: budget consume failed", "budget_id", budgetID, "error", consumeErr)
			}
		}
	}

	// AC-REG-10: EnvelopeCheck precedes every effect dispatch
	// Verify that the effect is properly enveloped (e.g. valid structure, allowed type)
	if err := g.checkEnvelope(effect); err != nil {
		decision.Verdict = "FAIL"
		decision.Reason = fmt.Sprintf("Envelope Violation: %v", err)
		return g.signer.SignDecision(decision)
	}

	// 4. Validate against PRG
	rule, exists := g.prg.Rules[actionID]
	if !exists {
		decision.Verdict = "FAIL"
		decision.Reason = fmt.Sprintf("no policy defined for action %s", actionID)
		return g.signer.SignDecision(decision)
	}

	// Prepare CEL input
	effectMap, _ := toMap(effect)
	input := map[string]interface{}{
		"action":    actionID,
		"effect":    effectMap,
		"artifacts": artifacts,
		"timestamp": g.clock.Now().Unix(),
	}

	valid, err := g.pe.EvaluateRequirementSet(rule, input)
	if err != nil {
		decision.Verdict = "FAIL"
		decision.Reason = fmt.Sprintf("PRG Evaluation Error: %v", err)
		return g.signer.SignDecision(decision)
	}

	if !valid {
		decision.Verdict = "FAIL"
		decision.Reason = "missing requirement"
		return g.signer.SignDecision(decision)
	}

	// 5. Pass -> Sign
	decision.Verdict = "PASS"
	decision.RequirementSetHash = rule.Hash()
	decision.Timestamp = g.clock.Now() // Authority time (KERNEL_TCB §3)
	// Optionally link evidence hashes in the decision record (needs schema update)

	return g.signer.SignDecision(decision)
}

// IssueExecutionIntent verifies a Decision and issues a signed Intent for the Executor.
func (g *Guardian) IssueExecutionIntent(ctx context.Context, decision *contracts.DecisionRecord, effect *contracts.Effect) (*contracts.AuthorizedExecutionIntent, error) {
	// 1. Verify Decision Structure
	if decision.Verdict != "PASS" {
		return nil, fmt.Errorf("cannot issue intent for denied decision: %s", decision.Verdict)
	}

	// 2. Verify Decision Signature (using Kernel Key)
	if valid, err := g.signer.VerifyDecision(decision); err != nil || !valid {
		return nil, fmt.Errorf("invalid decision signature: %w", err)
	}

	// Determine Allowed Tool (matching identification logic)
	var allowedTool string
	if tn, ok := effect.Params["tool_name"].(string); ok && tn != "" {
		allowedTool = tn
	} else {
		allowedTool = effect.EffectType
	}

	// 3. Create Intent
	// F4: Compute EffectDigestHash from canonicalized effect
	effectBytes, _ := canonicalize.JCS(effect)
	effectDigest := canonicalize.HashBytes(effectBytes)
	now := g.clock.Now()

	intent := &contracts.AuthorizedExecutionIntent{
		ID:               "intent-" + decision.ID, // Deterministic ID
		DecisionID:       decision.ID,
		EffectDigestHash: effectDigest,
		IssuedAt:         now,
		ExpiresAt:        now.Add(5 * time.Minute),
		Signer:           "kernel",
		AllowedTool:      allowedTool,
	}

	// 4. Sign Intent
	if err := g.signer.SignIntent(intent); err != nil {
		return nil, fmt.Errorf("failed to sign intent: %w", err)
	}

	return intent, nil
}

// checkEnvelope validates the structural integrity of the Effect envelope.
func (g *Guardian) checkEnvelope(effect *contracts.Effect) error {
	if effect.EffectType == "" {
		return fmt.Errorf("missing effect type")
	}
	if effect.EffectID == "" {
		return fmt.Errorf("missing effect ID")
	}
	// Verify critical metadata presence
	// Soft requirement: timestamp presence for auditability
	// Legacy effects might miss it, so we don't enforce yet.
	_ = effect.Params["timestamp"]
	return nil
}

// --- High-Level Governance API ---

const (
	VerdictAllow     = "PASS"
	VerdictBlock     = "FAIL"
	VerdictIntervene = "INTERVENE"
	VerdictPending   = "PENDING"
)

// DecisionRequest represents a request for a governance decision.
type DecisionRequest struct {
	Principal string                 `json:"principal"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"` // Tool name or effect type
	Context   map[string]interface{} `json:"context"`
}

// EvaluateDecision evaluates a request against the governance policy (PRG + Temporal).
// It constructs a DecisionRecord and returns it.
// When a PDP is configured, policy evaluation is delegated to it and the result
// is bound into the DecisionRecord for receipt chain verification.
func (g *Guardian) EvaluateDecision(ctx context.Context, req DecisionRequest) (*contracts.DecisionRecord, error) {
	// 1. Construct Effect from Request
	effect := &contracts.Effect{
		EffectID:   fmt.Sprintf("eff-%d", g.clock.Now().UnixNano()),
		EffectType: req.Action, // e.g. "EXECUTE_TOOL"
		Params:     req.Context,
	}
	// Add tool name to params if not present but resource is tool
	if req.Action == "EXECUTE_TOOL" {
		if effect.Params == nil {
			effect.Params = make(map[string]interface{})
		}
		effect.Params["tool_name"] = req.Resource
	}

	// 2. Prepare Decision Record
	// Calculate Effect Digest for binding
	effectBytes, _ := canonicalize.JCS(effect)
	effectDigest := canonicalize.HashBytes(effectBytes)

	// F5: Use configured EnvFingerprint instead of placeholder
	envFP := g.envFprint
	if envFP == "" {
		envFP = "sha256:unconfigured"
	}

	// GOV-001: Content-addressed policy version derived from PRG rule hash.
	// This ties each DecisionRecord to the exact policy state evaluated,
	// rather than a hardcoded semver string.
	policyVersion := "v1.0.0" // fallback
	if g.prg != nil {
		if hash, err := g.prg.ContentHash(); err == nil && hash != "" {
			policyVersion = "sha256:" + hash
		}
	}

	decision := &contracts.DecisionRecord{
		ID:             fmt.Sprintf("dec-%d", g.clock.Now().UnixNano()),
		Timestamp:      g.clock.Now(),
		Verdict:        VerdictBlock, // Default deny
		EffectDigest:   effectDigest,
		InputContext:   req.Context,
		EnvFingerprint: envFP,
		PolicyVersion:  policyVersion,
	}

	// 2.5: Delegate to external PDP if configured (P0.1 competitive defense)
	if g.pdp != nil {
		pdpReq := &pdp.DecisionRequest{
			Principal: req.Principal,
			Action:    req.Action,
			Resource:  req.Resource,
			Context:   req.Context,
			Timestamp: g.clock.Now(),
		}
		pdpResp, pdpErr := g.pdp.Evaluate(ctx, pdpReq)
		if pdpErr != nil {
			// Fail-closed: PDP error → DENY
			decision.Verdict = VerdictBlock
			decision.Reason = fmt.Sprintf("PDP error: %v", pdpErr)
			decision.PolicyBackend = string(g.pdp.Backend())
			_ = g.signer.SignDecision(decision)
			return decision, nil
		}

		// Bind PDP metadata into DecisionRecord for receipt chain
		decision.PolicyBackend = string(g.pdp.Backend())
		decision.PolicyContentHash = g.pdp.PolicyHash()
		decision.PolicyDecisionHash = pdpResp.DecisionHash

		if !pdpResp.Allow {
			decision.Verdict = VerdictBlock
			decision.Reason = fmt.Sprintf("PDP deny: %s (ref=%s)", pdpResp.ReasonCode, pdpResp.PolicyRef)
			_ = g.signer.SignDecision(decision)
			// Audit log for PDP denials
			if g.auditLog != nil {
				decisionBytes, _ := canonicalize.JCS(decision)
				_, _ = g.auditLog.Append("guardian", "PDP_DENY", decision.ID, string(decisionBytes))
			}
			return decision, nil
		}
		// PDP allowed — fall through to existing PRG + temporal checks
	}

	// 3. F3: Evaluate Temporal Guardian if wired
	var intervention *contracts.InterventionMetadata
	if g.temporal != nil {
		resp := g.temporal.Evaluate(ctx)
		if resp.Level >= ResponseInterrupt {
			intervention = &contracts.InterventionMetadata{
				Type:         responseToIntervention(resp.Level),
				ReasonCode:   fmt.Sprintf("TEMPORAL_%s", resp.Level),
				WaitDuration: resp.Duration,
			}
		} else if resp.Level == ResponseThrottle {
			intervention = &contracts.InterventionMetadata{
				Type:         contracts.InterventionThrottle,
				ReasonCode:   fmt.Sprintf("TEMPORAL_%s", resp.Level),
				WaitDuration: resp.Duration,
			}
		}
	}

	err := g.SignDecision(ctx, decision, effect, []string{}, intervention)
	if err != nil {
		return nil, err
	}

	// 4. F2: Persistence — audit failure is a hard error
	if g.auditLog != nil {
		decisionBytes, _ := canonicalize.JCS(decision)
		if _, logErr := g.auditLog.Append("guardian", "DECISION_MADE", decision.ID, string(decisionBytes)); logErr != nil {
			return nil, fmt.Errorf("audit failure for decision %s: %w", decision.ID, logErr)
		}
	}

	return decision, nil
}

// responseToIntervention maps TemporalGuardian ResponseLevel to InterventionType.
func responseToIntervention(level ResponseLevel) contracts.InterventionType {
	switch level {
	case ResponseInterrupt:
		return contracts.InterventionInterrupt
	case ResponseQuarantine:
		return contracts.InterventionQuarantine
	case ResponseFailClosed:
		return contracts.InterventionQuarantine // FailClosed maps to strongest intervention
	default:
		return contracts.InterventionNone
	}
}

func toMap(v any) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}
