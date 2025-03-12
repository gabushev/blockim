// Package types contains common data types for client and server
package types

// ChallengeRequest represents a request to get a new challenge
type ChallengeRequest struct {
}

// ChallengeResponse represents a response containing a challenge
type ChallengeResponse struct {
	// Challenge contains a serialized challenge in "data:difficulty:signature" format
	// example: YWJjZGVmZ2hpamtsbW5vcA==:20:c2lnbmF0dXJl
	Challenge string `json:"challenge"`
}

// SolutionRequest represents a request containing a solution
type SolutionRequest struct {
	// Challenge is a serialized challenge received from the server
	// example: YWJjZGVmZ2hpamtsbW5vcA==:20:c2lnbmF0dXJl
	// required: true
	Challenge string `json:"challenge" binding:"required"`

	// Nonce is the found solution that satisfies the difficulty condition
	// example: 42387
	// required: true
	Nonce string `json:"nonce" binding:"required"`
}

// QuoteResponse represents a response containing a quote
type QuoteResponse struct {
	// Quote is a random quote from
	// example: Not all those who wander are lost.
	Quote string `json:"quote"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	// Error description
	// example: Invalid solution
	Error string `json:"error"`
}
