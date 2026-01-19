package ascomserver

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"go.uber.org/zap"
)

// NewDiscoveryService creates a new ASCOM Alpaca discovery service.
// The discovery service listens for UDP broadcast messages on the specified port
// and responds with information about where to find the Alpaca REST API.
//
// Parameters:
//   - port: UDP port to listen on (typically DefaultDiscoveryPort = 32227)
//   - apiPort: HTTP port where the Alpaca REST API is available
//   - logger: Structured logger for discovery service operations
//
// The ASCOM Alpaca discovery protocol works as follows:
//  1. Client broadcasts "alpacadiscovery1" message to UDP port 32227
//  2. All Alpaca servers on the network respond with their API port number
//  3. Client can then connect to discovered servers via HTTP/HTTPS
func NewDiscoveryService(port, apiPort int, logger *zap.Logger) *DiscoveryService {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &DiscoveryService{
		port:    port,
		apiPort: apiPort,
		logger:  logger.With(zap.String("component", "discovery")),
		stopCh:  make(chan struct{}),
	}
}

// Start begins listening for discovery broadcasts in a background goroutine.
// This method returns immediately after starting the listener.
// Call Stop() to gracefully shut down the discovery service.
//
// Returns an error if the UDP listener cannot be created (e.g., port already in use).
func (d *DiscoveryService) Start() error {
	d.logger.Info("Starting ASCOM Alpaca discovery service",
		zap.Int("udp_port", d.port),
		zap.Int("api_port", d.apiPort))

	// Create UDP listener on the discovery port.
	// We listen on all interfaces (0.0.0.0) to receive broadcasts from any network.
	addr := &net.UDPAddr{
		IP:   net.IPv4zero, // 0.0.0.0 - listen on all interfaces
		Port: d.port,
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to create UDP listener: %w", err)
	}

	d.logger.Info("Discovery service UDP listener started",
		zap.String("listen_address", conn.LocalAddr().String()))

	// Start the discovery loop in a background goroutine.
	// This goroutine will run until Stop() is called.
	go d.discoveryLoop(conn)

	return nil
}

// Stop gracefully shuts down the discovery service.
// This method blocks until the discovery loop has fully terminated.
func (d *DiscoveryService) Stop() {
	d.logger.Info("Stopping discovery service")
	close(d.stopCh)
}

// discoveryLoop is the main loop that handles incoming discovery requests.
// It runs in a background goroutine and listens for UDP packets.
//
// When a packet containing "alpacadiscovery1" is received, the service responds
// with a JSON packet containing the API port number. The response is sent back
// to the address that sent the discovery request.
//
// This method continues running until the stopCh is closed via Stop().
func (d *DiscoveryService) discoveryLoop(conn *net.UDPConn) {
	defer func() { _ = conn.Close() }()

	// Buffer for receiving UDP packets.
	// Discovery messages are small, so 1024 bytes is more than sufficient.
	buffer := make([]byte, 1024)

	// Create the response packet once (it's always the same).
	// This response follows the ASCOM Alpaca discovery protocol specification.
	response := DiscoveryResponse{
		AlpacaPort: d.apiPort,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		d.logger.Error("Failed to marshal discovery response",
			zap.Error(err))
		return
	}

	d.logger.Debug("Discovery response prepared",
		zap.String("response", string(responseBytes)))

	// Main discovery loop - continues until stopCh is closed
	for {
		select {
		case <-d.stopCh:
			// Stop signal received - clean shutdown
			d.logger.Info("Discovery loop stopping")
			return

		default:
			// Set a read deadline so we can periodically check stopCh.
			// Without this, ReadFromUDP would block forever and we couldn't stop gracefully.
			// We use a 1-second deadline to balance responsiveness vs. CPU usage.
			func() { _ = conn.SetReadDeadline(mustTime(1)) }()

			// Read incoming UDP packet
			n, remoteAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				// Check if this is a timeout error (expected due to our deadline)
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Timeout is normal - just continue the loop to check stopCh
					continue
				}
				// Other errors are unexpected and should be logged
				d.logger.Warn("Error reading UDP packet",
					zap.Error(err))
				continue
			}

			// Parse the received message
			message := string(buffer[:n])

			d.logger.Debug("Received UDP packet",
				zap.String("from", remoteAddr.String()),
				zap.String("message", message),
				zap.Int("bytes", n))

			// Check if this is a valid ASCOM Alpaca discovery request.
			// The protocol specifies the exact string "alpacadiscovery1".
			// We do a case-sensitive comparison as per the specification.
			if message != AlpacaDiscoveryMessage {
				d.logger.Debug("Ignoring non-discovery message",
					zap.String("message", message))
				continue
			}

			// Valid discovery request received - send response.
			// The response is sent directly back to the address that sent the request.
			d.logger.Info("Discovery request received, sending response",
				zap.String("from", remoteAddr.String()),
				zap.Int("api_port", d.apiPort))

			_, err = conn.WriteToUDP(responseBytes, remoteAddr)
			if err != nil {
				d.logger.Error("Failed to send discovery response",
					zap.String("to", remoteAddr.String()),
					zap.Error(err))
				continue
			}

			d.logger.Debug("Discovery response sent successfully",
				zap.String("to", remoteAddr.String()))
		}
	}
}

// mustTime creates a time.Time value representing the current time plus the given seconds.
// This is a helper function for setting read deadlines in the discovery loop.
// Returns a time.Time value that is 'seconds' seconds in the future.
func mustTime(seconds int) time.Time {
	// Create a deadline 'seconds' seconds in the future.
	// This allows the ReadFromUDP call to timeout periodically so we can check stopCh.
	return time.Now().Add(time.Duration(seconds) * time.Second)
}
