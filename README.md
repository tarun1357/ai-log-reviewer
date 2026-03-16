# AI-Powered Distributed Log Analyzer 🔍🤖

A high-performance, concurrent log analysis pipeline built in Go. This tool simulates a distributed microservice environment, ingests massive volumes of raw logs, correlates them by `RequestID`, detects system anomalies using sliding-window heuristics, and leverages Large Language Models (LLMs) to perform automated Root Cause Analysis (RCA).

## 🌟 Key Features

*   **Microservice Simulator:** Generates highly realistic, concurrent logs across 5 mock services (`api-gateway`, `auth-service`, `order-service`, `payment-service`, `inventory-service`) complete with programmable latency, timeouts, and a statistical chance of catastrophic cascading failures.
*   **Concurrent Log Parsing:** Efficiently parses raw log files using Go concurrency (`goroutines` and `channels`), handling malformed inputs gracefully.
*   **Intelligent Correlation Engine:** Groups individual log entries into unified chronological request paths using Hash Maps, automatically filtering out duplicate logs.
*   **Multi-Tier Anomaly Detection:**
    *   **Rule-Based:** Detects timeouts, excessive latency, missing microservice hops, and error spikes.
    *   **Global Sliding Window:** Aggregates system-wide errors and triggers a SEV-1 `GLOBAL_SYSTEM_SPIKE` if >10 errors occur within any 1-minute window natively.
*   **AI Root Cause Analysis:** Integrates with the OpenAI API. It mathematically slices token windows to prevent overflows on massive stack traces and uses robust Regex fallback extraction to guarantee stable JSON parsing from the LLM.
*   **CLI Dashboard:** Beautifully styled, ANSI-colored terminal output featuring a high-level Incident Dashboard and specific recommended actions per anomalous request.

## 🏗️ Architecture Pipeline

1.  **Simulate:** `simulator.go` generates raw `.log` files in the `/logs` directory.
2.  **Parse:** `parser.go` transforms raw text strings into structured `LogEntry` Go structs.
3.  **Correlate:** `correlator.go` groups entries by `RequestID` and sorts them chronologically.
4.  **Detect:** `detector.go` scans correlated groups against latency thresholds and rules.
5.  **Analyze:** `llm.go` feeds anomalous groups into GPT-4o for natural language root cause summaries and actionable recommendations.
6.  **Alert:** `alerter.go` prints the colored ASCII dashboard to `stdout` and writes to `alerts.json`.

## 🚀 Getting Started

### Prerequisites
*   [Go 1.21+](https://go.dev/) installed.
*   An [OpenAI API Key](https://platform.openai.com/) (Required for the AI Analysis phase. Without it, the tool falls back to rule-based analysis).

### Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/ai-log-analyzer.git
   cd ai-log-analyzer
   ```

2. Configure the environment:
   Edit `configs/config.yaml` to set your desired simulation duration and provide your `openai_key`.
   > ⚠️ **SECURITY WARNING:** Never commit your real `openai_key` to GitHub! Make sure `configs/config.yaml` is added to your `.gitignore`.

### Execution
Run the full simulation and analysis pipeline step-by-step:
```bash
go run cmd/analyzer/main.go
```

## 🧪 Running Tests
The project includes unit tests checking token truncation limits, parsing logic, and correlation grouping.
```bash
go test ./... -v
```

## 📝 License
This project is open-source and available under the MIT License.
