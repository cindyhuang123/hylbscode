package gui

import (
	"strings"

	"github.com/cindyhuang123/hylbscode/internal/message"
)

func roleLabel(role message.MessageRole) string {
	switch role {
	case message.User:
		return "You"
	case message.Assistant:
		return "Assistant"
	case message.Tool:
		return "Tool"
	case message.System:
		return "System"
	default:
		return string(role)
	}
}

func messageText(m message.Message) string {
	var sb strings.Builder
	for _, part := range m.Parts {
		switch p := part.(type) {
		case message.TextContent:
			sb.WriteString(p.Text)
		case message.ReasoningContent:
			if strings.TrimSpace(p.Thinking) != "" {
				sb.WriteString("Thinking: ")
				sb.WriteString(p.Thinking)
				sb.WriteString("\n")
			}
		case message.ToolCall:
			sb.WriteString("[tool: ")
			sb.WriteString(p.Name)
			sb.WriteString("]\n")
			sb.WriteString(p.Input)
			sb.WriteString("\n")
		case message.ToolResult:
			name := p.Name
			if name == "" {
				name = p.ToolCallID
			}
			if p.IsError {
				sb.WriteString("[tool ")
				sb.WriteString(name)
				sb.WriteString(" error]\n")
			} else {
				sb.WriteString("[tool ")
				sb.WriteString(name)
				sb.WriteString("]\n")
			}
			sb.WriteString(p.Content)
			sb.WriteString("\n")
		case message.ImageURLContent:
			sb.WriteString(p.URL)
			sb.WriteString("\n")
		case message.BinaryContent:
			sb.WriteString("[attachment: ")
			sb.WriteString(p.MIMEType)
			sb.WriteString("]\n")
		}
	}
	return sb.String()
}
