package types

// T0-8: Synthesis Policy Review Queue
// Implements the UNVERIFIED → PENDING_REVIEW → APPROVED/REJECTED workflow
// for inferred policies that emerge from synthesis heuristics.

// ReviewStatus represents the lifecycle state of an inferred policy.
type ReviewStatus string

const (
	ReviewUnverified    ReviewStatus = "UNVERIFIED"     // Inferred by synthesis, not yet reviewed
	ReviewPendingReview ReviewStatus = "PENDING_REVIEW" // Submitted for human review
	ReviewApproved      ReviewStatus = "APPROVED"       // Explicitly approved by authority
	ReviewRejected      ReviewStatus = "REJECTED"       // Explicitly rejected by authority
)

// PolicyReviewItem represents a single inferred policy awaiting review.
type PolicyReviewItem struct {
	// ReviewID uniquely identifies this review item.
	ReviewID string `json:"review_id"`

	// PolicyType categorizes the inferred policy.
	// e.g. "capability_grant", "regulation_rule", "module_activation"
	PolicyType string `json:"policy_type"`

	// Description is a human-readable summary of the inferred policy.
	Description string `json:"description"`

	// InferredBy identifies which synthesis rule produced this policy.
	InferredBy string `json:"inferred_by"`

	// Status is the current review status.
	Status ReviewStatus `json:"status"`

	// AffectedModules lists the modules this policy affects.
	AffectedModules []string `json:"affected_modules,omitempty"`

	// RiskScore is the computed risk of adopting this policy (0.0 = safe, 1.0 = critical).
	RiskScore float64 `json:"risk_score"`

	// InferredAt is when the policy was first inferred.
	InferredAt string `json:"inferred_at"`

	// ReviewedAt is when a human made a decision (empty if still unreviewed).
	ReviewedAt string `json:"reviewed_at,omitempty"`

	// ReviewedBy is the identity of the reviewer.
	ReviewedBy string `json:"reviewed_by,omitempty"`

	// Reason is the reviewer's reason for approval/rejection.
	Reason string `json:"reason,omitempty"`

	// GenomeHash is the genome hash at time of inference.
	GenomeHash string `json:"genome_hash"`
}

// PolicyReviewQueue manages the lifecycle of inferred policies (T0-8).
type PolicyReviewQueue struct {
	// Items is the ordered list of review items.
	Items []PolicyReviewItem `json:"items"`
}

// Submit adds a new inferred policy to the review queue.
func (q *PolicyReviewQueue) Submit(item PolicyReviewItem) {
	item.Status = ReviewPendingReview
	q.Items = append(q.Items, item)
}

// Approve marks a review item as approved.
func (q *PolicyReviewQueue) Approve(reviewID, reviewedBy, reason string, reviewedAt string) bool {
	for i := range q.Items {
		if q.Items[i].ReviewID == reviewID && q.Items[i].Status == ReviewPendingReview {
			q.Items[i].Status = ReviewApproved
			q.Items[i].ReviewedBy = reviewedBy
			q.Items[i].Reason = reason
			q.Items[i].ReviewedAt = reviewedAt
			return true
		}
	}
	return false
}

// Reject marks a review item as rejected.
func (q *PolicyReviewQueue) Reject(reviewID, reviewedBy, reason string, reviewedAt string) bool {
	for i := range q.Items {
		if q.Items[i].ReviewID == reviewID && q.Items[i].Status == ReviewPendingReview {
			q.Items[i].Status = ReviewRejected
			q.Items[i].ReviewedBy = reviewedBy
			q.Items[i].Reason = reason
			q.Items[i].ReviewedAt = reviewedAt
			return true
		}
	}
	return false
}

// Pending returns all items awaiting review.
func (q *PolicyReviewQueue) Pending() []PolicyReviewItem {
	var pending []PolicyReviewItem
	for _, item := range q.Items {
		if item.Status == ReviewPendingReview {
			pending = append(pending, item)
		}
	}
	return pending
}

// Unverified returns all items that haven't been submitted for review yet.
func (q *PolicyReviewQueue) Unverified() []PolicyReviewItem {
	var unverified []PolicyReviewItem
	for _, item := range q.Items {
		if item.Status == ReviewUnverified {
			unverified = append(unverified, item)
		}
	}
	return unverified
}

// AllApproved returns true if all items have been approved (no pending or rejected).
func (q *PolicyReviewQueue) AllApproved() bool {
	for _, item := range q.Items {
		if item.Status != ReviewApproved {
			return false
		}
	}
	return len(q.Items) > 0
}

// HasRejected returns true if any items have been rejected.
func (q *PolicyReviewQueue) HasRejected() bool {
	for _, item := range q.Items {
		if item.Status == ReviewRejected {
			return true
		}
	}
	return false
}
