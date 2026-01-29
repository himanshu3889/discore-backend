package websocketApp

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Connection metrics
	activeConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "websocket_active_connections",
		Help: "Current number of active WebSocket connections",
	})

	// Message throughput
	messagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "websocket_messages_total",
			Help: "Total messages by direction",
		},
		[]string{"direction"}, // "inbound" or "outbound"
	)

	// Processing latency
	messageDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "websocket_message_duration_ms",
		Help:    "Time from receive to broadcast",
		Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1ms to ~1s
	})

	// Errors
	connectionErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "websocket_errors_total",
		Help: "Total connection errors",
	})
)

// func WsHandler(ctx *gin.Context) {
// 	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
// 	if err != nil {
// 		connectionErrors.Inc()
// 		logrus.WithFields(logrus.Fields{
// 			"event": "ERROR",
// 		}).WithError(err).Error("Failed to upgrade WebSocket connection")
// 		return
// 	}

// 	// Register client and increment metrics
// 	mutex.Lock()
// 	clients[conn] = true
// 	activeConnections.Inc()
// 	totalClients := len(clients)
// 	mutex.Unlock()

// 	logrus.WithFields(logrus.Fields{
// 		"event":          "CONNECT",
// 		"remote_address": conn.RemoteAddr(),
// 		"total_clients":  totalClients,
// 	}).Info("WebSocket client connected")

// 	// Cleanup on disconnect
// 	defer func() {
// 		mutex.Lock()
// 		delete(clients, conn)
// 		activeConnections.Dec()
// 		totalClients := len(clients)
// 		mutex.Unlock()

// 		conn.Close()

// 		logrus.WithFields(logrus.Fields{
// 			"event":          "DISCONNECT",
// 			"remote_address": conn.RemoteAddr(),
// 			"total_clients":  totalClients,
// 		}).Info("WebSocket client disconnected")
// 	}()

// 	for {
// 		recvStart := time.Now()
// 		_, message, err := conn.ReadMessage()
// 		if err != nil {
// 			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
// 				logrus.WithFields(logrus.Fields{
// 					"event":          "READ_ERROR",
// 					"remote_address": conn.RemoteAddr(),
// 				}).Warn("Unexpected WebSocket close")
// 			}
// 			break
// 		}

// 		// Track metrics (no per-message Info log)
// 		messagesTotal.WithLabelValues("inbound").Inc()
// 		queueDuration := time.Since(recvStart)

// 		logrus.WithFields(logrus.Fields{
// 			"event":         "MESSAGE",
// 			"client":        conn.RemoteAddr(),
// 			"message_size":  len(message),
// 			"queue_ms":      queueDuration.Milliseconds(),
// 		}).Debug("Message received")

// 		broadcast <- message
// 	}
// }

// func HandleWebsocketMessages() {
// 	for {
// 		message := <-broadcast

// 		// Get client count safely
// 		mutex.Lock()
// 		clientCount := len(clients)
// 		mutex.Unlock()

// 		broadcastStart := time.Now()

// 		// Broadcast to all clients
// 		mutex.Lock()
// 		for client := range clients {
// 			err := client.WriteMessage(websocket.TextMessage, message)
// 			if err != nil {
// 				logrus.WithFields(logrus.Fields{
// 					"event":          "WRITE_ERROR",
// 					"remote_address": client.RemoteAddr(),
// 				}).Warn("Failed to send message")
// 				client.Close()
// 			}
// 		}
// 		mutex.Unlock()

// 		// Track outbound metrics
// 		messagesTotal.WithLabelValues("outbound").Add(float64(clientCount))
// 		broadcastDuration := time.Since(broadcastStart)
// 		messageDuration.Observe(float64(broadcastDuration.Milliseconds()))

// 		logrus.WithFields(logrus.Fields{
// 			"event":          "BROADCAST",
// 			"clients":        clientCount,
// 			"duration_ms":    broadcastDuration.Milliseconds(),
// 			"message_size":   len(message),
// 		}).Debug("Broadcast completed")
// 	}
// }
