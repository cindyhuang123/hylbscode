package models

import "testing"

func TestReconcileExtraModels(t *testing.T) {
	const id = ModelID("test.extra.model")
	base := Model{ID: id, Name: "Test", Provider: ProviderOpenAI, APIModel: "test-model", ContextWindow: 8192}

	if err := ReconcileExtraModels([]Model{base}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, ok := SupportedModels[id]; !ok {
		t.Fatal("model not registered")
	}

	updated := base
	updated.ContextWindow = 16384
	if err := ReconcileExtraModels([]Model{updated}); err != nil {
		t.Fatalf("replace: %v", err)
	}
	if got := SupportedModels[id].ContextWindow; got != 16384 {
		t.Fatalf("replace: contextWindow = %d, want 16384", got)
	}

	if err := ReconcileExtraModels(nil); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if _, ok := SupportedModels[id]; ok {
		t.Fatal("model not removed after clear")
	}
}

func TestReconcileExtraModelsInvalidKeepsRegistry(t *testing.T) {
	bad := Model{ID: "test.extra.invalid", Provider: ProviderOpenAI}
	if err := ReconcileExtraModels([]Model{bad}); err == nil {
		t.Fatal("want validation error for missing api_model")
	}
	if _, ok := SupportedModels["test.extra.invalid"]; ok {
		t.Fatal("invalid model leaked into registry")
	}
}
