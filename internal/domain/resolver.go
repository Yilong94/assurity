package domain

import (
	"fmt"
	"time"
)

const (
	defaultInterval = 30 * time.Second
	defaultTimeout  = 15 * time.Second
	defaultRetries  = 0

	minInterval = time.Second
	minTimeout  = time.Second
	maxRetries  = 20
)

// ResolveServiceDefinitions applies defaults and validates raw file config.
func ResolveServiceDefinitions(fc *RawFileConfig) ([]ServiceDefinition, error) {
	out := make([]ServiceDefinition, 0, len(fc.Services))
	for i, s := range fc.Services {
		if s.Name == "" {
			return nil, fmt.Errorf("services[%d]: name is required", i)
		}
		if s.Endpoint == "" {
			return nil, fmt.Errorf("services[%d]: endpoint is required", i)
		}
		rs, err := resolveOne(i, s)
		if err != nil {
			return nil, err
		}
		out = append(out, rs)
	}
	return out, nil
}

func resolveOne(i int, s RawServiceConfig) (ServiceDefinition, error) {
	interval := defaultInterval
	if s.Interval != "" {
		d, err := time.ParseDuration(s.Interval)
		if err != nil {
			return ServiceDefinition{}, fmt.Errorf("services[%d] interval: %w", i, err)
		}
		if d < minInterval {
			return ServiceDefinition{}, fmt.Errorf("services[%d]: interval must be at least %v", i, minInterval)
		}
		interval = d
	}

	timeout := defaultTimeout
	if s.Timeout != "" {
		d, err := time.ParseDuration(s.Timeout)
		if err != nil {
			return ServiceDefinition{}, fmt.Errorf("services[%d] timeout: %w", i, err)
		}
		if d < minTimeout {
			return ServiceDefinition{}, fmt.Errorf("services[%d]: timeout must be at least %v", i, minTimeout)
		}
		timeout = d
	}

	retries := defaultRetries
	if s.Retries != nil {
		retries = *s.Retries
		if retries < 0 || retries > maxRetries {
			return ServiceDefinition{}, fmt.Errorf("services[%d]: retries must be between 0 and %d", i, maxRetries)
		}
	}

	return ServiceDefinition{
		Name:               s.Name,
		Endpoint:           s.Endpoint,
		IntervalSeconds:    int(interval / time.Second),
		TimeoutSeconds:     int(timeout / time.Second),
		ExtraRetryAttempts: retries,
	}, nil
}
