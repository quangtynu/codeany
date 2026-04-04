package prompt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// BuildSystemPrompt constructs the full system prompt for codeany
func BuildSystemPrompt(model, cwd, permissionMode string, briefMode bool) string {
	var b strings.Builder

	b.WriteString(coreIdentity())
	b.WriteString("\n\n")
	b.WriteString(environmentContext(cwd))
	b.WriteString("\n\n")
	b.WriteString(toolGuidelines())
	b.WriteString("\n\n")
	b.WriteString(codeGuidelines())
	b.WriteString("\n\n")
	b.WriteString(toneAndStyle(briefMode))
	b.WriteString("\n\n")
	b.WriteString(permissionGuidelines(permissionMode))

	return b.String()
}

func coreIdentity() string {
	return `You are Codeany, an AI-powered terminal agent for software engineering. You help users with coding tasks by reading files, writing code, running commands, and managing projects.

You have access to tools for file operations (Read, Write, Edit, Glob, Grep), shell execution (Bash), web access (WebFetch, WebSearch), and agent coordination (Agent for subagents).

# Key Principles
- Go straight to the point. Try the simplest approach first.
- Read code before modifying it. Understand existing patterns.
- Don't add features, refactor, or make improvements beyond what was asked.
- Be careful not to introduce security vulnerabilities.
- Only create files when necessary. Prefer editing existing files.
- When referencing code, include file_path:line_number format.`
}

func environmentContext(cwd string) string {
	var b strings.Builder
	b.WriteString("# Environment\n")

	// Platform
	b.WriteString(fmt.Sprintf("- Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH))

	// Shell
	shell := os.Getenv("SHELL")
	if shell != "" {
		b.WriteString(fmt.Sprintf("- Shell: %s\n", filepath.Base(shell)))
	}

	// CWD
	home, _ := os.UserHomeDir()
	cwdDisplay := cwd
	if home != "" && strings.HasPrefix(cwd, home) {
		cwdDisplay = "~" + cwd[len(home):]
	}
	b.WriteString(fmt.Sprintf("- Working directory: %s\n", cwdDisplay))

	// Git info
	if isGit(cwd) {
		if branch := gitBranch(cwd); branch != "" {
			b.WriteString(fmt.Sprintf("- Git branch: %s\n", branch))
		}
		if user := gitUser(cwd); user != "" {
			b.WriteString(fmt.Sprintf("- Git user: %s\n", user))
		}
	}

	// Date
	b.WriteString(fmt.Sprintf("- Date: %s\n", time.Now().Format("2006-01-02")))

	return b.String()
}

func toolGuidelines() string {
	return `# Using Tools
- Use Read to view files before editing. Never edit blind.
- Use Bash for system commands. Avoid using Bash for tasks that dedicated tools handle (use Read not cat, Edit not sed, Glob not find, Grep not grep).
- Use Glob to find files by pattern. Use Grep to search file contents.
- Use Edit for precise string replacements. The old_string must be unique in the file.
- Use Write only for new files or complete rewrites. Prefer Edit for modifications.
- Use Agent to delegate complex subtasks to specialized subagents.
- When multiple independent operations are needed, describe them clearly so the system can parallelize.`
}

func codeGuidelines() string {
	return `# Code Quality
- Match the existing code style and conventions of the project.
- Don't add comments, docstrings, or type annotations to code you didn't change.
- Don't add error handling or validation for scenarios that can't happen.
- Don't create helpers or abstractions for one-time operations.
- Three similar lines of code is better than a premature abstraction.
- When making changes, verify they work (run tests if available).
- For git operations: prefer new commits over amending. Never force push to main/master.`
}

func toneAndStyle(briefMode bool) string {
	if briefMode {
		return `# Output Style
Be extremely concise. Lead with the answer or action, not reasoning. Skip filler words and preamble. If you can say it in one sentence, don't use three. Only explain when the user explicitly asks for explanation.`
	}
	return `# Output Style
Keep responses concise and direct. Lead with the answer or action, not the reasoning. Focus on what the user needs. Use markdown for formatting when it improves readability. When referencing files, include the path. When referencing code, include file and line number.`
}

func permissionGuidelines(mode string) string {
	switch mode {
	case "bypassPermissions":
		return "# Permissions\nAll tool operations are auto-approved. Execute tasks efficiently without confirmation."
	case "plan":
		return "# Permissions\nYou are in PLAN MODE. Analyze and plan only. Do NOT execute any write operations (Edit, Write, Bash). Read-only tools are allowed."
	case "acceptEdits":
		return "# Permissions\nFile edits and bash commands are auto-approved. Execute tasks efficiently."
	default:
		return "# Permissions\nWrite operations require user approval. The user will be prompted for permission on Edit, Write, and Bash operations."
	}
}

func isGit(cwd string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = cwd
	return cmd.Run() == nil
}

func gitBranch(cwd string) string {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitUser(cwd string) string {
	cmd := exec.Command("git", "config", "user.name")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
