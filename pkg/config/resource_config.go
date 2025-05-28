package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	log "github.com/jonkeyguan/vfio-device-plugin/pkg/log"
	"gopkg.in/yaml.v2"
)

const (
	ConfigFilePath = "/etc/vfio/config.yaml"
	// ConfigFilePath = "/root/config.yaml"
)

type ResourceConfig struct {
	config *Config
}

// Config structure representing the root of the configuration file
type Config struct {
	Resources []Resource `yaml:"resources"` // List of resources
}

// Resource structure representing each resource in the configuration
type Resource struct {
	Name      string   `yaml:"resourceName"` // Name of the resource
	Addresses []string `yaml:"addresses"`    // List of device addresses for the resource
}

func NewResourceConfig() (*ResourceConfig, error) {
	logger := log.DefaultLogger()
	config, err := readConfig(ConfigFilePath)

	if err != nil {
		logger.Reason(err).Error("Error reading config file")
		return nil, err
	}

	logger.Infof("Config file loaded successfully: %s", ConfigFilePath)
	logger.Infof("Resources: %v", config.Resources)

	resourceConfig := &ResourceConfig{
		config: config,
	}

	return resourceConfig, nil
}

func (c *ResourceConfig) GetResources() []Resource {
	return c.config.Resources
}

// readConfig function to read and parse the YAML configuration file
func readConfig(filePath string) (*Config, error) {
	// Read the YAML file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Define a Config variable to hold the parsed data
	var config Config

	// Unmarshal the YAML data into the Config structure
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	// Process each resource's addresses to expand ranges into actual device addresses
	for i, resource := range config.Resources {
		var expandedAddresses []string
		for _, address := range resource.Addresses {
			expandedAddresses = append(expandedAddresses, parseDeviceAddress(address)...)
		}
		// Update the resource's addresses with the expanded list
		config.Resources[i].Addresses = expandedAddresses
	}

	return &config, nil
}

// parseDeviceAddress function to parse a device address like "0000:86:00.0#0-1,3,4" into multiple addresses
func parseDeviceAddress(device string) []string {
	log := log.DefaultLogger()

	// Split the device address into base device address and the range part (e.g., 0000:86:00.0 and 0-1,3,4)
	parts := strings.Split(device, "#")
	deviceAddress := parts[0] // The base device address (e.g., "0000:86:00.0")

	// Remove the trailing 0 (keep the dot)
	re := regexp.MustCompile(`0$`)
	baseAddress := re.ReplaceAllString(deviceAddress, "") // e.g., "0000:86:00."

	// If there is no range part or it's empty, just return the base address
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		return []string{baseAddress}
	}

	rangePart := parts[1] // The range part (e.g., "0-1,3,4")
	// Split the range part into individual ranges (e.g., "0-1", "3", "4")
	ranges := strings.Split(rangePart, ",")
	var addresses []string

	// Iterate over each range and expand it into individual addresses
	for _, r := range ranges {
		// If the range is in the format "start-end" (e.g., "0-1"), we need to generate a range of addresses
		if strings.Contains(r, "-") {
			rangeBounds := strings.Split(r, "-")
			start, err := strconv.Atoi(rangeBounds[0]) // Start of the range
			if err != nil {
				log.Reason(err).Error("Error parsing range start")
			}
			end, err := strconv.Atoi(rangeBounds[1]) // End of the range
			if err != nil {
				log.Reason(err).Error("Error parsing range end")
			}
			// Add all addresses in the range from start to end
			for i := start; i <= end; i++ {
				addresses = append(addresses, fmt.Sprintf("%s%d", baseAddress, i))
			}
		} else {
			// If it's a single address (e.g., "3"), just add it directly
			addresses = append(addresses, fmt.Sprintf("%s%s", baseAddress, r))
		}
	}

	return addresses
}
