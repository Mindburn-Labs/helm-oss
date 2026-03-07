package finance_test

import (
	"testing"
	"time"

	"github.com/Mindburn-Labs/helm-oss/core/pkg/finance"
)

func TestBudget_TokenEnforcement(t *testing.T) {
	tracker := finance.NewInMemoryTracker()
	budgetID := "budget-tokens-daily"

	// 1. Set Budget: 1000 Tokens
	tracker.SetBudget(finance.Budget{
		ID:           budgetID,
		ResourceType: "TOKENS",
		Limit:        1000,
		Window:       finance.WindowDaily,
		ResetAt:      time.Now().Add(24 * time.Hour),
	})

	// 2. Check Valid Cost (500 tokens)
	costSmall := finance.Cost{Tokens: 500}
	ok, err := tracker.Check(budgetID, costSmall)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if !ok {
		t.Error("Expected cost to be allowed")
	}

	// 3. Consume 500 tokens
	if err := tracker.Consume(budgetID, costSmall); err != nil {
		t.Fatalf("Consume failed: %v", err)
	}

	// 4. Check Remaining (Start 1000 - 500 = 500 left)
	// Try to consume 600 -> Should fail
	costLarge := finance.Cost{Tokens: 600}
	ok, err = tracker.Check(budgetID, costLarge)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if ok {
		t.Error("Expected cost to be blocked (exceeds remaining budget)")
	}

	// 5. Attempt Consume -> Error
	if err := tracker.Consume(budgetID, costLarge); err == nil {
		t.Error("Expected error when consuming beyond budget")
	}

	// 6. Consume exact remaining (500) -> OK
	if err := tracker.Consume(budgetID, costSmall); err != nil {
		t.Errorf("Expected exact remaining consumption to pass, got: %v", err)
	}
}

func TestBudget_MoneyEnforcement(t *testing.T) {
	tracker := finance.NewInMemoryTracker()
	budgetID := "budget-usd-monthly"

	// 1. Set Budget: $10.00 (1000 cents)
	tracker.SetBudget(finance.Budget{
		ID:           budgetID,
		ResourceType: "USD",
		Limit:        1000, // $10.00
		Window:       finance.WindowMonthly,
	})

	// 2. Cost: $2.50
	cost1 := finance.Cost{Money: finance.NewMoney(250, "USD")}
	if err := tracker.Consume(budgetID, cost1); err != nil {
		t.Fatalf("Failed to consume $2.50: %v", err)
	}

	// 3. Cost: $8.00 (Total $10.50) -> Should fail
	cost2 := finance.Cost{Money: finance.NewMoney(800, "USD")}
	if err := tracker.Consume(budgetID, cost2); err == nil {
		t.Error("Allowed spending $10.50 on $10.00 budget")
	}

	// 4. Currency Mismatch
	costEUR := finance.Cost{Money: finance.NewMoney(100, "EUR")}
	if err := tracker.Consume(budgetID, costEUR); err == nil {
		t.Error("Allowed EUR spending on USD budget")
	}
}
