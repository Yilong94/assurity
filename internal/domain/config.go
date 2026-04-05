package domain

// RawServiceConfig is one entry from an external configuration file (e.g. YAML).
type RawServiceConfig struct {
	Name     string `yaml:"name"`
	Endpoint string `yaml:"endpoint"`
	Interval string `yaml:"interval,omitempty"`
	Timeout  string `yaml:"timeout,omitempty"`
	Retries  *int   `yaml:"retries,omitempty"`
}

// RawFileConfig is the root structure of a services configuration file.
type RawFileConfig struct {
	Services []RawServiceConfig `yaml:"services"`
}
