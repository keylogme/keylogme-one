package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	k1 "github.com/keylogme/keylogme-one"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 10 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
	// Maximum message size allowed from peer.
	// maxMessageSize = 512
	//
	reconnectWait = 1 * time.Second
)

type Sender struct {
	ctx             context.Context
	origin_endpoint string
	url_ws          string
	apikey          string
	ws              *websocket.Conn
	reader          chan k1.PayloadLogger
	writer          chan []byte
	mu              sync.Mutex
}

func MustGetNewSender(ctx context.Context, origin, apikey string) *Sender {
	if origin == "" {
		log.Fatal("Origin endpoint is empty string")
	}
	if apikey == "" {
		log.Fatal("ApiKey endpoint is empty string")
	}

	trimmedOrigin := strings.TrimPrefix(origin, "http")
	url_ws := fmt.Sprintf("ws%s?apikey=%s", trimmedOrigin, apikey)
	s := &Sender{
		ctx:             ctx,
		origin_endpoint: origin,
		url_ws:          url_ws,
		apikey:          apikey,
		ws:              nil,
		reader:          make(chan k1.PayloadLogger),
		writer: make(
			chan []byte,
			100,
		), // buffered channel to store payloads when there is no connection
		mu: sync.Mutex{},
	}
	context.AfterFunc(ctx, s.Close)
	go s.processMessageQueue()
	return s
}

func (s *Sender) updateURL(url string) error {
	trimmedOrigin := strings.TrimPrefix(url, "http")
	url_ws := fmt.Sprintf("ws%s?apikey=%s", trimmedOrigin, s.apikey)
	s.origin_endpoint = url
	s.url_ws = url_ws

	s.closeWS()
	return s.connectWS()
}

func (s *Sender) closeWS() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ws != nil {
		slog.Info("Closing ws connection")
		_ = s.ws.WriteMessage(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		)
		s.ws.Close()
	}
	s.ws = nil
}

func (s *Sender) connectWS() error {
	if s.url_ws == "" {
		return fmt.Errorf("url_ws is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	ws, _, err := websocket.DefaultDialer.Dial(s.url_ws, nil)
	if err != nil {
		slog.Error("Could not dial server")
		return err
	}
	s.ws = ws
	return nil
}

func (s *Sender) run() error {
	if s.ws == nil {
		return fmt.Errorf("ws is nil")
	}
	closeChan := make(chan struct{})

	go s.read(closeChan)
	go s.write(closeChan)

	<-closeChan
	return nil
}

func (s *Sender) processMessageQueue() {
	for {
		if len(s.writer) > 0 {
			s.handleReconnects()
		}
		select {
		case <-time.After(reconnectWait):
			continue
		case <-s.ctx.Done():
			slog.Info("Stopping sender")
			return
		}
	}
}

func (s *Sender) handleReconnects() {
	slog.Info("Sender reconnecting...")
	defer s.closeWS()
	if err := s.connectWS(); err != nil {
		slog.Info(fmt.Sprintf("Could not connect to ws: %s\n", err.Error()))
		return
	}
	err := s.run()
	if err != nil {
		slog.Info(fmt.Sprintf("Run error : %s\n", err.Error()))
	}
}

func (s *Sender) write(closeConn chan struct{}) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer func() {
		slog.Info("------------------Closing write------------------")
	}()

	for {
		select {
		case <-closeConn:
			return
		case <-s.ctx.Done():
			return
		case p, ok := <-s.writer:
			if !ok {
				return
			}
			if s.ws == nil {
				slog.Info(fmt.Sprintf("ws disconnected and payload %s lost\n", string(p)))
				continue
			}
			slog.Info(fmt.Sprintf("Sending payload %s, queue %d\n", string(p), len(s.writer)))
			err := s.ws.WriteMessage(websocket.BinaryMessage, p)
			if err != nil {
				slog.Error(
					fmt.Sprintf("Failed to send %s : details %s\n", string(p), err.Error()),
				)
				return
			}
		case <-ticker.C:
			if err := s.ws.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				slog.Error(fmt.Sprintf("Could no set write deadline. %s\n", err.Error()))
				return
			}
			if err := s.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Error(fmt.Sprintf("Could no set ping message. %s\n", err.Error()))
				return
			}
		}
	}
}

func (s *Sender) read(closeConn chan struct{}) {
	defer close(closeConn)
	defer func() {
		slog.Info("------------------Closing read------------------")
	}()
	if err := s.ws.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		slog.Error(fmt.Sprintf("Could not set read deadline: %s\n", err.Error()))
		return
	}
	s.ws.SetPongHandler(
		func(string) error { _ = s.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil },
	)
	for {
		if s.ws == nil {
			slog.Info("reader: ws is closed")
			return
		}
		_, msg, err := s.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseInternalServerErr,
				websocket.CloseAbnormalClosure,
			) {
				slog.Error(fmt.Sprintf("Unexpected close error: %s\n", err.Error()))
			}
			slog.Info(fmt.Sprintf("sender:read: %s\n", err.Error()))
			return
		}
		slog.Info(fmt.Sprintf("received message: %s", msg))

		var payload k1.PayloadLogger
		err = json.Unmarshal(msg, &payload)
		if err != nil {
			slog.Info("failed to parse payload")
			continue
		}
		s.reader <- payload
	}
}

func (s *Sender) Send(p []byte) error {
	s.writer <- p
	return nil
}

func (s *Sender) Close() {
	slog.Info("💤 Close sender")
	close(s.reader)
	close(s.writer)
	s.closeWS()
}
