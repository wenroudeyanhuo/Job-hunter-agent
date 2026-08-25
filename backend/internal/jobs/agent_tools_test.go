package jobs

import "testing"

func TestAgentToolRegistryDescribesAllowedActions(t *testing.T) {
	registry := NewDefaultAgentToolRegistry()

	tool, ok := registry.Get("run_crawl")
	if !ok {
		t.Fatal("expected run_crawl tool to be registered")
	}
	if tool.Name != "run_crawl" || tool.RiskLevel != AgentToolRiskMedium || !tool.RequiresApproval {
		t.Fatalf("unexpected run_crawl metadata: %#v", tool)
	}
	if tool.InputSchema == "" || tool.Preview == "" || tool.Description == "" {
		t.Fatalf("expected tool schema, preview, and description: %#v", tool)
	}

	for _, action := range AllowedModelActionTypes() {
		if _, ok := registry.Get(action.Type); !ok {
			t.Fatalf("allowed model action %q is missing from tool registry", action.Type)
		}
	}
}

func TestAttachAgentToolMetadataToActionRequest(t *testing.T) {
	request := AgentActionRequest{ActionType: "send_feishu_report"}
	request = AttachAgentToolMetadata(request, NewDefaultAgentToolRegistry())

	if request.ToolName != "send_feishu_report" {
		t.Fatalf("expected tool name, got %#v", request)
	}
	if request.RiskLevel != AgentToolRiskMedium || !request.RequiresApproval {
		t.Fatalf("expected approval-gated medium-risk tool metadata, got %#v", request)
	}
	if request.ToolPreview == "" || request.ToolDescription == "" {
		t.Fatalf("expected tool preview and description, got %#v", request)
	}
}
