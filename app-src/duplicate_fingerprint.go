package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

func verifyDuplicateOriginal(path string, rec MoveRecord) error {
	file, err := openGuardedRegularFile(path, false)
	if err != nil {
		return fmt.Errorf("surviving original copy is unavailable or in use: %w", err)
	}
	defer file.Close()
	return verifyDuplicateOriginalHandle(file, rec)
}

func verifyDuplicateOriginalHandle(file *os.File, rec MoveRecord) error {
	want, err := hex.DecodeString(rec.OriginalSHA256)
	if err != nil || len(want) != sha256.Size || rec.OriginalSize < 0 {
		return errors.New("original fingerprint is missing or invalid; manual recovery is required")
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != rec.OriginalSize {
		return errors.New("surviving copy no longer matches the original size")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != hex.EncodeToString(want) {
		return errors.New("surviving copy no longer matches the original SHA-256 fingerprint")
	}
	return nil
}
