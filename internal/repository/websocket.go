package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"api/types"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var _ WebSocketRepository = (*WebSocket)(nil)

type WebSocket struct {
	connections map[uuid.UUID][]*websocket.Conn
	logger      *zap.Logger
}

type WebSocketParams struct {
	fx.In
	Logger *zap.Logger
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewWebSocket(p WebSocketParams) WebSocketRepository {
	connections := make(map[uuid.UUID][]*websocket.Conn)
	return &WebSocket{
		connections: connections,
		logger:      p.Logger,
	}
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

	i := len(ws.connections[retrospectiveID])
	ws.connections[retrospectiveID] = append(ws.connections[retrospectiveID], conn)

	for {
		err := conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		if err != nil {
			ws.logger.Error("error setting read deadline", zap.Error(err))
			break
		}

		var message types.WebSocketMessage
		err = conn.ReadJSON(&message)

		if err == nil {
			if message.Type == "ping" {
				errWrite := conn.WriteJSON(types.WebSocketMessage{Type: "pong"})
				if errWrite != nil {
					ws.logger.Error("error writing pong message", zap.Error(errWrite))
				}
			}
			continue
		}

		if netErr, ok := err.(net.Error); (ok && netErr.Timeout()) || websocket.IsUnexpectedCloseError(err) ||
			errors.Is(err, io.EOF) {
			break
		}

		ws.logger.Error("error reading json message", zap.Error(err))
	}
	conn.Close()
	ws.connections[retrospectiveID][i] = nil

	return nil
}

// GetRetrospective implements WebSocketRepository.
func (*WebSocket) GetRetrospective(ctx context.Context, id uuid.UUID) (*types.Retrospective, error) {
	panic("unimplemented")
}

func (w *WebSocket) sendMessageToRetro(ctx context.Context, message types.WebSocketMessage, retrospectiveID *uuid.UUID) error {
	if retrospectiveID == nil {
		id, ok := ctx.Value("retrospective_id").(uuid.UUID)
		if !ok {
			return fmt.Errorf("retrospective id not found")
		}
		retrospectiveID = &id
	}

	connections := w.connections[*retrospectiveID]
	if connections == nil {
		return nil
	}

	for _, conn := range connections {
		if conn == nil {
			continue
		}
		err := conn.WriteJSON(message)
		if err != nil {
			w.logger.Error("error sending message to connection", zap.Any("message", message), zap.Error(err))
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

	return w.sendMessageToRetro(ctx, message, nil)
}

// CreateQuestion implements Repository.
func (w *WebSocket) CreateQuestion(ctx context.Context, question *types.Question) error {
	message := types.WebSocketMessage{
		Action: "create",
		Type:   "question",
		Value:  question,
	}

	return w.sendMessageToRetro(ctx, message, nil)
}

// DeleteAnswer implements Repository.
func (w *WebSocket) DeleteAnswer(ctx context.Context, answer *types.Answer) error {
	message := types.WebSocketMessage{
		Action: "delete",
		Type:   "answer",
		Value:  answer,
	}

	return w.sendMessageToRetro(ctx, message, nil)
}

// DeleteQuestion implements Repository.
func (w *WebSocket) DeleteQuestion(ctx context.Context, id uuid.UUID) (*types.Question, error) {
	message := types.WebSocketMessage{
		Action: "delete",
		Type:   "question",
		Value:  types.Object{ID: id},
	}

	return nil, w.sendMessageToRetro(ctx, message, nil)
}

func (*WebSocket) GetOldRetrospectives(ctx context.Context, date time.Time) ([]uuid.UUID, error) {
	panic("unimplemented")
}

func (*WebSocket) GetAllRetrospectives(ctx context.Context) ([]uuid.UUID, error) {
	panic("unimplemented")
}

// CreateRetrospective implements Repository.
func (w *WebSocket) CreateRetrospective(ctx context.Context, retro *types.Retrospective) error {
	w.connections[retro.ID] = make([]*websocket.Conn, 0)
	return nil
}

// DeleteRetrospective implements Repository.
func (w *WebSocket) DeleteRetrospective(ctx context.Context, id uuid.UUID) (*types.Retrospective, error) {
	delete(w.connections, id)

	message := types.WebSocketMessage{
		Action: "delete",
		Type:   "retrospective",
		Value:  types.Object{ID: id},
	}

	return nil, w.sendMessageToRetro(ctx, message, &id)
}

// UpdateAnswer implements Repository.
func (w *WebSocket) UpdateAnswer(ctx context.Context, answer *types.Answer) error {
	message := types.WebSocketMessage{
		Action: "update",
		Type:   "answer",
		Value:  answer,
	}

	return w.sendMessageToRetro(ctx, message, nil)
}

// UpdateQuestion implements Repository.
func (w *WebSocket) UpdateQuestion(ctx context.Context, question *types.Question) error {
	message := types.WebSocketMessage{
		Action: "update",
		Type:   "question",
		Value:  question,
	}

	return w.sendMessageToRetro(ctx, message, nil)
}

// UpdateRetrospective implements Repository.
func (w *WebSocket) UpdateRetrospective(ctx context.Context, retro *types.Retrospective) error {
	message := types.WebSocketMessage{
		Action: "update",
		Type:   "retrospective",
		Value:  retro,
	}

	return w.sendMessageToRetro(ctx, message, &retro.ID)
}

// AddVoteToAnswer implements Repository.
func (w *WebSocket) AddVoteToAnswer(ctx context.Context, _ uuid.UUID, answer *types.Answer, _ string) error {
	message := types.WebSocketMessage{
		Action: "add_vote",
		Type:   "answer",
		Value:  answer,
	}

	return w.sendMessageToRetro(ctx, message, nil)
}

// RemoveVoteFromAnswer implements Repository.
func (w *WebSocket) RemoveVoteFromAnswer(ctx context.Context, answer *types.Answer, _ string) error {
	message := types.WebSocketMessage{
		Action: "remove_vote",
		Type:   "answer",
		Value:  answer,
	}

	return w.sendMessageToRetro(ctx, message, nil)
}
