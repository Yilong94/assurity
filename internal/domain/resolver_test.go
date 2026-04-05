package domain

import (
	"strings"
	"testing"
)

func intPtr(n int) *int { return &n }

func TestResolveServiceDefinitions_empty(t *testing.T) {
	out, err := ResolveServiceDefinitions(&RawFileConfig{Services: nil})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("len = %d", len(out))
	}
}

func TestResolveServiceDefinitions_defaults(t *testing.T) {
	out, err := ResolveServiceDefinitions(&RawFileConfig{
		Services: []RawServiceConfig{{
			Name:     "a",
			Endpoint: "https://a",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatal(len(out))
	}
	s := out[0]
	if s.Name != "a" || s.Endpoint != "https://a" {
		t.Fatalf("%+v", s)
	}
	if s.IntervalSeconds != 30 || s.TimeoutSeconds != 15 || s.ExtraRetryAttempts != 0 {
		t.Fatalf("defaults: %+v", s)
	}
}

func TestResolveServiceDefinitions_full(t *testing.T) {
	out, err := ResolveServiceDefinitions(&RawFileConfig{
		Services: []RawServiceConfig{{
			Name:     "x",
			Endpoint: "https://x",
			Interval: "2m",
			Timeout:  "20s",
			Retries:  intPtr(3),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := out[0]
	if s.IntervalSeconds != 120 || s.TimeoutSeconds != 20 || s.ExtraRetryAttempts != 3 {
		t.Fatalf("%+v", s)
	}
}

func TestResolveServiceDefinitions_missingName(t *testing.T) {
	_, err := ResolveServiceDefinitions(&RawFileConfig{
		Services: []RawServiceConfig{{Endpoint: "https://a"}},
	})
	if err == nil || !strings.Contains(err.Error(), "name is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveServiceDefinitions_missingEndpoint(t *testing.T) {
	_, err := ResolveServiceDefinitions(&RawFileConfig{
		Services: []RawServiceConfig{{Name: "a"}},
	})
	if err == nil || !strings.Contains(err.Error(), "endpoint is required") {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveServiceDefinitions_badInterval(t *testing.T) {
	_, err := ResolveServiceDefinitions(&RawFileConfig{
		Services: []RawServiceConfig{{
			Name:     "a",
			Endpoint: "https://a",
			Interval: "not-a-duration",
		}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveServiceDefinitions_intervalTooShort(t *testing.T) {
	_, err := ResolveServiceDefinitions(&RawFileConfig{
		Services: []RawServiceConfig{{
			Name:     "a",
			Endpoint: "https://a",
			Interval: "500ms",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "at least") {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveServiceDefinitions_timeoutTooShort(t *testing.T) {
	_, err := ResolveServiceDefinitions(&RawFileConfig{
		Services: []RawServiceConfig{{
			Name:     "a",
			Endpoint: "https://a",
			Timeout:  "1ms",
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "timeout must be at least") {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveServiceDefinitions_retriesOutOfRange(t *testing.T) {
	_, err := ResolveServiceDefinitions(&RawFileConfig{
		Services: []RawServiceConfig{{
			Name:     "a",
			Endpoint: "https://a",
			Retries:  intPtr(21),
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "retries must be between") {
		t.Fatalf("err = %v", err)
	}
	_, err = ResolveServiceDefinitions(&RawFileConfig{
		Services: []RawServiceConfig{{
			Name:     "a",
			Endpoint: "https://a",
			Retries:  intPtr(-1),
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "retries must be between") {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveServiceDefinitions_multipleServices(t *testing.T) {
	out, err := ResolveServiceDefinitions(&RawFileConfig{
		Services: []RawServiceConfig{
			{Name: "a", Endpoint: "https://a"},
			{Name: "b", Endpoint: "https://b", Interval: "60s"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].Name != "a" || out[1].IntervalSeconds != 60 {
		t.Fatalf("%+v", out)
	}
}
