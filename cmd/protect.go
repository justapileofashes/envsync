package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var protectUninstall bool

var protectCmd = &cobra.Command{
	Use:   "protect",
	Short: "Install a git pre-commit hook that blocks committing .env files",
	Long: `Installs a git pre-commit hook in the current repository. If anyone
accidentally stages a .env file (or .env.local, .env.production, …) and tries to
commit, the hook blocks the commit with a loud red warning — preventing a
catastrophic secret leak to GitHub.

Files matching .env.example / .env.schema / .env.sample are allowed. The hook is
idempotent and can be removed with --uninstall.`,
	RunE: runProtect,
}

func init() {
	protectCmd.Flags().BoolVar(&protectUninstall, "uninstall", false, "remove the EnvSync pre-commit guard")
	rootCmd.AddCommand(protectCmd)
}

const (
	protectMarkerStart = "# >>> envsync protect >>>"
	protectMarkerEnd   = "# <<< envsync protect <<<"
)

const protectGuard = `# >>> envsync protect >>>
# Installed by ` + "`envsync protect`" + ` — blocks accidental .env commits.
blocked=$(git diff --cached --name-only --diff-filter=ACM \
  | grep -E '(^|/)\.env([._-].*)?$' \
  | grep -vE '\.(example|schema|sample|template)$' || true)
if [ -n "$blocked" ]; then
  printf '\033[31m\n  ⛔ EnvSync: refusing to commit secret file(s):\033[0m\n' >&2
  printf '     %s\n' $blocked >&2
  printf '\033[33m  These look like real environment files. To proceed:\033[0m\n' >&2
  printf '     git rm --cached <file>   # unstage it, and add it to .gitignore\n' >&2
  printf '     git commit --no-verify   # bypass (only if you are CERTAIN)\n\n' >&2
  exit 1
fi
# <<< envsync protect <<<
`

func runProtect(_ *cobra.Command, _ []string) error {
	gitDir, err := findGitDir()
	if err != nil {
		return err
	}
	hookPath := filepath.Join(gitDir, "hooks", "pre-commit")

	if protectUninstall {
		return uninstallProtect(hookPath)
	}

	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		return fmt.Errorf("cannot create hooks directory: %w", err)
	}

	existing, _ := os.ReadFile(hookPath)
	content := string(existing)

	if strings.Contains(content, protectMarkerStart) {
		fmt.Println("EnvSync pre-commit guard already installed — nothing to do.")
		return nil
	}

	var out string
	if strings.TrimSpace(content) == "" {
		out = "#!/bin/sh\n" + protectGuard
	} else {
		// Append our guard to the existing hook, preserving it.
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		out = content + "\n" + protectGuard
	}

	if err := os.WriteFile(hookPath, []byte(out), 0o755); err != nil {
		return fmt.Errorf("cannot write pre-commit hook: %w", err)
	}

	fmt.Printf("Installed EnvSync pre-commit guard at %s\n", hookPath)
	fmt.Println("Accidental `git add .env` commits will now be blocked.")
	return nil
}

func uninstallProtect(hookPath string) error {
	data, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No pre-commit hook present — nothing to remove.")
			return nil
		}
		return err
	}
	content := string(data)
	start := strings.Index(content, protectMarkerStart)
	end := strings.Index(content, protectMarkerEnd)
	if start == -1 || end == -1 {
		fmt.Println("EnvSync guard not found in pre-commit hook — leaving it untouched.")
		return nil
	}
	end += len(protectMarkerEnd)
	cleaned := strings.TrimRight(content[:start], "\n") + content[end:]
	cleaned = strings.TrimSpace(cleaned)

	// If nothing but the shebang remains, remove the file entirely.
	if cleaned == "" || cleaned == "#!/bin/sh" {
		if err := os.Remove(hookPath); err != nil {
			return fmt.Errorf("cannot remove hook: %w", err)
		}
		fmt.Println("Removed EnvSync pre-commit guard (hook was otherwise empty).")
		return nil
	}
	if err := os.WriteFile(hookPath, []byte(cleaned+"\n"), 0o755); err != nil {
		return fmt.Errorf("cannot update hook: %w", err)
	}
	fmt.Println("Removed EnvSync pre-commit guard; kept the rest of your hook.")
	return nil
}

// findGitDir walks up from the cwd to locate the .git directory (handling both a
// normal repo and a worktree/file-based .git pointer).
func findGitDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, ".git")
		info, err := os.Stat(candidate)
		if err == nil {
			if info.IsDir() {
				return candidate, nil
			}
			// .git file (worktree): "gitdir: <path>"
			data, rerr := os.ReadFile(candidate)
			if rerr == nil {
				line := strings.TrimSpace(strings.TrimPrefix(string(data), "gitdir:"))
				if line != "" {
					if !filepath.IsAbs(line) {
						line = filepath.Join(dir, line)
					}
					return line, nil
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a git repository (no .git found from here upward); run `git init` first")
		}
		dir = parent
	}
}
