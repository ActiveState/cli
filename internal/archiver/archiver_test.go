package archiver

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"
)

func TestSanitizeArchivePath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		expected    string
	}{
		// Valid paths
		{
			name:        "valid simple file",
			input:       "file.txt",
			expectError: false,
			expected:    "file.txt",
		},
		{
			name:        "valid nested file",
			input:       "dir/subdir/file.txt",
			expectError: false,
			expected:    "dir/subdir/file.txt",
		},
		{
			name:        "valid file with dots in name",
			input:       "file.backup.txt",
			expectError: false,
			expected:    "file.backup.txt",
		},
		{
			name:        "valid directory",
			input:       "dir/subdir/",
			expectError: false,
			expected:    "dir/subdir",
		},

		// Path traversal attacks
		{
			name:        "parent directory traversal",
			input:       "../file.txt",
			expectError: true,
		},
		{
			name:        "multiple parent directory traversal",
			input:       "../../file.txt",
			expectError: true,
		},
		{
			name:        "parent traversal in middle",
			input:       "dir/../file.txt",
			expectError: true,
		},
		{
			name:        "parent traversal at end",
			input:       "dir/..",
			expectError: true,
		},
		{
			name:        "parent traversal with file",
			input:       "dir/../other/file.txt",
			expectError: true,
		},

		// Absolute paths
		{
			name:        "absolute path unix",
			input:       "/etc/passwd",
			expectError: true,
		},
		{
			name:        "absolute path windows",
			input:       "C:\\Windows\\System32",
			expectError: true,
		},

		// Paths starting with separators
		{
			name:        "path starting with separator",
			input:       "/file.txt",
			expectError: true,
		},
		{
			name:        "path starting with backslash",
			input:       "\\file.txt",
			expectError: true,
		},

		// Empty and invalid paths
		{
			name:        "empty path",
			input:       "",
			expectError: true,
		},
		{
			name:        "current directory",
			input:       ".",
			expectError: false,
			expected:    ".",
		},
		{
			name:        "root directory",
			input:       "/",
			expectError: false,
			expected:    "/",
		},

		// Edge cases
		{
			name:        "path with only dots",
			input:       "...",
			expectError: false,
			expected:    "...",
		},
		{
			name:        "path with mixed separators",
			input:       "dir\\subdir/file.txt",
			expectError: false,
			expected:    "dir/subdir/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeArchivePath(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for input %q, but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFile_Name(t *testing.T) {
	tests := []struct {
		name     string
		header   interface{}
		expected string
	}{
		{
			name: "valid tar header",
			header: &tar.Header{
				Name: "dir/file.txt",
			},
			expected: "file.txt",
		},
		{
			name: "valid zip header",
			header: zip.FileHeader{
				Name: "dir/subdir/file.txt",
			},
			expected: "file.txt",
		},
		{
			name: "malicious tar header with path traversal",
			header: &tar.Header{
				Name: "../etc/passwd",
			},
			expected: "invalid_path",
		},
		{
			name: "malicious zip header with path traversal",
			header: zip.FileHeader{
				Name: "../../../etc/passwd",
			},
			expected: "invalid_path",
		},
		{
			name:     "unknown header type",
			header:   "not a header",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := File{
				Header: tt.header,
			}

			result := file.Name()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFile_FullPath(t *testing.T) {
	tests := []struct {
		name        string
		header      interface{}
		expectError bool
		expected    string
	}{
		{
			name: "valid tar header",
			header: &tar.Header{
				Name: "dir/file.txt",
			},
			expectError: false,
			expected:    "dir/file.txt",
		},
		{
			name: "valid zip header",
			header: zip.FileHeader{
				Name: "dir/subdir/file.txt",
			},
			expectError: false,
			expected:    "dir/subdir/file.txt",
		},
		{
			name: "malicious tar header with path traversal",
			header: &tar.Header{
				Name: "../etc/passwd",
			},
			expectError: true,
		},
		{
			name: "malicious zip header with path traversal",
			header: zip.FileHeader{
				Name: "../../../etc/passwd",
			},
			expectError: true,
		},
		{
			name:        "unknown header type",
			header:      "not a header",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := File{
				Header: tt.header,
			}

			result, err := file.FullPath()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestZip_Read_WithMaliciousEntries(t *testing.T) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Add a valid file
	validFile, err := zipWriter.Create("valid/file.txt")
	if err != nil {
		t.Fatalf("failed to create valid file: %v", err)
	}
	validFile.Write([]byte("valid content"))

	maliciousFile, err := zipWriter.Create("../etc/passwd")
	if err != nil {
		t.Fatalf("failed to create malicious file: %v", err)
	}
	maliciousFile.Write([]byte("malicious content"))

	maliciousFile2, err := zipWriter.Create("dir/../../sensitive.txt")
	if err != nil {
		t.Fatalf("failed to create malicious file 2: %v", err)
	}
	maliciousFile2.Write([]byte("more malicious content"))

	zipWriter.Close()

	zipReader := NewZip()
	err = zipReader.Open(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer zipReader.Close()

	file, err := zipReader.Read()
	if err != nil {
		t.Fatalf("failed to read first file: %v", err)
	}

	if file.Name() != "file.txt" {
		t.Errorf("expected first file to be 'file.txt', got %q", file.Name())
	}

	_, err = zipReader.Read()
	if err == nil {
		t.Error("expected error when reading malicious file, but got none")
	}
	if !strings.Contains(err.Error(), "invalid file path in archive") {
		t.Errorf("expected path validation error, got: %v", err)
	}

	_, err = zipReader.Read()
	if err == nil {
		t.Error("expected error when reading second malicious file, but got none")
	}
	if !strings.Contains(err.Error(), "invalid file path in archive") {
		t.Errorf("expected path validation error, got: %v", err)
	}
}

func TestTarGz_Read_WithMaliciousEntries(t *testing.T) {
	var buf bytes.Buffer

	var tarBuf bytes.Buffer
	tarWriter := tar.NewWriter(&tarBuf)

	validHeader := &tar.Header{
		Name: "valid/file.txt",
		Size: 12,
		Mode: 0644,
	}
	tarWriter.WriteHeader(validHeader)
	tarWriter.Write([]byte("valid content"))

	maliciousHeader := &tar.Header{
		Name: "../etc/passwd",
		Size: 16,
		Mode: 0644,
	}
	tarWriter.WriteHeader(maliciousHeader)
	tarWriter.Write([]byte("malicious content"))

	tarWriter.Close()

	gzWriter := gzip.NewWriter(&buf)
	gzWriter.Write(tarBuf.Bytes())
	gzWriter.Close()

	tarGzReader := NewTarGz()
	err := tarGzReader.Open(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("failed to open tar: %v", err)
	}
	defer tarGzReader.Close()

	file, err := tarGzReader.Read()
	if err != nil {
		t.Fatalf("failed to read first file: %v", err)
	}

	if file.Name() != "file.txt" {
		t.Errorf("expected first file to be 'file.txt', got %q", file.Name())
	}

	_, err = tarGzReader.Read()
	if err == nil {
		t.Error("expected error when reading malicious file, but got none")
	}
	if !strings.Contains(err.Error(), "invalid file path in archive") {
		t.Errorf("expected path validation error, got: %v", err)
	}
}

func TestZip_Read_ValidEntries(t *testing.T) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	files := []string{
		"file1.txt",
		"dir/file2.txt",
		"dir/subdir/file3.txt",
		"file.backup.txt",
	}

	for _, filename := range files {
		file, err := zipWriter.Create(filename)
		if err != nil {
			t.Fatalf("failed to create file %s: %v", filename, err)
		}
		file.Write([]byte("content for " + filename))
	}

	zipWriter.Close()

	zipReader := NewZip()
	err := zipReader.Open(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer zipReader.Close()

	readFiles := make([]string, 0, len(files))
	for {
		file, err := zipReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		readFiles = append(readFiles, file.Name())
	}

	expectedFiles := []string{"file1.txt", "file2.txt", "file3.txt", "file.backup.txt"}
	if len(readFiles) != len(expectedFiles) {
		t.Errorf("expected %d files, got %d", len(expectedFiles), len(readFiles))
	}

	for _, expected := range expectedFiles {
		found := false
		for _, actual := range readFiles {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected file %s not found in read files", expected)
		}
	}
}

func TestSanitizeArchivePath_EdgeCases(t *testing.T) {
	edgeCases := []struct {
		name        string
		input       string
		expectError bool
	}{
		{"unicode path", "файл.txt", false},
		{"path with spaces", "file with spaces.txt", false},
		{"path with special chars", "file!@#$%^&*().txt", false},
		{"very long path", strings.Repeat("dir/", 100) + "file.txt", false},
		{"path with null bytes", "file\x00.txt", true},        // null bytes should be rejected
		{"path with control chars", "file\x01\x02.txt", true}, // control chars should be rejected
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := sanitizeArchivePath(tc.input)
			if tc.expectError && err == nil {
				t.Errorf("expected error for %q, but got none", tc.input)
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error for %q: %v", tc.input, err)
			}
		})
	}
}
