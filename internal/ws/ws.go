package ws

import (
	"context"
	"net/http"

	"github.com/centrifugal/centrifuge"
)

type Hub struct {
	Node *centrifuge.Node
}

func NewHub() (*Hub, error) {
	node, err := centrifuge.New(centrifuge.Config{})
	if err != nil {
		return nil, err
	}

	node.OnConnecting(func(ctx context.Context, e centrifuge.ConnectEvent) (centrifuge.ConnectReply, error) {
		return centrifuge.ConnectReply{}, nil
	})

	if err := node.Run(); err != nil {
		return nil, err
	}

	return &Hub{Node: node}, nil
}

func (h *Hub) Handler() http.Handler {
	return centrifuge.NewWebsocketHandler(h.Node, centrifuge.WebsocketConfig{})
}

func (h *Hub) Publish(channel string, data []byte) error {
	_, err := h.Node.Publish(channel, data)
	return err
}
