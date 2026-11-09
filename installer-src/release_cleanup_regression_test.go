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
	"strings"
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
	switch variant {
	case "no-delay":
		script = strings.ReplaceAll(script, "timeout /t 2 /nobreak >nul\r\n", "")
	case "no-self-delete":
		script = strings.ReplaceAll(script, "del /f /q \"%~f0\" >nul 2>nul\r\n", "")
	case "neither":
		script = strings.ReplaceAll(script, "timeout /t 2 /nobreak >nul\r\n", "")
		script = strings.ReplaceAll(script, "del /f /q \"%~f0\" >nul 2>nul\r\n", "")
	}
	if err := os.WriteFile(batch, []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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
	return code, os.IsNotExist(installErr), os.IsNotExist(batchErr), string(output)
}

func TestReleaseUninstallCleanupReportsSuccessfulRemoval(t *testing.T) {
	code, removed, selfRemoved, output := runCleanupFixture(t, "production")
	t.Logf("production: exit=%d install_removed=%v script_removed=%v output=%q", code, removed, selfRemoved, output)
	if code != 0 || !removed || !selfRemoved {
		t.Errorf("successful cleanup requires exit 0 plus absent owned install/script; got exit=%d removed=%v self_removed=%v", code, removed, selfRemoved)
	}
}

func TestReleaseUninstallCleanupCommandDiagnosis(t *testing.T) {
	for _, variant := range []string{"no-delay", "no-self-delete", "neither"} {
		t.Run(variant, func(t *testing.T) {
			code, removed, selfRemoved, output := runCleanupFixture(t, variant)
			t.Logf("diagnostic only: exit=%d install_removed=%v script_removed=%v output=%q", code, removed, selfRemoved, output)
			if !removed {
				t.Error("diagnostic did not remove its disposable install fixture")
			}
		})
	}
}
