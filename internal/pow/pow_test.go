package pow

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
)

// MockChallengeMaker implements ChallengeMaker interface for testing
type MockChallengeMaker struct {
	ctrl          *gomock.Controller
	challenge     *Challenge
	generateError error
	validNonce    string
}

func NewMockChallengeMaker(ctrl *gomock.Controller) *MockChallengeMaker {
	return &MockChallengeMaker{
		ctrl: ctrl,
	}
}

func (m *MockChallengeMaker) GenerateChallenge() (*Challenge, error) {
	if m.generateError != nil {
		return nil, m.generateError
	}
	return m.challenge, nil
}

func (m *MockChallengeMaker) VerifySolution(challenge, nonce string) bool {
	return nonce == m.validNonce
}

func (m *MockChallengeMaker) EXPECT() *MockChallengeMaker {
	return m
}

func TestChallengeMaker_GenerateChallenge(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name          string
		setupMock     func(*MockChallengeMaker)
		wantChallenge *Challenge
		wantError     error
	}{
		{
			name: "Successful generation",
			setupMock: func(m *MockChallengeMaker) {
				m.challenge = &Challenge{
					Data:       "test-data",
					Difficulty: 20,
					Signature:  "test-signature",
				}
			},
			wantChallenge: &Challenge{
				Data:       "test-data",
				Difficulty: 20,
				Signature:  "test-signature",
			},
			wantError: nil,
		},
		{
			name: "Generation error",
			setupMock: func(m *MockChallengeMaker) {
				m.generateError = errors.New("generation error")
			},
			wantChallenge: nil,
			wantError:     errors.New("generation error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			maker := NewMockChallengeMaker(ctrl)
			tc.setupMock(maker)

			challenge, err := maker.GenerateChallenge()

			if (err != nil && tc.wantError == nil) || (err == nil && tc.wantError != nil) {
				t.Errorf("GenerateChallenge() error = %v, wantError %v", err, tc.wantError)
				return
			}
			if err != nil && tc.wantError != nil && err.Error() != tc.wantError.Error() {
				t.Errorf("GenerateChallenge() error = %v, wantError %v", err, tc.wantError)
				return
			}
			if tc.wantChallenge != nil {
				if challenge == nil {
					t.Error("Expected non-nil challenge")
					return
				}
				if challenge.Data != tc.wantChallenge.Data {
					t.Errorf("Challenge.Data = %v, want %v", challenge.Data, tc.wantChallenge.Data)
				}
				if challenge.Difficulty != tc.wantChallenge.Difficulty {
					t.Errorf("Challenge.Difficulty = %v, want %v", challenge.Difficulty, tc.wantChallenge.Difficulty)
				}
				if challenge.Signature != tc.wantChallenge.Signature {
					t.Errorf("Challenge.Signature = %v, want %v", challenge.Signature, tc.wantChallenge.Signature)
				}
			}
		})
	}
}

func TestChallengeMaker_VerifySolution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name      string
		setupMock func(*MockChallengeMaker)
		challenge string
		nonce     string
		want      bool
	}{
		{
			name: "Valid solution",
			setupMock: func(m *MockChallengeMaker) {
				m.validNonce = "42"
			},
			challenge: "test-challenge",
			nonce:     "42",
			want:      true,
		},
		{
			name: "Invalid solution",
			setupMock: func(m *MockChallengeMaker) {
				m.validNonce = "42"
			},
			challenge: "test-challenge",
			nonce:     "invalid",
			want:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			maker := NewMockChallengeMaker(ctrl)
			tc.setupMock(maker)

			if got := maker.VerifySolution(tc.challenge, tc.nonce); got != tc.want {
				t.Errorf("VerifySolution() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestChallenge_SerializeDeserialize(t *testing.T) {
	original := &Challenge{
		Data:       "test-data",
		Difficulty: 20,
		Signature:  "test-signature",
	}

	serialized := original.Serialize()
	deserialized := &Challenge{}
	if err := deserialized.Deserialize(serialized); err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	if deserialized.Data != original.Data {
		t.Errorf("Data mismatch: got %s, want %s", deserialized.Data, original.Data)
	}
	if deserialized.Difficulty != original.Difficulty {
		t.Errorf("Difficulty mismatch: got %d, want %d", deserialized.Difficulty, original.Difficulty)
	}
	if deserialized.Signature != original.Signature {
		t.Errorf("Signature mismatch: got %s, want %s", deserialized.Signature, original.Signature)
	}
}
