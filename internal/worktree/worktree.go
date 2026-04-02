package worktree

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Worktree represents an active git worktree
type Worktree struct {
	Name           string    `json:"name"`
	Path           string    `json:"path"`
	OriginalCWD    string    `json:"originalCWD"`
	OriginalBranch string    `json:"originalBranch"`
	Branch         string    `json:"branch"`
	SessionID      string    `json:"sessionId"`
	CreatedAt      time.Time `json:"createdAt"`
}

// WorktreeDir returns the worktrees directory
func WorktreeDir(configDir string) string {
	return filepath.Join(configDir, "worktrees")
}

// Create creates a new git worktree
func Create(configDir, name, sessionID string) (*Worktree, error) {
	cwd, _ := os.Getwd()

	// Verify we're in a git repo
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return nil, fmt.Errorf("not a git repository")
	}
	gitRoot := strings.TrimSpace(string(out))

	// Get current branch
	branchOut, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	currentBranch := strings.TrimSpace(string(branchOut))

	// Create worktree
	wtDir := filepath.Join(WorktreeDir(configDir), sessionID, name)
	if err := os.MkdirAll(filepath.Dir(wtDir), 0755); err != nil {
		return nil, err
	}

	branchName := fmt.Sprintf("codeany/%s/%s", sessionID[:8], name)
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, wtDir)
	cmd.Dir = gitRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git worktree add failed: %s", strings.TrimSpace(string(output)))
	}

	wt := &Worktree{
		Name:           name,
		Path:           wtDir,
		OriginalCWD:    cwd,
		OriginalBranch: currentBranch,
		Branch:         branchName,
		SessionID:      sessionID,
		CreatedAt:      time.Now(),
	}

	// Save metadata
	data, _ := json.MarshalIndent(wt, "", "  ")
	os.WriteFile(filepath.Join(wtDir, ".codeany-worktree.json"), data, 0644)

	return wt, nil
}

// Enter changes CWD to the worktree
func (wt *Worktree) Enter() error {
	return os.Chdir(wt.Path)
}

// Exit returns to original CWD and optionally removes worktree
func (wt *Worktree) Exit(remove bool) error {
	if err := os.Chdir(wt.OriginalCWD); err != nil {
		return err
	}

	if remove {
		cmd := exec.Command("git", "worktree", "remove", wt.Path, "--force")
		cmd.Dir = wt.OriginalCWD
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git worktree remove failed: %s", strings.TrimSpace(string(output)))
		}

		// Remove branch
		exec.Command("git", "branch", "-D", wt.Branch).Run()
	}

	return nil
}

// LoadActive loads active worktree for a session
func LoadActive(configDir, sessionID string) *Worktree {
	dir := filepath.Join(WorktreeDir(configDir), sessionID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(dir, entry.Name(), ".codeany-worktree.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var wt Worktree
		if json.Unmarshal(data, &wt) == nil {
			return &wt
		}
	}
	return nil
}

// ListAll lists all worktrees
func ListAll(configDir string) []Worktree {
	dir := WorktreeDir(configDir)
	var worktrees []Worktree

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.Name() == ".codeany-worktree.json" {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			var wt Worktree
			if json.Unmarshal(data, &wt) == nil {
				worktrees = append(worktrees, wt)
			}
		}
		return nil
	})

	return worktrees
}
