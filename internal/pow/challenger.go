package pow

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/bits"
	"sync"
)

type ChallengeMaker struct {
	serverKey  []byte
	difficulty int
	challenges sync.Map
}

func NewChallengeMaker(serverKey string, difficulty int) *ChallengeMaker {
	return &ChallengeMaker{
		serverKey:  []byte(serverKey),
		difficulty: difficulty,
	}
}

func (cm *ChallengeMaker) GenerateChallenge() (*Challenge, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	randomData := base64.StdEncoding.EncodeToString(b)
	dataToSign := fmt.Sprintf("%s:%d", randomData, cm.difficulty)
	h := hmac.New(sha256.New, cm.serverKey)
	h.Write([]byte(dataToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	challenge := &Challenge{
		Data:       randomData,
		Difficulty: cm.difficulty,
		Signature:  signature,
	}
	cm.challenges.Store(randomData, challenge)

	return challenge, nil
}

func (cm *ChallengeMaker) VerifyChallenge(serializedChallenge string) (*Challenge, bool) {
	c := Challenge{}
	err := c.Deserialize(serializedChallenge)
	if err != nil {
		return nil, false
	}

	dataToSign := fmt.Sprintf("%s:%d", c.Data, c.Difficulty)
	h := hmac.New(sha256.New, cm.serverKey)
	h.Write([]byte(dataToSign))
	expectedSignature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	if c.Signature != expectedSignature {
		return nil, false
	}
	if c.Difficulty != cm.difficulty {
		return nil, false
	}
	if _, exists := cm.challenges.Load(c.Data); !exists {
		return nil, false
	}

	return &c, true
}

func (cm *ChallengeMaker) VerifySolution(serializedChallenge, nonce string) bool {
	challenge, valid := cm.VerifyChallenge(serializedChallenge)
	if !valid {
		return false
	}
	hash := sha256.Sum256([]byte(challenge.Data + nonce))
	isSolved := countLeadingZeroBits(hash[:]) >= challenge.Difficulty
	if isSolved {
		defer cm.challenges.Delete(challenge.Data)
	}

	return isSolved
}

func countLeadingZeroBits(hash []byte) int {
	var count int
	for _, b := range hash {
		if b == 0 {
			count += 8
			continue
		}
		count += bits.LeadingZeros8(b)
		break
	}

	return count
}
