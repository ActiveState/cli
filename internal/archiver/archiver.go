// Package archiver provides archive functionality using the modern archives library
package archiver

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/mholt/archives"
)

// sanitizeArchivePath validates and sanitizes archive entry paths to prevent path traversal attacks
func sanitizeArchivePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path not allowed")
	}

	for _, r := range path {
		if r == 0 || (r < 32 && r != '\t' && r != '\n' && r != '\r') {
			return "", fmt.Errorf("path contains invalid characters: %s", path)
		}
	}

	// Check for raw ".." sequences in the original path (before cleaning)
	// This catches cases like "../file.txt" or "dir/../../file.txt"
	// But allow "..." as it's a valid filename
	if strings.Contains(path, "..") && !strings.Contains(path, "...") {
		return "", fmt.Errorf("path contains directory traversal sequence: %s", path)
	}

	// Check for Windows absolute paths (C:, D:, etc.)
	if len(path) >= 2 && path[1] == ':' && ((path[0] >= 'A' && path[0] <= 'Z') || (path[0] >= 'a' && path[0] <= 'z')) {
		return "", fmt.Errorf("absolute path not allowed: %s", path)
	}

	// Check for paths starting with backslashes (Windows)
	if strings.HasPrefix(path, "\\") {
		return "", fmt.Errorf("path cannot start with backslash: %s", path)
	}

	// Normalize separators to forward slashes first (cross-platform)
	normalizedPath := strings.ReplaceAll(path, "\\", "/")

	cleanPath := filepath.Clean(normalizedPath)

	// Check if the cleaned path contains any remaining ".." components
	// This is a double-check in case filepath.Clean didn't catch everything
	if strings.Contains(cleanPath, "..") && !strings.Contains(cleanPath, "...") {
		return "", fmt.Errorf("path contains directory traversal sequence after cleaning: %s", path)
	}

	if cleanPath == "" {
		return "", fmt.Errorf("empty or invalid path: %s", path)
	}

	// Allow root directory "/" as it's a valid entry in TAR archives
	if cleanPath == "/" {
		return cleanPath, nil
	}

	if filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("absolute path not allowed: %s", path)
	}

	// Allow "." as it represents the current directory (common in TAR archives)
	if cleanPath == "." {
		return cleanPath, nil
	}

	// Strip leading path separator if present (common in some archive formats)
	if strings.HasPrefix(cleanPath, string(filepath.Separator)) {
		cleanPath = cleanPath[1:]
	}

	// Additional check: ensure the path doesn't start with path separators after stripping
	if strings.HasPrefix(cleanPath, string(filepath.Separator)) {
		return "", fmt.Errorf("path cannot start with path separator: %s", path)
	}

	return cleanPath, nil
}

// File represents a file in an archive
type File struct {
	io.ReadCloser
	Header interface{}
}

// getHeaderName safely extracts the name from archive headers and validates it
func (f File) getHeaderName() (string, error) {
	var rawName string
	if header, ok := f.Header.(*tar.Header); ok {
		rawName = header.Name
	} else if header, ok := f.Header.(zip.FileHeader); ok {
		rawName = header.Name
	} else {
		return "", fmt.Errorf("unknown header type")
	}

	// Always sanitize the path to prevent path traversal attacks
	return sanitizeArchivePath(rawName)
}

// getRawHeaderName extracts the raw name from archive headers without validation
// This is used for cases where we need to handle unknown header types gracefully
func (f File) getRawHeaderName() (string, bool) {
	if header, ok := f.Header.(*tar.Header); ok {
		return header.Name, true
	} else if header, ok := f.Header.(zip.FileHeader); ok {
		return header.Name, true
	}
	return "", false
}

// Name returns the name of the file
func (f File) Name() string {
	// First check if we have a known header type
	if rawName, ok := f.getRawHeaderName(); ok {
		// We have a known header type, sanitize the path
		sanitizedPath, err := sanitizeArchivePath(rawName)
		if err != nil {
			// If sanitization fails, return a safe default
			return "invalid_path"
		}
		return filepath.Base(sanitizedPath)
	}

	// Unknown header type, return empty string for backward compatibility
	return ""
}

// FullPath returns the full sanitized path of the file
func (f File) FullPath() (string, error) {
	return f.getHeaderName()
}

// Size returns the size of the file
func (f File) Size() int64 {
	if header, ok := f.Header.(*tar.Header); ok {
		return header.Size
	}
	if header, ok := f.Header.(zip.FileHeader); ok {
		return header.FileInfo().Size()
	}
	return 0
}

// IsDir checks if the file is a directory
func (f File) IsDir() bool {
	if header, ok := f.Header.(*tar.Header); ok {
		return header.FileInfo().IsDir()
	}
	if header, ok := f.Header.(zip.FileHeader); ok {
		return header.FileInfo().IsDir()
	}
	return false
}

// Mode returns the file mode
func (f File) Mode() os.FileMode {
	if header, ok := f.Header.(*tar.Header); ok {
		return header.FileInfo().Mode()
	}
	if header, ok := f.Header.(zip.FileHeader); ok {
		return header.FileInfo().Mode()
	}
	return 0
}

// FileInfo represents file metadata
type FileInfo struct {
	os.FileInfo
	CustomName string
}

// Name returns the custom name if set, otherwise the original name
func (fi FileInfo) Name() string {
	if fi.CustomName != "" {
		return fi.CustomName
	}
	return fi.FileInfo.Name()
}

// Reader interface for reading archives
type Reader interface {
	Open(archiveStream io.Reader, archiveSize int64) error
	Read() (File, error)
	Close() error
}

// Archiver interface for creating archives
type Archiver interface {
	Archive(files []string, destination string) error
}

// Zip implements the Archiver interface for ZIP files
type Zip struct {
	OverwriteExisting bool
	reader            *zip.Reader
	currentFile       int
	data              []byte
}

// NewZip creates a new ZIP archiver
func NewZip() *Zip {
	return &Zip{}
}

// Archive creates a ZIP archive from the given files
func (z *Zip) Archive(files []string, destination string) error {
	ctx := context.Background()

	// Create output file
	outFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Convert file paths to FileInfo slice
	fileMap := make(map[string]string)
	for _, file := range files {
		fileMap[file] = filepath.Base(file)
	}

	fileInfos, err := archives.FilesFromDisk(ctx, nil, fileMap)
	if err != nil {
		return err
	}

	// Create ZIP archive
	zip := &archives.Zip{}
	return zip.Archive(ctx, outFile, fileInfos)
}

// CheckExt checks if the file extension is appropriate for ZIP
func (z *Zip) CheckExt(archiveName string) error {
	if !strings.HasSuffix(strings.ToLower(archiveName), ".zip") {
		return fmt.Errorf("file %s does not have .zip extension", archiveName)
	}
	return nil
}

// Ext returns the file extension for ZIP files
func (z *Zip) Ext() string {
	return ".zip"
}

// Open opens a ZIP archive for reading
func (z *Zip) Open(archiveStream io.Reader, archiveSize int64) error {
	// Read the entire stream into memory since zip.NewReader requires io.ReaderAt
	data, err := io.ReadAll(archiveStream)
	if err != nil {
		return fmt.Errorf("failed to read archive data: %w", err)
	}

	// Create a reader from the data
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	z.reader = reader
	z.currentFile = 0
	z.data = data
	return nil
}

// Read reads the next file from the ZIP archive
func (z *Zip) Read() (File, error) {
	if z.reader == nil {
		return File{}, fmt.Errorf("archive not opened")
	}

	if z.currentFile >= len(z.reader.File) {
		return File{}, io.EOF
	}

	// Access file object - path validation happens immediately after
	file := z.reader.File[z.currentFile]

	// Validate and sanitize the file path to prevent path traversal attacks
	// This validation happens before any file operations to ensure security
	_, err := sanitizeArchivePath(file.Name)
	if err != nil {
		return File{}, fmt.Errorf("invalid file path in archive: %w", err)
	}

	z.currentFile++

	rc, err := file.Open()
	if err != nil {
		return File{}, fmt.Errorf("failed to open file in zip: %w", err)
	}

	return File{
		ReadCloser: rc,
		Header:     file.FileHeader,
	}, nil
}

// Close closes the ZIP archive
func (z *Zip) Close() error {
	z.reader = nil
	z.currentFile = 0
	z.data = nil
	return nil
}

// TarGz implements the Archiver interface for tar.gz files
type TarGz struct {
	OverwriteExisting bool
	Tar               *Tar
	reader            *tar.Reader
	gzipReader        io.ReadCloser
}

// NewTarGz creates a new tar.gz archiver
func NewTarGz() *TarGz {
	return &TarGz{}
}

// Archive creates a tar.gz archive from the given files
func (tgz *TarGz) Archive(files []string, destination string) error {
	ctx := context.Background()

	// Create output file
	outFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Convert file paths to FileInfo slice
	fileMap := make(map[string]string)
	for _, file := range files {
		fileMap[file] = filepath.Base(file)
	}

	fileInfos, err := archives.FilesFromDisk(ctx, nil, fileMap)
	if err != nil {
		return err
	}

	// Create tar.gz archive using CompressedArchive
	compressedArchive := archives.CompressedArchive{
		Compression: archives.Gz{},
		Archival:    archives.Tar{},
	}
	return compressedArchive.Archive(ctx, outFile, fileInfos)
}

// CheckExt checks if the file extension is appropriate for tar.gz
func (tgz *TarGz) CheckExt(archiveName string) error {
	if !strings.HasSuffix(strings.ToLower(archiveName), ".tar.gz") {
		return fmt.Errorf("file %s does not have .tar.gz extension", archiveName)
	}
	return nil
}

// Ext returns the file extension for tar.gz files
func (tgz *TarGz) Ext() string {
	return ".tar.gz"
}

// Open opens a tar.gz archive for reading
func (tgz *TarGz) Open(archiveStream io.Reader, archiveSize int64) error {
	// Create gzip reader
	gzReader, err := gzip.NewReader(archiveStream)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	tgz.reader = tarReader
	tgz.gzipReader = gzReader
	return nil
}

// Read reads the next file from the tar.gz archive
func (tgz *TarGz) Read() (File, error) {
	if tgz.reader == nil {
		return File{}, fmt.Errorf("archive not opened")
	}

	header, err := tgz.reader.Next()
	if err != nil {
		if err == io.EOF {
			return File{}, io.EOF
		}
		return File{}, fmt.Errorf("failed to read tar header: %w", err)
	}

	// Validate and sanitize the file path to prevent path traversal attacks
	_, err = sanitizeArchivePath(header.Name)
	if err != nil {
		return File{}, fmt.Errorf("invalid file path in archive: %w", err)
	}

	return File{
		ReadCloser: &tarFileReader{
			reader: tgz.reader,
			size:   header.Size,
		},
		Header: header,
	}, nil
}

// Close closes the tar.gz archive
func (tgz *TarGz) Close() error {
	if tgz.gzipReader != nil {
		tgz.gzipReader.Close()
		tgz.gzipReader = nil
	}
	tgz.reader = nil
	return nil
}

// tarFileReader wraps a tar.Reader to implement io.ReadCloser
type tarFileReader struct {
	reader *tar.Reader
	size   int64
	read   int64
}

func (tfr *tarFileReader) Read(p []byte) (n int, err error) {
	if tfr.read >= tfr.size {
		return 0, io.EOF
	}

	// Limit read to remaining size
	remaining := tfr.size - tfr.read
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}

	n, err = tfr.reader.Read(p)
	tfr.read += int64(n)
	return n, err
}

func (tfr *tarFileReader) Close() error {
	// For tar files, we don't need to close anything specific
	return nil
}

// Tar represents a tar archive (used within TarGz)
type Tar struct {
	StripComponents int
}

// CreateTgz creates a tar.gz archive with the given file mappings
func CreateTgz(archivePath string, workDir string, fileMaps []FileMap) error {
	ctx := context.Background()

	// Create output file
	outFile, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Convert fileMaps to FileInfo slice
	fileMap := make(map[string]string)
	for _, fileMapItem := range fileMaps {
		source := fileMapItem.Source
		if !filepath.IsAbs(source) {
			source = filepath.Join(workDir, source)
		}
		fileMap[source] = fileMapItem.Target
	}

	fileInfos, err := archives.FilesFromDisk(ctx, nil, fileMap)
	if err != nil {
		return err
	}

	// Create tar.gz archive using CompressedArchive
	compressedArchive := archives.CompressedArchive{
		Compression: archives.Gz{},
		Archival:    archives.Tar{},
	}
	return compressedArchive.Archive(ctx, outFile, fileInfos)
}

// FileMap represents a source to target file mapping
type FileMap struct {
	Source string
	Target string
}

// FilesWithCommonParent creates file mappings with a common parent path
func FilesWithCommonParent(filepaths ...string) []FileMap {
	var fileMaps []FileMap
	common := fileutils.CommonParentPath(filepaths)
	for _, path := range filepaths {
		path = filepath.ToSlash(path)
		fileMaps = append(fileMaps, FileMap{
			Source: path,
			Target: strings.TrimPrefix(strings.TrimPrefix(path, common), "/"),
		})
	}
	return fileMaps
}
