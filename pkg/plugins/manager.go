package plugins

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"sync"
)

// PluginType represents different types of plugins
type PluginType string

const (
	PluginTypePreProcessor  PluginType = "pre_processor"
	PluginTypePostProcessor PluginType = "post_processor"
	PluginTypeTool          PluginType = "tool"
	PluginTypeValidator     PluginType = "validator"
)

// Plugin defines the interface that all plugins must implement
type Plugin interface {
	// GetName returns the plugin name
	GetName() string

	// GetType returns the plugin type
	GetType() PluginType

	// GetVersion returns the plugin version
	GetVersion() string

	// Initialize initializes the plugin with configuration
	Initialize(config map[string]interface{}) error

	// Execute executes the plugin functionality
	Execute(data interface{}) (interface{}, error)

	// Cleanup performs cleanup operations
	Cleanup() error
}

// PluginManager manages plugin loading and execution
type PluginManager struct {
	pluginDir string
	plugins   map[string]Plugin
	pluginMu  sync.RWMutex
	config    map[string]interface{}
}

// PluginInfo holds metadata about a plugin
type PluginInfo struct {
	Name    string                 `json:"name"`
	Type    PluginType             `json:"type"`
	Version string                 `json:"version"`
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(pluginDir string) *PluginManager {
	return &PluginManager{
		pluginDir: pluginDir,
		plugins:   make(map[string]Plugin),
		config:    make(map[string]interface{}),
	}
}

// LoadPlugins loads all plugins from the plugin directory
func (pm *PluginManager) LoadPlugins() error {
	pm.pluginMu.Lock()
	defer pm.pluginMu.Unlock()

	// Create plugin directory if it doesn't exist
	if err := os.MkdirAll(pm.pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Walk through plugin directory
	return filepath.Walk(pm.pluginDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .so files (compiled Go plugins)
		if filepath.Ext(path) != ".so" {
			return nil
		}

		// Load the plugin
		plug, err := plugin.Open(path)
		if err != nil {
			log.Printf("Warning: Failed to load plugin %s: %v", path, err)
			return nil
		}

		// Look for Plugin symbol
		symbol, err := plug.Lookup("Plugin")
		if err != nil {
			log.Printf("Warning: Plugin %s doesn't export Plugin symbol", path)
			return nil
		}

		// Type assert to Plugin interface
		pluginInstance, ok := symbol.(Plugin)
		if !ok {
			log.Printf("Warning: Plugin %s doesn't implement Plugin interface", path)
			return nil
		}

		// Initialize the plugin
		pluginName := pluginInstance.GetName()
		pluginConfig := pm.getPluginConfig(pluginName)

		if err := pluginInstance.Initialize(pluginConfig); err != nil {
			log.Printf("Warning: Failed to initialize plugin %s: %v", pluginName, err)
			return nil
		}

		// Store the plugin
		pm.plugins[pluginName] = pluginInstance
		log.Printf("Loaded plugin: %s (v%s)", pluginName, pluginInstance.GetVersion())

		return nil
	})
}

// GetPlugin returns a plugin by name
func (pm *PluginManager) GetPlugin(name string) (Plugin, bool) {
	pm.pluginMu.RLock()
	defer pm.pluginMu.RUnlock()

	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// ListPlugins returns information about all loaded plugins
func (pm *PluginManager) ListPlugins() []PluginInfo {
	pm.pluginMu.RLock()
	defer pm.pluginMu.RUnlock()

	var plugins []PluginInfo
	for name, plugin := range pm.plugins {
		plugins = append(plugins, PluginInfo{
			Name:    name,
			Type:    plugin.GetType(),
			Version: plugin.GetVersion(),
			Enabled: true,
			Config:  pm.getPluginConfig(name),
		})
	}

	return plugins
}

// ExecutePluginsOfType executes all plugins of a specific type
func (pm *PluginManager) ExecutePluginsOfType(pluginType PluginType, data interface{}) ([]interface{}, error) {
	pm.pluginMu.RLock()
	defer pm.pluginMu.RUnlock()

	var results []interface{}

	for _, plugin := range pm.plugins {
		if plugin.GetType() == pluginType {
			result, err := plugin.Execute(data)
			if err != nil {
				log.Printf("Plugin %s execution failed: %v", plugin.GetName(), err)
				continue
			}
			results = append(results, result)
		}
	}

	return results, nil
}

// ExecutePreProcessors runs all pre-processor plugins
func (pm *PluginManager) ExecutePreProcessors(data interface{}) (interface{}, error) {
	_, err := pm.ExecutePluginsOfType(PluginTypePreProcessor, data)
	if err != nil {
		return nil, err
	}

	// For simplicity, return the original data plus any modifications
	// In a real implementation, you'd want more sophisticated data merging
	return data, nil
}

// ExecutePostProcessors runs all post-processor plugins
func (pm *PluginManager) ExecutePostProcessors(data interface{}) (interface{}, error) {
	_, err := pm.ExecutePluginsOfType(PluginTypePostProcessor, data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// ExecuteValidators runs all validator plugins
func (pm *PluginManager) ExecuteValidators(data interface{}) (bool, []string) {
	results, err := pm.ExecutePluginsOfType(PluginTypeValidator, data)
	if err != nil {
		return false, []string{err.Error()}
	}

	var errors []string
	allValid := true

	for _, result := range results {
		if valid, ok := result.(bool); !ok || !valid {
			allValid = false
			errors = append(errors, fmt.Sprintf("validation failed: %v", result))
		}
	}

	return allValid, errors
}

// getPluginConfig retrieves configuration for a specific plugin
func (pm *PluginManager) getPluginConfig(pluginName string) map[string]interface{} {
	// In a real implementation, this would load from a config file
	// For now, return empty config
	return make(map[string]interface{})
}

// Cleanup unloads all plugins and performs cleanup
func (pm *PluginManager) Cleanup() error {
	pm.pluginMu.Lock()
	defer pm.pluginMu.Unlock()

	var errors []error

	for name, plugin := range pm.plugins {
		if err := plugin.Cleanup(); err != nil {
			errors = append(errors, fmt.Errorf("failed to cleanup plugin %s: %w", name, err))
		}
	}

	// Clear plugin map
	pm.plugins = make(map[string]Plugin)

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// Sample plugin implementations for demonstration

// CodeFormatterPlugin formats code
type CodeFormatterPlugin struct {
	name    string
	version string
	config  map[string]interface{}
}

func (p *CodeFormatterPlugin) GetName() string {
	return "code-formatter"
}

func (p *CodeFormatterPlugin) GetType() PluginType {
	return PluginTypePostProcessor
}

func (p *CodeFormatterPlugin) GetVersion() string {
	return "1.0.0"
}

func (p *CodeFormatterPlugin) Initialize(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (p *CodeFormatterPlugin) Execute(data interface{}) (interface{}, error) {
	// Simple code formatting logic
	if code, ok := data.(string); ok {
		// In reality, this would use a proper code formatter
		return fmt.Sprintf("// Formatted code:\n%s", code), nil
	}
	return data, nil
}

func (p *CodeFormatterPlugin) Cleanup() error {
	return nil
}

// SecurityScannerPlugin scans for security issues
type SecurityScannerPlugin struct {
	name    string
	version string
	config  map[string]interface{}
}

func (p *SecurityScannerPlugin) GetName() string {
	return "security-scanner"
}

func (p *SecurityScannerPlugin) GetType() PluginType {
	return PluginTypeValidator
}

func (p *SecurityScannerPlugin) GetVersion() string {
	return "1.0.0"
}

func (p *SecurityScannerPlugin) Initialize(config map[string]interface{}) error {
	p.config = config
	return nil
}

func (p *SecurityScannerPlugin) Execute(data interface{}) (interface{}, error) {
	// Simple security scanning logic
	if code, ok := data.(string); ok {
		// Check for common security issues
		if contains(code, "eval(") || contains(code, "exec(") {
			return false, nil
		}
		return true, nil
	}
	return true, nil
}

func (p *SecurityScannerPlugin) Cleanup() error {
	return nil
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || contains(s[1:], substr)))
}
