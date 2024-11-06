package internal

import (
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

type Sender struct {
	origin_endpoint string
	url_ws          string
	ws              *websocket.Conn
	max_retries     int64
	retry_duration  time.Duration
	isClosed        bool
	reader          chan bool
	writer          chan []byte
	// done            chan struct{}
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

	ws, err := websocket.Dial(url_ws, "", origin)
	if err != nil {
		log.Fatal(err.Error())
	}
	s := &Sender{
		origin_endpoint: origin,
		url_ws:          url_ws,
		ws:              ws,
		max_retries:     10000,
		retry_duration:  1 * time.Second,
		isClosed:        false,
		reader:          make(chan bool),
		writer:          make(chan []byte),
		// done:            make(chan struct{}),
	}
	s.read()
	s.write()

	return s
}

func (s *Sender) write() {
	go func(s *Sender) {
		// defer close(s.done)
		for p := range s.writer {
		retry:
			slog.Info(fmt.Sprintf("Sending payload %s\n", string(p)))
			_, err := s.ws.Write(p)
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to send %s : details %s\n", string(p), err.Error()))
				if err := s.reconnect(); err != nil {
					s.isClosed = true
				}
				// retry
				slog.Info(fmt.Sprintf("Retrying payload %s\n", string(p)))
				goto retry
			}
		}
	}(s)
}

func (s *Sender) read() {
	go func(s *Sender) {
		// defer close(s.done)
		for {
			var msg []byte
			_, err := s.ws.Read(msg)
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", msg)
			// TODO: add a select
			// first select to read EOF
			// second every 5 s ping server
			// for above better migrate and use gorilla websocket to handle these things easier?
			// ws/websocket also supports deadlines
			// test: could it be to replicate this, to shutdown server?
		}
	}(s)
}

func (s *Sender) Send(p []byte) error {
	s.writer <- p
	return nil
}

func (s *Sender) reconnect() error {
	for i := range s.max_retries {
		slog.Info("Waiting for reconnecting...")
		time.Sleep(s.retry_duration)
		ws_reconnect, err := websocket.Dial(s.url_ws, "", s.origin_endpoint)
		if err != nil {
			continue
		}
		slog.Info(fmt.Sprintf("Reconnected after %d retries\n", i+1))
		s.read()
		s.isClosed = false
		s.ws = ws_reconnect
		return nil
	}
	return fmt.Errorf("Maximum retries excedeed\n")
}

func (s *Sender) Close() error {
	close(s.reader)
	close(s.writer)
	// close(s.done)
	return s.ws.Close()
}
