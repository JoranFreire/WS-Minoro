// Package billing wraps the Pagar.me v5 REST API. Card data never reaches
// this service directly — the frontend tokenizes it client-side (via
// Pagar.me's tokenizecard.js) and only the resulting single-use card_token
// is sent here, keeping this service out of PCI-DSS scope.
//
// NOTE: this client was written from Pagar.me's public documentation
// (docs.pagar.me) without a live account to test against. The request/
// response shapes below (POST /subscriptions with plan_id + card_token,
// Basic-Auth with the secret key as username) match what the docs describe,
// but should be verified against a real test-mode account before relying
// on this in production — Pagar.me's dashboard shows the actual request/
// response of every API call, which is the fastest way to catch a mismatch.
package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultBaseURL = "https://api.pagar.me/core/v5"

type Client struct {
	secretKey string
	baseURL   string
	http      *http.Client
}

func NewClient(secretKey string) *Client {
	return &Client{secretKey: secretKey, baseURL: defaultBaseURL, http: &http.Client{}}
}

// NewClientWithBaseURL allows tests (or a sandbox environment) to point the
// client at something other than the production API.
func NewClientWithBaseURL(secretKey, baseURL string) *Client {
	return &Client{secretKey: secretKey, baseURL: baseURL, http: &http.Client{}}
}

type CreateSubscriptionInput struct {
	PlanID        string
	CustomerName  string
	CustomerEmail string
	CardToken     string
}

type Subscription struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Customer struct {
		ID string `json:"id"`
	} `json:"customer"`
}

// CreateSubscription creates a new recurring subscription for a customer,
// charging the tokenized card. Pagar.me creates the customer record
// implicitly from the `customer` object on first use.
func (c *Client) CreateSubscription(ctx context.Context, in CreateSubscriptionInput) (*Subscription, error) {
	body := map[string]any{
		"plan_id":        in.PlanID,
		"payment_method": "credit_card",
		"card_token":     in.CardToken,
		"customer": map[string]string{
			"name":  in.CustomerName,
			"email": in.CustomerEmail,
		},
	}

	var sub Subscription
	if err := c.post(ctx, "/subscriptions", body, &sub); err != nil {
		return nil, err
	}
	return &sub, nil
}

// CancelSubscription cancels an active subscription — used when a tenant
// downgrades back to the free plan.
func (c *Client) CancelSubscription(ctx context.Context, subscriptionID string) error {
	return c.delete(ctx, "/subscriptions/"+subscriptionID)
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.secretKey, "")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("pagar.me: %s %s returned %d: %s", http.MethodPost, path, resp.StatusCode, respBody)
	}
	if out != nil {
		return json.Unmarshal(respBody, out)
	}
	return nil
}

func (c *Client) delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.secretKey, "")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pagar.me: %s %s returned %d: %s", http.MethodDelete, path, resp.StatusCode, respBody)
	}
	return nil
}
