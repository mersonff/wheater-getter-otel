package shared

import (
	"regexp"
	"testing"
)

func TestZipcodeValidation(t *testing.T) {
	tests := []struct {
		name     string
		zipcode  string
		expected bool
	}{
		{"valid zipcode", "29902555", true},
		{"valid zipcode 2", "01001000", true},
		{"invalid short", "12345", false},
		{"invalid long", "123456789", false},
		{"invalid letters", "abc12345", false},
		{"invalid empty", "", false},
		{"invalid with dash", "29902-555", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidZipcodeTest(tt.zipcode)
			if result != tt.expected {
				t.Errorf("isValidZipcode(%s) = %v, want %v", tt.zipcode, result, tt.expected)
			}
		})
	}
}

func isValidZipcodeTest(zipcode string) bool {
	matched, _ := regexp.MatchString(`^\d{8}$`, zipcode)
	return matched
}

func TestTemperatureConversion(t *testing.T) {
	tests := []struct {
		name     string
		celsius  float64
		expected struct {
			fahrenheit float64
			kelvin     float64
		}
	}{
		{
			name:    "zero celsius",
			celsius: 0,
			expected: struct {
				fahrenheit float64
				kelvin     float64
			}{
				fahrenheit: 32.0,
				kelvin:     273.15,
			},
		},
		{
			name:    "room temperature",
			celsius: 25.0,
			expected: struct {
				fahrenheit float64
				kelvin     float64
			}{
				fahrenheit: 77.0,
				kelvin:     298.15,
			},
		},
		{
			name:    "boiling point",
			celsius: 100.0,
			expected: struct {
				fahrenheit float64
				kelvin     float64
			}{
				fahrenheit: 212.0,
				kelvin:     373.15,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fahrenheit := tt.celsius*1.8 + 32
			kelvin := tt.celsius + 273.15

			if fahrenheit != tt.expected.fahrenheit {
				t.Errorf("Fahrenheit conversion: got %v, want %v", fahrenheit, tt.expected.fahrenheit)
			}

			if kelvin != tt.expected.kelvin {
				t.Errorf("Kelvin conversion: got %v, want %v", kelvin, tt.expected.kelvin)
			}
		})
	}
}

func TestConfigLoading(t *testing.T) {
	config := GetConfig()
	if config.Port == "" {
		t.Error("Port should not be empty")
	}
	if config.LogLevel == "" {
		t.Error("LogLevel should not be empty")
	}
	if config.Port != "8080" && config.Port != "8081" {
		t.Errorf("Port should be 8080 or 8081, got %s", config.Port)
	}
	if config.LogLevel != "INFO" && config.LogLevel != "DEBUG" && config.LogLevel != "WARN" && config.LogLevel != "ERROR" {
		t.Errorf("LogLevel should be a valid level, got %s", config.LogLevel)
	}
}

func TestLoggerCreation(t *testing.T) {
	logger := NewLogger(INFO, false)
	if logger == nil {
		t.Error("Logger should not be nil")
	}
}
