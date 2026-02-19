package library

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// Hashes represents all hash types for a game file
type Hashes struct {
	MD5    string
	SHA1   string
	SHA256 string
	Blake3 string // Reserved for future use when blake3 library is available
}

// CalculateFileHashes computes all hashes for a file
// Currently computes MD5, SHA1, SHA256. Blake3 is reserved for future use.
func (svc *LibraryService) CalculateFileHashes(filePath string) (*Hashes, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file %s: %w", filePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			svc.logger.WarnContext(nil, "failed to close file", "path", filePath, "error", err)
		}
	}()

	// Create hash writers
	md5Hash := md5.New()
	sha1Hash := sha1.New()
	sha256Hash := sha256.New()

	// Create a multi-writer to hash all at once
	multiWriter := io.MultiWriter(md5Hash, sha1Hash, sha256Hash)

	// Copy file to hash writers (streaming to avoid loading entire file in memory)
	if _, err := io.Copy(multiWriter, file); err != nil {
		return nil, fmt.Errorf("failed to read file for hashing %s: %w", filePath, err)
	}

	return &Hashes{
		MD5:    fmt.Sprintf("%x", md5Hash.Sum(nil)),
		SHA1:   fmt.Sprintf("%x", sha1Hash.Sum(nil)),
		SHA256: fmt.Sprintf("%x", sha256Hash.Sum(nil)),
		Blake3: "", // Reserved for future use
	}, nil
}

// HashesToMap converts Hashes struct to map for easy storage/lookup
func (h *Hashes) ToMap() map[string]string {
	return map[string]string{
		"md5":    h.MD5,
		"sha1":   h.SHA1,
		"sha256": h.SHA256,
		"blake3": h.Blake3,
	}
}
