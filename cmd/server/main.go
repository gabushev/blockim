package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"blockim/internal/config"
	"blockim/internal/logger"
	"blockim/internal/pow"
	"blockim/internal/quotes"
	"blockim/internal/types"
)

type Challenger interface {
	GenerateChallenge() (*pow.Challenge, error)
	VerifyChallenge(serializedChallenge string) (*pow.Challenge, bool)
	VerifySolution(serializedChallenge, nonce string) bool
}

type Quoter interface {
	GetRandomQuote() (string, error)
	Initialized() bool
}

type ChallengeService struct {
	cm     Challenger
	q      Quoter
	logger *slog.Logger
}

func NewChallengeService(cm Challenger, q Quoter, logger *slog.Logger) *ChallengeService {
	return &ChallengeService{
		cm:     cm,
		q:      q,
		logger: logger,
	}
}

func (sc *ChallengeService) handleConnection(conn net.Conn) {
	defer conn.Close()
	sc.logger.Info("New client connected", "addr", conn.RemoteAddr())
	reader := bufio.NewReader(conn)
	for {
		cmd, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				sc.logger.Info("Client disconnected", "addr", conn.RemoteAddr())
				return
			}
			sc.logger.Error("Failed to read command", "error", err)
			return
		}
		cmd = strings.TrimSpace(cmd)
		if len(cmd) == 0 {
			continue
		}
		switch cmd {
		case "challenge":
			challenge, err := sc.cm.GenerateChallenge()
			if err != nil {
				sc.logger.Error("Failed to generate challenge", "error", err)
				json.NewEncoder(conn).Encode(types.ErrorResponse{Error: "Failed to generate challenge"})
				continue
			}
			sc.logger.Info("Generated new challenge",
				"data", challenge.Data,
				"difficulty", challenge.Difficulty)

			json.NewEncoder(conn).Encode(types.ChallengeResponse{
				Challenge: challenge.Serialize(),
			})
		case "solution":
			var req types.SolutionRequest
			if err := json.NewDecoder(reader).Decode(&req); err != nil {
				sc.logger.Error("Failed to parse solution request", "error", err)
				json.NewEncoder(conn).Encode(types.ErrorResponse{Error: "Invalid request format"})
				continue
			}
			if !sc.cm.VerifySolution(req.Challenge, req.Nonce) {
				sc.logger.Error("Invalid solution",
					"challenge", req.Challenge,
					"nonce", req.Nonce)
				json.NewEncoder(conn).Encode(types.ErrorResponse{Error: "Invalid solution"})
				continue
			}
			quote, err := sc.q.GetRandomQuote()
			if err != nil {
				sc.logger.Error("Failed to get quote", "error", err)
				json.NewEncoder(conn).Encode(types.ErrorResponse{Error: "Failed to get quote"})
				continue
			}
			sc.logger.Info("Solution verified, quote provided",
				"challenge", req.Challenge,
				"nonce", req.Nonce,
				"quote", quote)
			json.NewEncoder(conn).Encode(types.QuoteResponse{Quote: quote})
		case "health":
			isReady := sc.q.Initialized()
			if !isReady {
				sc.logger.Warn("Health check failed", "reason", "quotes dictionary not ready")
				json.NewEncoder(conn).Encode(map[string]interface{}{
					"status": "error",
					"error":  "quotes dictionary is not ready",
					"time":   time.Now().Format(time.RFC3339),
				})
				continue
			}
			sc.logger.Info("Health check passed")
			json.NewEncoder(conn).Encode(map[string]interface{}{
				"status": "ok",
				"quotes": "ready",
				"time":   time.Now().Format(time.RFC3339),
			})
		case "quit":
			sc.logger.Info("Client disconnected", "addr", conn.RemoteAddr())
			return
		default:
			sc.logger.Warn("Unknown command received", "command", cmd)
			json.NewEncoder(conn).Encode(types.ErrorResponse{Error: "Unknown command"})
		}
	}
}

func main() {
	configPath := flag.String("config", "./config.yaml", "Path to config file")
	flag.Parse()
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal("Failed to load config", "error", err)
	}
	if err := logger.Setup(cfg.Logger); err != nil {
		logger.Fatal("Failed to setup logger", "error", err)
	}
	log := logger.Get()
	quoter := quotes.NewService(cfg.API.URL, cfg.API.Key, log)
	challengeMaker := pow.NewChallengeMaker(cfg.PoW.ServerSecret, cfg.PoW.Difficulty)
	server := NewChallengeService(challengeMaker, quoter, log)
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", cfg.Server.Port))
	if err != nil {
		log.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
	defer listener.Close()
	log.Info("TCP server started", "port", cfg.Server.Port)
	// sigterm chan
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					log.Error("Failed to accept connection", "error", err)
					continue
				}
			}
			go server.handleConnection(conn)
		}
	}()
	<-quit
	log.Info("Shutting down server...")
	cancel()
	time.Sleep(time.Second)
	log.Info("Server exiting")
}
