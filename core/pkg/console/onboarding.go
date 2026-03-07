package console

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/api"
)

// Onboarding request/response types

type signupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type signupResponse struct {
	Message  string `json:"message"`
	TenantID string `json:"tenantId"`
}

type onboardingVerifyRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type onboardingVerifyResponse struct {
	TenantID string `json:"tenantId"`
	APIKey   string `json:"apiKey"`
}

type resendRequest struct {
	Email string `json:"email"`
}

// handleSignupAPI handles POST /api/signup
// Creates a pending signup with a verification code.
func (s *Server) handleSignupAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.WriteMethodNotAllowed(w)
		return
	}

	var req signupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		api.WriteBadRequest(w, "Email and password are required")
		return
	}

	if len(req.Password) < 8 {
		api.WriteBadRequest(w, "Password must be at least 8 characters")
		return
	}

	// Generate tenant ID and verification code
	h := sha256.Sum256([]byte(req.Email + fmt.Sprintf("%d", time.Now().UnixNano())))
	tenantID := "tenant-" + hex.EncodeToString(h[:8])
	code := fmt.Sprintf("%06d", rand.Intn(1000000)) //nolint:gosec // verification code, not crypto

	s.onboardingMu.Lock()
	if existing, ok := s.pendingSignups[req.Email]; ok && existing.Verified {
		s.onboardingMu.Unlock()
		api.WriteConflict(w, "Email already verified")
		return
	}
	s.pendingSignups[req.Email] = &pendingSignup{
		Email:    req.Email,
		Password: req.Password,
		Code:     code,
		TenantID: tenantID,
		Verified: false,
	}
	s.onboardingMu.Unlock()

	// In production: send verification email here
	slog.Info("signup received", "email", req.Email, "tenant_id", tenantID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(signupResponse{
		Message:  "Verification code sent",
		TenantID: tenantID,
	})
}

// handleOnboardingVerifyAPI handles POST /api/onboarding/verify
// Verifies the signup code and provisions the tenant.
func (s *Server) handleOnboardingVerifyAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.WriteMethodNotAllowed(w)
		return
	}

	var req onboardingVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.Email == "" || req.Code == "" {
		api.WriteBadRequest(w, "Email and code are required")
		return
	}

	s.onboardingMu.Lock()
	signup, ok := s.pendingSignups[req.Email]
	if !ok {
		s.onboardingMu.Unlock()
		api.WriteNotFound(w, "No pending signup for this email")
		return
	}

	if signup.Verified {
		s.onboardingMu.Unlock()
		api.WriteConflict(w, "Email already verified")
		return
	}

	if signup.Code != req.Code {
		s.onboardingMu.Unlock()
		api.WriteUnauthorized(w, "Invalid verification code")
		return
	}

	signup.Verified = true
	s.onboardingMu.Unlock()

	// Generate API key
	keyHash := sha256.Sum256([]byte(signup.TenantID + fmt.Sprintf("%d", time.Now().UnixNano())))
	apiKey := "hk_" + hex.EncodeToString(keyHash[:16])

	slog.Info("signup verified", "email", req.Email, "tenant_id", signup.TenantID)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(onboardingVerifyResponse{
		TenantID: signup.TenantID,
		APIKey:   apiKey,
	})
}

// handleResendVerificationAPI handles POST /api/resend-verification
// Regenerates the verification code for a pending signup.
func (s *Server) handleResendVerificationAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.WriteMethodNotAllowed(w)
		return
	}

	var req resendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteBadRequest(w, "Invalid request body")
		return
	}

	if req.Email == "" {
		api.WriteBadRequest(w, "Email is required")
		return
	}

	s.onboardingMu.Lock()
	signup, ok := s.pendingSignups[req.Email]
	if !ok {
		s.onboardingMu.Unlock()
		// Silently succeed to not leak user existence
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "If the email exists, a new code was sent"})
		return
	}

	if signup.Verified {
		s.onboardingMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Email already verified"})
		return
	}

	newCode := fmt.Sprintf("%06d", rand.Intn(1000000)) //nolint:gosec // verification code
	signup.Code = newCode
	s.onboardingMu.Unlock()

	slog.Info("verification code resent", "email", req.Email)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"message": "Verification code resent"})
}
