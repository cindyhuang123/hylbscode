package agent

import (
	"testing"

	"github.com/cindyhuang123/hylbscode/internal/message"
)

func TestEmptyAssistant(t *testing.T) {
	finishOnly := message.Message{Role: message.Assistant}
	finishOnly.AddFinish(message.FinishReasonError)

	withText := message.Message{Role: message.Assistant}
	withText.AppendContent("hello")

	withToolCall := message.Message{Role: message.Assistant}
	withToolCall.AddToolCall(message.ToolCall{ID: "call_1", Name: "ls"})

	withReasoning := message.Message{Role: message.Assistant}
	withReasoning.AppendReasoningContent("thinking")

	userMsg := message.Message{Role: message.User}
	userMsg.AppendContent("hi")

	cases := []struct {
		name string
		msg  message.Message
		want bool
	}{
		{"assistant finish only is empty", finishOnly, true},
		{"assistant with text is not empty", withText, false},
		{"assistant with tool call is not empty", withToolCall, false},
		{"assistant with reasoning is not empty", withReasoning, false},
		{"user message is never dropped", userMsg, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := emptyAssistant(c.msg); got != c.want {
				t.Errorf("emptyAssistant() = %v, want %v", got, c.want)
			}
		})
	}
}
