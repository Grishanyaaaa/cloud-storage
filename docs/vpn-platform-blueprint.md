# VPN Platform Blueprint

## Production-Ready Stealth VPN SaaS with Telegram Bot Management

**Version:** 1.0  
**Stack:** Go 1.22+, PostgreSQL 16, Redis 7, gRPC, Kubernetes  
**Architecture:** Clean Architecture, Domain-Driven Design  

---

## Table of Contents

1. [High-Level Architecture](#1-high-level-architecture)
2. [Control Plane / Data Plane](#2-control-plane--data-plane)
3. [VPN Transport Design](#3-vpn-transport-design)
4. [Node Lifecycle](#4-node-lifecycle)
5. [Capacity Calculation Algorithms](#5-capacity-calculation-algorithms)
6. [Database Schema](#6-database-schema)
7. [gRPC Contracts](#7-grpc-contracts)
8. [Telegram Bot Architecture](#8-telegram-bot-architecture)
9. [Billing Architecture](#9-billing-architecture)
10. [Security Architecture](#10-security-architecture)
11. [Observability Stack](#11-observability-stack)
12. [Kubernetes Deployment](#12-kubernetes-deployment)
13. [Failure Scenarios](#13-failure-scenarios)
14. [Scaling Strategy](#14-scaling-strategy)
15. [Future Extensibility](#15-future-extensibility)

---

## 1. High-Level Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         CONTROL PLANE                                │
│                                                                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────┐   │
│  │ Telegram │  │  Billing  │  │   Node   │  │   Config         │   │
│  │ Bot Svc  │  │  Service  │  │  Manager │  │   Distributor    │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────────┬─────────┘   │
│       │              │              │                  │             │
│  ┌────┴──────────────┴──────────────┴──────────────────┴────────┐   │
│  │                    API Gateway (gRPC + REST)                   │   │
│  └────┬──────────────┬──────────────┬──────────────────┬────────┘   │
│       │              │              │                  │             │
│  ┌────┴─────┐  ┌────┴─────┐  ┌────┴─────┐  ┌────────┴─────────┐   │
│  │   Auth   │  │   User   │  │ Subscrip │  │    Analytics     │   │
│  │  Service │  │  Service │  │  Service │  │    Service       │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────────┘   │
│                                                                     │
│  ┌─────────────┐  ┌─────────────┐  ┌────────────────────────┐     │
│  │ PostgreSQL  │  │    Redis    │  │    NATS JetStream      │     │
│  │   Cluster   │  │   Cluster   │  │    (Event Bus)         │     │
│  └─────────────┘  └─────────────┘  └────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                          DATA PLANE                                  │
│                                                                     │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐    │
│  │   VPN Node      │  │   VPN Node      │  │   VPN Node      │    │
│  │   (EU-West)     │  │   (US-East)     │  │   (AP-South)    │    │
│  │                 │  │                 │  │                 │    │
│  │ ┌─────────────┐ │  │ ┌─────────────┐ │  │ ┌─────────────┐ │    │
│  │ │ Node Agent  │ │  │ │ Node Agent  │ │  │ │ Node Agent  │ │    │
│  │ ├─────────────┤ │  │ ├─────────────┤ │  │ ├─────────────┤ │    │
│  │ │ Transport   │ │  │ │ Transport   │ │  │ │ Transport   │ │    │
│  │ │ Engine      │ │  │ │ Engine      │ │  │ │ Engine      │ │    │
│  │ ├─────────────┤ │  │ ├─────────────┤ │  │ ├─────────────┤ │    │
│  │ │ Session Mgr │ │  │ │ Session Mgr │ │  │ │ Session Mgr │ │    │
│  │ ├─────────────┤ │  │ ├─────────────┤ │  │ ├─────────────┤ │    │
│  │ │ Metrics     │ │  │ │ Metrics     │ │  │ │ Metrics     │ │    │
│  │ │ Exporter    │ │  │ │ Exporter    │ │  │ │ Exporter    │ │    │
│  │ └─────────────┘ │  │ └─────────────┘ │  │ └─────────────┘ │    │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

### Services Registry

| Service | Responsibility | Port (gRPC) | Port (HTTP) |
|---------|---------------|-------------|-------------|
| `api-gateway` | External routing, auth, rate limiting | — | 8080 |
| `auth-service` | JWT, sessions, device binding | 9001 | — |
| `user-service` | User profiles, referrals | 9002 | — |
| `subscription-service` | Plans, billing state, entitlements | 9003 | — |
| `billing-service` | Payment processing, invoices | 9004 | — |
| `node-manager` | Node registry, health, load balancing | 9005 | — |
| `config-distributor` | VPN config generation, key management | 9006 | — |
| `telegram-bot` | Telegram interface, FSM, notifications | 9007 | 8081 (webhook) |
| `analytics-service` | Usage stats, anomaly detection | 9008 | — |
| `node-agent` | On-node daemon, transport engine | 9010 | 9011 (metrics) |

---

## 2. Control Plane / Data Plane

### Control Plane

Responsible for:
- User lifecycle management
- Subscription/billing orchestration
- Node fleet management
- Config generation & distribution
- Policy enforcement
- Observability aggregation

**Communication:** gRPC with mTLS between all control plane services. NATS JetStream for async events.

**Event Bus Topics:**

```
vpn.user.created
vpn.user.deleted
vpn.subscription.activated
vpn.subscription.expired
vpn.subscription.renewed
vpn.node.registered
vpn.node.unhealthy
vpn.node.drained
vpn.session.started
vpn.session.ended
vpn.config.rotated
vpn.billing.payment_received
vpn.billing.payment_failed
vpn.security.abuse_detected
```

### Data Plane

Responsible for:
- VPN tunnel termination
- Traffic forwarding
- Session management per node
- Local metrics collection
- Health reporting

**Communication with Control Plane:**
- gRPC streaming for config updates (node-agent → config-distributor)
- gRPC unary for heartbeat (node-agent → node-manager)
- Prometheus pull from control plane metrics aggregator

### Separation Guarantees

```go
// Control Plane never handles user traffic
// Data Plane never stores persistent user data
// Config flows: Control → Data (push via gRPC stream)
// Metrics flow: Data → Control (pull via Prometheus)
// Events flow: Data → Control (push via NATS)
```

---

## 3. VPN Transport Design

### Selected Transport: REALITY + HTTP/2 Multiplexing with Browser Fingerprint Camouflage

#### Why This Stack

| Attack Vector | Mitigation |
|--------------|-----------|
| Protocol signature detection | TLS 1.3 REALITY — no custom ciphers, real server cert |
| Statistical traffic analysis | HTTP/2 framing + padding + multiplexed streams |
| Active probing | REALITY auth — unauthenticated connections see real HTTPS site |
| TLS fingerprinting | uTLS with browser impersonation (Chrome/Firefox/Safari) |
| Connection pattern analysis | Randomized connection pooling, keepalive jitter |
| SNI-based blocking | ECH (Encrypted Client Hello) with fallback to domain fronting |

#### Architecture

```
Client                                          VPN Node
  │                                                │
  │  ┌─────────────────────────────────────────┐   │
  │  │ 1. TLS 1.3 ClientHello                 │   │
  │  │    - uTLS Chrome 120 fingerprint        │   │
  │  │    - SNI: legitimate-domain.com         │   │
  │  │    - REALITY short ID in SessionID      │   │
  │  ├─────────────────────────────────────────┤   │
  │  │ 2. Server validates REALITY auth        │   │
  │  │    - If invalid → proxy to real site    │   │
  │  │    - If valid → establish tunnel        │   │
  │  ├─────────────────────────────────────────┤   │
  │  │ 3. HTTP/2 CONNECT established           │   │
  │  │    - Multiplexed streams                │   │
  │  │    - Random padding per frame           │   │
  │  ├─────────────────────────────────────────┤   │
  │  │ 4. VPN payload inside HTTP/2 DATA       │   │
  │  │    - ChaCha20-Poly1305 inner encryption │   │
  │  │    - Per-packet sequence numbers        │   │
  │  │    - Anti-replay window (2048 packets)  │   │
  │  └─────────────────────────────────────────┘   │
  │                                                │
```

#### REALITY Implementation

```go
package transport

// RealityConfig defines the REALITY transport parameters
type RealityConfig struct {
    // ServerName is the SNI of the camouflage destination
    ServerName    string        `json:"server_name"`
    // PrivateKey is the x25519 private key for REALITY auth
    PrivateKey    [32]byte      `json:"-"`
    // ShortIDs are valid 8-byte authentication identifiers
    ShortIDs      [][8]byte     `json:"short_ids"`
    // Dest is the real HTTPS backend for unauthenticated connections
    Dest          string        `json:"dest"`
    // MaxTimeDiff is the maximum allowed time difference for handshake
    MaxTimeDiff   time.Duration `json:"max_time_diff"`
    // FingerprintPool rotates browser fingerprints
    FingerprintPool []Fingerprint `json:"fingerprint_pool"`
}

type Fingerprint struct {
    Browser     string   // "chrome_120", "firefox_121", "safari_17"
    Extensions  []uint16
    Curves      []tls.CurveID
    CipherOrder []uint16
    ALPN        []string
}
```

#### Handshake Flow

```
1. Client generates ephemeral x25519 keypair
2. Client computes authKey = ECDH(ephemeral_private, server_public)
3. Client places shortID XOR'd with first 8 bytes of authKey into TLS SessionID
4. Client sends TLS 1.3 ClientHello with uTLS fingerprint
5. Server extracts SessionID, tries all shortIDs with ECDH verification
6. If no match → TCP proxy to Dest (real site serves genuine response)
7. If match → proceed with modified TLS handshake
8. Server generates ServerHello with REALITY-specific extensions
9. Both derive session keys: HKDF-SHA256(ECDH_shared, "reality-vpn-session")
10. HTTP/2 connection established over the TLS session
11. Client sends HTTP/2 CONNECT to pseudo-endpoint
12. Bidirectional tunnel established
```

#### Inner Encryption Layer

```go
type SessionCrypto struct {
    // Outer: TLS 1.3 (handled by REALITY)
    // Inner: Additional encryption for defense-in-depth
    SendKey     [32]byte  // ChaCha20-Poly1305
    RecvKey     [32]byte
    SendNonce   uint64    // Monotonically increasing
    RecvWindow  *ReplayWindow // Sliding window, 2048 packets
}

type ReplayWindow struct {
    Base    uint64
    Bitmap  [32]uint64 // 2048-bit sliding window
}

// Packet format inside HTTP/2 DATA frames:
// [2 bytes: length] [8 bytes: nonce] [N bytes: ciphertext] [16 bytes: Poly1305 tag]
// Padding: random 0-64 bytes appended before encryption
```

#### Key Rotation

```
- Session keys: rotated every 1 hour or 1 GB transferred (whichever first)
- REALITY shortIDs: rotated every 24 hours
- Server x25519 keys: rotated every 7 days
- uTLS fingerprints: rotated per connection from pool
- SNI targets: rotated weekly, validated against real sites
```

#### Anti-Active-Probing

```
1. No response difference for invalid vs. non-existent shortIDs
2. Timing-constant comparison for auth validation
3. Unauthenticated connections get full real website response
4. Connection behavior indistinguishable from legitimate HTTPS
5. Server supports HTTP/1.1, HTTP/2 for non-tunnel traffic
6. Valid TLS certificate for the camouflage domain (Let's Encrypt or stolen^Wborrowed)
7. Identical TCP window sizes, TTL, MSS to real web servers
```

#### Fallback Chain

```
Primary:   REALITY + HTTP/2 (port 443)
Fallback1: QUIC/HTTP3 with masquerading (port 443/udp)
Fallback2: WebSocket over CDN (Cloudflare/Fastly)
Fallback3: meek-style domain fronting
Fallback4: Raw TCP with obfs4 (last resort)
```

---

## 4. Node Lifecycle

### State Machine

```
┌──────────┐    register    ┌──────────┐    health_ok    ┌──────────┐
│DISCOVERED├───────────────►│REGISTERED├────────────────►│  ACTIVE  │
└──────────┘                └──────────┘                 └────┬─────┘
                                                              │
                            ┌──────────┐    overloaded   ┌────┴─────┐
                            │ DRAINING │◄────────────────┤ DEGRADED │
                            └────┬─────┘                 └──────────┘
                                 │
                            ┌────┴─────┐    removed      ┌──────────┐
                            │  DRAINED │────────────────►│ REMOVED  │
                            └──────────┘                 └──────────┘
```

### Auto-Registration Flow

```go
type NodeRegistrationRequest struct {
    NodeID        string            `json:"node_id"`       // UUID generated on first boot
    Region        string            `json:"region"`        // e.g. "eu-west-1"
    Country       string            `json:"country"`       // ISO 3166-1 alpha-2
    City          string            `json:"city"`
    Provider      string            `json:"provider"`      // "hetzner", "vultr", "aws"
    PublicIP      net.IP            `json:"public_ip"`
    Specs         NodeSpecs         `json:"specs"`
    TransportCaps []TransportType   `json:"transport_caps"`
    AuthToken     string            `json:"auth_token"`    // Pre-shared registration token
}

type NodeSpecs struct {
    CPUCores      int     `json:"cpu_cores"`
    CPUModel      string  `json:"cpu_model"`
    RAMBytes      int64   `json:"ram_bytes"`
    BandwidthMbps int     `json:"bandwidth_mbps"`
    DiskGB        int     `json:"disk_gb"`
    KernelVersion string  `json:"kernel_version"`
}
```

### Node Agent Bootstrap

```bash
#!/bin/bash
# node-agent bootstrap (runs on fresh VPS)

# 1. Download agent binary (signed, verified)
curl -sL https://releases.vpn-platform.internal/node-agent/latest/linux-amd64 \
  -o /usr/local/bin/node-agent
sha256sum -c /usr/local/bin/node-agent.sha256

# 2. Generate node identity
node-agent init --control-plane=cp.vpn-platform.internal:9005 \
  --registration-token=${REG_TOKEN}

# 3. Agent starts, registers, receives config via gRPC stream
systemctl enable --now node-agent
```

### Heartbeat Protocol

```go
type HeartbeatPayload struct {
    NodeID          string    `json:"node_id"`
    Timestamp       time.Time `json:"timestamp"`
    ActiveSessions  int       `json:"active_sessions"`
    CPUPercent      float64   `json:"cpu_percent"`
    MemoryPercent   float64   `json:"memory_percent"`
    BandwidthIn     int64     `json:"bandwidth_in_bps"`
    BandwidthOut    int64     `json:"bandwidth_out_bps"`
    P95LatencyMs    float64   `json:"p95_latency_ms"`
    PacketLossRate  float64   `json:"packet_loss_rate"`
    OpenConnections int       `json:"open_connections"`
    UptimeSeconds   int64     `json:"uptime_seconds"`
}

// Heartbeat interval: 10 seconds
// Miss threshold: 3 consecutive misses → DEGRADED
// Recovery: 5 consecutive successes → back to ACTIVE
```

### Graceful Drain

```go
func (n *NodeAgent) StartDrain(ctx context.Context) error {
    // 1. Stop accepting new connections
    n.listener.SetAcceptDeadline(time.Now())
    
    // 2. Notify control plane
    n.nodeManager.NotifyDrainStart(ctx, n.nodeID)
    
    // 3. Wait for existing sessions to terminate (max 30 min)
    deadline := time.Now().Add(30 * time.Minute)
    for n.sessionManager.ActiveCount() > 0 && time.Now().Before(deadline) {
        time.Sleep(5 * time.Second)
        // Send migration hints to clients
        n.sessionManager.BroadcastMigrationHint()
    }
    
    // 4. Force-close remaining sessions
    n.sessionManager.ForceCloseAll()
    
    // 5. Report drained
    n.nodeManager.NotifyDrained(ctx, n.nodeID)
    return nil
}
```

### Geo-Routing

```go
type GeoRouter struct {
    nodes     map[string]*NodeInfo  // nodeID → info
    geoIndex  *rtree.RTree         // spatial index for lat/lng
    latencyDB map[string]map[string]float64  // region→region latency matrix
}

func (g *GeoRouter) SelectNode(userRegion string, preferences UserPreferences) (*NodeInfo, error) {
    candidates := g.findCandidatesByRegion(userRegion)
    
    // Score each candidate
    scored := make([]ScoredNode, 0, len(candidates))
    for _, node := range candidates {
        score := g.calculateScore(node, userRegion, preferences)
        scored = append(scored, ScoredNode{Node: node, Score: score})
    }
    
    // Weighted random selection from top-3 (avoid thundering herd)
    sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
    top := scored[:min(3, len(scored))]
    return weightedRandom(top), nil
}

func (g *GeoRouter) calculateScore(node *NodeInfo, userRegion string, prefs UserPreferences) float64 {
    score := 100.0
    
    // Latency penalty (0-40 points)
    latency := g.latencyDB[userRegion][node.Region]
    score -= math.Min(40, latency/5.0)
    
    // Load penalty (0-30 points)
    loadRatio := float64(node.ActiveSessions) / float64(node.MaxSessions)
    score -= loadRatio * 30.0
    
    // Health bonus (0-20 points)
    if node.P95LatencyMs < 10 {
        score += 20
    } else if node.P95LatencyMs < 50 {
        score += 10
    }
    
    // Region preference match (0-10 points)
    if prefs.PreferredRegion == node.Region {
        score += 10
    }
    
    return score
}
```

---

## 5. Capacity Calculation Algorithms

### Node Capacity Model

Each node has a calculated `MaxSessions` based on hardware specs and real-time telemetry.

#### Static Capacity Formula

```
MaxSessions = min(
    BandwidthCapacity,
    CPUCapacity,
    RAMCapacity,
    ConnectionCapacity
)
```

Where:

```
BandwidthCapacity = (BandwidthMbps × BandwidthUtilTarget) / AvgSessionBandwidthMbps

CPUCapacity = (CPUCores × 1000 × CPUUtilTarget) / AvgCPUPerSessionMillicores

RAMCapacity = (TotalRAM - ReservedRAM) / AvgRAMPerSession

ConnectionCapacity = MaxFileDescriptors × ConnectionUtilTarget / ConnectionsPerSession
```

#### Constants & Coefficients

| Parameter | Value | Notes |
|-----------|-------|-------|
| `BandwidthUtilTarget` | 0.75 | Leave 25% headroom for bursts |
| `AvgSessionBandwidthMbps` | 5.0 | Measured p50 per active session |
| `CPUUtilTarget` | 0.70 | Target CPU at 70% to handle spikes |
| `AvgCPUPerSessionMillicores` | 15 | ChaCha20 + TLS overhead per session |
| `ReservedRAM` | 512 MB | OS + agent + buffers |
| `AvgRAMPerSession` | 2.5 MB | Buffers + session state |
| `ConnectionUtilTarget` | 0.80 | FD headroom |
| `ConnectionsPerSession` | 3 | Main + control + keepalive |
| `ProtocolOverheadFactor` | 1.15 | 15% overhead from REALITY + HTTP/2 framing |
| `PaddingOverheadFactor` | 1.08 | 8% average padding overhead |

#### Example Calculation

**Node specs:** 4 vCPU, 8 GB RAM, 1 Gbps, 65535 max FDs

```
BandwidthCapacity = (1000 × 0.75) / (5.0 × 1.15 × 1.08)
                  = 750 / 6.21
                  = 120 sessions

CPUCapacity = (4 × 1000 × 0.70) / 15
            = 2800 / 15
            = 186 sessions

RAMCapacity = (8192 - 512) / 2.5
            = 7680 / 2.5
            = 3072 sessions

ConnectionCapacity = (65535 × 0.80) / 3
                   = 52428 / 3
                   = 17476 sessions

MaxSessions = min(120, 186, 3072, 17476) = 120 sessions
```

**Bottleneck:** Bandwidth. This is typical for VPN workloads.

#### Dynamic Capacity Adjustment

Real-time capacity adjusts based on telemetry:

```go
type DynamicCapacity struct {
    StaticMax       int
    CurrentMax      int
    AdjustInterval  time.Duration // 30 seconds
}

func (dc *DynamicCapacity) Recalculate(metrics NodeMetrics) int {
    penalties := 1.0
    
    // P95 latency penalty
    if metrics.P95LatencyMs > 50 {
        penalties *= 0.8
    } else if metrics.P95LatencyMs > 20 {
        penalties *= 0.9
    }
    
    // Packet loss penalty
    if metrics.PacketLossRate > 0.01 {
        penalties *= 0.7
    } else if metrics.PacketLossRate > 0.001 {
        penalties *= 0.85
    }
    
    // CPU thermal throttling
    if metrics.CPUPercent > 85 {
        penalties *= 0.6
    }
    
    // Memory pressure
    if metrics.MemoryPercent > 80 {
        penalties *= 0.75
    }
    
    dc.CurrentMax = int(float64(dc.StaticMax) * penalties)
    return dc.CurrentMax
}
```

#### Autoscaling Algorithm

```go
type AutoscalerConfig struct {
    ScaleUpThreshold    float64       // 0.75 — scale up when avg load > 75%
    ScaleDownThreshold  float64       // 0.30 — scale down when avg load < 30%
    ScaleUpCooldown     time.Duration // 3 minutes
    ScaleDownCooldown   time.Duration // 10 minutes
    MinNodes            int           // per region
    MaxNodes            int           // per region, cost cap
    ScaleStep           int           // nodes to add/remove per decision
}

func (a *Autoscaler) Evaluate(region string) ScaleDecision {
    nodes := a.nodeManager.GetActiveNodes(region)
    
    totalCapacity := 0
    totalSessions := 0
    for _, n := range nodes {
        totalCapacity += n.CurrentMaxSessions
        totalSessions += n.ActiveSessions
    }
    
    loadRatio := float64(totalSessions) / float64(totalCapacity)
    
    switch {
    case loadRatio > a.config.ScaleUpThreshold:
        if time.Since(a.lastScaleUp[region]) < a.config.ScaleUpCooldown {
            return ScaleDecision{Action: Hold}
        }
        needed := int(math.Ceil(float64(totalSessions) / a.config.ScaleUpThreshold)) - totalCapacity
        newNodes := int(math.Ceil(float64(needed) / float64(a.avgNodeCapacity(region))))
        newNodes = min(newNodes, a.config.ScaleStep)
        return ScaleDecision{Action: ScaleUp, Count: newNodes, Region: region}
        
    case loadRatio < a.config.ScaleDownThreshold && len(nodes) > a.config.MinNodes:
        if time.Since(a.lastScaleDown[region]) < a.config.ScaleDownCooldown {
            return ScaleDecision{Action: Hold}
        }
        excess := totalCapacity - int(float64(totalSessions)/a.config.ScaleUpThreshold)
        removeNodes := int(math.Floor(float64(excess) / float64(a.avgNodeCapacity(region))))
        removeNodes = min(removeNodes, a.config.ScaleStep)
        return ScaleDecision{Action: ScaleDown, Count: removeNodes, Region: region}
        
    default:
        return ScaleDecision{Action: Hold}
    }
}
```

---

## 6. Database Schema

### ER Diagram

```
┌──────────────┐       ┌──────────────────┐       ┌───────────────────┐
│    users     │       │  subscriptions   │       │     payments      │
├──────────────┤       ├──────────────────┤       ├───────────────────┤
│ id (PK)      │──┐    │ id (PK)          │──┐    │ id (PK)           │
│ telegram_id  │  │    │ user_id (FK)     │  │    │ subscription_id   │
│ username     │  ├───►│ plan_id (FK)     │  ├───►│ amount            │
│ referrer_id  │  │    │ status           │  │    │ currency          │
│ created_at   │  │    │ started_at       │  │    │ provider          │
│ updated_at   │  │    │ expires_at       │  │    │ status            │
│ status       │  │    │ auto_renew       │  │    │ external_id       │
│ tier         │  │    │ trial_ends_at    │  │    │ created_at        │
└──────────────┘  │    │ grace_ends_at    │  │    └───────────────────┘
                  │    └──────────────────┘  │
                  │                          │
┌──────────────┐  │    ┌──────────────────┐  │    ┌───────────────────┐
│   devices    │  │    │   vpn_configs    │  │    │   vpn_sessions    │
├──────────────┤  │    ├──────────────────┤  │    ├───────────────────┤
│ id (PK)      │  │    │ id (PK)          │  │    │ id (PK)           │
│ user_id (FK) │◄─┤    │ user_id (FK)     │◄─┘    │ config_id (FK)    │
│ name         │  │    │ device_id (FK)   │       │ node_id (FK)      │
│ platform     │  │    │ node_id (FK)     │       │ started_at        │
│ fingerprint  │  │    │ private_key      │       │ ended_at          │
│ created_at   │  │    │ public_key       │       │ bytes_in          │
│ last_seen    │  │    │ short_id         │       │ bytes_out         │
│ is_revoked   │  │    │ status           │       │ disconnect_reason │
└──────────────┘  │    │ created_at       │       └───────────────────┘
                  │    │ revoked_at       │
                  │    └──────────────────┘
                  │                               ┌───────────────────┐
┌──────────────┐  │    ┌──────────────────┐       │  node_metrics     │
│    plans     │  │    │      nodes       │       ├───────────────────┤
├──────────────┤  │    ├──────────────────┤       │ id (PK)           │
│ id (PK)      │  │    │ id (PK)          │──────►│ node_id (FK)      │
│ name         │  │    │ region           │       │ timestamp         │
│ price        │  │    │ country          │       │ cpu_percent       │
│ currency     │  │    │ city             │       │ memory_percent    │
│ duration_days│  │    │ provider         │       │ bandwidth_in      │
│ max_devices  │  │    │ public_ip        │       │ bandwidth_out     │
│ bandwidth_cap│  │    │ status           │       │ active_sessions   │
│ features     │  │    │ specs (jsonb)    │       │ p95_latency_ms    │
└──────────────┘  │    │ max_sessions     │       │ packet_loss_rate  │
                  │    │ current_sessions │       └───────────────────┘
                  │    │ registered_at    │
┌──────────────┐  │    │ last_heartbeat   │       ┌───────────────────┐
│  referrals   │  │    └──────────────────┘       │    promo_codes    │
├──────────────┤  │                               ├───────────────────┤
│ id (PK)      │  │    ┌──────────────────┐       │ id (PK)           │
│ referrer_id  │◄─┘    │   audit_logs     │       │ code              │
│ referred_id  │       ├──────────────────┤       │ discount_percent  │
│ reward_type  │       │ id (PK)          │       │ max_uses          │
│ reward_amount│       │ actor_id         │       │ current_uses      │
│ status       │       │ action           │       │ valid_from        │
│ created_at   │       │ resource_type    │       │ valid_until       │
└──────────────┘       │ resource_id      │       │ plan_ids (array)  │
                       │ metadata (jsonb) │       │ created_by        │
                       │ ip_address       │       └───────────────────┘
                       │ created_at       │
                       └──────────────────┘
```

### DDL (Key Tables)

```sql
-- Users
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_id     BIGINT UNIQUE NOT NULL,
    username        VARCHAR(255),
    referrer_id     UUID REFERENCES users(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'active',  -- active, suspended, deleted
    tier            VARCHAR(20) NOT NULL DEFAULT 'free',    -- free, basic, premium, enterprise
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_telegram_id ON users(telegram_id);
CREATE INDEX idx_users_referrer_id ON users(referrer_id) WHERE referrer_id IS NOT NULL;
CREATE INDEX idx_users_status ON users(status) WHERE status != 'deleted';

-- Subscriptions
CREATE TABLE subscriptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    plan_id         UUID NOT NULL REFERENCES plans(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',  -- pending, trial, active, grace, expired, cancelled
    started_at      TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    trial_ends_at   TIMESTAMPTZ,
    grace_ends_at   TIMESTAMPTZ,
    auto_renew      BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX idx_subscriptions_status_expires ON subscriptions(status, expires_at)
    WHERE status IN ('active', 'trial', 'grace');

-- Nodes
CREATE TABLE nodes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region          VARCHAR(50) NOT NULL,
    country         CHAR(2) NOT NULL,
    city            VARCHAR(100),
    provider        VARCHAR(50) NOT NULL,
    public_ip       INET NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'registered',
    specs           JSONB NOT NULL,
    max_sessions    INT NOT NULL DEFAULT 0,
    current_sessions INT NOT NULL DEFAULT 0,
    registered_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_heartbeat  TIMESTAMPTZ
);

CREATE INDEX idx_nodes_status_region ON nodes(status, region) WHERE status = 'active';
CREATE INDEX idx_nodes_public_ip ON nodes(public_ip);

-- VPN Sessions (partitioned by month)
CREATE TABLE vpn_sessions (
    id              UUID NOT NULL DEFAULT gen_random_uuid(),
    config_id       UUID NOT NULL,
    node_id         UUID NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at        TIMESTAMPTZ,
    bytes_in        BIGINT NOT NULL DEFAULT 0,
    bytes_out       BIGINT NOT NULL DEFAULT 0,
    disconnect_reason VARCHAR(50),
    PRIMARY KEY (id, started_at)
) PARTITION BY RANGE (started_at);

-- Create monthly partitions
CREATE TABLE vpn_sessions_2025_01 PARTITION OF vpn_sessions
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
-- ... auto-created by pg_partman

-- Node Metrics (partitioned, with retention)
CREATE TABLE node_metrics (
    id              BIGSERIAL,
    node_id         UUID NOT NULL,
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cpu_percent     REAL NOT NULL,
    memory_percent  REAL NOT NULL,
    bandwidth_in    BIGINT NOT NULL,
    bandwidth_out   BIGINT NOT NULL,
    active_sessions INT NOT NULL,
    p95_latency_ms  REAL NOT NULL,
    packet_loss_rate REAL NOT NULL,
    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

-- Audit Logs (append-only, partitioned)
CREATE TABLE audit_logs (
    id              BIGSERIAL,
    actor_id        UUID,
    action          VARCHAR(100) NOT NULL,
    resource_type   VARCHAR(50) NOT NULL,
    resource_id     UUID,
    metadata        JSONB,
    ip_address      INET,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);
```

### Partitioning Strategy

| Table | Partition By | Interval | Retention |
|-------|-------------|----------|-----------|
| `vpn_sessions` | `started_at` | Monthly | 12 months |
| `node_metrics` | `timestamp` | Daily | 30 days (raw), 1 year (1min aggregates) |
| `audit_logs` | `created_at` | Monthly | 24 months |
| `payments` | `created_at` | Quarterly | Forever |

### Indexes Strategy

- **Hot queries:** Covered indexes for session lookups, node selection
- **Partial indexes:** Only index active/relevant rows (WHERE clauses)
- **GIN indexes:** For JSONB columns (specs, metadata)
- **BRIN indexes:** For timestamp columns in partitioned tables

---

## 7. gRPC Contracts

### Proto Definitions

```protobuf
syntax = "proto3";
package vpn.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/duration.proto";

// ==================== Auth Service ====================

service AuthService {
    rpc Authenticate(AuthRequest) returns (AuthResponse);
    rpc RefreshToken(RefreshRequest) returns (AuthResponse);
    rpc RevokeSession(RevokeSessionRequest) returns (RevokeSessionResponse);
    rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
    rpc BindDevice(BindDeviceRequest) returns (BindDeviceResponse);
}

message AuthRequest {
    int64 telegram_id = 1;
    string telegram_hash = 2;  // Telegram login widget hash
    DeviceInfo device = 3;
}

message AuthResponse {
    string access_token = 1;   // JWT, 15 min TTL
    string refresh_token = 2;  // opaque, 30 day TTL
    google.protobuf.Timestamp expires_at = 3;
    UserInfo user = 4;
}

message DeviceInfo {
    string fingerprint = 1;
    string platform = 2;       // "ios", "android", "macos", "windows", "linux"
    string app_version = 3;
}

// ==================== Node Manager ====================

service NodeManager {
    rpc RegisterNode(RegisterNodeRequest) returns (RegisterNodeResponse);
    rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
    rpc GetOptimalNode(GetOptimalNodeRequest) returns (GetOptimalNodeResponse);
    rpc ListNodes(ListNodesRequest) returns (ListNodesResponse);
    rpc DrainNode(DrainNodeRequest) returns (DrainNodeResponse);
    rpc RemoveNode(RemoveNodeRequest) returns (RemoveNodeResponse);
    
    // Streaming
    rpc StreamConfig(stream ConfigAck) returns (stream NodeConfig);
    rpc StreamMetrics(stream NodeMetricsReport) returns (stream MetricsAck);
}

message RegisterNodeRequest {
    string node_id = 1;
    string region = 2;
    string country = 3;
    string city = 4;
    string provider = 5;
    string public_ip = 6;
    NodeSpecs specs = 7;
    repeated string transport_capabilities = 8;
    string auth_token = 9;
}

message NodeSpecs {
    int32 cpu_cores = 1;
    string cpu_model = 2;
    int64 ram_bytes = 3;
    int32 bandwidth_mbps = 4;
    int32 disk_gb = 5;
    string kernel_version = 6;
}

message HeartbeatRequest {
    string node_id = 1;
    google.protobuf.Timestamp timestamp = 2;
    int32 active_sessions = 3;
    double cpu_percent = 4;
    double memory_percent = 5;
    int64 bandwidth_in_bps = 6;
    int64 bandwidth_out_bps = 7;
    double p95_latency_ms = 8;
    double packet_loss_rate = 9;
    int32 open_connections = 10;
}

message HeartbeatResponse {
    NodeDirective directive = 1;  // CONTINUE, DRAIN, RECONFIGURE
    NodeConfig config = 2;       // optional new config
}

message GetOptimalNodeRequest {
    string user_id = 1;
    string preferred_region = 2;
    string transport_type = 3;
}

message GetOptimalNodeResponse {
    NodeInfo node = 1;
    VPNConfig config = 2;
}

message NodeConfig {
    string version = 1;
    TransportConfig transport = 2;
    RateLimitConfig rate_limit = 3;
    repeated string blocked_ips = 4;
    int32 max_sessions_override = 5;
}

// ==================== Subscription Service ====================

service SubscriptionService {
    rpc GetSubscription(GetSubscriptionRequest) returns (Subscription);
    rpc CreateSubscription(CreateSubscriptionRequest) returns (Subscription);
    rpc CancelSubscription(CancelSubscriptionRequest) returns (Subscription);
    rpc RenewSubscription(RenewSubscriptionRequest) returns (Subscription);
    rpc ApplyPromoCode(ApplyPromoCodeRequest) returns (ApplyPromoCodeResponse);
    rpc ListPlans(ListPlansRequest) returns (ListPlansResponse);
}

message Subscription {
    string id = 1;
    string user_id = 2;
    string plan_id = 3;
    SubscriptionStatus status = 4;
    google.protobuf.Timestamp started_at = 5;
    google.protobuf.Timestamp expires_at = 6;
    google.protobuf.Timestamp trial_ends_at = 7;
    google.protobuf.Timestamp grace_ends_at = 8;
    bool auto_renew = 9;
}

enum SubscriptionStatus {
    SUBSCRIPTION_STATUS_UNSPECIFIED = 0;
    SUBSCRIPTION_STATUS_PENDING = 1;
    SUBSCRIPTION_STATUS_TRIAL = 2;
    SUBSCRIPTION_STATUS_ACTIVE = 3;
    SUBSCRIPTION_STATUS_GRACE = 4;
    SUBSCRIPTION_STATUS_EXPIRED = 5;
    SUBSCRIPTION_STATUS_CANCELLED = 6;
}

// ==================== Config Distributor ====================

service ConfigDistributor {
    rpc GenerateConfig(GenerateConfigRequest) returns (VPNConfig);
    rpc RevokeConfig(RevokeConfigRequest) returns (RevokeConfigResponse);
    rpc RotateKeys(RotateKeysRequest) returns (RotateKeysResponse);
    rpc GetActiveConfigs(GetActiveConfigsRequest) returns (GetActiveConfigsResponse);
}

message VPNConfig {
    string id = 1;
    string node_address = 2;
    int32 node_port = 3;
    string transport_type = 4;
    bytes client_private_key = 5;
    bytes server_public_key = 6;
    bytes short_id = 7;
    string sni = 8;
    string fingerprint = 9;
    string http2_path = 10;
    map<string, string> extra_params = 11;
    google.protobuf.Timestamp valid_until = 12;
}

// ==================== Billing Service ====================

service BillingService {
    rpc CreatePayment(CreatePaymentRequest) returns (Payment);
    rpc GetPayment(GetPaymentRequest) returns (Payment);
    rpc ProcessWebhook(WebhookRequest) returns (WebhookResponse);
    rpc GetInvoice(GetInvoiceRequest) returns (Invoice);
    rpc ListPayments(ListPaymentsRequest) returns (ListPaymentsResponse);
}

message CreatePaymentRequest {
    string user_id = 1;
    string plan_id = 2;
    PaymentProvider provider = 3;
    string promo_code = 4;
    string currency = 5;
}

enum PaymentProvider {
    PAYMENT_PROVIDER_UNSPECIFIED = 0;
    PAYMENT_PROVIDER_STRIPE = 1;
    PAYMENT_PROVIDER_CRYPTO = 2;
    PAYMENT_PROVIDER_TELEGRAM_STARS = 3;
}

message Payment {
    string id = 1;
    string user_id = 2;
    string subscription_id = 3;
    int64 amount = 4;           // in smallest currency unit (cents)
    string currency = 5;
    PaymentProvider provider = 6;
    PaymentStatus status = 7;
    string external_id = 8;     // Stripe charge ID, crypto tx hash, etc
    string checkout_url = 9;    // redirect URL for payment
    google.protobuf.Timestamp created_at = 10;
}

enum PaymentStatus {
    PAYMENT_STATUS_UNSPECIFIED = 0;
    PAYMENT_STATUS_PENDING = 1;
    PAYMENT_STATUS_COMPLETED = 2;
    PAYMENT_STATUS_FAILED = 3;
    PAYMENT_STATUS_REFUNDED = 4;
}
```

---

## 8. Telegram Bot Architecture

### Architecture Overview

```
┌──────────────────────────────────────────────────────┐
│                 Telegram Bot Service                   │
│                                                      │
│  ┌────────────┐  ┌─────────────┐  ┌──────────────┐  │
│  │  Webhook   │  │    FSM      │  │  Middleware  │  │
│  │  Handler   │──│  Engine     │──│  Chain       │  │
│  └────────────┘  └─────────────┘  └──────────────┘  │
│                                                      │
│  ┌────────────┐  ┌─────────────┐  ┌──────────────┐  │
│  │  Command   │  │  Callback   │  │  Inline      │  │
│  │  Router    │  │  Router     │  │  Handler     │  │
│  └────────────┘  └─────────────┘  └──────────────┘  │
│                                                      │
│  ┌────────────┐  ┌─────────────┐  ┌──────────────┐  │
│  │  Config    │  │  Notifier   │  │  Job         │  │
│  │  Renderer  │  │  (async)    │  │  Scheduler   │  │
│  └────────────┘  └─────────────┘  └──────────────┘  │
└──────────────────────────────────────────────────────┘
```

### FSM States

```go
type BotState string

const (
    StateIdle               BotState = "idle"
    StateRegistration       BotState = "registration"
    StateSelectingPlan      BotState = "selecting_plan"
    StatePaymentPending     BotState = "payment_pending"
    StateSelectingRegion    BotState = "selecting_region"
    StateSelectingDevice    BotState = "selecting_device"
    StateViewingConfig      BotState = "viewing_config"
    StateManagingDevices    BotState = "managing_devices"
    StateViewingStats       BotState = "viewing_stats"
    StateEnteringPromo      BotState = "entering_promo"
    StateReferralMenu       BotState = "referral_menu"
    StateSupport            BotState = "support"
    StateConfirmAction      BotState = "confirm_action"
)

type FSMTransition struct {
    From    BotState
    Event   string
    To      BotState
    Action  func(ctx *BotContext) error
}

var transitions = []FSMTransition{
    {StateIdle, "cmd_start", StateRegistration, handleStart},
    {StateIdle, "cmd_subscribe", StateSelectingPlan, handleSubscribe},
    {StateIdle, "cmd_connect", StateSelectingRegion, handleConnect},
    {StateIdle, "cmd_devices", StateManagingDevices, handleDevices},
    {StateIdle, "cmd_stats", StateViewingStats, handleStats},
    {StateIdle, "cmd_referral", StateReferralMenu, handleReferral},
    {StateIdle, "cmd_promo", StateEnteringPromo, handlePromo},
    {StateSelectingPlan, "plan_selected", StatePaymentPending, handlePlanSelected},
    {StatePaymentPending, "payment_success", StateIdle, handlePaymentSuccess},
    {StatePaymentPending, "payment_failed", StateSelectingPlan, handlePaymentFailed},
    {StateSelectingRegion, "region_selected", StateViewingConfig, handleRegionSelected},
    {StateManagingDevices, "device_revoke", StateConfirmAction, handleDeviceRevoke},
    {StateConfirmAction, "confirmed", StateIdle, handleConfirmed},
    {StateConfirmAction, "cancelled", StateIdle, handleCancelled},
}
```

### Command Handlers

```go
type CommandRegistry struct {
    commands map[string]CommandHandler
}

// Bot commands:
// /start        — Registration + welcome
// /subscribe    — View/buy plans
// /connect      — Get VPN config for device
// /regions      — List available regions with load
// /devices      — Manage connected devices
// /stats        — Usage statistics
// /referral     — Referral link + stats
// /promo        — Enter promo code
// /support      — Help & FAQ
// /settings     — Preferences
// /revoke       — Revoke specific device/config

// Deep links:
// t.me/bot?start=ref_<user_id>     — Referral registration
// t.me/bot?start=promo_<code>      — Promo code auto-apply
// t.me/bot?start=plan_<plan_id>    — Direct plan purchase
```

### Inline Keyboards

```go
func buildMainMenu(hasSubscription bool) *tgbotapi.InlineKeyboardMarkup {
    rows := [][]tgbotapi.InlineKeyboardButton{}
    
    if hasSubscription {
        rows = append(rows,
            tgbotapi.NewInlineKeyboardRow(
                tgbotapi.NewInlineKeyboardButtonData("🌐 Connect", "action:connect"),
                tgbotapi.NewInlineKeyboardButtonData("📊 Stats", "action:stats"),
            ),
            tgbotapi.NewInlineKeyboardRow(
                tgbotapi.NewInlineKeyboardButtonData("📱 Devices", "action:devices"),
                tgbotapi.NewInlineKeyboardButtonData("🌍 Regions", "action:regions"),
            ),
        )
    } else {
        rows = append(rows,
            tgbotapi.NewInlineKeyboardRow(
                tgbotapi.NewInlineKeyboardButtonData("⚡ Get VPN", "action:subscribe"),
            ),
        )
    }
    
    rows = append(rows,
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("👥 Referral", "action:referral"),
            tgbotapi.NewInlineKeyboardButtonData("⚙️ Settings", "action:settings"),
        ),
    )
    
    kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
    return &kb
}
```

### Async Job System

```go
type AsyncJobType string

const (
    JobSubscriptionExpiring  AsyncJobType = "subscription_expiring"
    JobSubscriptionExpired   AsyncJobType = "subscription_expired"
    JobPaymentReminder       AsyncJobType = "payment_reminder"
    JobNodeMigration         AsyncJobType = "node_migration"
    JobKeyRotation           AsyncJobType = "key_rotation"
    JobUsageAlert            AsyncJobType = "usage_alert"
    JobReferralReward        AsyncJobType = "referral_reward"
    JobBroadcast             AsyncJobType = "broadcast"
)

type NotificationScheduler struct {
    queue    *nats.JetStreamContext
    workers  int
}

// Scheduled notifications:
// - 3 days before expiry: renewal reminder
// - 1 day before expiry: urgent renewal
// - On expiry: grace period notification
// - Grace period end: disconnection warning
// - Monthly: usage stats summary
// - On referral conversion: reward notification
```

### Config Delivery

```go
func (h *ConfigHandler) DeliverConfig(ctx *BotContext, config *VPNConfig) error {
    // Generate shareable config based on client platform
    switch ctx.Device.Platform {
    case "ios", "macos":
        // Generate .mobileconfig or custom app deep link
        return h.sendAppleConfig(ctx, config)
    case "android":
        // Generate intent:// deep link for custom app
        return h.sendAndroidConfig(ctx, config)
    case "windows", "linux":
        // Generate config file + connection URI
        return h.sendDesktopConfig(ctx, config)
    }
    
    // Fallback: send as copyable text block
    configText := formatConfigText(config)
    msg := tgbotapi.NewMessage(ctx.ChatID, configText)
    msg.ParseMode = "MarkdownV2"
    _, err := ctx.Bot.Send(msg)
    return err
}
```

---

## 9. Billing Architecture

### Payment Flow

```
User selects plan
       │
       ▼
┌─────────────────┐
│ Create Payment  │
│ (pending)       │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │         ┌─────────────┐
    ▼         ▼         ▼             │
┌───────┐ ┌───────┐ ┌───────────┐    │
│Stripe │ │Crypto │ │TG Stars   │    │
│       │ │       │ │           │    │
│Checkout│ │Invoice│ │In-app     │    │
│Session │ │+ addr │ │purchase   │    │
└───┬───┘ └───┬───┘ └─────┬─────┘    │
    │         │            │          │
    ▼         ▼            ▼          │
┌──────────────────────────────┐      │
│    Webhook / Callback        │      │
│    Validates payment         │      │
└──────────────┬───────────────┘      │
               │                      │
               ▼                      │
┌──────────────────────────────┐      │
│  Payment confirmed           │      │
│  → Activate subscription     │──────┘ (retry on failure)
│  → Generate VPN config       │
│  → Notify user               │
└──────────────────────────────┘
```

### Provider Implementations

```go
type PaymentProcessor interface {
    CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*Payment, error)
    ValidateWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)
    Refund(ctx context.Context, paymentID string, amount int64) error
}

// Stripe implementation
type StripeProcessor struct {
    client    *stripe.Client
    webhookSecret string
}

// Crypto implementation (BTC, ETH, USDT via NOWPayments or BTCPay)
type CryptoProcessor struct {
    client    *nowpayments.Client
    callbacks chan CryptoCallback
}

// Telegram Stars implementation
type TelegramStarsProcessor struct {
    bot       *tgbotapi.BotAPI
    invoices  map[string]*TelegramInvoice
}
```

### Subscription Lifecycle

```go
type SubscriptionLifecycle struct {
    TrialDays     int           // 3 days
    GracePeriod   time.Duration // 3 days after expiry
    RenewalWindow time.Duration // 7 days before expiry (auto-charge window)
}

// State transitions:
// PENDING → TRIAL (on first activation with trial eligible)
// PENDING → ACTIVE (on payment confirmation, no trial)
// TRIAL → ACTIVE (on trial expiry if payment method exists)
// TRIAL → EXPIRED (on trial expiry if no payment)
// ACTIVE → GRACE (on expiry if payment fails)
// ACTIVE → ACTIVE (on successful renewal)
// GRACE → ACTIVE (on successful retry payment)
// GRACE → EXPIRED (on grace period end)
// ACTIVE → CANCELLED (on user cancellation, access until expires_at)
// * → EXPIRED (terminal, after all grace periods)

func (s *SubscriptionService) ProcessExpiry(ctx context.Context, sub *Subscription) error {
    if sub.AutoRenew {
        // Attempt auto-renewal
        payment, err := s.billing.ChargeStoredMethod(ctx, sub.UserID, sub.PlanID)
        if err == nil {
            return s.Renew(ctx, sub.ID, payment.ID)
        }
        // Payment failed → enter grace period
        return s.EnterGrace(ctx, sub.ID)
    }
    return s.Expire(ctx, sub.ID)
}
```

### Fraud Prevention

```go
type FraudChecker struct {
    rules []FraudRule
}

type FraudRule interface {
    Check(ctx context.Context, payment *Payment) (FraudScore, error)
}

// Rules:
// 1. VelocityRule: >3 payment attempts in 1 hour → block
// 2. DeviceMismatchRule: payment device != registered device → flag
// 3. GeoMismatchRule: payment IP geo != account geo → flag
// 4. AmountAnomalyRule: unusual amount pattern → flag
// 5. CardTestingRule: multiple small charges → block
// 6. RefundAbuseRule: >2 refunds in 30 days → flag

type FraudScore struct {
    Score    float64 // 0.0 - 1.0
    Decision FraudDecision // ALLOW, REVIEW, BLOCK
    Reasons  []string
}
```

### Pricing Model

| Plan | Monthly | Quarterly | Annual | Devices | Bandwidth |
|------|---------|-----------|--------|---------|-----------|
| Basic | $4.99 | $12.99 | $39.99 | 2 | 100 GB/mo |
| Premium | $9.99 | $24.99 | $79.99 | 5 | Unlimited |
| Enterprise | $19.99 | $49.99 | $149.99 | 10 | Unlimited + Priority |

---

## 10. Security Architecture

### Defense Layers

```
Layer 1: Network Edge
├── Cloudflare/CDN (DDoS mitigation for control plane)
├── GeoIP filtering (block known hostile ranges)
├── TCP SYN flood protection (kernel tuning)
└── IP reputation database

Layer 2: Transport
├── REALITY authentication (no response to invalid clients)
├── uTLS fingerprint validation
├── Active probing resistance
└── Rate limiting per IP

Layer 3: Application
├── JWT with short TTL (15 min)
├── Device binding (hardware fingerprint)
├── Session pinning
└── Request signing

Layer 4: Data
├── AES-256-GCM at rest (database)
├── ChaCha20-Poly1305 in transit (VPN)
├── Key rotation (automatic)
└── Secret management (HashiCorp Vault)

Layer 5: Operations
├── Audit logging (immutable)
├── Anomaly detection
├── Automated incident response
└── Compliance monitoring
```

### Rate Limiting

```go
type RateLimiter struct {
    // Per-IP limits
    IPLimits map[string]*rate.Limiter
    // Per-user limits
    UserLimits map[string]*rate.Limiter
    // Global limits
    GlobalLimit *rate.Limiter
}

type RateLimitConfig struct {
    // API Gateway
    APIRequestsPerMinute   int // 60 per user
    APIRequestsPerHour     int // 1000 per user
    
    // Auth
    LoginAttemptsPerHour   int // 5 per IP
    TokenRefreshPerHour    int // 10 per user
    
    // VPN Connections
    ConnectionsPerMinute   int // 3 per user (prevent connection storms)
    ConfigRequestsPerHour  int // 10 per user
    
    // Bot
    BotMessagesPerMinute   int // 20 per user
    
    // Node registration
    RegistrationsPerHour   int // 1 per IP (prevent enumeration)
}
```

### Anti-Abuse System

```go
type AbuseDetector struct {
    rules     []AbuseRule
    actions   map[AbuseSeverity]AbuseAction
    store     AbuseStore
}

type AbuseRule interface {
    Evaluate(ctx context.Context, event *Event) (*AbuseViolation, error)
}

// Rules:
// 1. BandwidthAbuse: >10x avg bandwidth sustained → throttle
// 2. ConnectionChurn: >100 connects/disconnects per hour → temp ban
// 3. PortScanning: detected by node IDS → immediate ban
// 4. TorrentAbuse: BitTorrent traffic on restricted plans → warning
// 5. SharedCredentials: same config from >3 IPs simultaneously → revoke
// 6. BotBehavior: automated patterns on Telegram bot → captcha

type AbuseAction struct {
    Severity    AbuseSeverity
    Action      string    // "warn", "throttle", "suspend", "ban"
    Duration    time.Duration
    NotifyUser  bool
    NotifyAdmin bool
}
```

### Anti-Probing

```go
type ProbingDefense struct {
    // All ports except 443 return nothing (DROP, not REJECT)
    // Port 443: REALITY ensures non-authenticated traffic sees real site
    
    // Honeypot detection
    honeypotPorts []int    // Common VPN ports that we DON'T use → detect scanners
    
    // Behavioral detection
    connPatterns  *ConnPatternAnalyzer
}

// iptables rules (applied by node-agent)
var firewallRules = []string{
    // Allow established connections
    "-A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT",
    // Allow SSH from control plane only
    "-A INPUT -p tcp --dport 22 -s <control_plane_ip> -j ACCEPT",
    // Allow HTTPS (our VPN port)
    "-A INPUT -p tcp --dport 443 -j ACCEPT",
    // Allow node-agent metrics (internal)
    "-A INPUT -p tcp --dport 9011 -s <control_plane_ip> -j ACCEPT",
    // Honeypot ports (log and drop)
    "-A INPUT -p tcp --dport 1194 -j LOG --log-prefix 'PROBE:' && DROP",
    "-A INPUT -p udp --dport 51820 -j LOG --log-prefix 'PROBE:' && DROP",
    // Drop everything else
    "-A INPUT -j DROP",
}
```

### Device Binding

```go
type DeviceFingerprint struct {
    HardwareID   string // Derived from hardware identifiers
    Platform     string
    AppVersion   string
    InstallID    string // Generated on first install
}

// Binding flow:
// 1. On first connect, client sends DeviceFingerprint
// 2. Server generates device_id = HMAC-SHA256(user_secret, fingerprint)
// 3. Config is bound to device_id
// 4. Subsequent connections must present matching fingerprint
// 5. Mismatch → config revoked, user notified
```

### Secret Rotation

```go
type SecretRotator struct {
    vault    *vault.Client
    schedule map[SecretType]time.Duration
}

// Rotation schedule:
// JWT signing keys:     7 days (overlap period: 1 day)
// Database passwords:   30 days
// API keys:             90 days
// REALITY keys:         7 days (per node)
// Node auth tokens:     24 hours
// TLS certificates:     60 days (via cert-manager)
// Encryption keys:      180 days (envelope encryption)
```

---

## 11. Observability Stack

### Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                        Observability                              │
│                                                                  │
│  ┌──────────┐   ┌───────────┐   ┌──────────┐   ┌───────────┐   │
│  │Prometheus│   │   Loki    │   │  Tempo   │   │ Grafana   │   │
│  │(metrics) │   │  (logs)   │   │(traces)  │   │(dashboard)│   │
│  └────┬─────┘   └─────┬─────┘   └────┬─────┘   └─────┬─────┘   │
│       │                │              │                │         │
│  ┌────┴────────────────┴──────────────┴────────────────┴────┐    │
│  │              OpenTelemetry Collector                       │    │
│  └────┬────────────────┬──────────────┬────────────────┬────┘    │
│       │                │              │                │         │
│  ┌────┴─────┐   ┌─────┴────┐   ┌────┴─────┐   ┌─────┴────┐    │
│  │ Control  │   │   Data   │   │   Bot    │   │  Infra   │    │
│  │ Plane    │   │   Plane  │   │  Service │   │  Agents  │    │
│  │ Services │   │   Nodes  │   │          │   │          │    │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

### Metrics (Prometheus)

```go
// Key metrics exposed by services

// Node Agent metrics
var (
    vpnActiveSessions = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "vpn_active_sessions",
        Help: "Number of active VPN sessions",
    }, []string{"node_id", "region", "transport"})
    
    vpnBandwidthBytes = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "vpn_bandwidth_bytes_total",
        Help: "Total bytes transferred",
    }, []string{"node_id", "direction"}) // direction: "in", "out"
    
    vpnSessionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "vpn_session_duration_seconds",
        Help:    "VPN session duration",
        Buckets: []float64{60, 300, 900, 3600, 7200, 14400, 28800, 86400},
    }, []string{"node_id", "disconnect_reason"})
    
    vpnHandshakeLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "vpn_handshake_latency_seconds",
        Help:    "Time to complete VPN handshake",
        Buckets: prometheus.ExponentialBuckets(0.01, 2, 10),
    }, []string{"node_id", "transport"})
    
    vpnPacketLoss = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "vpn_packet_loss_ratio",
        Help: "Current packet loss ratio",
    }, []string{"node_id"})
)

// Control Plane metrics
var (
    apiRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name:    "api_request_duration_seconds",
        Buckets: prometheus.DefBuckets,
    }, []string{"service", "method", "status"})
    
    subscriptionCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "subscription_active_count",
        Help: "Active subscriptions by plan",
    }, []string{"plan", "status"})
    
    paymentTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "payment_total",
        Help: "Total payments processed",
    }, []string{"provider", "status", "currency"})
    
    nodeHealthStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "node_health_status",
        Help: "Node health (1=healthy, 0=unhealthy)",
    }, []string{"node_id", "region"})
)
```

### OpenTelemetry Integration

```go
func initTracer(serviceName string) (*sdktrace.TracerProvider, error) {
    exporter, err := otlptracehttp.New(context.Background(),
        otlptracehttp.WithEndpoint("otel-collector:4318"),
        otlptracehttp.WithInsecure(),
    )
    if err != nil {
        return nil, err
    }
    
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(resource.NewWithAttributes(
            semconv.SchemaURL,
            semconv.ServiceNameKey.String(serviceName),
            attribute.String("environment", os.Getenv("ENV")),
        )),
        sdktrace.WithSampler(sdktrace.ParentBased(
            sdktrace.TraceIDRatioBased(0.1), // 10% sampling in production
        )),
    )
    
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

### SLO/SLA Definitions

| Service | SLI | SLO | SLA |
|---------|-----|-----|-----|
| VPN Connectivity | Successful connection rate | 99.9% | 99.5% |
| VPN Latency | p95 handshake time | < 500ms | < 1s |
| API Availability | Successful response rate | 99.95% | 99.9% |
| Bot Response | Response time to command | < 2s | < 5s |
| Payment Processing | Successful charge rate | 99.5% | 99% |
| Node Health | Heartbeat success rate | 99.99% | 99.9% |
| Config Delivery | Config generation time | < 1s | < 3s |

### Alerting Rules

```yaml
# Prometheus alerting rules
groups:
  - name: vpn_critical
    rules:
      - alert: NodeDown
        expr: up{job="node-agent"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "VPN node {{ $labels.node_id }} is down"
          
      - alert: HighSessionFailureRate
        expr: rate(vpn_handshake_errors_total[5m]) / rate(vpn_handshake_total[5m]) > 0.05
        for: 3m
        labels:
          severity: warning
          
      - alert: NodeCapacityHigh
        expr: vpn_active_sessions / vpn_max_sessions > 0.85
        for: 5m
        labels:
          severity: warning
          
      - alert: PaymentFailureSpike
        expr: rate(payment_total{status="failed"}[10m]) > 5
        for: 2m
        labels:
          severity: critical
          
      - alert: BandwidthSaturation
        expr: rate(vpn_bandwidth_bytes_total[1m]) * 8 / (node_bandwidth_mbps * 1e6) > 0.9
        for: 2m
        labels:
          severity: critical
```

### Logging (Loki)

```go
// Structured logging with slog
type contextKey string

func LoggerMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        logger := slog.With(
            slog.String("trace_id", trace.SpanFromContext(r.Context()).SpanContext().TraceID().String()),
            slog.String("service", serviceName),
            slog.String("method", r.Method),
            slog.String("path", r.URL.Path),
        )
        ctx := context.WithValue(r.Context(), contextKey("logger"), logger)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Log levels:
// ERROR: Payment failures, node crashes, security events
// WARN: Rate limits hit, degraded performance, approaching capacity
// INFO: User actions, subscription changes, node state changes
// DEBUG: Request details, config generation steps (disabled in prod)
```

---

## 12. Kubernetes Deployment

### Cluster Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                             │
│                                                                 │
│  ┌──────────────── Namespace: vpn-control ────────────────────┐ │
│  │                                                            │ │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌──────────────────┐│ │
│  │  │api-gw   │ │auth-svc │ │user-svc │ │subscription-svc  ││ │
│  │  │(3 pods) │ │(2 pods) │ │(2 pods) │ │(2 pods)          ││ │
│  │  └─────────┘ └─────────┘ └─────────┘ └──────────────────┘│ │
│  │                                                            │ │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌──────────────────┐│ │
│  │  │billing  │ │node-mgr │ │cfg-dist │ │telegram-bot      ││ │
│  │  │(2 pods) │ │(2 pods) │ │(2 pods) │ │(2 pods)          ││ │
│  │  └─────────┘ └─────────┘ └─────────┘ └──────────────────┘│ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌──────────────── Namespace: vpn-data ───────────────────────┐ │
│  │                                                            │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐     │ │
│  │  │ PostgreSQL   │  │    Redis     │  │    NATS      │     │ │
│  │  │ (HA cluster) │  │  (Sentinel)  │  │  (JetStream) │     │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘     │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌──────────────── Namespace: vpn-observability ──────────────┐ │
│  │                                                            │ │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐     │ │
│  │  │Prometheus│ │ Grafana  │ │   Loki   │ │  Tempo   │     │ │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘     │ │
│  │                                                            │ │
│  │  ┌──────────────────────────────────────────────┐          │ │
│  │  │       OpenTelemetry Collector (DaemonSet)    │          │ │
│  │  └──────────────────────────────────────────────┘          │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│              VPN Nodes (External, managed via node-agent)         │
│                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │ EU-West  │  │ US-East  │  │ AP-South │  │ EU-North │       │
│  │ (Hetzner)│  │ (Vultr)  │  │ (AWS)    │  │(DigitalO)│       │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

### Deployment Manifests

```yaml
# api-gateway deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
  namespace: vpn-control
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: api-gateway
  template:
    metadata:
      labels:
        app: api-gateway
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
    spec:
      serviceAccountName: api-gateway
      containers:
        - name: api-gateway
          image: registry.vpn-platform.internal/api-gateway:v1.0.0
          ports:
            - containerPort: 8080
              name: http
            - containerPort: 9090
              name: metrics
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: postgres-credentials
                  key: url
            - name: REDIS_URL
              valueFrom:
                secretKeyRef:
                  name: redis-credentials
                  key: url
            - name: NATS_URL
              value: "nats://nats.vpn-data:4222"
          resources:
            requests:
              cpu: 200m
              memory: 256Mi
            limits:
              cpu: 1000m
              memory: 512Mi
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8080
            initialDelaySeconds: 3
            periodSeconds: 5
      topologySpreadConstraints:
        - maxSkew: 1
          topologyKey: topology.kubernetes.io/zone
          whenUnsatisfiable: DoNotSchedule
          labelSelector:
            matchLabels:
              app: api-gateway
---
# HPA
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-gateway-hpa
  namespace: vpn-control
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-gateway
  minReplicas: 3
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Pods
      pods:
        metric:
          name: http_requests_per_second
        target:
          type: AverageValue
          averageValue: "1000"
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
        - type: Pods
          value: 2
          periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Pods
          value: 1
          periodSeconds: 120
```

### Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-gateway-ingress
  namespace: vpn-control
  annotations:
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/rate-limit-window: "1m"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
    - hosts:
        - api.vpn-platform.com
      secretName: api-tls
  rules:
    - host: api.vpn-platform.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: api-gateway
                port:
                  number: 8080
```

### Secrets Management

```yaml
# External Secrets Operator + HashiCorp Vault
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: postgres-credentials
  namespace: vpn-control
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-backend
    kind: ClusterSecretStore
  target:
    name: postgres-credentials
  data:
    - secretKey: url
      remoteRef:
        key: vpn/data/postgres
        property: connection_string
    - secretKey: password
      remoteRef:
        key: vpn/data/postgres
        property: password
```

### Blue-Green Deployment (for major releases)

```yaml
# Using Argo Rollouts for blue-green
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: api-gateway-rollout
  namespace: vpn-control
spec:
  replicas: 3
  strategy:
    blueGreen:
      activeService: api-gateway-active
      previewService: api-gateway-preview
      autoPromotionEnabled: false
      scaleDownDelaySeconds: 300
      prePromotionAnalysis:
        templates:
          - templateName: success-rate
        args:
          - name: service-name
            value: api-gateway-preview
      postPromotionAnalysis:
        templates:
          - templateName: success-rate
        args:
          - name: service-name
            value: api-gateway-active
  selector:
    matchLabels:
      app: api-gateway
  template:
    metadata:
      labels:
        app: api-gateway
    spec:
      containers:
        - name: api-gateway
          image: registry.vpn-platform.internal/api-gateway:v2.0.0
```

---

## 13. Failure Scenarios

### Failure Matrix

| Scenario | Detection | Impact | Mitigation | RTO |
|----------|-----------|--------|------------|-----|
| Single node crash | Heartbeat miss (30s) | Users on that node disconnected | Auto-reconnect to next best node | < 60s |
| Control plane DB down | Health check fail | No new connections, bot offline | PostgreSQL HA failover (Patroni) | < 30s |
| Redis failure | Sentinel detection | Degraded rate limiting, slow auth | Redis Sentinel auto-failover | < 15s |
| Node network partition | Heartbeat + client reports | Existing sessions may survive, no new | Mark degraded, redirect new users | < 30s |
| DPI starts blocking | Client connection failures spike | Region becomes unavailable | Auto-fallback to alternative transport | < 5min |
| Billing provider outage | Webhook timeout | New subscriptions delayed | Queue payments, retry with backoff | < 1hr |
| Certificate expiry | cert-manager alert | All TLS connections fail | Auto-renewal 30 days before expiry | 0 (preventive) |
| Key compromise | Audit detection | Potential session interception | Emergency key rotation, revoke all configs | < 5min |
| DDoS on control plane | Traffic spike detection | Bot/API unresponsive | Cloudflare WAF, auto-scaling | < 2min |
| Datacenter failure | Multiple node heartbeat loss | Regional outage | Geo-routing redirects to other regions | < 2min |

### Recovery Procedures

```go
// Automatic client reconnection
type ReconnectStrategy struct {
    MaxAttempts     int           // 10
    InitialDelay    time.Duration // 1 second
    MaxDelay        time.Duration // 30 seconds
    BackoffFactor   float64       // 2.0
    JitterPercent   float64       // 0.2
}

func (c *VPNClient) Reconnect(ctx context.Context) error {
    strategy := c.reconnectStrategy
    
    for attempt := 0; attempt < strategy.MaxAttempts; attempt++ {
        // Try current node first
        if err := c.connect(ctx, c.currentNode); err == nil {
            return nil
        }
        
        // Request new node from control plane
        newNode, err := c.controlPlane.GetOptimalNode(ctx, c.userID, c.region)
        if err == nil {
            if err := c.connect(ctx, newNode); err == nil {
                return nil
            }
        }
        
        // Backoff with jitter
        delay := time.Duration(float64(strategy.InitialDelay) * math.Pow(strategy.BackoffFactor, float64(attempt)))
        if delay > strategy.MaxDelay {
            delay = strategy.MaxDelay
        }
        jitter := time.Duration(float64(delay) * strategy.JitterPercent * (rand.Float64()*2 - 1))
        
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(delay + jitter):
        }
    }
    
    return ErrMaxReconnectAttemptsExceeded
}
```

### Circuit Breaker Pattern

```go
type CircuitBreaker struct {
    state           CircuitState // CLOSED, OPEN, HALF_OPEN
    failureCount    int
    successCount    int
    failureThreshold int          // 5
    successThreshold int          // 3
    timeout         time.Duration // 30 seconds
    lastFailure     time.Time
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
    if cb.state == Open {
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = HalfOpen
        } else {
            return ErrCircuitOpen
        }
    }
    
    err := fn()
    if err != nil {
        cb.failureCount++
        cb.lastFailure = time.Now()
        if cb.failureCount >= cb.failureThreshold {
            cb.state = Open
        }
        return err
    }
    
    if cb.state == HalfOpen {
        cb.successCount++
        if cb.successCount >= cb.successThreshold {
            cb.state = Closed
            cb.failureCount = 0
            cb.successCount = 0
        }
    }
    return nil
}
```

---

## 14. Scaling Strategy

### Horizontal Scaling Dimensions

```
┌────────────────────────────────────────────────────────────────┐
│                     Scaling Dimensions                          │
│                                                                │
│  Users      ──────►  Control Plane pods (HPA)                  │
│  Sessions   ──────►  VPN Nodes (custom autoscaler)             │
│  Bandwidth  ──────►  Node count + node size                    │
│  Regions    ──────►  New node deployments                      │
│  Events     ──────►  NATS partitions + consumers               │
│  Queries    ──────►  Read replicas + connection pooling         │
└────────────────────────────────────────────────────────────────┘
```

### Scaling Tiers

| Users | VPN Nodes | Control Plane | Database | Monthly Cost (est.) |
|-------|-----------|---------------|----------|---------------------|
| 1K | 3 (1/region) | 2 pods/svc | Single primary | $500 |
| 10K | 12 (4/region) | 3 pods/svc | Primary + 2 replicas | $3,000 |
| 100K | 50 (multi-region) | 5 pods/svc, 3 zones | HA cluster + read replicas | $20,000 |
| 1M | 200+ (auto-scaled) | 10 pods/svc, multi-cluster | Sharded + Citus | $150,000 |

### Database Scaling

```
Phase 1 (< 100K users):
- Single PostgreSQL primary + 2 synchronous replicas
- PgBouncer connection pooling (max 200 connections)
- Partitioned tables for time-series data

Phase 2 (100K - 1M users):
- Citus distributed PostgreSQL
- Shard key: user_id for user-centric tables
- Shard key: node_id for node-centric tables
- Reference tables: plans, promo_codes

Phase 3 (> 1M users):
- Multi-region Citus clusters
- Read replicas per region
- Hot data in Redis (sessions, rate limits)
- Cold data archival to S3 + Parquet
```

### Node Scaling Automation

```go
type NodeProvisioner struct {
    providers map[string]CloudProvider // "hetzner", "vultr", "aws"
    templates map[string]NodeTemplate  // per provider + region
}

type CloudProvider interface {
    CreateServer(ctx context.Context, opts CreateServerOpts) (*Server, error)
    DeleteServer(ctx context.Context, serverID string) error
    ListServers(ctx context.Context) ([]*Server, error)
}

type NodeTemplate struct {
    Provider     string
    Region       string
    MachineType  string  // e.g. "cpx31" for Hetzner
    Image        string  // pre-baked with node-agent
    UserData     string  // cloud-init script
    Tags         map[string]string
}

func (p *NodeProvisioner) ScaleUp(ctx context.Context, region string, count int) error {
    template := p.selectTemplate(region)
    provider := p.providers[template.Provider]
    
    for i := 0; i < count; i++ {
        server, err := provider.CreateServer(ctx, CreateServerOpts{
            Name:        fmt.Sprintf("vpn-node-%s-%s", region, uuid.New().String()[:8]),
            Region:      template.Region,
            MachineType: template.MachineType,
            Image:       template.Image,
            UserData:    template.UserData,
            Tags:        template.Tags,
        })
        if err != nil {
            return fmt.Errorf("provision node %d: %w", i, err)
        }
        
        // Node will auto-register via node-agent on boot
        log.Info("provisioned new node", "server_id", server.ID, "ip", server.PublicIP)
    }
    return nil
}
```

---

## 15. Future Extensibility

### Extension Points

```
1. Transport Plugins
   - Interface: TransportPlugin
   - Hot-reload via gRPC plugin system
   - Allow adding new stealth transports without recompile
   - Example: QUIC-based transport, SSH camouflage, DNS tunnel

2. Payment Providers
   - Interface: PaymentProcessor  
   - Add new providers via config
   - Planned: Apple Pay, Google Pay, direct bank transfer

3. Client Platforms
   - Config renderer per platform
   - Planned: browser extension, router firmware, smart TV

4. Regions
   - Zero-config new region addition
   - Just provision VPS + set registration token
   - Auto-discovery handles the rest

5. Bot Platforms
   - Abstract bot interface beyond Telegram
   - Planned: Discord bot, web dashboard, mobile app API

6. Analytics
   - Plugin system for custom analytics processors
   - Export to external BI tools
   - Machine learning pipeline for anomaly detection

7. Compliance
   - Modular compliance checks per jurisdiction
   - GDPR data export/deletion automation
   - Audit trail export
```

### Plugin Architecture

```go
// Transport plugin interface
type TransportPlugin interface {
    Name() string
    Version() string
    
    // Lifecycle
    Init(config json.RawMessage) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // Connection handling
    Accept(ctx context.Context) (net.Conn, error)
    Dial(ctx context.Context, addr string) (net.Conn, error)
    
    // Config
    GenerateClientConfig(userKey []byte) ([]byte, error)
    ValidateConnection(conn net.Conn) (bool, error)
    
    // Metadata
    Capabilities() TransportCapabilities
    HealthCheck(ctx context.Context) error
}

type TransportCapabilities struct {
    SupportsUDP       bool
    SupportsMux       bool
    MaxBandwidthMbps  int
    AntiDPILevel      int // 1-5
    Protocols         []string
}
```

### Roadmap

```
v1.0 - MVP
├── REALITY + HTTP/2 transport
├── Telegram bot (core commands)
├── Stripe billing
├── 3 regions
└── Manual node management

v1.5 - Scale
├── Auto-provisioning (Hetzner + Vultr)
├── Crypto payments
├── Referral system
├── 10 regions
└── Capacity autoscaler

v2.0 - Enterprise
├── Multi-transport (REALITY + QUIC + WebSocket)
├── Team accounts
├── Admin dashboard (web)
├── Custom branding / white-label
├── API for resellers
└── 25+ regions

v3.0 - Platform
├── Plugin marketplace
├── Self-hosted option
├── Mobile SDK
├── Router support
├── Split tunneling rules engine
└── AI-powered traffic optimization
```

---

## Appendix A: Project Structure

```
vpn-platform/
├── cmd/
│   ├── api-gateway/
│   │   └── main.go
│   ├── auth-service/
│   │   └── main.go
│   ├── user-service/
│   │   └── main.go
│   ├── subscription-service/
│   │   └── main.go
│   ├── billing-service/
│   │   └── main.go
│   ├── node-manager/
│   │   └── main.go
│   ├── config-distributor/
│   │   └── main.go
│   ├── telegram-bot/
│   │   └── main.go
│   ├── analytics-service/
│   │   └── main.go
│   └── node-agent/
│       └── main.go
├── internal/
│   ├── domain/           # Domain entities
│   │   ├── user.go
│   │   ├── subscription.go
│   │   ├── node.go
│   │   ├── session.go
│   │   ├── payment.go
│   │   └── config.go
│   ├── usecase/          # Business logic
│   │   ├── auth/
│   │   ├── subscription/
│   │   ├── billing/
│   │   ├── node/
│   │   └── vpn/
│   ├── adapter/          # External adapters
│   │   ├── postgres/
│   │   ├── redis/
│   │   ├── nats/
│   │   ├── stripe/
│   │   ├── crypto/
│   │   ├── telegram/
│   │   ├── hetzner/
│   │   └── vultr/
│   ├── transport/        # VPN transport implementations
│   │   ├── reality/
│   │   ├── quic/
│   │   ├── websocket/
│   │   └── plugin.go
│   └── pkg/              # Shared utilities
│       ├── logger/
│       ├── crypto/
│       ├── ratelimit/
│       ├── circuitbreaker/
│       └── metrics/
├── proto/                # gRPC definitions
│   ├── auth/v1/
│   ├── node/v1/
│   ├── subscription/v1/
│   ├── billing/v1/
│   └── config/v1/
├── deploy/
│   ├── kubernetes/
│   │   ├── base/
│   │   ├── overlays/
│   │   │   ├── dev/
│   │   │   ├── staging/
│   │   │   └── production/
│   │   └── kustomization.yaml
│   ├── terraform/
│   │   ├── modules/
│   │   │   ├── k8s-cluster/
│   │   │   ├── vpn-node/
│   │   │   └── dns/
│   │   └── environments/
│   └── docker/
│       ├── Dockerfile.service
│       └── Dockerfile.node-agent
├── migrations/
│   ├── 001_initial.up.sql
│   ├── 001_initial.down.sql
│   └── ...
├── docs/
│   ├── architecture.md
│   ├── api.md
│   └── runbook.md
├── go.mod
├── go.sum
├── Makefile
└── docker-compose.yml
```

## Appendix B: Key Dependencies

```go
// go.mod (key dependencies)
module github.com/vpn-platform/vpn-platform

go 1.22

require (
    // gRPC
    google.golang.org/grpc v1.62.0
    google.golang.org/protobuf v1.33.0
    
    // Database
    github.com/jackc/pgx/v5 v5.5.5
    github.com/redis/go-redis/v9 v9.5.1
    
    // NATS
    github.com/nats-io/nats.go v1.34.0
    
    // Telegram
    github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
    
    // Crypto
    golang.org/x/crypto v0.21.0
    github.com/refraction-networking/utls v1.6.4
    
    // Observability
    go.opentelemetry.io/otel v1.24.0
    github.com/prometheus/client_golang v1.19.0
    
    // Payments
    github.com/stripe/stripe-go/v78 v78.0.0
    
    // Infrastructure
    github.com/hashicorp/vault/api v1.12.0
    github.com/hetznercloud/hcloud-go/v2 v2.6.0
    
    // Utils
    github.com/google/uuid v1.6.0
    golang.org/x/time v0.5.0  // rate limiting
    github.com/sony/gobreaker v0.5.0  // circuit breaker
)
```

## Appendix C: Environment Variables

```bash
# Common
ENV=production
LOG_LEVEL=info
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318

# Database
DATABASE_URL=postgres://vpn:${DB_PASSWORD}@postgres:5432/vpn_platform?sslmode=require
DATABASE_MAX_CONNS=50
DATABASE_MIN_CONNS=10

# Redis
REDIS_URL=redis://:${REDIS_PASSWORD}@redis-sentinel:26379/0
REDIS_SENTINEL_MASTER=mymaster

# NATS
NATS_URL=nats://nats:4222
NATS_CLUSTER_NAME=vpn-cluster

# Auth
JWT_PRIVATE_KEY_PATH=/secrets/jwt/private.pem
JWT_PUBLIC_KEY_PATH=/secrets/jwt/public.pem
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

# Billing
STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
STRIPE_WEBHOOK_SECRET=${STRIPE_WEBHOOK_SECRET}
CRYPTO_API_KEY=${NOWPAYMENTS_API_KEY}

# Telegram
TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
TELEGRAM_WEBHOOK_URL=https://api.vpn-platform.com/bot/webhook

# VPN Transport
REALITY_PRIVATE_KEY=${REALITY_PRIVATE_KEY}
REALITY_SHORT_IDS=aabbccdd,11223344
REALITY_SERVER_NAME=www.microsoft.com
REALITY_DEST=www.microsoft.com:443

# Node Agent
CONTROL_PLANE_ADDR=node-manager.vpn-control:9005
NODE_AUTH_TOKEN=${NODE_AUTH_TOKEN}
HEARTBEAT_INTERVAL=10s
```

---

*Blueprint generated for production VPN SaaS platform. All components designed for horizontal scalability, anti-DPI resistance, and operational excellence.*
