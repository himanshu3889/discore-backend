package websocketApp

import (
	userCacheStore "discore/internal/modules/websocket/cacheStore/user"
	"encoding/json"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
	"golang.org/x/net/context"
)

// Typer state that need to be broadcast
type Typer struct {
	ID   UserID `json:"id"`
	Name string `json:"name"`
}

// Room typing coalescer
type TypingCoalescer struct {
	typers map[UserID]bool
	total  int
	timer  *time.Timer
	mu     sync.Mutex
}

const (
	flushTypingDelay = 500 * time.Millisecond
	broadcastTimeout = 350 * time.Millisecond
	maxTrackedTypers = 4
)

// Add the user in the room typing state
func (room *RoomState) AddTyper(userID UserID) {
	room.typing.mu.Lock()
	defer room.typing.mu.Unlock()

	if _, exists := room.typing.typers[userID]; exists {
		return
	}

	if len(room.typing.typers) < maxTrackedTypers {
		room.typing.typers[userID] = true
	}
	room.typing.total++

	if room.typing.timer != nil {
		room.typing.timer.Stop()
	}
	room.typing.timer = time.AfterFunc(flushTypingDelay, room.flushTyping)
}

// Flush the typing in the room
func (room *RoomState) flushTyping() {
	// Load shedding
	// Check if room's outBuffer is nearly full
	if len(room.outBuffer) >= cap(room.outBuffer)*9/10 {
		return
	}

	// Grab data quickly, release lock
	room.typing.mu.Lock()
	typerIDs := make([]snowflake.ID, 0, len(room.typing.typers))
	for id := range room.typing.typers {
		typerIDs = append(typerIDs, id)
	}
	total := room.typing.total
	room.typing.typers = make(map[UserID]bool)
	room.typing.total = 0
	room.typing.timer = nil
	room.typing.mu.Unlock()

	if total == 0 {
		return
	}

	// Now do work without lock
	ctx := context.Background()
	usersMap, err := userCacheStore.GetUsersBatch(ctx, typerIDs)
	if err != nil {
		return
	}

	typersList := make([]Typer, 0, len(typerIDs))
	for _, id := range typerIDs {
		name := ""
		if user, ok := usersMap[id]; ok {
			name = user.Name
		}
		typersList = append(typersList, Typer{ID: id, Name: name})
	}

	data, _ := json.Marshal(map[string]interface{}{
		"users": typersList,
		"total": total,
	})
	raw := json.RawMessage(data)

	req := &BroadcastRequest{
		Event: EventRoomTyping,
		Room:  room.name,
		Data:  &raw,
	}

	// Non-blocking send with timeout
	select {
	case room.outBuffer <- req:
		// Sent
	case <-time.After(broadcastTimeout):
		// dropped
	}
}
