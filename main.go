package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------- КОНФИГ ----------------
const API_URL = "http://localhost:8080/api/v1/messages?channel=dev-local"

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
// То же самое, что на бэкенде
type Message struct {
	ID        int    `json:"id"`
	Channel   string `json:"channel"`
	Text      string `json:"text"`
	Severity  string `json:"severity"`
	CreatedAt string `json:"created_at"`
}

type model struct {
	messages []Message
	err      error
}

// Сообщение для обновления цикла (Tick)
type tickMsg time.Time

// ---------------- BUBBLE TEA ЛОГИКА ----------------

func initialModel() model {
	return model{
		messages: []Message{},
	}
}

// Init запускает таймер сразу при старте
func (m model) Init() tea.Cmd {
	return tea.Batch(
		fetchMessagesCmd(),
		tickCmd(),
	)
}

// Update обрабатывает события (нажатие кнопок, прилет данных, таймер)
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Нажали кнопку
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	// Прилетели сообщения с сервера
	case []Message:
		m.messages = msg
		return m, nil

	// Ошибка
	case error:
		m.messages = []Message{}
		m.err = msg
		return m, nil

	// Таймер сработал (прошло 2 сек) -> запускаем новый феч и новый таймер
	case tickMsg:
		return m, tea.Batch(
			fetchMessagesCmd(),
			tickCmd(),
		)
	}

	return m, nil
}

// View рисует интерфейс
func (m model) View() string {
	s := titleStyle.Render("⚡ NERVE MONITOR: #dev-local") + "\n\n"

	if m.err != nil {
		s += fmt.Sprintf("Error: %v\n", m.err)
	} else if len(m.messages) == 0 {
		s += "Waiting for data...\n"
	} else {
		for _, msg := range m.messages {
			// Красим уровень опасности
			sev := msg.Severity
			if sev == "critical" {
				sev = severityCritical.Render("[CRITICAL]")
			} else {
				sev = severityInfo.Render("[INFO]")
			}
			
			// Формируем строку
			// [INFO] 2025-12-14... : Text
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
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

// Команда: Сходи на сервер и возьми JSON
func fetchMessagesCmd() tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get(API_URL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var msgs []Message
		if err := json.NewDecoder(resp.Body).Decode(&msgs); err != nil {
			return err
		}
		return msgs
	}
}

// Команда: Подожди 2 секунды
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
