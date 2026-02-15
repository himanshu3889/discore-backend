package websocketApp

import (
	"time"

	"github.com/gorilla/websocket"
)

// readPump reads messages from the WebSocket connection.
func (client *Client) ReadPump(hub *Hub) {
	defer func() {
		// Clean up the client connection when the goroutine exits
		// logrus.Infof("Client disconnected: %s", client.conn.RemoteAddr())
		client.conn.Close()
	}()

	client.conn.SetReadDeadline(time.Now().Add(pongWait)) // Add deadline
	client.conn.SetPongHandler(func(string) error {       // Add pong handler
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		select {
		case <-client.done: // LISTEN for shutdown
			// logrus.Warn("ReadPump: shutdown signal")
			return

		default:
			// Continue reading
		}

		_, recMessage, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// logrus.WithError(err).Error("Websocket read error")
			}
			break
		}

		// Rate limiting
		canProceed := hub.ApplyRateLimit(client)
		if !canProceed {
			// logrus.Warn("Rate limit exceeded!")
			continue
		}

		// based on event we will do some more things before broadcast
		hub.HandleIncomingMessage(client, recMessage)

		// No broadcasting directly

	}
}

// writePump writes messages to the WebSocket connection.
func (client *Client) WritePump() {
	// TODO: remove from outside
	ticker := time.NewTicker(pingInterval) // Ping interval
	defer func() {
		// logrus.Infof("Client disconnected by server: %s", client.conn.RemoteAddr())
		ticker.Stop()
		client.conn.Close()
	}()

	client.conn.EnableWriteCompression(true)
	client.conn.SetCompressionLevel(3) // 40% compression, will add CPU overhead but save bandwidth

	for {
		select {
		case preparedMsg, ok := <-client.send:
			if !ok {
				// Hub closed the channel
				client.conn.SetWriteDeadline(time.Now().Add(writeWait))
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WritePreparedMessage(preparedMsg); err != nil {
				// logrus.WithError(err).Error("Write error")
				return
			}
		case <-ticker.C:
			// Send ping messages to keep the connection alive
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				// logrus.WithError(err).Error("Websocket ping error")
				return
			}

		case <-client.done: // Listen for shutdown signal
			// logrus.Warn("WritePump: shutdown signal received")
			return
		}
	}
}
