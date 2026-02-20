
# 📚 Discore – Backend

**Discore** is a backend system designed to simulate how a production-scale chat service behaves under heavy load.

The goal of this project is not feature completeness, but **architectural correctness under stress**: handling high connection counts, bursty traffic, and large fan-out workloads. It explores practical distributed systems patterns such as message streaming, request coalescing, rate limiting, and WebSocket session management.

![System Design Architecture](https://drive.google.com/thumbnail?id=1l7gvNXhPW8277iZFMiOoQteJFKrEgJkV&sz=w5000)

> **Looking for the frontend?** Check out the [Discore Frontend Repository](https://github.com/himanshu3889/discore-frontend).
---

## 🏗 Architecture & The "Why"

This system follows a **Modular Monolith** architecture. While it runs as a single deployable unit for operational simplicity, the internal domain logic (Auth, Chat, Guilds) is strictly isolated. This approach allows for efficient local development while ensuring that future extraction into microservices is straightforward.

### Why Go?
Go fits the workload pattern of chat systems: many idle connections with occasional massive bursts.
- **Concurrency:** Goroutines are lightweight enough to support tens of thousands of concurrent WebSocket sessions on a single node.
- **Control:** Channels and `context` primitives make cancellation, backpressure, and lifecycle management straightforward.
- **Performance:** Predictable latency and low memory overhead compared to managed runtimes.

### Why PostgreSQL?
Used for **relational data** (Users, Sessions, Servers, Channels, Memberships, Conversations, Server invites).
- **Consistency:** Strong consistency is required for identity and authorization.
- **Integrity:** Relational constraints prevent invalid states (e.g., orphan memberships).
- **Complex Queries:** Joins are occasionally needed for efficient permission resolution (e.g., "Get all channels user X can see").

### Why MongoDB for Messages?
Chat history is write-heavy, append-only, and rarely updated.
- **Flexible Schema:** Document storage avoids schema migration friction for message metadata/attachments.
- **Write Throughput:** Horizontal scaling and sharding are easier for large collections.


### Why Kafka?
Kafka handles the **Fan-Out on Write** problem.
- **The Problem:** When a user sends a message in a large server, synchronously delivering it to 100k connected clients would block the sender and overload the API node.
- **The Solution:** The API produces a message event to Kafka. Consumer workers handle the distribution asynchronously. This decouples ingestion from delivery and ensures the API layer remains responsive.

### Why Redis?
Redis serves three distinct roles:
1.  **Standard Caching:** Server, User, Channel and Invite lookups to reduce DB round trips.
2.  **Redis Cell (Rate Limiting):** Implements the GCRA algorithm (Generic Cell Rate Algorithm). Unlike fixed-window limits, GCRA provides smooth traffic shaping without "burst" edges.
3.  **Redis Bloom:** Used for fast existence checks (e.g., "Does this Invite ID exist?"). This protects the database from query floods for invalid keys.

---

### Why Prometheus?

Prometheus handles the **"Is my backend healthy?"** problem.

- **The Problem:** Your Discord backend has WebSocket connections, message handlers, and API endpoints. When users report lag or dropped messages, you don't know if it's high CPU, memory leaks, or goroutine spikes. You're blind to performance until someone complains.

- **The Solution:** Prometheus scrapes metrics from your Go backend every 15 seconds. You track goroutine count, memory usage, WebSocket connection count, message throughput, and API latency. When message delivery slows down, you see exactly if it's a resource bottleneck or a handler issue.

---

### Why Grafana?

Grafana serves three distinct roles:

1. **Live Dashboards:** Turns Prometheus metrics into real-time charts showing active WebSocket connections, messages per second, and API response times. You see your Discord backend health at a glance.

2. **Alert Notifications:** Sends Discord/Slack alerts when WebSocket connections drop suddenly or memory usage spikes. You catch issues before users notice lag.

3. **Query Interface:** Lets you explore metrics with PromQL. When investigating slow message delivery, you filter by handler, time range, or specific channels without writing code.

---

### Why Loki?

Loki handles the **"Find that error in my WebSocket handler logs"** problem.

- **The Problem:** Your backend generates logs for connection events, message processing, and errors. When a user reports a failed message send, searching through log files with `grep` is slow. Storing everything in a heavy search engine costs too much for a side project.

- **The Solution:** Loki only indexes labels like service name, log level, or handler type—not every word. You filter by `handler=websocket` and `level=error` first, then search content. Storage stays cheap and you still find that connection timeout error fast. Works directly with Grafana so you click from a metric spike to the exact log lines that caused it.

---

## 🛠 Deep Dive - Engineering Patterns

### Rate Limiting (Token Bucket)
To prevent spam, we use a distributed rate limiter via **Redis Cell**.
- Each user has a "token bucket" that replenishes at a fixed rate.
- Every message send attempt consumes a token.
- If the bucket is empty, the request is rejected immediately (429 Too Many Requests).
- **Benefit:** No local state is required on the Go servers, allowing consistent enforcement across a cluster.

### Request Coalescing (Typing Indicators)
**The Problem:** Typing indicators generate massive noise. If 1,000 users type, sending 1,000 individual events to all subscribers causes an $O(N^2)$ network flood.

**The Strategy:**
- The server does not broadcast every "start typing" event immediately.
- Instead, it aggregates these events into a short time window (e.g., 500ms).
- A single consolidated update (e.g., "User A, B, and C are typing") is emitted to the channel.

### Cache-Stampede Prevention
**The Problem:** If a popular cache key expires (e.g., metadata for a massive server), thousands of concurrent requests might try to hit the DB simultaneously to refresh it.
**The Solution:** We use **Singleflight** (via `golang.org/x/sync/singleflight`).
- This pattern suppresses duplicate function calls.
- If 1,000 routines request the same key, only *one* actually calls the database. The result is shared with all 1,000 waiting routines.

### Cache Penetration Protection
**The Problem:** Malicious actors may repeatedly query random or non-existent IDs to bypass cache layers and overload the database.
**The Solution:** **Bloom Filters + Negative Caching**.
- Before querying Postgres, the system first checks a Redis Bloom filter.
- If the filter returns **"No"**, the ID definitely does not exist → immediately return `404` without touching the database.
- If the filter returns **"Maybe"**, the request proceeds to the cache layer.
    - If not found in cache → query Postgres.
    - If DB confirms it does not exist → store a short-TTL **null cache entry** (negative cache).

### Backpressure & Bounded Buffers
**The Problem:** If a client is on a slow connection, the server's outbound message queue for that client will grow indefinitely, eventually consuming all available RAM (OOM).
**The Solution:**
- Every WebSocket connection has a bounded Go channel (buffer).
- If the buffer fills up (client cannot read fast enough), the server creates backpressure.
- We deliberately **drop the connection** rather than slowing down the entire server. This creates a hard limit on the memory footprint per user.

### Timeout Pattern
Every external call (DB, Redis) is wrapped in a Go `context.WithTimeout`.
- This ensures that if a subsystem stalls, it doesn't cause cascading latency (thread starvation) across the entire platform.
- Resources are freed immediately when the deadline is exceeded.

### gRPC
Authentication is handled separately at a different service, so gRPC is used for fast communication between the two services.


### REST Api's
Client services for User, Server, Channel, Member, and Messages.


---

## 📝 Roadmap (To-Do)

Future improvements planned to enhance scalability and user experience.

* [x] **Performance Monitoring & Charts** (Prometheus/Grafana)
* [ ] **Elasticsearch Message Search** (Full-text search across message history)
* [ ] **Mongodb to Scylladb** (Impove the read and write latency of the messages using Scylladb)
* [ ] **User Online Status & Presence System** (Real-time online/offline/idle tracking)
* [ ] **Smart Notifications** (Batching `@all` / `@here` mentions to prevent notification storms)
* [ ] **System Resiliency & Fault Tolerance** (Circuit Breakers)

---

## 🚀 How to Run

### Prerequisites
* Go 1.22+
* Docker & Docker Compose

### Quick Start

1.  **Clone the repository:**
    ```bash
    git clone [https://github.com/himanshu3889/discore-backend.git](https://github.com/himanshu3889/discore-backend.git)
    cd discore-backend
    ```

2.  **Start Infrastructure:**

    ```Build the env file as the .env.example```
    
    Spin up Redis, Postgres, MongoDB, and Kafka.
    
    using makerfile
    ```bash
    make up-sys
    ```
    or directly with docker compose
    ```bash
    docker compose up -d postgres mongodb redis kafka
    ```

    Spin up Prometheus, Grafana and Loki.
    
    using makerfile
    ```bash
    make up-mon
    ```
    or directly with docker compose
    ```bash
    docker compose up -d prometheus grafana loki
    ```

3.  **Run the Server:**
    
    With go run
    ```bash
    go run cmd/gateway/main.go
    go run cmd/modules/main.go
    ```

    With air
    ```
    air -c .air.gateway.toml
    air -c .air.modules.toml
    ```

    with maker command
    ```
    make air-gateway
    make air-modules
    ```

4.  **Connect:**
    * Gateway server: http://localhost:8090
    * Module server: http://localhost:8080
    * Prometheus: http://localhost:3000
    * Grafana: http://localhost:9090
    * Loki: http://localhost:3100

    ```REST APIs via gateway server: http://localhost:8090```
    
    ```WebSocket via module server: ws://localhost:8080/ws```

---

## 📜 Design Philosophy

This project prioritizes **correctness under load**, **isolation of responsibilities**, and **explicit failure handling**.

Feature completeness is intentionally secondary to architectural resilience. The goal is not just to build a chat app, but to build a system that stays up when the chat app goes viral.

---