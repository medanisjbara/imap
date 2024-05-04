package msgconv

import (
	"context"
)

// PortalMethods defines methods for interacting with the bridge's portal.
type PortalMethods interface {
	// Add methods here for uploading media, fetching replies, getting clients, etc.
}

// MessageConverter converts messages between different formats.
type MessageConverter struct {
	PortalMethods

	MaxFileSize int64
}

// IsPrivateChat determines whether the conversation is a private chat.
func (mc *MessageConverter) IsPrivateChat(ctx context.Context) bool {
	// Add logic to determine if it's a private chat.
	// Return true or false based on the logic.
	return true // Placeholder return, replace with actual logic
}
