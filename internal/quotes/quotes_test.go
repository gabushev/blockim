package quotes

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
)

// MockQuoteService implements QuoteService interface for testing
type MockQuoteService struct {
	ctrl        *gomock.Controller
	quotes      []string
	initialized bool
}

func NewMockQuoteService(ctrl *gomock.Controller) *MockQuoteService {
	return &MockQuoteService{
		ctrl:        ctrl,
		quotes:      make([]string, 0),
		initialized: false,
	}
}

func (m *MockQuoteService) GetRandomQuote() (string, error) {
	if len(m.quotes) == 0 {
		return "", errors.New("no quotes available")
	}
	return m.quotes[0], nil
}

func (m *MockQuoteService) Initialized() bool {
	return m.initialized
}

func (m *MockQuoteService) EXPECT() *MockQuoteService {
	return m
}

func (m *MockQuoteService) Return(quote string, err error) *gomock.Call {
	m.quotes = []string{quote}
	return &gomock.Call{}
}

func TestGetRandomQuote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name      string
		setupMock func(*MockQuoteService)
		wantQuote string
		wantError error
	}{
		{
			name: "Successfully get quote",
			setupMock: func(m *MockQuoteService) {
				m.quotes = []string{"Test quote"}
			},
			wantQuote: "Test quote",
			wantError: nil,
		},
		{
			name: "Error getting quote",
			setupMock: func(m *MockQuoteService) {
				m.quotes = []string{}
			},
			wantQuote: "",
			wantError: errors.New("no quotes available"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := NewMockQuoteService(ctrl)
			tc.setupMock(mockService)

			quote, err := mockService.GetRandomQuote()

			if (err != nil && tc.wantError == nil) || (err == nil && tc.wantError != nil) {
				t.Errorf("GetRandomQuote() error = %v, wantError %v", err, tc.wantError)
				return
			}
			if err != nil && tc.wantError != nil && err.Error() != tc.wantError.Error() {
				t.Errorf("GetRandomQuote() error = %v, wantError %v", err, tc.wantError)
				return
			}
			if quote != tc.wantQuote {
				t.Errorf("GetRandomQuote() = %v, want %v", quote, tc.wantQuote)
			}
		})
	}
}

func TestInitialized(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testCases := []struct {
		name      string
		setupMock func(*MockQuoteService)
		want      bool
	}{
		{
			name: "Service is initialized",
			setupMock: func(m *MockQuoteService) {
				m.initialized = true
			},
			want: true,
		},
		{
			name: "Service is not initialized",
			setupMock: func(m *MockQuoteService) {
				m.initialized = false
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockService := NewMockQuoteService(ctrl)
			tc.setupMock(mockService)

			if got := mockService.Initialized(); got != tc.want {
				t.Errorf("Initialized() = %v, want %v", got, tc.want)
			}
		})
	}
}
