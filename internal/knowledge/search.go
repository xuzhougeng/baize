package knowledge

import (
	"slices"
	"strings"
	"unicode"
)

const maxGeneratedKeywords = 48

var (
	leadingNoisePhrases = []string{
		"请问", "问下", "帮我", "怎么", "如何", "怎样", "为何", "为啥", "有没有", "能不能", "能否", "可否", "是否", "什么", "什么时候",
	}
	trailingNoisePhrases = []string{
		"吗", "呢", "呀", "啊", "嘛", "么", "什么", "怎么样", "如何", "多少", "哪个", "哪儿", "哪里",
	}
	hanSplitRunes = map[rune]struct{}{
		'的': {}, '了': {}, '和': {}, '与': {}, '及': {}, '并': {}, '就': {}, '都': {}, '也': {}, '把': {}, '给': {}, '向': {}, '从': {}, '到': {}, '在': {}, '于': {},
	}
	preferredHanTerms = []string{
		"环境变量", "局域网", "服务器", "浏览器", "知识库", "二维码", "用户名", "家目录",
		"共享", "查看", "测试", "英伟达", "推理", "登录", "账号", "密码", "权限", "目录", "文件", "镜像", "配置", "方法", "工具", "自动化", "构建",
		"安装", "版本", "接口", "提醒", "微信", "终端", "模型", "翻译", "摘要", "检索", "向量", "问题", "错误", "报错", "日志", "状态", "路径",
		"时间", "内容", "图片", "方式", "解封", "禁止", "支持", "删除", "补充", "追加", "创建", "运行", "启动", "停止", "命令", "窗口", "谷歌", "出品",
	}
)

type SearchResult struct {
	Entry   Entry
	Score   int
	Matches []string
}

func GenerateKeywords(text string) []string {
	var (
		out      []string
		seen     = make(map[string]struct{})
		asciiBuf []rune
		hanBuf   []rune
	)

	flushASCII := func() {
		if len(asciiBuf) == 0 {
			return
		}
		addKeyword(&out, seen, strings.ToLower(string(asciiBuf)))
		asciiBuf = asciiBuf[:0]
	}
	flushHan := func() {
		if len(hanBuf) == 0 {
			return
		}
		addHanKeywords(&out, seen, hanBuf)
		hanBuf = hanBuf[:0]
	}

	for _, r := range text {
		switch {
		case isASCIIKeywordRune(r):
			flushHan()
			asciiBuf = append(asciiBuf, unicode.ToLower(r))
		case isHan(r):
			flushASCII()
			hanBuf = append(hanBuf, r)
		default:
			flushASCII()
			flushHan()
		}
		if len(out) >= maxGeneratedKeywords {
			return out[:maxGeneratedKeywords]
		}
	}

	flushASCII()
	flushHan()
	if len(out) > maxGeneratedKeywords {
		return out[:maxGeneratedKeywords]
	}
	return out
}

func MergeKeywords(groups ...[]string) []string {
	var (
		out  []string
		seen = make(map[string]struct{})
	)
	for _, group := range groups {
		for _, keyword := range group {
			addKeyword(&out, seen, keyword)
			if len(out) >= maxGeneratedKeywords {
				return out[:maxGeneratedKeywords]
			}
		}
	}
	return out
}

func RankEntries(entries []Entry, query string, extraKeywords []string, limit int) []SearchResult {
	terms := MergeKeywords(GenerateKeywords(query), extraKeywords)
	if len(terms) == 0 {
		return nil
	}

	results := make([]SearchResult, 0, len(entries))
	for _, entry := range entries {
		scored := scoreEntry(entry, terms)
		if scored.Score == 0 {
			continue
		}
		results = append(results, scored)
	}

	slices.SortFunc(results, func(a, b SearchResult) int {
		if a.Score != b.Score {
			return b.Score - a.Score
		}
		switch {
		case a.Entry.RecordedAt.After(b.Entry.RecordedAt):
			return -1
		case a.Entry.RecordedAt.Before(b.Entry.RecordedAt):
			return 1
		default:
			return strings.Compare(a.Entry.ID, b.Entry.ID)
		}
	})

	if limit > 0 && len(results) > limit {
		return results[:limit]
	}
	return results
}

func scoreEntry(entry Entry, terms []string) SearchResult {
	entryKeywords := MergeKeywords(GenerateKeywords(entry.Text), entry.Keywords)
	keywordSet := make(map[string]struct{}, len(entryKeywords))
	for _, keyword := range entryKeywords {
		keywordSet[keyword] = struct{}{}
	}

	var (
		score     int
		matches   []string
		matchSeen = make(map[string]struct{})
		lowerText = strings.ToLower(entry.Text)
	)
	for _, term := range terms {
		switch {
		case term == "":
			continue
		case containsKeyword(keywordSet, term):
			score += 8 + min(len([]rune(term)), 6)
			addKeyword(&matches, matchSeen, term)
		case strings.Contains(lowerText, term):
			score += 4 + min(len([]rune(term)), 4)
			addKeyword(&matches, matchSeen, term)
		}
	}

	return SearchResult{
		Entry:   entry,
		Score:   score,
		Matches: matches,
	}
}

func addHanKeywords(out *[]string, seen map[string]struct{}, runes []rune) {
	for _, part := range splitHanRun(runes) {
		part = trimHanNoise(part)
		if len([]rune(part)) < 2 {
			continue
		}
		addHanPartKeywords(out, seen, []rune(part))
		if len(*out) >= maxGeneratedKeywords {
			return
		}
	}
}

func splitHanRun(runes []rune) []string {
	var (
		parts []string
		buf   []rune
	)
	flush := func() {
		if len(buf) == 0 {
			return
		}
		parts = append(parts, string(buf))
		buf = buf[:0]
	}

	for _, r := range runes {
		if _, ok := hanSplitRunes[r]; ok {
			flush()
			continue
		}
		buf = append(buf, r)
	}
	flush()
	return parts
}

func trimHanNoise(text string) string {
	text = strings.TrimSpace(text)
	for {
		trimmed := false
		for _, prefix := range leadingNoisePhrases {
			if strings.HasPrefix(text, prefix) {
				text = strings.TrimSpace(strings.TrimPrefix(text, prefix))
				trimmed = true
			}
		}
		for _, suffix := range trailingNoisePhrases {
			if strings.HasSuffix(text, suffix) {
				text = strings.TrimSpace(strings.TrimSuffix(text, suffix))
				trimmed = true
			}
		}
		if !trimmed {
			return text
		}
	}
}

func addHanPartKeywords(out *[]string, seen map[string]struct{}, runes []rune) {
	if len(runes) < 2 {
		return
	}

	var gap []rune
	flushGap := func() {
		if len(gap) == 0 {
			return
		}
		addHanGapKeywords(out, seen, gap)
		gap = gap[:0]
	}

	for len(runes) > 0 {
		term, width := longestPreferredPrefix(runes)
		if width > 0 {
			flushGap()
			addKeyword(out, seen, term)
			runes = runes[width:]
			if len(*out) >= maxGeneratedKeywords {
				return
			}
			continue
		}
		gap = append(gap, runes[0])
		runes = runes[1:]
	}
	flushGap()
}

func longestPreferredPrefix(runes []rune) (string, int) {
	remaining := string(runes)
	for _, term := range preferredHanTerms {
		if strings.HasPrefix(remaining, term) {
			return term, len([]rune(term))
		}
	}
	return "", 0
}

func addHanGapKeywords(out *[]string, seen map[string]struct{}, gap []rune) {
	text := trimHanNoise(string(gap))
	runes := []rune(text)
	switch length := len(runes); {
	case length < 2:
		return
	case length <= 3:
		addKeyword(out, seen, text)
	case length == 4 || length == 6:
		for start := 0; start+2 <= length; start += 2 {
			addKeyword(out, seen, string(runes[start:start+2]))
		}
	case length == 5:
		addKeyword(out, seen, string(runes[:2]))
		addKeyword(out, seen, string(runes[2:]))
	default:
		addKeyword(out, seen, string(runes[:2]))
		addKeyword(out, seen, string(runes[length-2:]))
	}
}

func addKeyword(out *[]string, seen map[string]struct{}, keyword string) {
	keyword = strings.TrimSpace(strings.ToLower(keyword))
	if len([]rune(keyword)) < 2 {
		return
	}
	if _, ok := seen[keyword]; ok {
		return
	}
	seen[keyword] = struct{}{}
	*out = append(*out, keyword)
}

func containsKeyword(set map[string]struct{}, keyword string) bool {
	_, ok := set[strings.ToLower(strings.TrimSpace(keyword))]
	return ok
}

func isASCIIKeywordRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func isHan(r rune) bool {
	return unicode.Is(unicode.Han, r)
}
