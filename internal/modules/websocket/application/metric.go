package websocketApp

import (
	"time"

	websocketMetrics "github.com/himanshu3889/discore-backend/internal/modules/websocket/metric"
)

// trackConnect updates connection counts
func (h *Hub) MetricTrackConnect(joining bool) {
	if joining {
		websocketMetrics.ActiveConnections.Inc()
	} else {
		websocketMetrics.ActiveConnections.Dec()
	}
}

// trackRoom updates room counts
func (h *Hub) MetricTrackRoom(created bool) {
	if created {
		websocketMetrics.ActiveRooms.Inc()
	} else {
		websocketMetrics.ActiveRooms.Dec()
	}
}

// recordBroadcastMetrics handles the latency and throughput tracking
func (h *Hub) MetricRecordBroadcast(eventType EventType, broadcastStart time.Time, pipelineStart time.Time) {

	// Record broadcast Latency
	websocketMetrics.BroadcastDuration.Observe(time.Since(broadcastStart).Seconds())

	// Record Throughput
	websocketMetrics.MessagesSent.WithLabelValues(string(eventType)).Inc()

	// Record overall latency
	// Only record if PipelineStart is set (not zero)
	if !pipelineStart.IsZero() {
		totalLatency := time.Since(pipelineStart).Seconds()
		websocketMetrics.PipelineLatency.WithLabelValues(string(eventType)).Observe(totalLatency)
	}

}

// handles the subscribe queue depth
func (h *Hub) MetricSubscribeQueueDepth(workerPicked bool) {
	if workerPicked {
		websocketMetrics.SubscribeQueueDepth.Dec()
	} else {
		websocketMetrics.SubscribeQueueDepth.Inc()
	}
}

// handles the latency and throughput tracking
func (h *Hub) MetricSubscribeQueueDrop() {
	websocketMetrics.SubscribeTimeouts.Inc()
}

func (h *Hub) MetricRoomTyping() {
	websocketMetrics.TypingCoalesced.Inc()
}
