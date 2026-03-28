package skilllib

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportArchiveExtractsSkillDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	archivePath := filepath.Join(root, "writer.zip")
	writeSkillArchive(t, archivePath, map[string]string{
		"writer/SKILL.md": strings.TrimSpace(`
---
name: writer
description: 帮助输出更清晰的中文写作
---
# Writer
给出简洁、结构清晰、少废话的中文输出。
`),
		"writer/examples/example.txt": "sample",
	})

	imported, err := ImportArchive(archivePath, filepath.Join(root, "skills"))
	if err != nil {
		t.Fatalf("import archive: %v", err)
	}
	if imported.Name != "writer" {
		t.Fatalf("unexpected skill name: %q", imported.Name)
	}
	if !strings.HasSuffix(imported.Dir, filepath.Join("skills", "writer")) {
		t.Fatalf("unexpected skill dir: %q", imported.Dir)
	}

	skills, err := NewLoader(filepath.Join(root, "skills")).List()
	if err != nil {
		t.Fatalf("list imported skills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "writer" {
		t.Fatalf("unexpected listed skill: %#v", skills[0])
	}

	if _, err := os.Stat(filepath.Join(root, "skills", "writer", "examples", "example.txt")); err != nil {
		t.Fatalf("stat extracted file: %v", err)
	}
}

func TestImportArchiveSupportsRootLevelPackage(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	archivePath := filepath.Join(root, "translator.zip")
	writeSkillArchive(t, archivePath, map[string]string{
		"SKILL.md": strings.TrimSpace(`
---
name: translator
description: 翻译时优先保留技术术语
---
# Translator
保留术语，避免过度意译。
`),
		"assets/terms.txt": "API\nSDK",
	})

	imported, err := ImportArchive(archivePath, filepath.Join(root, "skills"))
	if err != nil {
		t.Fatalf("import root-level archive: %v", err)
	}
	if !strings.HasSuffix(imported.Dir, filepath.Join("skills", "translator")) {
		t.Fatalf("unexpected imported dir: %q", imported.Dir)
	}
	if _, err := os.Stat(filepath.Join(root, "skills", "translator", "assets", "terms.txt")); err != nil {
		t.Fatalf("stat extracted asset: %v", err)
	}
}

func TestInspectArchiveRejectsInvalidSkillDefinition(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	archivePath := filepath.Join(root, "broken.zip")
	writeSkillArchive(t, archivePath, map[string]string{
		"writer/SKILL.md": strings.TrimSpace(`
---
name: writer
---
# Writer
给出简洁输出。
`),
	})

	_, err := InspectArchive(archivePath)
	if err == nil {
		t.Fatal("expected invalid skill archive to fail")
	}
	if !strings.Contains(err.Error(), "description") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInspectArchiveRejectsMultipleTopLevelRoots(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	archivePath := filepath.Join(root, "broken-roots.zip")
	writeSkillArchive(t, archivePath, map[string]string{
		"writer/SKILL.md": strings.TrimSpace(`
---
name: writer
description: 帮助输出更清晰的中文写作
---
# Writer
给出简洁输出。
`),
		"notes/readme.txt": "extra root",
	})

	_, err := InspectArchive(archivePath)
	if err == nil {
		t.Fatal("expected archive with multiple roots to fail")
	}
	if !strings.Contains(err.Error(), "一个顶层技能目录") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeSkillArchive(t *testing.T, archivePath string, files map[string]string) {
	t.Helper()

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}

	writer := zip.NewWriter(file)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create archive entry %s: %v", name, err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("write archive entry %s: %v", name, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close archive writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close archive file: %v", err)
	}
}
