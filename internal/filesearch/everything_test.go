package filesearch

import "testing"

func TestCompileQueryFromSemanticInput(t *testing.T) {
	t.Parallel()

	query := CompileQuery(ToolInput{
		Keywords:     []string{"单细胞", "相关"},
		Drives:       []string{"D盘"},
		Extensions:   []string{"pdf"},
		DateField:    "",
		DateValue:    "",
		KnownFolders: nil,
		Paths:        nil,
		Limit:        DefaultLimit,
	})
	if query != `file: D:\ *.pdf 单细胞` {
		t.Fatalf("unexpected query: %q", query)
	}
}

func TestDescribeExecutionUsesPathOptionForSingleDrive(t *testing.T) {
	t.Parallel()

	display := DescribeExecution(ToolInput{
		Drives:     []string{"d"},
		Extensions: []string{"csv"},
		Limit:      DefaultLimit,
	})
	if display != `-path D:\ file: *.csv` {
		t.Fatalf("unexpected execution display: %q", display)
	}
}

func TestCompileQueryNormalizesDownloadsAndRecentDays(t *testing.T) {
	t.Parallel()

	query := CompileQuery(ToolInput{
		Paths:      []string{"Downloads"},
		DateField:  "created",
		DateValue:  "last2days",
		Limit:      DefaultLimit,
		Keywords:   []string{"下载目录"},
		Extensions: []string{"pdf"},
	})
	if query != "file: shell:Downloads *.pdf dc:last48hours" {
		t.Fatalf("unexpected query: %q", query)
	}
}

func TestDefinitionContainsToolContract(t *testing.T) {
	t.Parallel()

	spec := Definition()
	if spec.Name != ToolName {
		t.Fatalf("unexpected tool name: %q", spec.Name)
	}
	if spec.Purpose == "" || spec.Description == "" || spec.InputContract == "" || spec.OutputContract == "" || spec.Usage == "" || spec.InputJSONExample == "" || spec.OutputJSONExample == "" {
		t.Fatalf("tool definition should be self descriptive: %#v", spec)
	}
}
