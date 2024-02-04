package repository

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"api/types"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type WebSocket struct {
	connections map[uuid.UUID][]*websocket.Conn
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// AddConnection implements WebSocketRepository.
func (ws *WebSocket) AddConnection(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	retrospectiveID, ok := ctx.Value("retrospective_id").(uuid.UUID)
	if !ok {
		return fmt.Errorf("retrospective id not found")
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	if _, ok := ws.connections[retrospectiveID]; !ok {
		return fmt.Errorf("retrospective doesn't exist")
	}

	ws.connections[retrospectiveID] = append(ws.connections[retrospectiveID], conn)

	<-ctx.Done()
	conn.Close()
	// TODO: remove connection from ws.connections[retrospectiveID] when closed

	return nil
}

// GetRetrospective implements WebSocketRepository.
func (*WebSocket) GetRetrospective(ctx context.Context, id uuid.UUID) (*types.Retrospective, error) {
	panic("unimplemented")
}

func NewWebSocket() (*WebSocket, error) {
	connections := make(map[uuid.UUID][]*websocket.Conn)
	return &WebSocket{
		connections: connections,
	}, nil
}

func (w *WebSocket) sendMessageToRetro(ctx context.Context, message types.WebSocketMessage) error {
	retrospectiveID, ok := ctx.Value("retrospective_id").(uuid.UUID)
	if !ok {
		return fmt.Errorf("retrospective id not found")
	}

	connections := w.connections[retrospectiveID]
	if connections == nil {
		return nil
	}

	for _, conn := range connections {
		err := conn.WriteJSON(message)
		if err != nil {
			log.Printf("Error sending message %+v to connection: %v", message, err)
		}
	}
	return nil
}

// CreateAnswer implements Repository.
func (w *WebSocket) CreateAnswer(ctx context.Context, answer *types.Answer) error {
	message := types.WebSocketMessage{
		Action: "create",
		Type:   "answer",
		Value:  answer,
	}

	return w.sendMessageToRetro(ctx, message)
}

// CreateQuestion implements Repository.
func (w *WebSocket) CreateQuestion(ctx context.Context, question *types.Question) error {
	message := types.WebSocketMessage{
		Action: "create",
		Type:   "question",
		Value:  question,
	}

	return w.sendMessageToRetro(ctx, message)
}

// DeleteAnswer implements Repository.
func (w *WebSocket) DeleteAnswer(ctx context.Context, answer *types.Answer) error {
	message := types.WebSocketMessage{
		Action: "delete",
		Type:   "answer",
		Value:  answer,
	}

	return w.sendMessageToRetro(ctx, message)
}

// DeleteQuestion implements Repository.
func (w *WebSocket) DeleteQuestion(ctx context.Context, id uuid.UUID) (*types.Question, error) {
	message := types.WebSocketMessage{
		Action: "delete",
		Type:   "question",
		Value:  types.Object{ID: id},
	}

	return nil, w.sendMessageToRetro(ctx, message)
}

// DeleteRetrospective implements Repository.
func (w *WebSocket) CreateRetrospective(ctx context.Context, retro *types.Retrospective) error {
	w.connections[retro.ID] = make([]*websocket.Conn, 0)
	return nil
}

// DeleteRetrospective implements Repository.
func (w *WebSocket) DeleteRetrospective(ctx context.Context, id uuid.UUID) (*types.Retrospective, error) {
	delete(w.connections, id)
	return nil, nil
}

// UpdateAnswer implements Repository.
func (w *WebSocket) UpdateAnswer(ctx context.Context, answer *types.Answer) error {
	message := types.WebSocketMessage{
		Action: "update",
		Type:   "answer",
		Value:  answer,
	}

	return w.sendMessageToRetro(ctx, message)
}

// UpdateQuestion implements Repository.
func (w *WebSocket) UpdateQuestion(ctx context.Context, question *types.Question) error {
	message := types.WebSocketMessage{
		Action: "update",
		Type:   "question",
		Value:  question,
	}

	return w.sendMessageToRetro(ctx, message)
}

// UpdateRetrospective implements Repository.
func (*WebSocket) UpdateRetrospective(ctx context.Context, retro *types.Retrospective) error {
	panic("unimplemented")
}
