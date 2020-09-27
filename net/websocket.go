package net

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/flarco/gutil"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"net/http"
	"os"
	"strings"
	"time"
)

// WsConn is a websocket server/client
type WsConn struct {
	ID            string
	Conn          *websocket.Conn
	ConnectTime   time.Time
	Closed        bool
	Context       gutil.Context
	ClientConns   map[string]*WsConn                             // RW
	Handlers      map[MessageType]func(Message) Message          // R
	SuperHandlers map[MessageType]func(*WsConn, Message) Message // R
	ReplyHandlers map[string]func(msg Message)                   // RW
	ParentWc      *WsConn
	Props         map[string]interface{} // R
	DisableHb     bool
	LastHeartbeat time.Time
	Reconnects    int // number of time worker reconnects
	done          chan struct{}
	counter       uint64
	heartBeating  bool
}

// NewWsConn returns a websocket empty connection
func NewWsConn() (wc *WsConn) {
	return NewWsConnContext(context.Background())
}

// NewWsConnContext returns a websocket empty connection
func NewWsConnContext(ctx context.Context) (wc *WsConn) {
	wc = &WsConn{
		ConnectTime:   time.Now(),
		Context:       gutil.NewContext(ctx, 100),
		Handlers:      map[MessageType]func(msg Message) (respMsg Message){},
		SuperHandlers: map[MessageType]func(*WsConn, Message) Message{},
		ReplyHandlers: map[string]func(msg Message){},
		ClientConns:   map[string]*WsConn{},
		LastHeartbeat: time.Now(),
		Props:         map[string]interface{}{},
		done:          make(chan struct{}),
	}
	return
}

// NewServerClientEcho returns a websocket server for echo
func (wc *WsConn) NewServerClientEcho(clientID string, c echo.Context) (err error) {
	wc.Context.Mux.Lock()
	defer wc.Context.Mux.Unlock()
	upgrader := websocket.Upgrader{}
	if os.Getenv("SLINGELT_DEV") == "TRUE" {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
	}
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return gutil.ErrJSON(http.StatusInternalServerError, err, "Error upgrading to websocket")
	}

	wsConn := NewWsConnContext(wc.Context.Ctx)
	wsConn.ID = clientID
	wsConn.ParentWc = wc
	wsConn.Conn = conn
	wsConn.SuperHandle(ClientHeartbeatMsg, handleClientHeartbeat)

	for k, handler := range wc.Handlers {
		wsConn.Handlers[k] = handler
	}
	for k, superHandler := range wc.SuperHandlers {
		wsConn.SuperHandlers[k] = superHandler
	}

	if cwc, ok := wc.ClientConns[clientID]; ok {
		cwc.Close()
		rc := cwc.Reconnects
		wc.ClientConns[clientID] = wsConn
		wc.ClientConns[clientID].Reconnects = rc + 1
	} else {
		wc.ClientConns[clientID] = wsConn
	}

	go wsConn.Loop(0)

	return nil
}

// NewClient connects to a ws server
func (wc *WsConn) NewClient(URL string, header map[string]string) (err error) {
	headerHTTP := http.Header{}
	for k, v := range header {
		headerHTTP[k] = []string{v}
	}

	ctx, cancel := context.WithTimeout(wc.Context.Ctx, 10*time.Second)
	defer cancel()
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, URL, headerHTTP)
	if err != nil {
		err = gutil.Error(err, "Could not connect to %s", URL)
		return
	}

	if resp != nil && resp.StatusCode != 200 && resp.StatusCode != 101 {
		err = fmt.Errorf("Could not connect to %s. Got status code %d: %s", URL, resp.StatusCode, resp.Status)
		err = gutil.Error(err)
		return
	}

	wc.Conn = conn
	wc.ID = gutil.NewTsID()
	wc.ConnectTime = time.Now()
	wc.LastHeartbeat = time.Now()
	wc.Closed = false

	if !wc.DisableHb && !wc.heartBeating {
		// start heartbeating
		go wc.HeartbeatLoop()
	}

	return nil
}

// Broadcast send a message to all ClientConns
func (wc *WsConn) Broadcast(msg Message) error {
	eg := gutil.ErrorGroup{}
	for id, conn := range wc.ClientConns {
		err := conn.SendMessage(msg, nil)
		if err != nil {
			err = gutil.Error(err, "could not send message to: "+id)
		}
		eg.Capture(err)
	}
	return eg.Err()
}

// SendMessageToClient send a message to a client
func (wc *WsConn) SendMessageToClient(clientID string, msg Message, replyHandler func(msg Message)) (err error) {
	wc.Context.Mux.Lock()
	clientConn, ok := wc.ClientConns[clientID]
	wc.Context.Mux.Unlock()
	if !ok {
		err = gutil.Error(fmt.Errorf("client '%s' does not exists", clientID))
		return
	}
	err = clientConn.SendMessage(msg, replyHandler)
	if err != nil {
		err = gutil.Error(err, "could not send message to client "+clientID)
		if strings.Contains(err.Error(), "websocket: close") {
			clientConn.Closed = true
			clientConn.Close()
		}
	}
	return err
}

// SendMessageToClientWait send a message to a client and waits for a response
func (wc *WsConn) SendMessageToClientWait(clientID string, msg Message, timeOut ...int) (respMsg Message, err error) {
	wc.Context.Mux.Lock()
	clientConn, ok := wc.ClientConns[clientID]
	wc.Context.Mux.Unlock()
	if !ok {
		err = gutil.Error(fmt.Errorf("client '%s' does not exists", clientID))
		return
	}
	respMsg, err = clientConn.SendMessageWait(msg, timeOut...)
	if err != nil {
		err = gutil.Error(err, "could not send message to client "+clientID)
		if strings.Contains(err.Error(), "websocket: close") {
			clientConn.Closed = true
			clientConn.Close()
		}
	}
	return
}

// SendMessageWait send a message and waits for a response
// with default timeout of 60 sec
func (wc *WsConn) SendMessageWait(msg Message, timeOut ...int) (respMsg Message, err error) {

	to := 60 * time.Second
	if len(timeOut) > 0 {
		to = time.Duration(timeOut[0]) * time.Second
	}

	replyChn := make(chan Message)
	replyHandler := func(msg Message) {
		replyChn <- msg
	}

	err = wc.SendMessage(msg, replyHandler)
	if err != nil {
		err = gutil.Error(err, "could not send message")
		return
	}

	// wait for response with timeout
	timer := time.NewTimer(to)
	select {
	case <-timer.C:
		err = gutil.Error(fmt.Errorf("timeout. no response received for message %s", msg.Type))
		return
	case respMsg = <-replyChn:
		return
	}
}

func (wc *WsConn) writeMessage(msgType int, data []byte, timeoutSec int) (err error) {
	if wc.Closed {
		err = gutil.Error(fmt.Errorf("connection is closed"))
		return
	}

	wc.Context.Mux.Lock()
	defer wc.Context.Mux.Unlock()

	deadline := time.Now().Add(time.Second * time.Duration(timeoutSec))
	wc.Conn.SetWriteDeadline(deadline)
	err = wc.Conn.WriteMessage(msgType, data)
	if err != nil {
		gutil.LogError(err, "could not write message, resetting")
	}

	return err
}

// SendMessage send a message and does not wait for a response
// but runs the provided replyHandler on reply
func (wc *WsConn) SendMessage(msg Message, replyHandler func(Message)) (err error) {
	if wc.Closed {
		return fmt.Errorf("wc is closed")
	}
	if replyHandler != nil {
		wc.Context.Mux.Lock()
		wc.ReplyHandlers[msg.ReqID] = replyHandler
		wc.Context.Mux.Unlock()
	}
	if wc.Conn == nil {
		err = fmt.Errorf("wc.Conn not defined. In server mode?")
		return gutil.Error(err)
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		err = gutil.Error(err, "could not marshall message")
		return
	}
	err = wc.writeMessage(websocket.TextMessage, msgBytes, 4)
	// need to handle reconnects here?
	if err != nil {
		err = gutil.Error(err, "could not send message")
	}
	return
}

// SuperHandle adds a message type handler, providing the wc object
func (wc *WsConn) SuperHandle(msgType MessageType, handler func(*WsConn, Message) Message) {
	wc.SuperHandlers[msgType] = handler
}

// Handle adds a message type handler
func (wc *WsConn) Handle(msgType MessageType, handler func(msg Message) (respMsg Message)) {
	wc.Handlers[msgType] = handler
}

// Close closes the websocker connection
func (wc *WsConn) Close() {
	if wc == nil {
		return
	}

	if wc.ParentWc != nil {
		wc.ParentWc.Context.Mux.Lock()
		delete(wc.ParentWc.ClientConns, wc.ID)
		wc.ParentWc.Context.Mux.Unlock()
		wc.ParentWc = nil
	}

	if wc.Closed {
		return
	}

	wc.Context.Mux.Lock()
	for _, conn := range wc.ClientConns {
		conn.writeMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), 2)
	}
	wc.Context.Mux.Unlock()

	if wc.Conn != nil {
		wc.writeMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), 2)
	}

	wc.Closed = true
	gutil.Debug("closed Ws client connection for " + wc.ID)
}

// HeartbeatLoop runs infinitely sending heartbeats every 10 sec
func (wc *WsConn) HeartbeatLoop() {
	wc.heartBeating = true
	defer func() { wc.heartBeating = false }()

	for {
		hbMsg := NewMessage(ClientHeartbeatMsg, nil)
		err := wc.SendMessage(hbMsg, nil)
		if err != nil {
			gutil.LogError(err, "could not send heartbeat message")
			return
		}
		time.Sleep(10 * time.Second)
	}
}

// Loop runs for n number of times and waits for messages
// 0 is infinite
func (wc *WsConn) Loop(n int) {
	for {
		if n > 0 && wc.counter >= uint64(n) {
			return
		}

		if wc.Closed {
			return
		}

		msgType, msg, err := wc.Conn.ReadMessage()
		wc.counter++
		if err != nil {
			if strings.Contains(err.Error(), "websocket: close") || strings.Contains(err.Error(), "connection reset by peer") {
				gutil.Debug("closing ws '%s' connection due to: %s", wc.ID, err.Error())
				wc.Closed = true
				wc.Close()
				return
			}
			gutil.LogError(err, "could not read message from %s", wc.ID)
			continue
		}

		if msgType == websocket.CloseMessage {
			gutil.Debug("received close message from %s", wc.ID)
			wc.Close()
			return
		}

		// handle and send response
		go func() {
			respMsg := Message{}
			message, err := NewMessageFromJSON(msg)
			if err != nil {
				err = gutil.Error(err, "could not unmarshal message '%s' from %s", string(message.Type), wc.ID)
				respMsg.Type = ErrMsg
				respMsg.Payload = err.Error()
			} else if message.OrigReqID != "" {
				// then it's a reply
				if replyFunc, ok := wc.ReplyHandlers[message.OrigReqID]; ok {
					go replyFunc(message)
					wc.Context.Mux.Lock()
					delete(wc.ReplyHandlers, message.OrigReqID)
					wc.Context.Mux.Unlock()
				}
				return
			} else if superHandler, ok := wc.SuperHandlers[message.Type]; ok {
				respMsg = superHandler(wc, message)
				if respMsg.Type == NoReply {
					return
				}
			} else if handler, ok := wc.Handlers[message.Type]; ok {
				respMsg = handler(message)
				if respMsg.Type == NoReply {
					return
				}
			} else {
				err = gutil.Error(fmt.Errorf("message type '%s' not handled", message.Type))
				respMsg = NewMessageErr(err)
				gutil.LogError(err)
			}

			respMsg.OrigReqID = message.ReqID
			err = wc.SendMessage(respMsg, nil)
			if err != nil {
				gutil.LogError(err, "could not send response message '%s' to %s", string(respMsg.Type), wc.ID)
			}
		}()
	}
}

func handleClientHeartbeat(wc *WsConn, msg Message) (rMsg Message) {
	rMsg = NoReplyMsg
	wc.LastHeartbeat = time.Now()
	return
}
