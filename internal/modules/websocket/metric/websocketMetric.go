package websocketMetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// --- Health & Capacity ---

	ActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ws_connections_active",
		Help: "Current number of active WebSocket clients",
	})

	ActiveRooms = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ws_rooms_active",
		Help: "Current number of active chat rooms",
	})

	SubscribeQueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ws_subscribe_queue_depth",
		Help: "Current pending requests in the subscribe worker pool",
	})

	// COUNTER: Tracks dropped connections due to full/slow subscribe queue
	SubscribeTimeouts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ws_subscribe_timeouts_total",
		Help: "Total number of room subscription attempts that timed out",
	})

	// --- Throughput ---

	// Track volume of messages. Labels: "type" (chat, typing, presence)
	MessagesSent = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ws_messages_out_total",
		Help: "Total messages sent to clients",
	}, []string{"type"})

	// Track how many typing events were SAVED (not sent) by the coalescer
	TypingCoalesced = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ws_typing_coalesced_total",
		Help: "Number of typing events dropped/merged to save bandwidth",
	})

	// --- Latency (Performance) ---

	// How long it takes to fan-out a message to a room.
	// We removed the 'room' label. This tracks GLOBAL performance.
	BroadcastDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "ws_broadcast_duration_seconds",
		Help:    "Time taken to push a message to all clients in a room",
		Buckets: []float64{.0001, .0005, .001, .005, .01, .05, .1, .2}, // Focused on sub-millisecond speed
	})

	// Total end-to-end pipeline time (from Kafka consume -> Client receive)
	// End-to-end time (Producer -> Kafka -> Consumer -> Broadcast)
	PipelineLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ws_pipeline_latency_seconds",
		Help:    "Total time from Kafka Producer to Broadcast completion",
		Buckets: []float64{.01, .05, .1, .25, .5, 1, 2.5, 5},
	}, []string{"type"})
)
