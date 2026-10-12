package detector

import "path/filepath"

// Inspects a file path extension and maps it to a supported programming language identifier string.
func detectLanguage(path string) string {
	ext := filepath.Ext(path)
	languageMap := map[string]string{
		".py":    "python",
		".java":  "java",
		".js":    "javascript",
		".ts":    "typescript",
		".go":    "go",
		".rs":    "rust",
		".cpp":   "cpp",
		".c":     "c",
		".h":     "c",
		".rb":    "ruby",
		".php":   "php",
		".cs":    "csharp",
		".kt":    "kotlin",
		".swift": "swift",
	}
	if lang, ok := languageMap[ext]; ok {
		return lang
	}
	return "unknown"
}
