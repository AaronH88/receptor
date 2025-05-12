package utils

import (
	"bufio"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestReadStringContext(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		delim       byte
		timeout     time.Duration
		expected    string
		expectError bool
	}{
		{
			name:        "Simple string with newline",
			input:       "hello\nworld",
			delim:       '\n',
			timeout:     time.Second,
			expected:    "hello",
			expectError: false,
		},
		{
			name:        "String without delimiter",
			input:       "hello world",
			delim:       '\n',
			timeout:     time.Second,
			expected:    "",
			expectError: true, // EOF error
		},
		{
			name:        "Empty string",
			input:       "",
			delim:       '\n',
			timeout:     time.Second,
			expected:    "",
			expectError: true, // EOF error
		},
		{
			name:        "Multiple delimiters",
			input:       "hello\nworld\n",
			delim:       '\n',
			timeout:     time.Second,
			expected:    "hello",
			expectError: false,
		},
		{
			name:        "Context timeout",
			input:       "hello world", // No delimiter to force timeout
			delim:       '\n',
			timeout:     10 * time.Millisecond,
			expected:    "",
			expectError: true, // Context deadline exceeded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a reader with the test input
			reader := strings.NewReader(tt.input)
			bufReader := bufio.NewReader(reader)

			// Create a context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			// Call the function
			result, err := ReadStringContext(ctx, bufReader, tt.delim)

			// Check the result
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				// ReadString includes the delimiter in the result
				expected := tt.expected
				if tt.delim == '\n' {
					expected += "\n"
				}
				if result != expected {
					t.Errorf("Expected result %q but got %q", expected, result)
				}
			}
		})
	}
}

func TestReadStringContextCancelled(t *testing.T) {
	// Create a reader that will block
	pipeReader, pipeWriter := io.Pipe()
	bufReader := bufio.NewReader(pipeReader)

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Start the read in a goroutine
	resultCh := make(chan string)
	errCh := make(chan error)
	go func() {
		result, err := ReadStringContext(ctx, bufReader, '\n')
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	// Cancel the context
	cancel()

	// Check that we get a context canceled error
	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled but got: %v", err)
		}
	case result := <-resultCh:
		t.Errorf("Expected error but got result: %q", result)
	case <-time.After(time.Second):
		t.Errorf("Test timed out")
	}

	// Clean up
	pipeWriter.Close()
	pipeReader.Close()
}

func TestReadStringContextSuccess(t *testing.T) {
	// Create a pipe
	pipeReader, pipeWriter := io.Pipe()
	bufReader := bufio.NewReader(pipeReader)

	// Create a context
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Start the read in a goroutine
	resultCh := make(chan string)
	errCh := make(chan error)
	go func() {
		result, err := ReadStringContext(ctx, bufReader, '\n')
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	// Write data to the pipe
	go func() {
		time.Sleep(10 * time.Millisecond) // Small delay to ensure reader is waiting
		pipeWriter.Write([]byte("hello\n"))
	}()

	// Check that we get the expected result
	select {
	case err := <-errCh:
		t.Errorf("Expected success but got error: %v", err)
	case result := <-resultCh:
		// ReadString includes the delimiter in the result
		if result != "hello\n" {
			t.Errorf("Expected 'hello\\n' but got: %q", result)
		}
	case <-time.After(time.Second):
		t.Errorf("Test timed out")
	}

	// Clean up
	pipeWriter.Close()
	pipeReader.Close()
}
