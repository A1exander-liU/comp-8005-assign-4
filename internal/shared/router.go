package shared

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"

	"go.uber.org/zap"
)

type Handler func(Message, net.Conn) (Message, error)

type Router struct {
	ID        string
	l         *zap.Logger
	handlers  map[MessageType]Handler
	readHooks []chan Message

	conn net.Conn

	// Send
	encoder *gob.Encoder
	// Receive
	decoder *gob.Decoder

	// Pending messages to write
	sendChannel chan Message
}

// NewRouter creates a new router for the associated connection.
func NewRouter(l *zap.Logger, conn net.Conn) *Router {
	return &Router{
		ID:       conn.RemoteAddr().String(),
		l:        l,
		handlers: map[MessageType]Handler{},
		conn:     conn,

		encoder: gob.NewEncoder(conn),
		decoder: gob.NewDecoder(conn),

		sendChannel: make(chan Message, 64),
	}
}

func (r *Router) dispatch(m Message) (Message, error) {
	h, ok := r.handlers[m.Type]
	if !ok {
		// start := time.Now()
		// err := fmt.Errorf("unknown message type: %s", m.Type)
		// return Message{ID: m.ID, Type: MessageError, Timestamp: start, Message: err.Error()}, err
		return Message{}, nil
	}

	return h(m, r.conn)
}

// Send adds a new message that should be sent.
//
// Messages are sent through a channel, where they are later asynchronously sent.
func (r *Router) Send(m Message) {
	r.sendChannel <- m
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

// writeLoop handles sending news sent through the channel.
func (r *Router) writeLoop(workerCtx context.Context) {
	for {
		select {
		case <-workerCtx.Done():
			return
		case message := <-r.sendChannel:
			if err := r.encoder.Encode(message); err != nil {
				r.l.Error("failed to send", zap.Error(err))
				return
			}
		}
	}
}

// HookRead attaches a function that can receive a copy of messages sent from the connection.
//
// Once a new message is received, the hook will be called, passing the messsage to it.
func (r *Router) HookRead(hook chan Message) {
	r.readHooks = append(r.readHooks, hook)
}

// Start listens for messages in a loop, responding using the registered handlers.
//
// call 'Router.Handle' for desired routes to handle before calling this method.
//
// An error is returned during failure to read or write from the connection.
func (r *Router) Start(ctx context.Context, cancel context.CancelFunc) error {
	// read loop routes message to handlers
	// handlers return a message that are sent a channel
	// write loop selects over the channel to send the messages to the connection
	//
	// reads and writes are cenralized
	// how to decouple heartbeats
	// - need to record last heartbeat time to calc time since last heartbeat
	// - need to hook to when messages are received

	defer cancel()
	go r.writeLoop(ctx)

	for {
		var message Message

		// read messages from workers
		// handle closed connection and other errors
		err := r.decoder.Decode(&message)
		if err == io.EOF {
			r.l.Info("EOF: Connection closed")
			return nil
		}
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			// m := fmt.Sprintf("From %s", r.ID)
			// r.l.Info(m, zap.String("type", string(message.Type)), zap.String("message", message.Message), zap.Time("timestamp", message.Timestamp))
		}

		// read hooks
		for _, hook := range r.readHooks {
			hook <- message
		}

		// Dispatch returns a message to be sent
		// skip empty messages
		res, _ := r.dispatch(message)
		if res == (Message{}) {
			continue
		}

		// Add the message to the channel
		r.Send(res)

		m := fmt.Sprintf("To %s", r.ID)
		r.l.Info(m, zap.String("type", string(res.Type)), zap.String("message", res.Message), zap.Time("timestamp", res.Timestamp))
	}
}
