package katago

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type DetectedSetup struct {
	BinaryPath string
	ModelPath  string
	ConfigPath string
	Version    string
	Errors     []string
}

func DetectKataGo() (*DetectedSetup, error) {
	setup := &DetectedSetup{
		Errors: []string{},
	}

	// 1. Find KataGo binary
	binaryPath, err := findKataGoBinary()
	if err != nil {
		setup.Errors = append(setup.Errors, fmt.Sprintf("Binary: %v", err))
	} else {
		setup.BinaryPath = binaryPath

		// Get version if binary found
		if version, vErr := getKataGoVersion(binaryPath); vErr == nil {
			setup.Version = version
		}
	}

	// 2. Find model files
	modelPath, err := findKataGoModel()
	if err != nil {
		setup.Errors = append(setup.Errors, fmt.Sprintf("Model: %v", err))
	} else {
		setup.ModelPath = modelPath
	}

	// 3. Find or generate config
	configPath, err := findOrGenerateConfig(setup.BinaryPath, setup.ModelPath)
	if err != nil {
		setup.Errors = append(setup.Errors, fmt.Sprintf("Config: %v", err))
	} else {
		setup.ConfigPath = configPath
	}

	// Return error if critical components missing
	if setup.BinaryPath == "" {
		return setup, fmt.Errorf("KataGo not found. Installation errors:\n%s", strings.Join(setup.Errors, "\n"))
	}

	return setup, nil
}

func findKataGoBinary() (string, error) {
	// Check common installation locations
	searchPaths := []string{
		// Environment variable
		os.Getenv("KATAGO_BINARY_PATH"),
		// System PATH
		"katago",
		// Common installation paths
		"/usr/local/bin/katago",
		"/usr/bin/katago",
		"/opt/homebrew/bin/katago",
		"/opt/local/bin/katago",
		// Windows paths
		"C:\\Program Files\\KataGo\\katago.exe",
		"C:\\KataGo\\katago.exe",
	}

	// Add user home paths
	if home, err := os.UserHomeDir(); err == nil {
		searchPaths = append(searchPaths,
			filepath.Join(home, "bin", "katago"),
			filepath.Join(home, ".local", "bin", "katago"),
			filepath.Join(home, "katago", "katago"),
		)
	}

	// Try each path
	for _, path := range searchPaths {
		if path == "" {
			continue
		}

		// If not absolute, try to find in PATH
		if !filepath.IsAbs(path) {
			if found, err := exec.LookPath(path); err == nil {
				path = found
			}
		}

		// Check if file exists and is executable
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			// On Windows, .exe files are always executable
			if runtime.GOOS == "windows" && strings.HasSuffix(path, ".exe") {
				return path, nil
			}
			// On Unix, check execute permission
			if runtime.GOOS != "windows" && info.Mode()&0o111 != 0 {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("not found in PATH or common locations. Install from https://github.com/lightvector/KataGo/releases")
}

func findKataGoModel() (string, error) {
	// Check environment variable first
	if modelPath := os.Getenv("KATAGO_MODEL_PATH"); modelPath != "" {
		if _, err := os.Stat(modelPath); err == nil {
			return modelPath, nil
		}
	}

	// Common model locations
	searchDirs := []string{}

	// Add home directory paths
	if home, err := os.UserHomeDir(); err == nil {
		searchDirs = append(searchDirs,
			filepath.Join(home, ".katago"),
			filepath.Join(home, ".katago", "models"),
			filepath.Join(home, "katago"),
			filepath.Join(home, "katago", "models"),
		)
	}

	// System paths
	searchDirs = append(searchDirs,
		"/usr/local/share/katago",
		"/usr/share/katago",
		"/opt/katago",
		"/opt/katago/models",
	)

	// Windows paths
	if runtime.GOOS == "windows" {
		searchDirs = append(searchDirs,
			"C:\\KataGo\\models",
			"C:\\Program Files\\KataGo\\models",
		)
	}

	// Look for model files
	modelExtensions := []string{".bin.gz", ".bin", ".txt.gz", ".txt"}

	for _, dir := range searchDirs {
		if _, err := os.Stat(dir); err != nil {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			for _, ext := range modelExtensions {
				if strings.HasSuffix(name, ext) && strings.Contains(name, "model") {
					return filepath.Join(dir, name), nil
				}
			}
		}
	}

	katagoHome := filepath.Join(getHomeDir(), ".katago")
	return "", fmt.Errorf("no model files found. Download from https://katagotraining.org/networks/ and place in %s", katagoHome)
}

func findOrGenerateConfig(binaryPath, modelPath string) (string, error) {
	// Check environment variable
	if configPath := os.Getenv("KATAGO_CONFIG_PATH"); configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	// Look for existing config
	configNames := []string{"analysis.cfg", "analysis_example.cfg", "gtp.cfg", "gtp_example.cfg"}
	searchDirs := []string{}

	if home, err := os.UserHomeDir(); err == nil {
		searchDirs = append(searchDirs,
			filepath.Join(home, ".katago"),
			filepath.Join(home, "katago"),
		)
	}

	searchDirs = append(searchDirs,
		"/usr/local/share/katago",
		"/usr/share/katago",
		"/etc/katago",
	)

	for _, dir := range searchDirs {
		for _, name := range configNames {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}

	// If we have binary and model, suggest generating config
	if binaryPath != "" && modelPath != "" {
		katagoHome := filepath.Join(getHomeDir(), ".katago")
		configPath := filepath.Join(katagoHome, "analysis.cfg")

		return "", fmt.Errorf("no config found. Generate one with: %s genconfig -model %s -output %s",
			binaryPath, modelPath, configPath)
	}

	return "", fmt.Errorf("no config found and cannot generate (missing binary or model)")
}

func getKataGoVersion(binaryPath string) (string, error) {
	cmd := exec.Command(binaryPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func GetInstallationInstructions() string {
	var instructions strings.Builder

	instructions.WriteString("KataGo Installation Instructions\n")
	instructions.WriteString("================================\n\n")

	switch runtime.GOOS {
	case "darwin":
		instructions.WriteString("macOS:\n")
		instructions.WriteString("  brew install katago\n")
		instructions.WriteString("  OR download from: https://github.com/lightvector/KataGo/releases\n\n")
	case "linux":
		instructions.WriteString("Linux:\n")
		instructions.WriteString("  Ubuntu/Debian: sudo apt install katago\n")
		instructions.WriteString("  OR download from: https://github.com/lightvector/KataGo/releases\n\n")
	case "windows":
		instructions.WriteString("Windows:\n")
		instructions.WriteString("  Download from: https://github.com/lightvector/KataGo/releases\n")
		instructions.WriteString("  Extract to C:\\KataGo\\ or C:\\Program Files\\KataGo\\\n\n")
	}

	instructions.WriteString("After installing KataGo:\n")
	instructions.WriteString("1. Download a neural network from: https://katagotraining.org/networks/\n")
	instructions.WriteString("2. Save it to ~/.katago/\n")
	instructions.WriteString("3. Generate config: katago genconfig -model <path-to-model> -output ~/.katago/analysis.cfg\n")
	instructions.WriteString("\nOr set environment variables:\n")
	instructions.WriteString("  export KATAGO_BINARY_PATH=/path/to/katago\n")
	instructions.WriteString("  export KATAGO_MODEL_PATH=/path/to/model.bin.gz\n")
	instructions.WriteString("  export KATAGO_CONFIG_PATH=/path/to/analysis.cfg\n")

	return instructions.String()
}
