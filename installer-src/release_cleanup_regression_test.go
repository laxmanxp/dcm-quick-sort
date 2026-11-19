//go:build windows

package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// Evaluate the committed uninstall script expression, not a duplicated fixture
// implementation. Every substituted path belongs to the test's temporary root.
func cleanupScriptFromSource(t *testing.T, installDir, uninstallExe string) string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "main_windows.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var expression ast.Expr
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "uninstall" {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			assignment, ok := node.(*ast.AssignStmt)
			if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
				return true
			}
			name, ok := assignment.Lhs[0].(*ast.Ident)
			if ok && name.Name == "script" {
				expression = assignment.Rhs[0]
			}
			return true
		})
	}
	if expression == nil {
		t.Fatal("production uninstall script expression not found")
	}
	values := map[string]string{"installDir": installDir, "uninstallExe": uninstallExe}
	var evaluate func(ast.Expr) (string, error)
	evaluate = func(expr ast.Expr) (string, error) {
		switch value := expr.(type) {
		case *ast.BasicLit:
			if value.Kind == token.STRING {
				return strconv.Unquote(value.Value)
			}
		case *ast.Ident:
			if text, ok := values[value.Name]; ok {
				return text, nil
			}
		case *ast.BinaryExpr:
			if value.Op == token.ADD {
				left, err := evaluate(value.X)
				if err != nil {
					return "", err
				}
				right, err := evaluate(value.Y)
				return left + right, err
			}
		}
		return "", fmt.Errorf("unsupported production cleanup expression %T", expr)
	}
	script, err := evaluate(expression)
	if err != nil {
		t.Fatal(err)
	}
	return script
}

func runCleanupFixture(t *testing.T, variant string) (int, bool, bool, string) {
	t.Helper()
	root := t.TempDir()
	installDir := filepath.Join(root, "Owned Install")
	uninstaller := filepath.Join(installDir, "Uninstall DCM Quick Sort.exe")
	batch := filepath.Join(root, "cleanup.cmd")
	if err := os.Mkdir(installDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(uninstaller, []byte("disposable uninstaller fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(root, "must-remain.txt")
	if err := os.WriteFile(sentinel, []byte("keep original bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	script := cleanupScriptFromSource(t, installDir, uninstaller)
	if err := os.WriteFile(batch, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	// Deny deletion of exactly one owned fixture to prove genuine failures
	// cannot be converted into an unconditional success exit.
	lockPath := ""
	switch variant {
	case "locked-uninstaller":
		lockPath = uninstaller
	case "locked-install-file":
		lockPath = filepath.Join(installDir, "busy.bin")
		if err := os.WriteFile(lockPath, []byte("locked original"), 0o600); err != nil {
			t.Fatal(err)
		}
	case "locked-script":
		lockPath = batch
	}
	if lockPath != "" {
		name, err := syscall.UTF16PtrFromString(lockPath)
		if err != nil {
			t.Fatal(err)
		}
		handle, err := syscall.CreateFile(name, syscall.GENERIC_READ, syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE, nil, syscall.OPEN_EXISTING, 0, 0)
		if err != nil {
			t.Fatal(err)
		}
		defer syscall.CloseHandle(handle)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "cmd.exe", "/D", "/V:OFF", "/C", batch)
	cmd.Dir = root
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: CREATE_NO_WINDOW}
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatal("owned cleanup fixture exceeded deadline")
	}
	code := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	_, installErr := os.Stat(installDir)
	_, batchErr := os.Stat(batch)
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "keep original bytes" {
		t.Fatalf("outside sentinel changed: %q, %v", data, err)
	}
	if lockPath != "" {
		if _, err := os.Stat(lockPath); err != nil {
			t.Fatalf("locked fixture changed: %v", err)
		}
		if variant == "locked-uninstaller" || variant == "locked-install-file" {
			want := "locked original"
			if variant == "locked-uninstaller" {
				want = "disposable uninstaller fixture"
			}
			data, err := os.ReadFile(lockPath)
			if err != nil || string(data) != want {
				t.Fatalf("locked bytes changed: %q %v", data, err)
			}
		}
	}

	return code, os.IsNotExist(installErr), os.IsNotExist(batchErr), string(output)
}

func TestReleaseUninstallCleanupReportsSuccessfulRemoval(t *testing.T) {
	started := time.Now()
	code, removed, selfRemoved, output := runCleanupFixture(t, "production")
	t.Logf("production: exit=%d install_removed=%v script_removed=%v output=%q", code, removed, selfRemoved, output)
	if time.Since(started) < 2*time.Second {
		t.Error("cleanup delay did not wait without console input")
	}
	if output != "" {
		t.Errorf("successful cleanup emitted an error: %q", output)
	}
	if code != 0 || !removed || !selfRemoved {
		t.Errorf("successful cleanup requires exit 0 plus absent owned install/script; got exit=%d removed=%v self_removed=%v", code, removed, selfRemoved)
	}
}

func TestReleaseUninstallCleanupReportsLockedTargetFailures(t *testing.T) {
	for _, tc := range []struct {
		name           string
		code           int
		installRemoved bool
	}{
		{"locked-uninstaller", 1, false},
		{"locked-install-file", 1, false},
		{"locked-script", 2, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, removed, selfRemoved, output := runCleanupFixture(t, tc.name)
			t.Logf("exit=%d install_removed=%v script_removed=%v output=%q", code, removed, selfRemoved, output)
			if code != tc.code || removed != tc.installRemoved || selfRemoved {
				t.Errorf("cleanup hid a blocked removal: exit=%d removed=%v self_removed=%v", code, removed, selfRemoved)
			}
		})
	}
}
