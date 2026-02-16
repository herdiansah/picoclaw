// Package handlers provides reference implementations for integrating
// SQLite conversation persistence with PicoClaw's channel handlers.
//
// This file demonstrates how to use the internal/db/sqlite package
// within the existing Telegram channel handler (pkg/channels/telegram.go).
//
// Integration points:
//   - On incoming message: call sqlite.SaveMessage(userID, text)
//   - Before building prompt: call sqlite.GetHistory(userID, maxHistory)
//   - On bot reply: call sqlite.SaveMessage(userID, reply)
//   - On /forget command: call sqlite.DeleteHistory(userID)
//
// To integrate, add these calls into pkg/channels/telegram.go's
// handleMessage() method and the TelegramCommander in telegram_commands.go.
package handlers

import (
	"log"
	"os"
	"strconv"

	"github.com/sipeed/picoclaw/internal/db/sqlite"
)

// maxHistoryFromEnv reads MAX_HISTORY from environment, defaulting to 20.
func maxHistoryFromEnv() int {
	v := os.Getenv("MAX_HISTORY")
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return n
	}
	return 20
}

// SaveIncomingMessage persists a user's incoming message to the DB.
// Call this from the Telegram handler after receiving a message.
func SaveIncomingMessage(userID int64, text string) {
	if err := sqlite.SaveMessage(userID, text); err != nil {
		log.Printf("failed to save incoming message: %v", err)
	}
}

// SaveBotReply persists the bot's reply to the DB.
// Call this from the Telegram handler after generating a response.
func SaveBotReply(userID int64, reply string) {
	if err := sqlite.SaveMessage(userID, reply); err != nil {
		log.Printf("failed to save bot reply: %v", err)
	}
}

// GetConversationHistory retrieves recent messages for context building.
// Returns an empty slice (not an error) if the DB is unavailable, so the
// bot can continue to operate without history.
func GetConversationHistory(userID int64) []string {
	limit := maxHistoryFromEnv()
	history, err := sqlite.GetHistory(userID, limit)
	if err != nil {
		log.Printf("failed to get history for user %d: %v", userID, err)
		return []string{}
	}
	return history
}

// ForgetUserHistory deletes all messages for a user (/forget command).
// Returns true if successful, false if an error occurred.
func ForgetUserHistory(userID int64) bool {
	if err := sqlite.DeleteHistory(userID); err != nil {
		log.Printf("failed to delete history for user %d: %v", userID, err)
		return false
	}
	return true
}
