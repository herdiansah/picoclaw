package prompt

import "strings"

// BuildPrompt composes the conversation context for PicoClaw
func BuildPrompt(history []string, current string) string {
    // history is returned newest-first; reverse or format as needed
    var b strings.Builder
    b.WriteString("Conversation history:\n")
    for i := len(history)-1; i >= 0; i-- {
        b.WriteString(history[i] + "\n")
    }
    b.WriteString("\nUser: " + current + "\nAssistant:")
    return b.String()
}
