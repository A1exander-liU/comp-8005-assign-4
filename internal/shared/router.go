package shared

import (
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Handler func(Message, net.Conn) (Message, error)

type Router struct {
	ID       string
	l        *zap.Logger
	handlers map[MessageType]Handler

	conn net.Conn
	mu   sync.Mutex

	encoder *gob.Encoder
	decoder *gob.Decoder

	done chan bool
}

// NewRouter creates a new router for the associated connection.
func NewRouter(l *zap.Logger, conn net.Conn) *Router {
	return &Router{
		ID:       conn.RemoteAddr().String(),
		l:        l,
		handlers: map[MessageType]Handler{},
		conn:     conn,

		// Send
		encoder: gob.NewEncoder(conn),
		// Receive
		decoder: gob.NewDecoder(conn),

		done: make(chan bool),
	}
}

func (r *Router) dispatch(m Message) (Message, error) {
	start := time.Now()
	h, ok := r.handlers[m.Type]
	if !ok {
		err := fmt.Errorf("unknown message type: %s", m.Type)
		return Message{ID: m.ID, Type: MessageError, Timestamp: start, Message: err.Error()}, err
	}

	return h(m, r.conn)
}

// Send utilises a mutex to send messages in a thread-safe manner.
//
// Can be used to independently send messages from the router.
func (r *Router) Send(m Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.encoder.Encode(m)
}

// Handle registers a function to be called when a message with the corresponding type is sent.
//
// Registering another handler with the same type will overwrite the existing one, only one handler
// of a type can exist.
//
// Return an empty message 'Message{}' if the handler is not expected to send a response back.
func (r *Router) Handle(m MessageType, h Handler) {
	r.handlers[m] = h
}

// Start listens for messages in a loop, responding using the registered handlers.
//
// call 'Router.Handle' for desired routes to handle before calling this method.
//
// An error is returned during failure to read or write from the connection.
func (r *Router) Start() error {
	for {
		var message Message

		err := r.decoder.Decode(&message)
		if err == io.EOF {
			r.l.Info("Connection closed")
			return nil
		}
		if err != nil {
			return err
		}

		m := fmt.Sprintf("From %s", r.ID)
		r.l.Info(m, zap.String("type", string(message.Type)), zap.String("message", message.Message), zap.Time("timestamp", message.Timestamp))

		res, _ := r.dispatch(message)
		if res == (Message{}) {
			continue
		}

		err = r.Send(res)
		if res.Type == MessageClose {
			r.l.Info("Connection closed")
			r.done <- true
			close(r.done)
			return r.conn.Close()
		}
		if err != nil {
			return err
		}

		m = fmt.Sprintf("To %s", r.ID)
		r.l.Info(m, zap.String("type", string(res.Type)), zap.String("message", res.Message), zap.Time("timestamp", res.Timestamp))
	}
}
