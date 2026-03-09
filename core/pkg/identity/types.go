package identity

// IdentityToken represents an authenticated identity via SSO.
type IdentityToken struct {
	Subject string                 `json:"sub"`
	Email   string                 `json:"email"`
	Issuer  string                 `json:"iss"`
	Claims  map[string]interface{} `json:"claims"`
}

type PrincipalType string

const (
	PrincipalUser    PrincipalType = "USER"
	PrincipalAgent   PrincipalType = "AGENT"
	PrincipalService PrincipalType = "SERVICE"
)

// Principal represents any entity that can be authenticated.
type Principal interface {
	ID() string
	Type() PrincipalType
}

// AgentIdentity represents a HELM agent.
type AgentIdentity struct {
	AgentID     string
	DelegatorID string // User who delegated execution
	Scopes      []string
}

func (a *AgentIdentity) ID() string          { return a.AgentID }
func (a *AgentIdentity) Type() PrincipalType { return PrincipalAgent }
