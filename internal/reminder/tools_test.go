package reminder

import "testing"

func TestAgentToolContractsHaveFamilyMetadata(t *testing.T) {
	t.Parallel()

	contracts := AgentToolContracts()
	if len(contracts) != 3 {
		t.Fatalf("expected 3 reminder tool contracts, got %d", len(contracts))
	}

	for _, contract := range contracts {
		contract = contract.Normalized()
		if contract.FamilyKey != ToolFamilyKey {
			t.Fatalf("tool %q FamilyKey = %q, want %q", contract.Name, contract.FamilyKey, ToolFamilyKey)
		}
		if contract.FamilyTitle != ToolFamilyTitle {
			t.Fatalf("tool %q FamilyTitle = %q, want %q", contract.Name, contract.FamilyTitle, ToolFamilyTitle)
		}
		if contract.DisplayTitle == "" {
			t.Fatalf("tool %q missing DisplayTitle", contract.Name)
		}
		if contract.OutputJSONExample == "" {
			t.Fatalf("tool %q missing OutputJSONExample", contract.Name)
		}
	}
}
