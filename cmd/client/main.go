package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"blockim/internal/logger"
	"blockim/internal/pow"
	"blockim/internal/types"
)

type Client struct {
	conn   net.Conn
	logger *slog.Logger
}

func NewClient(addr string, logger *slog.Logger) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	return &Client{
		conn:   conn,
		logger: logger,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) GetQuote() (string, error) {
	if err := c.sendCommand("challenge"); err != nil {
		return "", fmt.Errorf("failed to request challenge: %w", err)
	}
	reader := bufio.NewReader(c.conn)
	var challengeResp types.ChallengeResponse
	if err := json.NewDecoder(reader).Decode(&challengeResp); err != nil {
		return "", fmt.Errorf("failed to decode challenge response: %w", err)
	}
	c.logger.Debug("Received challenge", "challenge", challengeResp.Challenge)
	startTime := time.Now()
	nonce, err := c.solve(challengeResp.Challenge)
	if err != nil {
		return "", fmt.Errorf("failed to solve challenge: %w", err)
	}
	timeForSolution := time.Since(startTime)
	c.logger.Info("Found solution", "nonce", nonce, "timeTaken", timeForSolution)
	if err := c.sendCommand("solution"); err != nil {
		return "", fmt.Errorf("failed to send solution command: %w", err)
	}
	solution := types.SolutionRequest{
		Challenge: challengeResp.Challenge,
		Nonce:     nonce,
	}
	if err := json.NewEncoder(c.conn).Encode(solution); err != nil {
		return "", fmt.Errorf("failed to send solution: %w", err)
	}
	reader = bufio.NewReader(c.conn)
	var quoteResp types.QuoteResponse
	if err := json.NewDecoder(reader).Decode(&quoteResp); err != nil {
		return "", fmt.Errorf("failed to decode quote response: %w", err)
	}

	return quoteResp.Quote, nil
}

func (c *Client) CheckHealth() error {
	if err := c.sendCommand("health"); err != nil {
		return fmt.Errorf("failed to send health command: %w", err)
	}
	reader := bufio.NewReader(c.conn)
	var resp map[string]interface{}
	if err := json.NewDecoder(reader).Decode(&resp); err != nil {
		return fmt.Errorf("failed to decode health response: %w", err)
	}
	if status, ok := resp["status"].(string); !ok || status != "ok" {
		return fmt.Errorf("server is not ready: %v", resp["error"])
	}

	return nil
}

func (c *Client) sendCommand(cmd string) error {
	_, err := fmt.Fprintf(c.conn, "%s\n", cmd)

	return err
}

func (c *Client) Quit() error {
	if err := c.sendCommand("quit"); err != nil {
		return fmt.Errorf("failed to send quit command: %w", err)
	}
	return nil
}

func main() {
	var serverAddr string
	flag.StringVar(&serverAddr, "server", "localhost:8080", "server location and port")
	flag.Parse()
	logger.Setup(logger.Config{
		Level:  "info",
		Pretty: true,
	})
	log := logger.Get()
	client, err := NewClient(serverAddr, log)
	if err != nil {
		log.Error("Failed to create client", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := client.Quit(); err != nil {
			log.Error("Failed to quit gracefully", "error", err)
		}
		client.Close()
	}()
	if err := client.CheckHealth(); err != nil {
		log.Error("Health check failed", "error", err)
		os.Exit(1)
	}
	log.Info("Server is ready")
	quote, err := client.GetQuote()
	if err != nil {
		log.Error("Failed to get quote", "error", err)
		os.Exit(1)
	}
	log.Info("Received quote", "quote", quote)
}

func (c *Client) solve(serializedChallenge string) (string, error) {
	challenge := pow.Challenge{}
	if err := challenge.Deserialize(serializedChallenge); err != nil {
		return "", err
	}
	nonce, err := challenge.Solve()
	if err != nil {
		return "", err
	}

	return nonce, nil
}
