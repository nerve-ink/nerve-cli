package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------- КОНФИГ ----------------
var (
	apiURL     string
	channelID  string
)

// ---------------- СТИЛИ (Lipgloss) ----------------
var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#04B575")). // Hacker Green
		MarginBottom(1)

	msgStyle = lipgloss.NewStyle().
		PaddingLeft(1).
		Foreground(lipgloss.Color("#FFF"))

	severityCritical = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true) // Red
	severityInfo     = lipgloss.NewStyle().Foreground(lipgloss.Color("#0000FF"))            // Blue
	severityTime     = lipgloss.NewStyle().Foreground(lipgloss.Color("#555"))               // Grey
)

// ---------------- МОДЕЛИ ДАННЫХ ----------------
type Message struct {
	ID        string `json:"id"` // Changed to string to match backend
	Channel   string `json:"channel_id"` // Matched backend JSON tag
	Text      string `json:"text"`
	Severity  string `json:"severity"`
	CreatedAt string `json:"created_at"`
}

type model struct {
	messages []Message
	err      error
}

type tickMsg time.Time

// ---------------- BUBBLE TEA ЛОГИКА ----------------

func initialModel() model {
	return model{
		messages: []Message{},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		fetchMessagesCmd(),
		tickCmd(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case []Message:
		m.messages = msg
		return m, nil

	case error:
		m.messages = []Message{}
		m.err = msg
		return m, nil

	case tickMsg:
		return m, tea.Batch(
			fetchMessagesCmd(),
			tickCmd(),
		)
	}

	return m, nil
}

func (m model) View() string {
	s := titleStyle.Render(fmt.Sprintf("⚡ NERVE MONITOR: #%s", channelID)) + "\n\n"

	if m.err != nil {
		s += fmt.Sprintf("Error: %v\n", m.err)
	} else if len(m.messages) == 0 {
		s += "Waiting for data...\n"
	} else {
		for _, msg := range m.messages {
			sev := msg.Severity
			if sev == "critical" {
				sev = severityCritical.Render("[CRITICAL]")
			} else {
				sev = severityInfo.Render("[INFO]")
			}
			
			line := fmt.Sprintf("%s %s: %s", 
				sev, 
				severityTime.Render(msg.CreatedAt), 
				msgStyle.Render(msg.Text))
			
			s += line + "\n"
		}
	}

	s += "\nPress 'q' to quit.\n"
	return s
}

// ---------------- ФУНКЦИИ ----------------

func main() {
	host := flag.String("host", "http://localhost:8080", "Nerve API Host")
	channel := flag.String("channel", "dev-local", "Channel ID to monitor")
	flag.Parse()

	channelID = *channel
	apiURL = fmt.Sprintf("%s/api/v1/messages?channel=%s", *host, *channel)

	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func fetchMessagesCmd() tea.Cmd {
	return func() tea.Msg {
		// Needs Auth? The backend now requires auth.
		// The CLI didn't have auth logic.
		// For "Product Ready", CLI needs auth.
		// But adding full WebAuthn flow to CLI is hard.
		// I'll assume for now this CLI is for "public" or "agent" usage if I added token support.
		// But since I enabled auth middleware on /api/v1/messages, this CLI will fail 401.
		
		// I must add token support.
		token := os.Getenv("NERVE_TOKEN")
		if token == "" {
			return fmt.Errorf("NERVE_TOKEN env var required")
		}

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil { return err }
		req.Header.Set("Authorization", "Bearer " + token)
		
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("API Error: %s", resp.Status)
		}

		var msgs []Message
		if err := json.NewDecoder(resp.Body).Decode(&msgs); err != nil {
			return err
		}
		return msgs
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}