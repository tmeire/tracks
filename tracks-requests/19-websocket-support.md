# Feature Request: WebSocket Support

**Priority:** Low  
**Status:** Open

## Description

The framework lacks built-in WebSocket support for real-time bidirectional communication, which is needed for modern interactive applications.

## Current Gap

No WebSocket handlers for:
- Real-time updates
- Chat applications
- Live notifications
- Collaborative editing
- Live dashboards

## Required Functionality

1. **WebSocket Upgrade**: Handle HTTP upgrade to WebSocket
2. **Connection Management**: Track active connections
3. **Message Handling**: Text and binary message support
4. **Broadcasting**: Send to multiple connections
5. **Rooms/Channels**: Group connections by topic
6. **Authentication**: Secure WebSocket connections
7. **Heartbeat/Ping**: Keep connections alive
8. **Graceful Shutdown**: Close connections cleanly
9. **Connection Limits**: Prevent resource exhaustion

## Proposed API

```go
// Define WebSocket handler
type ChatHandler struct {
    rooms map[string]*tracks.Room
}

func (h *ChatHandler) Handle(conn *tracks.WebSocketConn) {
    // Authenticate
    user := conn.Context().Value("user").(*User)
    
    // Join room
    room := h.getRoom(conn.Param("room_id"))
    room.Join(conn)
    
    // Handle messages
    for {
        msg, err := conn.ReadMessage()
        if err != nil {
            room.Leave(conn)
            return
        }
        
        // Broadcast to room
        room.Broadcast(map[string]any{
            "user": user.Name,
            "message": string(msg),
            "timestamp": time.Now(),
        })
    }
}

// Register WebSocket route
router.WebSocket("/ws/chat/:room_id", &ChatHandler{})

// With middleware
router.WebSocket("/ws/notifications", &NotificationHandler{},
    tracks.AuthMiddleware(),
)

// Server-side broadcasting
func (h *Handler) NotifyUsers(r *http.Request) (any, error) {
    ws := tracks.WebSocketHub()
    
    // Broadcast to all
    ws.Broadcast(map[string]any{
        "type": "announcement",
        "message": "System maintenance in 5 minutes",
    })
    
    // Send to specific user
    ws.SendToUser("user-123", map[string]any{
        "type": "notification",
        "message": "You have a new message",
    })
    
    // Send to room
    ws.Room("room-456").Broadcast(map[string]any{
        "type": "update",
        "data": updatedData,
    })
    
    return nil, nil
}

// Configuration
config := tracks.Config{
    WebSocket: tracks.WebSocketConfig{
        MaxConnections: 10000,
        ReadTimeout:    60 * time.Second,
        WriteTimeout:   10 * time.Second,
        PingInterval:   30 * time.Second,
        MaxMessageSize: 1024 * 1024, // 1MB
    },
}
```

## Client Example

```javascript
const ws = new WebSocket('wss://example.com/ws/chat/room-123');

ws.onopen = () => {
    console.log('Connected');
    ws.send(JSON.stringify({type: 'join', user: 'john'}));
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('Received:', data);
};

ws.onclose = () => {
    console.log('Disconnected');
};
```

## Use Cases

- Real-time chat
- Live notifications
- Stock/crypto price tickers
- Collaborative editing
- Multiplayer games
- Live dashboards
- Real-time analytics

## Acceptance Criteria

- [ ] HTTP upgrade handling
- [ ] Text and binary message support
- [ ] Connection management
- [ ] Room/channel system
- [ ] Broadcasting to all/room/user
- [ ] Authentication middleware support
- [ ] Heartbeat/ping-pong
- [ ] Graceful connection closure
- [ ] Connection limits
- [ ] Message size limits
- [ ] Integration with existing middleware
- [ ] Documentation and examples

## Architecture

```
Client WebSocket -> HTTP Upgrade -> WebSocket Handler
                          |
                          v
                   Connection Manager
                          |
            +-------------+-------------+
            |                           |
            v                           v
     Message Handler               Room Manager
            |                           |
            v                           v
     Broadcast Hub              Individual Rooms
            |                           |
            +-------------+-------------+
                          |
                          v
                    Other Clients
```
