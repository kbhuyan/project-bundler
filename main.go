// project-bundler/main.go
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// --- Configuration Section ---

// ProjectConfig defines the bundling rules for a specific project type.
type ProjectConfig struct {
	IgnoreDirs []string
	IgnoreExts []string
	LangMap    map[string]string
}

// baseLangMap contains common language mappings for extensions.
var baseLangMap = map[string]string{
	".md":        "markdown",
	".sh":        "shell",
	".json":      "json",
	".yml":       "yaml",
	".yaml":      "yaml",
	".toml":      "toml",
	".txt":       "text",
	".gitignore": "text",
	".proto":     "protobuf",
}

// filenameLangMap contains mappings for well-known filenames that lack extensions.
var filenameLangMap = map[string]string{
	"Dockerfile": "dockerfile",
	"Makefile":   "makefile",
	"go.mod":     "go-mod",
	"go.sum":     "text",
	"LICENSE":    "text",
	"README":     "markdown",
}

// projectConfigs holds the presets for different project types.
var projectConfigs = map[string]ProjectConfig{
	"generic": {
		IgnoreDirs: []string{".git"},
		IgnoreExts: []string{".DS_Store", ".log", ".lock"},
	},
	"android": {
		IgnoreDirs: []string{".git", ".idea", "build", ".gradle", "gradle"},
		IgnoreExts: []string{".DS_Store", ".iml", ".jar", ".keystore", ".jks", ".apk", ".aab", ".so", ".png", ".jpg", ".jpeg", ".gif", ".webp"},
		LangMap: map[string]string{
			".java":   "java",
			".kt":     "kotlin",
			".kts":    "kotlin",
			".xml":    "xml",
			".gradle": "groovy",
			".pro":    "text",
		},
	},
	"go": {
		IgnoreDirs: []string{".git", "vendor", "build"},
		IgnoreExts: []string{".DS_Store", ".exe", ".so", ".a"},
		LangMap: map[string]string{
			".go": "go",
		},
	},
	"rust": {
		IgnoreDirs: []string{".git", "target"},
		IgnoreExts: []string{".DS_Store", ".rlib", ".so", ".a", ".exe"},
		LangMap: map[string]string{
			".rs": "rust",
		},
	},
	"ios": {
		IgnoreDirs: []string{".git", ".idea", "Pods", "build", "DerivedData", ".swiftpm", "Carthage"},
		IgnoreExts: []string{".DS_Store", ".mobileprovision", ".app", ".ipa", ".car", ".xcassets", ".storyboardc", ".nib", ".png", ".jpg", ".jpeg"},
		LangMap: map[string]string{
			".swift":      "swift",
			".m":          "objectivec",
			".h":          "objectivec",
			".storyboard": "xml",
			".xib":        "xml",
			".plist":      "xml",
		},
	},
}

// --- Helper Functions ---

// stringSet is a helper type for efficient lookups (O(1) average).
type stringSet map[string]struct{}

func newStringSet(items []string) stringSet {
	s := make(stringSet, len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}
	return s
}

func (s stringSet) Contains(item string) bool {
	_, ok := s[item]
	return ok
}

// mergeMaps combines multiple maps. Keys in later maps overwrite earlier ones.
func mergeMaps(maps ...map[string]string) map[string]string {
	result := make(map[string]string)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// detectProjectType checks for landmark files to determine the project type.
func detectProjectType(srcDir string) string {
	landmarkFiles := map[string]string{
		"go.mod":        "go",
		"Cargo.toml":    "rust",
		"build.gradle":  "android",
		"Package.swift": "ios",
		"Podfile":       "ios",
	}

	for landmark, projectType := range landmarkFiles {
		if _, err := os.Stat(filepath.Join(srcDir, landmark)); err == nil {
			fmt.Printf("Auto-detected project type: %s\n", projectType)
			return projectType
		}
	}

	if matches, _ := filepath.Glob(filepath.Join(srcDir, "*.xcodeproj")); len(matches) > 0 {
		fmt.Println("Auto-detected project type: ios")
		return "ios"
	}

	fmt.Println("Could not auto-detect project type, using 'generic' defaults.")
	return "generic"
}

// isBinaryFile checks the first 1KB for null bytes to detect binary content.
func isBinaryFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Read up to 1024 bytes from the file.
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return false, err
	}

	// The actual slice of bytes read might be smaller than the buffer.
	readBytes := buffer[:n]

	// A null byte is a strong indicator of a binary file.
	return bytes.Contains(readBytes, []byte{0}), nil
}

// --- Main Execution ---

func main() {
	var availableTypes []string
	for k := range projectConfigs {
		availableTypes = append(availableTypes, k)
	}

	// 1. Define and parse command-line flags.
	srcDir := flag.String("src", ".", "Source project directory.")
	outputFile := flag.String("output", "bundle.md", "Output markdown file.")
	projectType := flag.String("type", "auto", "Project type. Options: "+strings.Join(availableTypes, ", "))
	reportSkipped := flag.Bool("report-skipped", false, "Report all skipped files and reasons.")
	flag.Parse()

	// 2. Determine and load project configuration.
	finalProjectType := *projectType
	if finalProjectType == "auto" {
		finalProjectType = detectProjectType(*srcDir)
	}

	config, ok := projectConfigs[finalProjectType]
	if !ok {
		log.Fatalf("Invalid project type '%s'. Available types are: %s", finalProjectType, strings.Join(availableTypes, ", "))
	}

	ignoreDirs := newStringSet(config.IgnoreDirs)
	ignoreExts := newStringSet(config.IgnoreExts)
	langMap := mergeMaps(baseLangMap, config.LangMap)
	skippedFiles := make(map[string][]string)

	// 3. Setup output file and buffered writer.
	file, err := os.Create(*outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	fmt.Printf("Starting to bundle project from '%s' into '%s' (type: %s)...\n", *srcDir, *outputFile, finalProjectType)

	// 4. Walk the directory tree.
	walkErr := filepath.WalkDir(*srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err // Propagate errors like permission denied.
		}

		// Skip directories that are in the ignore list.
		if d.IsDir() {
			if ignoreDirs.Contains(d.Name()) {
				skippedFiles["Ignored Directory"] = append(skippedFiles["Ignored Directory"], path)
				return filepath.SkipDir // Efficiently prune this entire directory.
			}
			return nil
		}

		// Skip files based on extension or full filename.
		ext := filepath.Ext(d.Name())
		if ignoreExts.Contains(ext) || ignoreExts.Contains(d.Name()) {
			skippedFiles["Ignored Extension/File"] = append(skippedFiles["Ignored Extension/File"], path)
			return nil
		}

		// IMPORTANT: Perform binary file detection to prevent corruption.
		isBinary, err := isBinaryFile(path)
		if err != nil {
			skippedFiles["File Read Error"] = append(skippedFiles["File Read Error"], path)
			log.Printf("Could not check file type for %s: %v", path, err)
			return nil
		}
		if isBinary {
			skippedFiles["Detected Binary Content"] = append(skippedFiles["Detected Binary Content"], path)
			return nil // Safely skip this binary file.
		}

		// At this point, the file is considered valid for bundling.
		fmt.Printf("  + Bundling file: %s\n", path)
		content, err := os.ReadFile(path)
		if err != nil {
			skippedFiles["File Read Error"] = append(skippedFiles["File Read Error"], path)
			log.Printf("Could not read file %s: %v", path, err)
			return nil
		}

		// Determine language for syntax highlighting.
		var lang string
		lang, ok = langMap[ext] // 1. Try by extension.
		if !ok {
			lang, ok = filenameLangMap[d.Name()] // 2. Try by full filename.
			if !ok {
				lang = "text" // 3. Default to plain text.
			}
		}

		relativePath, err := filepath.Rel(*srcDir, path)
		if err != nil {
			relativePath = path // Fallback to full path on error.
		}

		// Write the formatted block to the output buffer.
		header := fmt.Sprintf("File: /%s\n```%s\n", relativePath, lang)
		if _, err := writer.WriteString(header); err != nil {
			return err
		}
		if _, err := writer.Write(content); err != nil {
			return err
		}
		if _, err := writer.WriteString("\n```\n\n"); err != nil {
			return err
		}

		return nil
	})

	if walkErr != nil {
		log.Fatalf("Error during directory walk: %v", walkErr)
	}

	// 5. Print the optional skipped files report.
	if *reportSkipped {
		fmt.Println("\n--- Skipped Files Report ---")
		if len(skippedFiles) == 0 {
			fmt.Println("No files were skipped.")
		} else {
			for reason, paths := range skippedFiles {
				fmt.Printf("\nReason: %s\n", reason)
				for _, path := range paths {
					fmt.Printf("  - %s\n", path)
				}
			}
		}
		fmt.Println("--------------------------")
	}

	fmt.Printf("\n✅ Successfully created project bundle at '%s'\n", *outputFile)
}
