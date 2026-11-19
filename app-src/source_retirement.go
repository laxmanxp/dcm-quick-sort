package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

// sourceSnapshot binds the bytes selected for a move to an open-file identity.
// Info comes from File.Stat, so a later pathname replacement cannot change the
// identity against which retirement is checked.
type sourceSnapshot struct {
	Info   os.FileInfo
	Size   int64
	SHA256 string
}

type verifiedRetireFunc func(source, target string, expected sourceSnapshot) error

func snapshotSource(path string) (sourceSnapshot, error) {
	file, err := openGuardedRegularFile(path, false)
	if err != nil {
		return sourceSnapshot{}, err
	}
	defer file.Close()
	return snapshotOpenFile(file)
}

// snapshotOpenFile reads the already guarded handle, never reopening its path.
// It leaves the file offset at EOF; callers that copy from it must seek to zero.
func snapshotOpenFile(file *os.File) (sourceSnapshot, error) {
	info, err := file.Stat()
	if err != nil {
		return sourceSnapshot{}, err
	}
	if !info.Mode().IsRegular() {
		return sourceSnapshot{}, errors.New("guarded path is not a regular file")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return sourceSnapshot{}, err
	}
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return sourceSnapshot{}, err
	}
	if size != info.Size() {
		return sourceSnapshot{}, errors.New("file size changed while its content was being verified")
	}
	return sourceSnapshot{Info: info, Size: size, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

// retireVerifiedSource keeps both source and destination protected from writes
// and pathname changes until the exact verified source handle has been retired.
// A refusal preserves both files and must not be reported as a successful move.
func retireVerifiedSource(source, target string, expected sourceSnapshot) error {
	if expected.Info == nil || expected.Size < 0 || len(expected.SHA256) != sha256.Size*2 {
		return errors.New("source retirement requires its original identity and SHA-256 snapshot")
	}
	if _, err := hex.DecodeString(expected.SHA256); err != nil {
		return errors.New("source retirement snapshot has an invalid SHA-256 digest")
	}
	sourceFile, err := openGuardedRegularFile(source, true)
	if err != nil {
		return fmt.Errorf("could not guard source for verified retirement: %w", err)
	}
	defer sourceFile.Close()
	current, err := snapshotOpenFile(sourceFile)
	if err != nil {
		return fmt.Errorf("could not verify guarded source: %w", err)
	}
	if !os.SameFile(expected.Info, current.Info) {
		return errors.New("source path now identifies a different file; both files were retained")
	}
	if current.Size != expected.Size || current.SHA256 != expected.SHA256 {
		return errors.New("source content changed after the move started; both files were retained")
	}

	targetFile, err := openGuardedRegularFile(target, false)
	if err != nil {
		return fmt.Errorf("could not guard destination before source retirement: %w", err)
	}
	defer targetFile.Close()
	destination, err := snapshotOpenFile(targetFile)
	if err != nil {
		return fmt.Errorf("could not verify guarded destination: %w", err)
	}
	if os.SameFile(current.Info, destination.Info) {
		return errors.New("source and destination identify the same file; nothing was removed")
	}
	if destination.Size != expected.Size || destination.SHA256 != expected.SHA256 {
		return errors.New("destination no longer contains the verified source bytes; both files were retained")
	}
	if err := deleteGuardedFile(sourceFile); err != nil {
		return fmt.Errorf("could not retire the verified source handle: %w", err)
	}
	// Close while the destination guard is still held. Deletion applies to this
	// handle's identity, not to a pathname that another process could replace.
	if err := sourceFile.Close(); err != nil {
		return fmt.Errorf("could not close the retired source handle: %w", err)
	}
	return nil
}
