package websocketApp

// Hub (1 goroutine)
//     ├── register chan *Client
//     ├── unregister chan *Client
//     └── broadcast chan []byte

// Each Client (2 goroutines per connection)
//     ├── readPump → reads from conn, sends to broadcast
//     └── writePump ← reads from client.send chan, writes to conn

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	baseKafka "github.com/himanshu3889/discore-backend/base/infrastructure/kafka"
	redisDatabase "github.com/himanshu3889/discore-backend/base/infrastructure/redis"
	"github.com/himanshu3889/discore-backend/configs"

	"github.com/bwmarrin/snowflake"
	"github.com/go-redis/redis_rate/v10"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Constants for connection management
const (
	pingPeriod   = 25 * time.Second
	pingInterval = 10 * time.Second
	missedPings  = 2

	writeWait = 10 * time.Second

	// Time to wait for a pong response
	pongWait = missedPings*pingPeriod + writeWait
)

type UserID = snowflake.ID
type ChannelID = snowflake.ID

// Client represents a single connected WebSocket client.
type Client struct {
	conn   *websocket.Conn
	send   chan *websocket.PreparedMessage // Channel to send messages to the client; // This can panic if you try to close again
	done   chan struct{}                   // signal to unregister the client
	userID UserID
	room   string //  the topic
}

// Subscriptions
type RoomRequest struct {
	client *Client
	name   string // room:unique_id
}

// RoomState holds per-room data with its own lock
type RoomState struct {
	name      string
	clients   map[*Client]bool
	outBuffer chan *BroadcastRequest // for broadcasting
	mu        sync.RWMutex           // Per-room lock
	typing    TypingCoalescer
}

type BroadcastRequest struct {
	Event  EventType        `json:"event"`
	Room   string           `json:"room"`
	Data   *json.RawMessage `json:"data"`
	Action *string          `json:"action"`

	// Internal: When did this message enter the system?
	PipelineStart time.Time `json:"-"`
}

type SubscribePool struct {
	workers int
	queue   chan *RoomRequest
	wg      sync.WaitGroup
}

// Constants for buffer and queue managements
const (
	subscribePoolQueueCnt = 10
	registerBufferLen     = 100
	unregisterBufferLen   = 100
	roomOutBufferLen      = 100
	clientBufferSize      = 20
)

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	rooms        map[string]*RoomState // Room states
	subscribe    *SubscribePool        //
	totalClients int32                 // total client connections

	register   chan *Client // Register clients to the hub
	unregister chan *Client // Unregister client from hub; cleanup its stuff from room

	// Queue producer and consumer
	producer        *baseKafka.KafkaProducer
	consumerManager *baseKafka.ConsumerManager

	limiter *redis_rate.Limiter // Rate limiting

	ctx context.Context // context
	wg  sync.WaitGroup  // wait group
	mu  sync.RWMutex    // Locking
}

// NewHub creates and returns a new Hub instance.
func newHub() *Hub {
	brokers := strings.Split(configs.Config.KAFKA_BROKERS, ",")
	redisClient := redisDatabase.RedisClient

	subscribePool := &SubscribePool{
		workers: 10,
		queue:   make(chan *RoomRequest, subscribePoolQueueCnt),
	}

	return &Hub{
		rooms:           make(map[string]*RoomState),
		register:        make(chan *Client, registerBufferLen),
		unregister:      make(chan *Client, unregisterBufferLen),
		subscribe:       subscribePool,
		producer:        baseKafka.NewProducer(brokers),
		consumerManager: baseKafka.NewConsumerManager("websocket-hub"),
		limiter:         redis_rate.NewLimiter(redisClient),
		ctx:             context.Background(),
		wg:              sync.WaitGroup{},
	}
}

// Global hub instance
var (
	globalHub *Hub
	once      sync.Once
)

// Initialize websocket hub
func InitializeHub(ctx context.Context) {
	once.Do(func() {
		// hubCtx, cancel := context.WithCancel(ctx)
		globalHub = newHub()

		// Start the hub's event loop
		go globalHub.run()
	})
}

// run starts the hub's main event loop.
func (hub *Hub) run() {
	logrus.Info("Running websocket hub...")

	// Start the subscriber workers
	go hub.InitSubscribeWorkers()

	// Start the consumers
	go hub.KafkaBroadcastConsumer()

	for {
		select {

		case _, ok := <-hub.register: // to register the client; not room joining
			if !ok {
				// Closed
				return
			}

			atomic.AddInt32(&hub.totalClients, 1)

			// [METRIC]
			hub.MetricTrackConnect(true)

			// logrus.Infof("Client registered: %s (Total: %d)", client.conn.RemoteAddr(), hub.totalClients)

		case client, ok := <-hub.unregister:
			if !ok {
				// Closed
				return
			}
			// Safely access
			hub.mu.RLock()
			roomState := hub.rooms[client.room]
			hub.mu.RUnlock()

			if roomState != nil {
				roomState.RemoveClients([]*Client{client})
			}

			atomic.AddInt32(&hub.totalClients, -1)

			// [METRIC]
			hub.MetricTrackConnect(false)

			// logrus.Infof("Client unregistered: %s (Total: %d)", client.conn.RemoteAddr(), hub.totalClients)

			// 	//  Now Workers will handle the subscriptions
			// case roomRequest := <-hub.subscribe.queue:
			// 	hub.BuildRoomBroadcaster(roomRequest.name)
			// 	hub.SubscribeRoom(roomRequest)

		}
	}
}

// Get a new room state
func (hub *Hub) NewRoomState(name string) *RoomState {
	return &RoomState{
		name:      name,
		clients:   make(map[*Client]bool),
		outBuffer: make(chan *BroadcastRequest, roomOutBufferLen),
		typing: TypingCoalescer{
			typers: make(map[UserID]bool),
		},
	}
}

// Get or Create room without safety; take hub lock before using this
func (hub *Hub) GetOrCreateRoom(name string) *RoomState {

	// Return existing if found
	if room, exists := hub.rooms[name]; exists {
		return room
	}

	// Create and Register new room
	newRoom := hub.NewRoomState(name)
	hub.rooms[name] = newRoom

	// [METRIC] : Increment Metric
	hub.MetricTrackRoom(true)

	return newRoom
}

// Cleanup when hub shuts down
func (hub *Hub) Shutdown() {
	if hub.producer != nil {
		hub.producer.Close()
	}
}
