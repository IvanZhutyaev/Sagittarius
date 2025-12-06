package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type BidRequest struct {
	AuctionID      string `json:"auction_id"`
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

type BidResponse struct {
	BidID         string `json:"bid_id"`
	ReservationID string `json:"reservation_id"`
	Success       bool   `json:"success"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

func main() {
	baseURL := "http://localhost:8080"
	userID := "test-user-1"

	// Test 1: Submit a bid
	fmt.Println("Test 1: Submitting a bid...")
	bidReq := BidRequest{
		AuctionID:      "test-auction-1",
		Amount:         1500,
		IdempotencyKey: fmt.Sprintf("test-key-%d", time.Now().Unix()),
	}

	resp, err := submitBid(baseURL, userID, bidReq)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %+v\n", resp)

	// Test 2: Submit duplicate bid (idempotency test)
	fmt.Println("\nTest 2: Submitting duplicate bid (idempotency test)...")
	resp2, err := submitBid(baseURL, userID, bidReq)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %+v\n", resp2)
	if resp2.BidID == resp.BidID {
		fmt.Println("✓ Idempotency works! Same bid ID returned.")
	}

	// Test 3: Submit higher bid
	fmt.Println("\nTest 3: Submitting higher bid...")
	bidReq3 := BidRequest{
		AuctionID:      "test-auction-1",
		Amount:         2000,
		IdempotencyKey: fmt.Sprintf("test-key-%d", time.Now().Unix()),
	}

	resp3, err := submitBid(baseURL, userID, bidReq3)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Response: %+v\n", resp3)
}

func submitBid(baseURL, userID string, req BidRequest) (*BidResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", baseURL+"/api/v1/bids", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-User-ID", userID)
	if req.IdempotencyKey != "" {
		httpReq.Header.Set("X-Idempotency-Key", req.IdempotencyKey)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var bidResp BidResponse
	if err := json.Unmarshal(body, &bidResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &bidResp, nil
}

