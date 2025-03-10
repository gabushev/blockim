package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"blockim/internal/pow"
	"blockim/internal/types"
)

func main() {
	serverAddr := flag.String("server", "http://localhost:8080", "Server address")
	flag.Parse()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	challengeResp, err := http.Post(*serverAddr+"/api/challenge", "application/json", nil)
	if err != nil {
		log.Fatalf("error on challenge request: %v", err)
	}
	defer challengeResp.Body.Close()

	if challengeResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(challengeResp.Body)
		log.Fatalf("server error: %s", body)
	}

	var challenge types.ChallengeResponse
	if err := json.NewDecoder(challengeResp.Body).Decode(&challenge); err != nil {
		log.Fatalf("challenge decode error: %v", err)
	}

	startedAt := time.Now()
	nonce := solveChallenge(challenge.Challenge)
	timeTaken := time.Since(startedAt).Seconds()
	log.Printf("solution found in %.2f sec, solution nonce is %s", timeTaken, nonce)

	solution := types.SolutionRequest{
		Challenge: challenge.Challenge,
		Nonce:     nonce,
	}

	solutionJSON, err := json.Marshal(solution)
	if err != nil {
		log.Fatalf("serialization error: %v", err)
	}

	solutionResp, err := client.Post(
		*serverAddr+"/api/solution",
		"application/json",
		bytes.NewBuffer(solutionJSON),
	)
	if err != nil {
		log.Fatalf("error on sending solution: %v", err)
	}
	defer solutionResp.Body.Close()

	if solutionResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(solutionResp.Body)
		log.Fatalf("server decided the solution is wrong: %s", body)
	}

	var quote types.QuoteResponse
	if err := json.NewDecoder(solutionResp.Body).Decode(&quote); err != nil {
		log.Fatalf("serialization error: %v", err)
	}

	fmt.Printf("\nGot quote from server: %s\n", quote.Quote)
}

func solveChallenge(serializedChallenge string) string {
	c := pow.Challenge{}
	err := c.Deserialize(serializedChallenge)
	if err != nil {
		log.Fatalf("error on challenge loading: %v", err)
	}

	nonce, err := c.Solve()
	if err != nil {
		log.Fatalf("error on solving challenge: %v", err)
	}

	return nonce
}
