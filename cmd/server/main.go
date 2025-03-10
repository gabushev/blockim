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
	"log"
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
	cm Challenger
	q  Quoter
}

func NewChallengeService(cm Challenger, q Quoter) *ChallengeService {
	return &ChallengeService{
		cm: cm,
		q:  q,
	}
}

func main() {
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("config loaing error: %v", err)
	}
	quoter := quotes.NewsService(cfg.API.URL, cfg.API.Key)
	challengeMaker := pow.NewChallengeMaker(cfg.PoW.ServerSecret, cfg.PoW.Difficulty)
	challengeService := NewChallengeService(challengeMaker, quoter)

	router := gin.Default()
	router.GET("/health", challengeService.healthcheck)
	api := router.Group("/api")
	{
		api.POST("/challenge", challengeService.getChallenge)
		api.POST("/solution", challengeService.submitSolution)
	}
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	log.Printf("HTTP server started on :%d", cfg.Server.Port)
	log.Printf("Swagger UI available by http://localhost:%d/swagger/index.html", cfg.Server.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}

func (cs *ChallengeService) healthcheck(c *gin.Context) {
	isReady := cs.q.Initialized()
	if !isReady {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "error",
			"error":  "quotes dictionary is not ready",
			"time":   time.Now().Format(time.RFC3339),
		})
		return
	}

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
	challenge, err := cs.cm.GenerateChallenge()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: "failed to generate challenge",
		})
		return
	}

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
	var req types.SolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "invalid request format",
		})
		return
	}

	if !cs.cm.VerifySolution(req.Challenge, req.Nonce) {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{
			Error: "invalid solution",
		})
		return
	}
	quote, err := cs.q.GetRandomQuote()
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{
			Error: "failed to get random quote",
		})
		return
	}
	c.JSON(http.StatusOK, types.QuoteResponse{
		Quote: quote,
	})
}
