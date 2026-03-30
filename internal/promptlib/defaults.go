package promptlib

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultPromptSeedVersion = "v1"

var builtinPrompts = []Prompt{
	{
		Title: "5题联想式心理测试",
		Content: strings.TrimSpace(`
你现在扮演“5题联想式心理测试引导器”。

严格遵守以下规则：

1. 这是一个轻量心理测试，只用于自我观察和娱乐，不是医学诊断。
2. 一次只能问 1 道题，总共 5 道题。
3. 第 1 题到第 5 题的“出题消息”，你的回复必须且只能是 1 个 JSON 对象。
4. 这个 JSON 对象的格式必须严格固定为：

{"question":"题目内容","questiontype":"singleselect","options":[{"value":"a","label":"选项A"},{"value":"b","label":"选项B"},{"value":"c","label":"选项C"},{"value":"d","label":"选项D"}]}

5. 字段要求如下：
- question：当前题目的文字内容
- questiontype：固定写 "singleselect"
- options：必须是长度为 4 的数组
- 每个选项对象必须且只能有两个字段：
  - value：简短标识，如 "a" "b" "c" "d"
  - label：给用户看到的选项文字

6. 不要输出代码块，不要输出 Markdown，不要输出解释，不要输出前言或后记，不要在 JSON 前后添加任何文字。
7. 不要把 JSON 放进数组，不要多包一层对象，不要添加 type、title、desc、analysis、result 等额外字段。
8. 第 1 题直接开始，不要等待，不要说明规则。
9. 从第 2 题开始，必须根据用户上一题的选择，联想生成下一题。
10. 联想方向要围绕情绪倾向、关系模式、安全感、控制感、决策方式、自我认同自然展开。
11. 题目要场景化、直觉化、简短，不要像标准量表，不要使用专业术语。
12. 每题固定 4 个选项，选项文案要自然、简短、有画面感。
13. 5 道题不能重复，不能只是换个说法。
14. 用户无论回复某个选项的 value，还是直接回复该选项的 label，都视为有效选择。
15. 用户回答完第 5 题后，你不要再输出 JSON，也不要再给选项，直接输出最终结果。
16. 最终结果必须是普通文本，格式严格如下：

【测试结果】类型名
一句总体判断。
3 条简短分析，每条 1 句话。
1 条具体建议。

17. 最终结果必须基于这 5 题的实际选择路径，不能写成通用套话。

现在开始，第 1 题直接输出 JSON。
`),
		RecordedAt: time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC),
	},
}

func DefaultPrompts() []Prompt {
	defaults := make([]Prompt, len(builtinPrompts))
	copy(defaults, builtinPrompts)
	return defaults
}

func DefaultPromptSeedMarker(dataDir string) string {
	return filepath.Join(dataDir, "prompts", ".seeded-"+defaultPromptSeedVersion)
}

func SeedDefaultPrompts(ctx context.Context, store *Store, markerPath string) error {
	if store == nil {
		return nil
	}
	if _, err := os.Stat(markerPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	existing, err := store.List(ctx)
	if err != nil {
		return err
	}

	existingTitles := make(map[string]struct{}, len(existing))
	for _, prompt := range existing {
		title := strings.TrimSpace(prompt.Title)
		if title == "" {
			continue
		}
		existingTitles[strings.ToLower(title)] = struct{}{}
	}

	for _, prompt := range DefaultPrompts() {
		key := strings.ToLower(strings.TrimSpace(prompt.Title))
		if _, ok := existingTitles[key]; ok {
			continue
		}
		if _, err := store.Add(ctx, prompt); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(markerPath, []byte(defaultPromptSeedVersion+"\n"), 0o644)
}
