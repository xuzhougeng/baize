package weixin

import (
	"context"
	"testing"
)

func TestFileSenderDefinitionIsSelfDescriptive(t *testing.T) {
	t.Parallel()

	spec := Definition()
	if spec.Name != FileSenderToolName {
		t.Fatalf("unexpected tool name: %q", spec.Name)
	}
	if spec.Purpose == "" || spec.Description == "" || spec.InputContract == "" || spec.OutputContract == "" || spec.Usage == "" || spec.InputJSONExample == "" || spec.OutputJSONExample == "" {
		t.Fatalf("filesender definition should be self descriptive: %#v", spec)
	}
}

func TestFileSenderExecuteReturnsStructuredResult(t *testing.T) {
	t.Parallel()

	sender := &FileSender{}
	called := false
	sender.SetSendFunc(func(_ context.Context, toUserID, contextToken, filePath string) error {
		called = true
		if toUserID != "wxid-1" || contextToken != "ctx-1" || filePath != `D:\exports\output.csv` {
			t.Fatalf("unexpected send args: to=%q ctx=%q path=%q", toUserID, contextToken, filePath)
		}
		return nil
	})

	result, err := sender.Execute(context.Background(), FileSenderInput{
		ToUserID:     "wxid-1",
		ContextToken: "ctx-1",
		FilePath:     `D:\exports\output.csv`,
	})
	if err != nil {
		t.Fatalf("execute filesender: %v", err)
	}
	if !called {
		t.Fatal("expected send func to be called")
	}
	if result.Tool != FileSenderToolName || result.Status != "sent" || result.FilePath != `D:\exports\output.csv` {
		t.Fatalf("unexpected send result: %#v", result)
	}
}
