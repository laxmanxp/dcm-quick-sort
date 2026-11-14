// Inspect package bytes without executing the installer or changing registration.
package main

import (
	"bytes"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	appPath := flag.String("app", "", "qualified app executable")
	installerPath := flag.String("installer", "", "candidate installer executable")
	flag.Parse()
	if *appPath == "" || *installerPath == "" || flag.NArg() != 0 {
		fatal(fmt.Errorf("-app and -installer are required"))
	}
	app, err := os.ReadFile(*appPath)
	if err != nil {
		fatal(err)
	}
	installer, err := os.ReadFile(*installerPath)
	if err != nil {
		fatal(err)
	}
	if len(app) == 0 || len(installer) == 0 {
		fatal(fmt.Errorf("empty executable"))
	}
	for _, path := range []string{*appPath, *installerPath} {
		p, err := pe.Open(path)
		if err != nil {
			fatal(err)
		}
		machine := p.FileHeader.Machine
		p.Close()
		if machine != pe.IMAGE_FILE_MACHINE_AMD64 {
			fatal(fmt.Errorf("%s is not a Windows amd64 PE", path))
		}
	}
	count := bytes.Count(installer, app)
	if count != 1 {
		fatal(fmt.Errorf("installer contains %d exact copies of the qualified app; want one", count))
	}
	appHash, installerHash := sha256.Sum256(app), sha256.Sum256(installer)
	result := struct {
		App                string `json:"app"`
		Installer          string `json:"installer"`
		AppSHA256          string `json:"app_sha256"`
		InstallerSHA256    string `json:"installer_sha256"`
		AppSize            int    `json:"app_size"`
		InstallerSize      int    `json:"installer_size"`
		ExactPayloadCopies int    `json:"exact_payload_copies"`
		PayloadOffset      int    `json:"payload_offset"`
	}{*appPath, *installerPath, hex.EncodeToString(appHash[:]), hex.EncodeToString(installerHash[:]), len(app), len(installer), count, bytes.Index(installer, app)}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
