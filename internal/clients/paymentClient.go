package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type IPaymentClient interface {
	CreateBill(ctx context.Context, email, name string, amount float64, description, callbackURL, collectionID string) (*CreateBillRes, error)
}

type CreateBillRes struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type billplzClient struct {
	apiURL string
	apiKey string
}

func NewBillplzClient(apiURL, apiKey string) IPaymentClient {
	return &billplzClient{
		apiURL: apiURL,
		apiKey: apiKey,
	}
}

func (c *billplzClient) CreateBill(ctx context.Context, email, name string, amount float64, description, callbackURL, collectionID string) (*CreateBillRes, error) {
	// Billplz amount is in cents, so convert from MYR (float64) to cents (int)
	amountInCents := int(amount * 100)

	data := url.Values{}
	data.Set("collection_id", collectionID)
	data.Set("email", email)
	data.Set("name", name)
	data.Set("amount", fmt.Sprintf("%d", amountInCents))
	data.Set("callback_url", callbackURL)
	data.Set("description", description)

	reqURL := fmt.Sprintf("%s/bills", c.apiURL)
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Set Basic Authorization using the api_key as username (password is empty)
	req.SetBasicAuth(c.apiKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("billplz API error (status: %s): %s", resp.Status, string(bodyBytes))
	}

	var billRes CreateBillRes
	if err := json.NewDecoder(resp.Body).Decode(&billRes); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &billRes, nil
}
