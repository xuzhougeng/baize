package skilllib

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type ArchivePackage struct {
	Skill Skill

	DirName     string
	archiveRoot string
}

func InspectArchive(zipPath string) (ArchivePackage, error) {
	zipPath = strings.TrimSpace(zipPath)
	if strings.ToLower(filepath.Ext(zipPath)) != ".zip" {
		return ArchivePackage{}, errors.New("skill 上传只支持 .zip 文件。")
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return ArchivePackage{}, fmt.Errorf("打开 skill zip 失败: %w", err)
	}
	defer reader.Close()

	files := make([]archiveFile, 0, len(reader.File))
	var skillFile *zip.File
	var skillPath string
	for _, file := range reader.File {
		normalized, ok, err := normalizeArchivePath(file.Name)
		if err != nil {
			return ArchivePackage{}, err
		}
		if !ok || shouldIgnoreArchiveEntry(normalized) {
			continue
		}

		files = append(files, archiveFile{
			file:       file,
			normalized: normalized,
		})

		if file.FileInfo().IsDir() {
			continue
		}
		if path.Base(normalized) != skillFileName {
			continue
		}
		if normalized != skillFileName && strings.Count(normalized, "/") != 1 {
			return ArchivePackage{}, errors.New("SKILL.md 必须位于 zip 根目录，或位于唯一顶层技能目录下。")
		}
		if skillFile != nil {
			return ArchivePackage{}, errors.New("zip 内只能包含一个 SKILL.md。")
		}

		skillFile = file
		skillPath = normalized
	}

	if skillFile == nil {
		return ArchivePackage{}, errors.New("zip 内缺少 SKILL.md。")
	}

	archiveRoot := ""
	if skillPath != skillFileName {
		archiveRoot = strings.Split(skillPath, "/")[0]
		for _, entry := range files {
			if entry.normalized == archiveRoot || strings.HasPrefix(entry.normalized, archiveRoot+"/") {
				continue
			}
			return ArchivePackage{}, errors.New("zip 内只能包含一个顶层技能目录。")
		}
	}

	content, err := readArchiveFile(skillFile)
	if err != nil {
		return ArchivePackage{}, err
	}
	if strings.TrimSpace(content) == "" {
		return ArchivePackage{}, errors.New("SKILL.md 不能为空。")
	}

	meta := parseFrontmatter(content)
	if len(meta) == 0 {
		return ArchivePackage{}, errors.New("SKILL.md 缺少 frontmatter，至少需要 name 和 description。")
	}

	name := strings.TrimSpace(meta["name"])
	if name == "" {
		return ArchivePackage{}, errors.New("SKILL.md frontmatter 缺少 name。")
	}

	description := strings.TrimSpace(meta["description"])
	if description == "" {
		return ArchivePackage{}, errors.New("SKILL.md frontmatter 缺少 description。")
	}

	if strings.TrimSpace(stripFrontmatter(content)) == "" {
		return ArchivePackage{}, errors.New("SKILL.md frontmatter 后必须包含技能说明内容。")
	}

	dirName := archiveRoot
	if dirName == "" {
		dirName = name
	}
	dirName, err = sanitizeArchiveDirName(dirName)
	if err != nil {
		return ArchivePackage{}, err
	}

	return ArchivePackage{
		Skill: Skill{
			Name:        name,
			Description: description,
			Content:     strings.TrimSpace(content),
		},
		DirName:     dirName,
		archiveRoot: archiveRoot,
	}, nil
}

func ImportArchive(zipPath, targetRoot string) (Skill, error) {
	pkg, err := InspectArchive(zipPath)
	if err != nil {
		return Skill{}, err
	}

	targetRoot = filepath.Clean(strings.TrimSpace(targetRoot))
	if targetRoot == "" {
		return Skill{}, errors.New("缺少 skill 目标目录。")
	}
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return Skill{}, fmt.Errorf("创建 skill 目录失败: %w", err)
	}

	targetDir := filepath.Join(targetRoot, pkg.DirName)
	if _, err := os.Stat(targetDir); err == nil {
		return Skill{}, fmt.Errorf("skill 目录 %q 已存在。", pkg.DirName)
	} else if !os.IsNotExist(err) {
		return Skill{}, fmt.Errorf("检查 skill 目录失败: %w", err)
	}

	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return Skill{}, fmt.Errorf("打开 skill zip 失败: %w", err)
	}
	defer reader.Close()

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return Skill{}, fmt.Errorf("创建 skill 目录失败: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(targetDir)
		}
	}()

	for _, file := range reader.File {
		normalized, ok, err := normalizeArchivePath(file.Name)
		if err != nil {
			return Skill{}, err
		}
		if !ok || shouldIgnoreArchiveEntry(normalized) {
			continue
		}

		relativePath := normalized
		if pkg.archiveRoot != "" {
			relativePath = strings.TrimPrefix(normalized, pkg.archiveRoot)
			relativePath = strings.TrimPrefix(relativePath, "/")
		}
		if relativePath == "" {
			continue
		}

		destination, err := resolveArchiveDestination(targetDir, relativePath)
		if err != nil {
			return Skill{}, err
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(destination, 0o755); err != nil {
				return Skill{}, fmt.Errorf("创建目录 %q 失败: %w", relativePath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return Skill{}, fmt.Errorf("创建目录 %q 失败: %w", filepath.Dir(relativePath), err)
		}
		if err := extractArchiveFile(file, destination); err != nil {
			return Skill{}, err
		}
	}

	pkg.Skill.Dir = targetDir
	success = true
	return pkg.Skill, nil
}

type archiveFile struct {
	file       *zip.File
	normalized string
}

func normalizeArchivePath(raw string) (string, bool, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	for strings.HasPrefix(normalized, "./") {
		normalized = strings.TrimPrefix(normalized, "./")
	}
	normalized = strings.TrimPrefix(normalized, "/")
	if normalized == "" {
		return "", false, nil
	}

	cleaned := path.Clean(normalized)
	if cleaned == "." {
		return "", false, nil
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, "/../") {
		return "", false, fmt.Errorf("zip 内包含非法路径 %q。", raw)
	}
	if strings.HasPrefix(cleaned, "/") {
		return "", false, fmt.Errorf("zip 内包含非法绝对路径 %q。", raw)
	}
	return cleaned, true, nil
}

func shouldIgnoreArchiveEntry(name string) bool {
	base := path.Base(name)
	if name == "__MACOSX" || strings.HasPrefix(name, "__MACOSX/") {
		return true
	}
	switch base {
	case ".DS_Store", "Thumbs.db":
		return true
	}
	return strings.HasPrefix(base, "._")
}

func readArchiveFile(file *zip.File) (string, error) {
	reader, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("打开 %s 失败: %w", file.Name, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("读取 %s 失败: %w", file.Name, err)
	}
	return string(data), nil
}

func sanitizeArchiveDirName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "", errors.New("skill 名称不能为空。")
	}
	if strings.ContainsAny(name, `/\`) {
		return "", errors.New("skill 名称不能包含路径分隔符。")
	}
	if strings.ContainsAny(name, "\x00\r\n") {
		return "", errors.New("skill 名称包含非法字符。")
	}
	if strings.Trim(name, ". ") == "" {
		return "", errors.New("skill 名称不合法。")
	}
	return name, nil
}

func stripFrontmatter(content string) string {
	trimmed := strings.TrimSpace(content)
	lines := strings.Split(trimmed, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return trimmed
	}

	for idx := 1; idx < len(lines); idx++ {
		if strings.TrimSpace(lines[idx]) != "---" {
			continue
		}
		return strings.TrimSpace(strings.Join(lines[idx+1:], "\n"))
	}
	return ""
}

func resolveArchiveDestination(rootDir, relativePath string) (string, error) {
	destination := filepath.Join(rootDir, filepath.FromSlash(relativePath))
	relative, err := filepath.Rel(rootDir, destination)
	if err != nil {
		return "", fmt.Errorf("解析 skill 文件路径失败: %w", err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("skill 文件路径越界: %s", relativePath)
	}
	return destination, nil
}

func extractArchiveFile(file *zip.File, destination string) error {
	reader, err := file.Open()
	if err != nil {
		return fmt.Errorf("打开 %s 失败: %w", file.Name, err)
	}
	defer reader.Close()

	mode := file.Mode().Perm()
	if mode == 0 {
		mode = 0o644
	}

	target, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("写入 %s 失败: %w", destination, err)
	}

	_, copyErr := io.Copy(target, reader)
	closeErr := target.Close()
	if copyErr != nil {
		return fmt.Errorf("写入 %s 失败: %w", destination, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("关闭 %s 失败: %w", destination, closeErr)
	}
	return nil
}
