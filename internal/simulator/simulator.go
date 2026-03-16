package simulator

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ServiceChain defines the order of services a request flows through.
var ServiceChain = []string{
	"api-gateway",
	"auth-service",
	"order-service",
	"payment-service",
	"inventory-service",
}

var levels = []string{"INFO", "INFO", "INFO", "INFO", "WARN", "ERROR"}

var messageTemplates = map[string]map[string][]string{
	"api-gateway": {
		"INFO":  {"Received incoming request", "Routing request to downstream", "Request forwarded successfully"},
		"WARN":  {"High request rate detected", "Slow upstream response"},
		"ERROR": {"Gateway timeout", "Connection refused from downstream"},
	},
	"auth-service": {
		"INFO":  {"Token validated successfully", "User authenticated", "Session created"},
		"WARN":  {"Token expiring soon", "Rate limit approaching for user"},
		"ERROR": {"Authentication failed: invalid token", "Auth service unavailable", "Token expired: invalid JWT structure", "Token expired: signature validation failed"},
	},
	"order-service": {
		"INFO":  {"Order created", "Order validated", "Order persisted to database"},
		"WARN":  {"Order total exceeds threshold", "Duplicate order detected"},
		"ERROR": {"Order creation failed: database timeout", "Order validation error"},
	},
	"payment-service": {
		"INFO":  {"Payment initiated", "Payment processed successfully", "Payment confirmation sent"},
		"WARN":  {"Payment gateway slow response", "Retry attempt for payment"},
		"ERROR": {"Payment declined", "Payment gateway connection refused", "context deadline exceeded", "Payment failed: insufficient funds", "Payment blocked: suspected fraud"},
	},
	"inventory-service": {
		"INFO":  {"Inventory checked", "Stock reserved", "Inventory updated"},
		"WARN":  {"Low stock warning", "Inventory sync delay"},
		"ERROR": {"Inventory update failed: timeout", "Stock reservation conflict"},
	},
}

// Run generates log files for the configured services over the given duration.
func Run(logDir string, services []string, duration time.Duration, ratePerSec int) error {
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	// Open one file per service.
	files := make(map[string]*os.File)
	for _, svc := range services {
		path := filepath.Join(logDir, svc+".log")
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("create log file for %s: %w", svc, err)
		}
		defer f.Close()
		files[svc] = f
	}

	fmt.Printf("🔄 Simulator: generating logs for %s at ~%d req/s\n", duration, ratePerSec)

	var mu sync.Mutex
	ticker := time.NewTicker(time.Second / time.Duration(ratePerSec))
	defer ticker.Stop()

	deadline := time.After(duration)

	for {
		select {
		case <-deadline:
			fmt.Println("✅ Simulator: log generation complete")
			return nil
		case <-ticker.C:
			requestID := generateRequestID()
			go simulateRequest(files, &mu, requestID, services)
		}
	}
}

func simulateRequest(files map[string]*os.File, mu *sync.Mutex, requestID string, services []string) {
	// Decide how far along the chain this request gets (to simulate missing hops).
	hops := len(ServiceChain)
	if rand.Float64() < 0.1 { // 10% chance of incomplete chain
		hops = rand.Intn(len(ServiceChain)-1) + 1
	}

	// 2% chance of a massive cascading failure
	isCascadingFailure := rand.Float64() < 0.02

	for i := 0; i < hops; i++ {
		svc := ServiceChain[i]
		// Check if we have a file for this service.
		f, ok := files[svc]
		if !ok {
			continue
		}

		level := pickLevel()
		msg := pickMessage(svc, level)
		latency := generateLatency(level)

		if isCascadingFailure {
			level = "ERROR"
			switch svc {
			case "api-gateway":
				msg = "Gateway timeout"
				latency = 5000 * time.Millisecond
			case "auth-service":
				msg = "Auth service unavailable"
			case "order-service":
				msg = "Order creation failed: database timeout"
			case "payment-service":
				msg = "Payment gateway connection refused"
			case "inventory-service":
				msg = "Inventory update failed: timeout"
			}
		}

		ts := time.Now().UTC().Format(time.RFC3339Nano)

		line := fmt.Sprintf("%s | %-17s | %-5s | %-19s | %-45s | latency=%dms\n",
			ts, svc, level, requestID, msg, latency.Milliseconds())

		mu.Lock()
		f.WriteString(line)
		mu.Unlock()

		// Small delay between service hops to make timestamps realistic.
		time.Sleep(time.Duration(rand.Intn(20)+5) * time.Millisecond)
	}
}

func generateRequestID() string {
	return fmt.Sprintf("req-%04x-%04x-%04x", rand.Intn(0xFFFF), rand.Intn(0xFFFF), rand.Intn(0xFFFF))
}

func pickLevel() string {
	return levels[rand.Intn(len(levels))]
}

func pickMessage(service, level string) string {
	msgs, ok := messageTemplates[service][level]
	if !ok || len(msgs) == 0 {
		return "unknown event"
	}
	return msgs[rand.Intn(len(msgs))]
}

func generateLatency(level string) time.Duration {
	base := rand.Intn(100) + 10 // 10-110ms normal
	switch level {
	case "WARN":
		base += rand.Intn(400) // up to ~510ms
	case "ERROR":
		base += rand.Intn(1500) // up to ~1610ms
	}
	return time.Duration(base) * time.Millisecond
}
