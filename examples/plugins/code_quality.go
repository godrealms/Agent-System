package main

import (
	"AI-agent/pkg/plugins"
	"fmt"
	"strings"
)

// CodeQualityPlugin analyzes code quality
type CodeQualityPlugin struct {
	name    string
	version string
	config  map[string]interface{}
}

func (p *CodeQualityPlugin) GetName() string {
	return "code-quality-analyzer"
}

func (p *CodeQualityPlugin) GetType() plugins.PluginType {
	return plugins.PluginTypeValidator
}

func (p *CodeQualityPlugin) GetVersion() string {
	return "1.0.0"
}

func (p *CodeQualityPlugin) Initialize(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (p *CodeQualityPlugin) Execute(data interface{}) (interface{}, error) {
	if code, ok := data.(string); ok {
		issues := p.analyzeCodeQuality(code)
		return map[string]interface{}{
			"quality_score": p.calculateScore(issues),
			"issues":        issues,
		}, nil
	}
	return nil, fmt.Errorf("invalid data type")
}

func (p *CodeQualityPlugin) Cleanup() error {
	return nil
}

func (p *CodeQualityPlugin) analyzeCodeQuality(code string) []string {
	var issues []string

	// Check for long lines
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if len(line) > 120 {
			issues = append(issues, fmt.Sprintf("Line %d: Line too long (%d chars)", i+1, len(line)))
		}
	}

	// Check for TODO comments
	if strings.Contains(strings.ToLower(code), "todo") {
		issues = append(issues, "TODO comments found")
	}

	// Check for commented out code
	if strings.Contains(code, "// ") && strings.Contains(code, "=") {
		issues = append(issues, "Potential commented out code found")
	}

	return issues
}

func (p *CodeQualityPlugin) calculateScore(issues []string) float64 {
	// Simple scoring: 100 - (number of issues * 10)
	score := 100.0 - float64(len(issues)*10)
	if score < 0 {
		score = 0
	}
	return score
}

// Plugin exports the plugin instance for use as a Go plugin (.so)
var Plugin plugins.Plugin = &CodeQualityPlugin{
	name:    "code-quality-analyzer",
	version: "1.0.0",
}

func main() {
	// Entry point when built as a standalone binary (not a .so plugin)
	fmt.Println("CodeQualityPlugin v1.0.0 — build with -buildmode=plugin to use as a plugin")
}
