package pow

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Challenge struct {
	Data       string
	Difficulty int
	Signature  string
}

func (c *Challenge) Serialize() string {
	// format randomData:difficulty:signature
	return fmt.Sprintf("%s:%d:%s", c.Data, c.Difficulty, c.Signature)
}

func (c *Challenge) Deserialize(data string) error {
	parts := strings.Split(data, ":")
	if len(parts) != 3 {
		return errors.New("invalid challenge format")
	}
	difficulty, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid challenge difficulty: %v", err)
	}
	c.Difficulty = difficulty
	c.Signature = parts[2]
	c.Data = parts[0]

	return nil
}

func (c *Challenge) Solve() (string, error) {
	var counter uint64 = 0
	for {
		nonce := fmt.Sprintf("%d", counter)
		hash := sha256.Sum256([]byte(c.Data + nonce))
		if countLeadingZeroBits(hash[:]) >= c.Difficulty {
			return nonce, nil
		}
		counter++
	}
}
