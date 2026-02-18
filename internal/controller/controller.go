// Package controller contains class for controller
package controller

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/A1exander-liU/comp-8005-assign-2/internal/shared"
	"go.uber.org/zap"
)

type ChunkStatus string

const (
	ChunkUnassigned = "unassigned"
	ChunkAssigned   = "assigned"
	ChunkCompleted  = "completed"
)

// Config holds parameters for controller setup.
type Config struct {
	// Path to the shadowfile
	Shadowfile string

	// User's name to crack
	Username string

	// Port number for the controller to listen on
	Port int

	// Period for sending a heartbeat
	HeartbeatSeconds int

	// Cracking job size for workers
	ChunkSize int
}

type workerConnection struct {
	Registered bool
	ChunkID    int
	Done       chan bool

	Conn   net.Conn
	Router *shared.Router
}

type chunk struct {
	passwords []string
	status    ChunkStatus
}

// Controller is reponsible for managing worker connections
// for sending and receiving password cracking jobs.
type Controller struct {
	// Public logger for sending log messages
	Logger *zap.Logger

	listener net.Listener
	workers  map[string]*workerConnection
	fs       *flag.FlagSet
	chunks   map[int]*chunk

	ShadowData       shared.ShadowData
	HeartbeatSeconds int
	Config           Config

	LatencyParse        time.Duration
	LatencyDispatch     time.Duration
	LatencyDispatchTime time.Time
	LatencyCrack        time.Duration
	LatencyReturn       time.Duration
}

// NewController creates a new Controller object.
func NewController(logger *zap.Logger) *Controller {
	return &Controller{
		Logger: logger,

		workers: map[string]*workerConnection{},
	}
}

// ParseArguments parses command line arguments.
func (c *Controller) ParseArguments() Config {
	var config Config
	fs := flag.NewFlagSet("controller CLI", flag.ExitOnError)

	fs.StringVar(&config.Shadowfile, "f", "", "path to shadowfile")
	fs.StringVar(&config.Username, "u", "", "username whose password to be cracked")
	fs.IntVar(&config.Port, "p", 0, "port number to listen on")
	fs.IntVar(&config.HeartbeatSeconds, "b", 0, "period (in seconds) to send a heartbeat")
	fs.IntVar(&config.ChunkSize, "c", 0, "chunk size of each cracking task for a worker")
	c.fs = fs

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return config
}

// HandleArguments performs validation on the arguments, the program will exit
// and print out a usage if any of the arguments failed validation.
func (c *Controller) HandleArguments(config Config) {
	parseStart := time.Now()
	if config.Shadowfile == "" {
		fmt.Println("Error: -f is required")
		c.fs.Usage()
		os.Exit(1)
	}
	if config.Username == "" {
		fmt.Println("Error: -u is required")
		c.fs.Usage()
		os.Exit(1)
	}
	if config.Port < 1 || config.Port > 65535 {
		fmt.Println("Error: -p is required and must be in range: 1 - 65535 (inclusive)")
		c.fs.Usage()
		os.Exit(1)
	}

	if config.HeartbeatSeconds < 1 {
		fmt.Println("Error: -b must be a non-zero positive integer")
		c.fs.Usage()
		os.Exit(1)
	}

	if config.ChunkSize < 1 {
		fmt.Println("Error: -c must be a non-zero positive integer")
		c.fs.Usage()
		os.Exit(1)
	}

	c.Config = config
	c.parseShadowFile()
	c.LatencyParse = time.Since(parseStart)
	c.partitionSearchSpace()
}

// parseShadowFile reads to the shadowfile to extracted the password hash elements
// of the desired user.
//
// This will return with an error if:
//   - it failed to read the shadowfile
//   - it could not find the user
func (c *Controller) parseShadowFile() {
	foundUser := false

	contents, err := os.ReadFile(c.Config.Shadowfile)
	if err != nil {
		fmt.Println("Failed to read shadowfile:", err)
		os.Exit(1)
	}

	entries := strings.SplitSeq(string(contents), "\n")
	for entry := range entries {
		user, hash, found := strings.Cut(entry, ":")
		if !found {
			continue
		}

		if user != c.Config.Username {
			continue
		}

		foundUser = true
		shadow, err := shared.ParseHash(hash)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		shadow.Username = user
		c.ShadowData = shadow
	}

	if !foundUser {
		fmt.Println("Failed to find user:", c.Config.Username)
		os.Exit(1)
	}
}

// partitionSearchSpace creates chunks configured through the chunk size CLI argument.
func (c *Controller) partitionSearchSpace() {
	passwords := shared.GenerateCandidatePasswords(shared.SearchSpace, 3)
	partitions := shared.PartitionArraySize(passwords, c.Config.ChunkSize)

	chunks := make(map[int]*chunk, 0)
	for i, c := range partitions {
		chunks[i] = &chunk{passwords: c, status: ChunkUnassigned}
	}

	c.chunks = chunks
}

// getUnassignedChunk gives a chunk of passwords that hasn't been assigned to another worker.
//
// It will return the chunk id and a boolean if the chunk was assigned.
// In the case that there are no more unassigned chunks, this will return
// a chunk id of -1 and false
func (c *Controller) getUnassignedChunk(workerID string) (int, bool) {
	for chunkID, chunk := range c.chunks {
		if chunk.status != ChunkUnassigned {
			continue
		}

		c.workers[workerID].ChunkID = chunkID
		c.chunks[chunkID].status = ChunkAssigned

		return chunkID, true
	}

	return -1, false
}

// SetupServer starts listening for connections on the specified port.
//
// This will exit with an error if listening failed.
func (c *Controller) SetupServer() {
	address := fmt.Sprintf("[::]:%d", c.Config.Port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Printf("Failed to listen on %s: %s\n", address, err)
		os.Exit(1)
	}

	c.listener = listener

	c.Logger.Info(fmt.Sprintf("Controller listening on %s", address))
}

// AcceptConnection accepts an incoming connection.
//
// This wraps around the net.Listener.Accept
func (c *Controller) AcceptConnection() (net.Conn, error) {
	return c.listener.Accept()
}

func (c *Controller) handleRegistration(m shared.Message, conn net.Conn) (shared.Message, error) {
	id := conn.RemoteAddr().String()
	ok := c.workers[id].Registered
	if ok {
		err := fmt.Errorf("worker %s is already registered", id)
		return shared.Message{ID: id, Type: shared.MessageError, Timestamp: time.Now(), Message: err.Error()}, err
	}

	c.workers[id].Registered = true
	c.workers[id].Done = make(chan bool)

	c.LatencyDispatchTime = time.Now()

	return shared.Message{ID: id, Type: shared.MessageRegister, Timestamp: time.Now(), Message: "Registration successful"}, nil
}

func (c *Controller) sendJob(_ shared.Message, conn net.Conn) (shared.Message, error) {
	timestamp := time.Now()

	chunkID, found := c.getUnassignedChunk(conn.RemoteAddr().String())
	var workerChunk []string
	if found {
		workerChunk = c.chunks[chunkID].passwords
	} else {
		workerChunk = make([]string, 0)
	}

	res := shared.Message{
		Version: shared.MessageVersion, Type: shared.MessageJobDetails, Message: "Cracking details",
		Timestamp: timestamp,
		Payload: shared.PayloadJobDetails{
			Algorithm:      c.ShadowData.Algorithm,
			Parameters:     c.ShadowData.Parameters,
			Salt:           c.ShadowData.Salt,
			Hash:           c.ShadowData.Hash,
			SearchSpace:    shared.SearchSpace,
			ChunkID:        chunkID,
			Chunk:          workerChunk,
			PasswordLength: 3,
		},
	}

	c.LatencyDispatch = timestamp.Sub(c.LatencyDispatchTime)

	go c.sendHeartbeat(conn, time.Duration(c.HeartbeatSeconds)*time.Second)

	return res, nil
}

func (c *Controller) handleJobResults(m shared.Message, conn net.Conn) (shared.Message, error) {
	timestamp := time.Now()

	payload, ok := m.Payload.(shared.PayloadJobResults)
	if !ok {
		return shared.Message{Version: shared.MessageVersion, Type: shared.MessageError, Message: "Bad payload"}, nil
	}

	c.chunks[payload.ChunkID].status = ChunkCompleted
	c.workers[conn.RemoteAddr().String()].ChunkID = -1

	res := shared.Message{
		Version:   shared.MessageVersion,
		Type:      shared.MessageJobResults,
		Message:   "Received message details",
		Timestamp: timestamp,
	}

	c.LatencyCrack = payload.Time
	c.LatencyReturn = m.Timestamp.Sub(timestamp)
	c.displayJobResults(payload.Password, payload.Err, payload.Time)

	return res, nil
}

func (c *Controller) handleClose(_ shared.Message, conn net.Conn) (shared.Message, error) {
	id := conn.RemoteAddr().String()
	_, ok := c.workers[id]
	if ok {
		delete(c.workers, id)
	}

	message := fmt.Sprintf("Closing connection for %s", id)
	return shared.Message{ID: id, Type: shared.MessageClose, Timestamp: time.Now(), Message: message}, nil
}

func (c *Controller) displayJobResults(result string, err error, _ time.Duration) {
	if err != nil {
		fmt.Println("PASSWORD: FAILED TO CRACK PASSWORD", err)
	} else {
		fmt.Printf("PASSWORD: %s\n", result)
	}

	c.reportFinalResults()
}

func (c *Controller) handleHeartbeat(m shared.Message, conn net.Conn) (shared.Message, error) {
	payload, _ := m.Payload.(shared.PayloadHearbeat)

	c.Logger.Info("Heartbeat info",
		zap.Int("total", payload.TotalTested),
		zap.Int("delta", payload.DeltaTested),
		zap.Int("activeThreads", payload.ActiveThreads),
	)

	return shared.Message{}, nil
}

func (c *Controller) sendHeartbeat(conn net.Conn, period time.Duration) {
	id := conn.RemoteAddr().String()
	m := shared.Message{
		Version:   shared.MessageVersion,
		ID:        id,
		Type:      shared.MessageHeartbeat,
		Timestamp: time.Now(),
		Message:   "Sending heartbeat",
	}
	worker := c.workers[id]
	ticker := time.NewTicker(period)

	for {
		select {
		case <-worker.Done:
			return
		case <-ticker.C:
			err := worker.Router.Send(m)
			if err != nil {
				ticker.Stop()
				return
			}
		}
	}
}

func (c *Controller) reportFinalResults() {
	total := c.LatencyParse + c.LatencyDispatch + c.LatencyCrack + c.LatencyReturn

	fmt.Println("FINAL RESULTS (seconds)")
	fmt.Println("Parse:", c.LatencyParse.Seconds())
	fmt.Println("Dispatch:", c.LatencyDispatch.Seconds())
	fmt.Println("Crack:", c.LatencyCrack.Seconds())
	fmt.Println("Return:", c.LatencyReturn.Seconds())
	fmt.Println("Total:", total.Seconds())
}

// HandleConnection manages communication with a single worker for the
// whole entire lifecycle.
func (c *Controller) HandleConnection(conn net.Conn) {
	r := shared.NewRouter(c.Logger, conn)
	r.Handle(shared.MessageRegister, c.handleRegistration)
	r.Handle(shared.MessageJobDetails, c.sendJob)
	r.Handle(shared.MessageJobResults, c.handleJobResults)
	r.Handle(shared.MessageClose, c.handleClose)
	r.Handle(shared.MessageHeartbeat, c.handleHeartbeat)

	c.workers[conn.RemoteAddr().String()] = &workerConnection{
		Registered: false,
		Conn:       conn, Router: r,
	}

	if err := r.Start(); err != nil {
		c.Logger.Error(err.Error())
	}
}
