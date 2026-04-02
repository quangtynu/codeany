package team

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Team represents a group of cooperating agents
type Team struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	LeadAgent   string   `json:"leadAgent"`
	Members     []Member `json:"members"`
	CreatedAt   time.Time `json:"createdAt"`
	dir         string
}

// Member is an agent in a team
type Member struct {
	Name      string `json:"name"`
	AgentType string `json:"agentType"`
	Model     string `json:"model,omitempty"`
	IsActive  bool   `json:"isActive"`
}

// Message is an inter-agent message
type Message struct {
	From      string    `json:"from"`
	To        string    `json:"to"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	Read      bool      `json:"read"`
}

var mu sync.Mutex

// TeamsDir returns the teams directory
func TeamsDir(configDir string) string {
	return filepath.Join(configDir, "teams")
}

// Create creates a new team
func Create(configDir, name, description string) (*Team, error) {
	dir := filepath.Join(TeamsDir(configDir), name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	os.MkdirAll(filepath.Join(dir, "inboxes"), 0755)

	t := &Team{
		Name:        name,
		Description: description,
		LeadAgent:   "lead",
		Members:     []Member{{Name: "lead", AgentType: "general-purpose", IsActive: true}},
		CreatedAt:   time.Now(),
		dir:         dir,
	}

	return t, t.Save()
}

// Load loads a team by name
func Load(configDir, name string) (*Team, error) {
	dir := filepath.Join(TeamsDir(configDir), name)
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return nil, err
	}
	var t Team
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	t.dir = dir
	return &t, nil
}

// ListTeams lists all teams
func ListTeams(configDir string) []Team {
	dir := TeamsDir(configDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var teams []Team
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t, err := Load(configDir, entry.Name())
		if err != nil {
			continue
		}
		teams = append(teams, *t)
	}
	return teams
}

// Delete removes a team
func Delete(configDir, name string) error {
	dir := filepath.Join(TeamsDir(configDir), name)
	return os.RemoveAll(dir)
}

// Save persists team config
func (t *Team) Save() error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(t.dir, "config.json"), data, 0644)
}

// AddMember adds a member to the team
func (t *Team) AddMember(name, agentType, model string) {
	t.Members = append(t.Members, Member{
		Name:      name,
		AgentType: agentType,
		Model:     model,
		IsActive:  true,
	})
	t.Save()
}

// SendMessage sends a message to an agent's inbox
func SendMsg(configDir, teamName, from, to, text string) error {
	mu.Lock()
	defer mu.Unlock()

	inboxPath := filepath.Join(TeamsDir(configDir), teamName, "inboxes", to+".json")

	var messages []Message
	if data, err := os.ReadFile(inboxPath); err == nil {
		json.Unmarshal(data, &messages)
	}

	messages = append(messages, Message{
		From:      from,
		To:        to,
		Text:      text,
		Timestamp: time.Now(),
	})

	data, _ := json.MarshalIndent(messages, "", "  ")
	return os.WriteFile(inboxPath, data, 0644)
}

// ReadInbox reads unread messages for an agent
func ReadInbox(configDir, teamName, agentName string) []Message {
	mu.Lock()
	defer mu.Unlock()

	inboxPath := filepath.Join(TeamsDir(configDir), teamName, "inboxes", agentName+".json")
	data, err := os.ReadFile(inboxPath)
	if err != nil {
		return nil
	}

	var messages []Message
	json.Unmarshal(data, &messages)

	// Mark as read
	var unread []Message
	for i := range messages {
		if !messages[i].Read {
			unread = append(unread, messages[i])
			messages[i].Read = true
		}
	}

	if len(unread) > 0 {
		data, _ := json.MarshalIndent(messages, "", "  ")
		os.WriteFile(inboxPath, data, 0644)
	}

	return unread
}

// FormatTeamList formats teams for display
func FormatTeamList(teams []Team) string {
	if len(teams) == 0 {
		return "No teams.\n\nCreate one with: /team create <name> [description]"
	}

	var b strings.Builder
	b.WriteString("Teams:\n\n")
	for _, t := range teams {
		active := 0
		for _, m := range t.Members {
			if m.IsActive {
				active++
			}
		}
		b.WriteString(fmt.Sprintf("  %s — %d members (%d active)\n", t.Name, len(t.Members), active))
		if t.Description != "" {
			b.WriteString(fmt.Sprintf("    %s\n", t.Description))
		}
	}
	return b.String()
}
