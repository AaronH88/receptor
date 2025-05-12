package types

import (
	"os"
	"testing"
	"time"
)

func TestMainInitNodeID(t *testing.T) {
	mainInitNodeIDTestCases := []struct {
		name        string
		nodeID      string
		expectedErr string
	}{
		{
			name:        "successful, no error",
			nodeID:      "t.e-s_t@1:234",
			expectedErr: "",
		},
		{
			name:        "failed, charactered not allowed",
			nodeID:      "test!#&123",
			expectedErr: "node id can only contain a-z, A-Z, 0-9 or special characters . - _ @ : but received: test!#&123",
		},
		{
			name:        "localhost not allowed",
			nodeID:      "localhost",
			expectedErr: "node ID \"localhost\" is reserved",
		},
	}

	for _, testCase := range mainInitNodeIDTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := NodeCfg{
				ID:      testCase.nodeID,
				DataDir: t.TempDir(),
			}
			err := cfg.Init()
			if err == nil && testCase.expectedErr != "" {
				t.Errorf("exected error but got no error")
			} else if err != nil && testCase.expectedErr == "" {
				t.Errorf("expected no error but got: %s", err.Error())
			} else if err != nil && err.Error() != testCase.expectedErr {
				t.Errorf("expected error to be %s, but got: %s", testCase.expectedErr, err.Error())
			}
			t.Cleanup(func() {
				cfg = NodeCfg{}
			})
		})
	}
}

func TestNodeCfgRun(t *testing.T) {
	// This is a simple test since the Run method doesn't do much
	cfg := NodeCfg{
		ID:      "test-node",
		DataDir: t.TempDir(),
	}

	// Initialize the node first
	err := cfg.Init()
	if err != nil {
		t.Fatalf("Failed to initialize node: %v", err)
	}

	// Test the Run method
	err = cfg.Run()
	if err != nil {
		t.Errorf("Expected no error from Run() but got: %v", err)
	}
}

func TestGetUploadRate(t *testing.T) {
	tests := []struct {
		name     string
		cfg      ReceptorPyroscopeCfg
		expected time.Duration
	}{
		{
			name: "Default upload rate",
			cfg: ReceptorPyroscopeCfg{
				UploadRate: "",
			},
			expected: 15 * time.Second,
		},
		{
			name: "Custom upload rate",
			cfg: ReceptorPyroscopeCfg{
				UploadRate: "uploadRate: 30s",
			},
			expected: 30 * time.Second,
		},
		{
			name: "Invalid upload rate",
			cfg: ReceptorPyroscopeCfg{
				UploadRate: "invalid",
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getUploadRate(tt.cfg)
			if result != tt.expected {
				t.Errorf("getUploadRate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetProfileTypes(t *testing.T) {
	tests := []struct {
		name     string
		cfg      ReceptorPyroscopeCfg
		expected int
	}{
		{
			name: "Default profile types",
			cfg: ReceptorPyroscopeCfg{
				ProfileTypes: []string{},
			},
			expected: 5, // Default has 5 profile types
		},
		{
			name: "Custom profile types",
			cfg: ReceptorPyroscopeCfg{
				ProfileTypes: []string{"ProfileGoroutines", "ProfileMutexCount"},
			},
			expected: 7, // 5 default + 2 custom
		},
		{
			name: "All profile types",
			cfg: ReceptorPyroscopeCfg{
				ProfileTypes: []string{
					"ProfileGoroutines",
					"ProfileMutexCount",
					"ProfileMutexDuration",
					"ProfileBlockCount",
					"ProfileBlockDuration",
				},
			},
			expected: 10, // 5 default + 5 custom
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getProfileTypes(tt.cfg)
			if len(result) != tt.expected {
				t.Errorf("getProfileTypes() returned %d profile types, want %d", len(result), tt.expected)
			}
		})
	}
}

func TestReceptorPyroscopeCfgInit(t *testing.T) {
	// Set up a temporary directory for testing
	tempDir := t.TempDir()
	receptorDataDir = tempDir

	tests := []struct {
		name      string
		cfg       ReceptorPyroscopeCfg
		expectErr bool
	}{
		{
			name: "Empty application name",
			cfg: ReceptorPyroscopeCfg{
				ApplicationName: "",
			},
			expectErr: false,
		},
		// Pyroscope doesn't validate the server address at initialization time,
		// so we can't test for an error with an invalid URL
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Init()
			if (err != nil) != tt.expectErr {
				t.Errorf("ReceptorPyroscopeCfg.Init() error = %v, expectErr %v", err, tt.expectErr)
			}

			// Clean up any log files created
			logFile := tempDir + "/pyroscope.log"
			if _, err := os.Stat(logFile); err == nil {
				os.Remove(logFile)
			}
		})
	}
}
