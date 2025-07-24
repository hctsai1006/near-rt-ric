package o1

import (
	"fmt"
)

// MessageHandler is a concrete implementation of NetconfMessageHandler.
type MessageHandler struct {
	datastore *NetconfDatastore
}

// NewMessageHandler creates a new MessageHandler.
func NewMessageHandler(datastore *NetconfDatastore) *MessageHandler {
	return &MessageHandler{datastore: datastore}
}

// HandleGetConfig handles the get-config operation.
func (h *MessageHandler) HandleGetConfig(sessionID uint32, req *GetConfigRequest) (interface{}, error) {
	return h.datastore.GetConfig(req.Datastore, req.Filter)
}

// HandleEditConfig handles the edit-config operation.
func (h *MessageHandler) HandleEditConfig(sessionID uint32, req *EditConfigRequest) error {
	return h.datastore.EditConfig(req.Datastore, req.Config, req.DefaultOperation)
}

// HandleGet handles the get operation.
func (h *MessageHandler) HandleGet(sessionID uint32, filter string) (interface{}, error) {
	// For now, we'll just treat get as get-config on the running datastore.
	return h.datastore.GetConfig(DatastoreRunning, filter)
}

// HandleCopyConfig handles the copy-config operation.
func (h *MessageHandler) HandleCopyConfig(sessionID uint32, source, target DatastoreType) error {
	return fmt.Errorf("copy-config not implemented")
}

// HandleDeleteConfig handles the delete-config operation.
func (h *MessageHandler) HandleDeleteConfig(sessionID uint32, target DatastoreType) error {
	return fmt.Errorf("delete-config not implemented")
}

// HandleLock handles the lock operation.
func (h *MessageHandler) HandleLock(sessionID uint32, target DatastoreType) error {
	return h.datastore.Lock(target, fmt.Sprintf("%d", sessionID))
}

// HandleUnlock handles the unlock operation.
func (h *MessageHandler) HandleUnlock(sessionID uint32, target DatastoreType) error {
	return h.datastore.Unlock(target, fmt.Sprintf("%d", sessionID))
}

// HandleCloseSession handles the close-session operation.
func (h *MessageHandler) HandleCloseSession(sessionID uint32) error {
	// The session is closed by the server, so we don't need to do anything here.
	return nil
}

// HandleKillSession handles the kill-session operation.
func (h *MessageHandler) HandleKillSession(sessionID uint32, targetSessionID uint32) error {
	return fmt.Errorf("kill-session not implemented")
}
