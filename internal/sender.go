package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"runtime"
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
	reconnectWait = 1 * time.Second
	// size queue (to have some buffer if there is loss internet)
	maxQueueSizeWrite = 100
)

type Sender struct {
	ctx             context.Context
	mu              sync.Mutex
	done            chan struct{}
	wg              sync.WaitGroup
	origin_endpoint string
	url_ws          string
	apikey          string
	ws              *websocket.Conn
	reader          chan k1.PayloadLogger
	writer          chan []byte
	closed          bool
}

func MustGetNewSender(ctx context.Context, origin, apikey string) *Sender {
	if origin == "" {
		log.Fatal("Origin endpoint is empty string")
	}
	if apikey == "" {
		log.Fatal("ApiKey endpoint is empty string")
	}

	trimmedOrigin := strings.TrimPrefix(origin, "http")
	url_ws := fmt.Sprintf("ws%s?apikey=%s&os=%s", trimmedOrigin, apikey, runtime.GOOS)
	s := &Sender{
		ctx:             ctx,
		mu:              sync.Mutex{},
		done:            make(chan struct{}),
		wg:              sync.WaitGroup{},
		origin_endpoint: origin,
		url_ws:          url_ws,
		apikey:          apikey,
		ws:              nil,
		reader:          make(chan k1.PayloadLogger),
		writer: make(
			chan []byte,
			maxQueueSizeWrite,
		), // buffered channel to store payloads when there is no connection
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

	s.Close()
	return s.connectWS()
}

func (s *Sender) closeWS() {
	slog.Debug("closeWS: closing ws...")
	s.mu.Lock()
	slog.Debug("closeWS: got lock")
	defer s.mu.Unlock()
	if s.ws == nil {
		slog.Debug("closeWS: ws is nil")
		return
	}
	slog.Debug("closeWS: closing ws connection")
	_ = s.ws.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)
	s.ws.Close()
	s.wg.Wait() // wait for goroutines to finish
	slog.Debug("closeWS: ws connection closed")
	s.ws = nil
}

func (s *Sender) connectWS() error {
	slog.Debug("connectWS: connecting ws...")
	if s.url_ws == "" {
		return fmt.Errorf("url_ws is empty")
	}
	s.mu.Lock()
	slog.Debug("connectWS: got lock")
	defer s.mu.Unlock()
	ws, resp, err := websocket.DefaultDialer.Dial(s.url_ws, nil)
	if resp != nil && resp.StatusCode == http.StatusUnauthorized {
		slog.Error("connectWS: Unauthorized access - check API key")
		s.Close()
		return fmt.Errorf("unauthorized access")
	}
	if err != nil {
		slog.Error("connectWS: Could not dial server")
		return err
	}
	s.ws = ws
	s.closed = false
	s.done = make(chan struct{})
	slog.Debug("connectWS: connected")
	return nil
}

func (s *Sender) run() error {
	if s.ws == nil {
		return fmt.Errorf("ws is nil")
	}

	s.wg.Add(1)
	go s.read()
	s.wg.Add(1)
	go s.write()

	s.wg.Wait()
	slog.Debug("Sender run completed")
	return nil
}

func (s *Sender) processMessageQueue() {
	defer func() {
		slog.Debug("processMessageQueue: closing")
	}()
	for {
		if len(s.writer) > 0 {
			s.handleReconnects()
		}
		if s.closed {
			slog.Debug("process: sender is closed")
			return
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
	if s.closed {
		return
	}
	s.closeWS()
	if err := s.connectWS(); err != nil {
		slog.Info(fmt.Sprintf("Could not connect to ws (%s): %s\n", s.url_ws, err.Error()))
		return
	}
	err := s.run()
	if err != nil {
		slog.Info(fmt.Sprintf("Run error : %s\n", err.Error()))
	}
}

func (s *Sender) writeMessage(messageType int, message []byte) error {
	// slog.Info("write message : getting lock...")
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ws == nil {
		slog.Debug("ws is nil")
		return websocket.ErrCloseSent // Return an error indicating the connection is closed
	}

	// Set the write deadline
	// slog.Info("setting write deadline")
	if err := s.ws.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		slog.Error(fmt.Sprintf("Error setting write deadline: %v", err))
		return err
	}

	// Write the message
	// slog.Info("writing message")
	err := s.ws.WriteMessage(messageType, message)
	if err != nil {
		log.Printf("Error writing message: %v", err)
		return err
	}
	// slog.Info("message sent")
	return nil
}

func (s *Sender) write() {
	defer s.wg.Done()
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer func() {
		slog.Debug("------------------Closing write------------------")
	}()

	for {
		select {
		case <-s.done:
			slog.Info("writer: done signal")
			return
		case <-s.ctx.Done():
			slog.Info("writer: ctx done")
			return
		case p, ok := <-s.writer:
			if !ok {
				return
			}
			slog.Debug(fmt.Sprintf("Sending payload %s, queue %d\n", string(p), len(s.writer)))
			err := s.writeMessage(websocket.BinaryMessage, p)
			if err != nil {
				slog.Error(
					fmt.Sprintf("Failed to send %s : details %s\n", string(p), err.Error()),
				)
				close(s.done)
				return
			}

		case <-ticker.C:
			slog.Debug("Sending ping")
			err := s.writeMessage(websocket.PingMessage, nil)
			if err != nil {
				slog.Error(fmt.Sprintf("Error sending ping: %v", err))
				close(s.done)
				return
			}
		}
	}
}

func (s *Sender) read() {
	defer s.wg.Done()

	defer func() {
		slog.Debug("------------------Closing read------------------")
	}()
	if err := s.ws.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		slog.Error(fmt.Sprintf("Could not set read deadline: %s\n", err.Error()))
		return
	}
	s.ws.SetPongHandler(
		func(string) error { _ = s.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil },
	)
	for {
		select {
		case <-s.done:
			slog.Info("reader: done signal")
			return
		case <-s.ctx.Done():
			slog.Info("reader: ctx done")
			return
		default:
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
				// s.done <- struct{}{} // Signal to stop other goroutines
				close(s.done)
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
}

func (s *Sender) Send(p []byte) error {
	if len(s.writer) == maxQueueSizeWrite {
		slog.Info("Sender queue is full")
	}
	if s.closed {
		slog.Info(
			"Sender is closed, check api key is valid or not being used somewhere else",
		)
	}
	s.writer <- p
	return nil
}

func (s *Sender) Close() {
	slog.Info("💤 Close sender ")
	if s.closed {
		slog.Info("close: sender already closed")
		return
	}
	s.closed = true
	s.closeWS()
	s.wg.Wait()
	slog.Debug("close: sender closed")
}
