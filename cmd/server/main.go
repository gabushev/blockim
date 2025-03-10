// @title Word of Wisdom API
// @version 1.0
// @description API server for providing quotes - backed with PoW Armor against DDOS
// @host localhost:8080
// @BasePath /api
// @schemes http https
//
// @tag.name challenge
// @tag.description Operations with Proof of Work challenge
//
// @tag.name solution
// @tag.description Operations with solutions and getting quotes
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "blockim/docs"
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

func main() {
	configPath := flag.String("config", "./config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatal("config loading error", "error", err)
	}
	if err := logger.Setup(cfg.Logger); err != nil {
		logger.Fatal("Failed to setup logger", "error", err)
	}
	log := logger.Get()
	quoter := quotes.NewService(cfg.API.URL, cfg.API.Key, log)
	challengeMaker := pow.NewChallengeMaker(cfg.PoW.ServerSecret, cfg.PoW.Difficulty)
	challengeService := NewChallengeService(challengeMaker, quoter, log)
	router := gin.New()
	router.Use(logger.GinMiddleware())
	router.Use(gin.Recovery())

	router.GET("/health", challengeService.healthcheck)
	api := router.Group("/api")
	{
		api.POST("/challenge", challengeService.getChallenge)
		api.POST("/solution", challengeService.submitSolution)
	}
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", cfg.Server.Port),
		Handler: router,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP server error", "error", err)
		}
	}()
	log.Info("HTTP server started", "port", cfg.Server.Port)
	log.Info("Swagger UI available", "url", fmt.Sprintf("http://localhost:%d/swagger/index.html", cfg.Server.Port))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("Server exiting")
}

func (cs *ChallengeService) healthcheck(c *gin.Context) {
	log := logger.WithContext(c.Request.Context())
	isReady := cs.q.Initialized()
	if !isReady {
		log.Warn("Health check failed", "reason", "quotes dictionary not ready")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "error",
			"error":  "quotes dictionary is not ready",
			"time":   time.Now().Format(time.RFC3339),
		})
		return
	}

	log.Info("Health check passed")
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"quotes": "ready",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// @Summary Get new challenge
// @Description Generates new Proof of Work challenge
// @ID get-challenge
// @Tags challenge
// @Accept json
// @Produce json
// @Success 200 {object} types.ChallengeResponse
// @Failure 500 {object} types.ErrorResponse
// @Router /challenge [post]
func (cs *ChallengeService) getChallenge(c *gin.Context) {
	log := logger.WithContext(c.Request.Context())
	challenge, err := cs.cm.GenerateChallenge()
	if err != nil {
		log.Error("Failed to generate challenge", "error", err)
		c.JSON(http.StatusInternalServerError,
			types.ErrorResponse{Error: "Failed to generate challenge"},
		)
		return
	}

	log.Info("Generated new challenge",
		"data", challenge.Data,
		"difficulty", challenge.Difficulty)

	c.JSON(http.StatusOK, types.ChallengeResponse{
		Challenge: challenge.Serialize(),
	})
}

// @Summary Send challenge solution
// @Description Checks solution by Proof of Work challenge and may return a quote if its correct
// @ID submit-solution
// @Tags solution
// @Accept json
// @Produce json
// @Param request body types.SolutionRequest true "Solution for challenge"
// @Success 200 {object} types.QuoteResponse
// @Failure 400 {object} types.ErrorResponse
// @Router /solution [post]
func (cs *ChallengeService) submitSolution(c *gin.Context) {
	log := logger.WithContext(c.Request.Context())
	var req types.SolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("Failed to parse solution request", "error", err)
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "Invalid request body"},
		)
		return
	}

	if !cs.cm.VerifySolution(req.Challenge, req.Nonce) {
		log.Error("Invalid solution",
			"challenge", req.Challenge,
			"nonce", req.Nonce)
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "Invalid solution"},
		)
		return
	}

	quote, err := cs.q.GetRandomQuote()
	if err != nil {
		log.Error("Failed to get quote", "error", err)
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: "Failed to get quote"})
		return
	}

	log.Info("Solution verified, quote provided",
		"challenge", req.Challenge,
		"nonce", req.Nonce,
		"quote", quote)

	c.JSON(http.StatusOK, types.QuoteResponse{Quote: quote})
}
