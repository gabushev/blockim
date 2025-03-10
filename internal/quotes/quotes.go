package quotes

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"
)

type APIResponse struct {
	Docs []Quote `json:"docs"`
}

type Quote struct {
	ID        string `json:"_id"`
	Dialog    string `json:"dialog"`
	Movie     string `json:"movie"`
	Character string `json:"character"`
}

var fallbackQuotes = []string{
	"Ludzie to potwory, Geralt. Gorsze niż te, na które polujesz.",
	"Coś się kończy, coś się zaczyna.",
	"Lepiej zaliczać się do niektórych, niż do wszystkich.",
	"Neutralność to nic innego jak wyparcie się człowieczeństwa.",
	"Tylko myślący człowiek może popełnić naprawdę wielką głupotę.",
	"Pierdol się, czarodzieju!",
	"Jesteś uparty jak osioł, wiedźminie!",
	"Idź się rypać, Wiedźminie!",
	"Gówno wiesz i gówno rozumiesz, Wiedźminie.",
	"Do kurwy nędzy!",
	"Spierdalaj!",
	"Kurwa mać!",
	"Ja pierdolę…",
}

type Service struct {
	ApiURL      string
	apiKey      string
	quotes      []string
	quotesMu    *sync.RWMutex
	initialized bool
}

func NewsService(apiURL string, apiKey string) *Service {
	qs := &Service{
		ApiURL:      apiURL,
		apiKey:      apiKey,
		quotesMu:    &sync.RWMutex{},
		initialized: false,
	}

	err := qs.initQuotes()
	if err != nil {
		fmt.Errorf("unable to initialize quotes: %v", err)
	}

	return qs
}

func (s *Service) initQuotes() error {
	if s.apiKey == "" || s.apiKey == "PUT_YOUR_VALID_API_KEY_HERE" {
		log.Printf("No valid API key provided, using fallback quotes")
		s.quotesMu.Lock()
		s.quotes = fallbackQuotes
		s.quotesMu.Unlock()
		s.initialized = true
		return nil
	}

	req, err := http.NewRequest("GET", s.ApiURL, nil)
	if err != nil {
		return fmt.Errorf("request creation error: %v", err)
	}
	log.Printf("making request to %s", s.ApiURL)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+s.apiKey)
	// so I got some troubles with connection probably the reason is my current network
	// it should not get worse but these settings is needed specifically for me right now
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
	fallback := func() {
		s.quotesMu.Lock()
		s.quotes = fallbackQuotes
		s.quotesMu.Unlock()
		s.initialized = true
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("error on request, err: %v", err)
		defer fallback()

		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("error on quetes request, unexpected response code %d", resp.StatusCode)
		defer fallback()

		return nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error on response reading, err: %v", err)
		defer fallback()

		return nil
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Printf("error on respinse parsing, err: %v", err)
		defer fallback()
		return nil
	}
	if len(apiResp.Docs) == 0 {
		log.Printf("empty quotes response")
		defer fallback()

		return nil
	}
	s.quotesMu.Lock()
	for _, quote := range apiResp.Docs {
		if quote.Dialog != "" {
			s.quotes = append(s.quotes, quote.Dialog)
		}
	}
	s.quotesMu.Unlock()
	s.initialized = true
	log.Printf("initialized quotes: %d", len(s.quotes))

	return nil
}

func (s *Service) GetRandomQuote() (string, error) {
	if !s.initialized {
		err := s.initQuotes()
		if err != nil {
			return "", fmt.Errorf("error initializing quotes: %v", err)
		}
	}
	s.quotesMu.RLock()
	defer s.quotesMu.RUnlock()
	if len(s.quotes) == 0 {
		return "", fmt.Errorf("no quotes available")
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	return s.quotes[r.Intn(len(s.quotes))], nil
}

func (s *Service) Initialized() bool {
	return s.initialized
}
