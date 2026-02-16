package handlers

import (
    "log"
    "yourmodule/internal/db/sqlite"
    "yourmodule/internal/prompt"
    // telegram lib imports...
)

// Example for webhook or polling handler
func HandleMessage(update *tgbotapi.Update) {
    if update.Message == nil {
        return
    }

    userID := update.Message.From.ID
    text := update.Message.Text

    // Persist incoming message
    if err := sqlite.SaveMessage(int64(userID), text); err != nil {
        log.Printf("failed to save message: %v", err)
    }

    // Build context for PicoClaw
    history, err := sqlite.GetHistory(int64(userID), 20)
    if err != nil {
        log.Printf("failed to get history: %v", err)
        history = []string{}
    }

    promptText := prompt.BuildPrompt(history, text) // assemble history + current message
    reply := GenerateReply(promptText)              // call your PicoClaw generator

    // Optionally save bot reply as well
    if err := sqlite.SaveMessage(int64(userID), reply); err != nil {
        log.Printf("failed to save bot reply: %v", err)
    }

    // Send reply back to Telegram
    // sendMessageToTelegram(userID, reply)
}
