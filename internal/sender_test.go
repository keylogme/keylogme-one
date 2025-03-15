package internal

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{} // use default options
type serverHandler struct {
	DataLog  *[]string
	Delay    time.Duration
	Deadline time.Duration
}

func (h serverHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		fmt.Printf("server test: %d %s\n", mt, string(message))
		*h.DataLog = append(*h.DataLog, string(message))
		if h.Delay != 0 {
			time.Sleep(h.Delay)
		}
		if h.Deadline != 0 {
			_ = c.SetReadDeadline(time.Now().Add(h.Deadline))
		}
	}
}

func getServer(p *[]string, delay time.Duration, deadline time.Duration) *httptest.Server {
	sh := serverHandler{DataLog: p, Delay: delay, Deadline: deadline}
	s := httptest.NewServer(sh)
	return s
}

func TestSender(t *testing.T) {
	expected := []string{}
	server := getServer(&expected, 0, 0)

	sender := MustGetNewSender(server.URL, "fake-key")
	// send payload
	payloads := []string{"1", "2", "3", "4", "5"}
	for _, p := range payloads {
		slog.Info(fmt.Sprintf("sending %s\n", p))
		err := sender.Send([]byte(p))
		if err != nil {
			t.Fatal(err.Error())
		}
	}
	// wait so server process the last payload
	time.Sleep(1 * time.Second)
	// check results
	if len(payloads) != len(expected) {
		t.Fatal("expected same len")
	}
	for idx := range payloads {
		fmt.Printf("Sent %#v , expected %#v\n", payloads[idx], expected[idx])
		if payloads[idx] != expected[idx] {
			t.Fatal("wrong expected value")
		}
	}
}

// TODO: test with client disconnection-> goal is to check server disconnects as well

func TestSenderWithDelay(t *testing.T) {
	expected := []string{}
	server := getServer(&expected, 1*time.Second, 0)
	defer server.Close()

	sender := MustGetNewSender(server.URL, "fake-key")
	// defer sender.Close()
	// send payload
	// server.CloseClientConnections()
	payloads := []string{"1", "2", "3", "4", "5"}
	for _, p := range payloads {
		fmt.Printf("sending %s\n", p)
		err := sender.Send([]byte(p))
		if err != nil {
			t.Fatal(err.Error())
		}
	}
	time.Sleep(10 * time.Second)
	fmt.Println(len(expected))
}

func TestWithDeadline(t *testing.T) {
	expected := []string{}
	deadline := 1 * time.Second
	server := getServer(&expected, 0, deadline)
	defer server.Close() // server close will close all client connections

	sender := MustGetNewSender(server.URL, "fake-key")
	// disconnect
	// sender.Close()
	payloads := []string{"1", "2", "3", "4", "5"}
	slog.Info("sending 1")
	_ = sender.Send([]byte("1"))
	slog.Info("waiting X seconds so server disconnects ws...")
	time.Sleep(2 * deadline)

	for _, p := range []string{"2", "3", "4", "5"} {
		slog.Info(fmt.Sprintf("sending %s\n", p))
		err := sender.Send([]byte(p))
		if err != nil {
			t.Fatal(err.Error())
		}
	}
	// wait so server process the last payload
	time.Sleep(1 * time.Second)
	// check results
	if len(payloads) != len(expected) {
		t.Fatalf("expected same len : %#v vs %#v\n", payloads, expected)
	}
	for idx := range payloads {
		fmt.Printf("Sent %#v , expected %#v\n", payloads[idx], expected[idx])
		if payloads[idx] != expected[idx] {
			t.Fatal("wrong expected value")
		}
	}
}

func TestServerNotAvailable(t *testing.T) {
	expected := []string{}
	server := getServer(&expected, 0, 0)
	slog.Info(server.URL)

	sender := MustGetNewSender(server.URL, "fake-key")

	payloads := []string{"1", "2", "3", "4", "5"}
	for idx, p := range payloads {
		if idx == 1 {
			server.Close()
		}
		slog.Info(fmt.Sprintf("sending %s\n", p))
		err := sender.Send([]byte(p))
		if err != nil {
			t.Fatal(err.Error())
		}
	}
	// start server again
	time.Sleep(5 * time.Second)
	slog.Info("Server started again")
	server = getServer(&expected, 0, 0)
	slog.Info(server.URL)
	err := sender.updateURL(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(3 * time.Second)
	// check results
	if len(payloads) != len(expected) {
		t.Fatalf("expected same len : %#v vs %#v\n", payloads, expected)
	}
}
