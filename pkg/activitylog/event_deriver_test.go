package activitylog

import (
	"strings"
	"testing"
)

func TestDefaultEventDeriver(t *testing.T) {
	category, name := DefaultEventDeriver(EventDeriverInput{
		Endpoint: "/api/v1/payment-request?x=1",
		Method:   "POST",
	})
	if category != "payment-request" || name != "payment-request" {
		t.Fatalf("expected payment-request fallback, got category=%q name=%q", category, name)
	}
}

func TestNewCoreLikeEventDeriver_TableMatch(t *testing.T) {
	deriver := NewCoreLikeEventDeriver(CoreLikeEventDeriverConfig{
		BasePath:   "/api/v1",
		TableNames: []string{"members", "payment_requests"},
	})

	category, name := deriver(EventDeriverInput{
		Endpoint: "/api/v1/payment-request/123",
		Method:   "GET",
	})
	if category != "PAYMENT_REQUESTS" {
		t.Fatalf("expected category PAYMENT_REQUESTS, got %q", category)
	}
	if name != "PAYMENT_REQUESTS_VIEW" {
		t.Fatalf("expected name PAYMENT_REQUESTS_VIEW, got %q", name)
	}
}

func TestNewCoreLikeEventDeriver_Fallback(t *testing.T) {
	deriver := NewCoreLikeEventDeriver(CoreLikeEventDeriverConfig{})

	category, name := deriver(EventDeriverInput{
		Endpoint: "/api/v1/project",
		Method:   "POST",
	})
	if category != "PROJECT" {
		t.Fatalf("expected fallback category PROJECT, got %q", category)
	}
	if name != "PROJECT_CREATE" {
		t.Fatalf("expected fallback name PROJECT_CREATE, got %q", name)
	}
}

func TestDefaultEventInfoDeriver_Description(t *testing.T) {
	statusCode := HTTPStatusCode(201)
	body := `{"amount":1000,"password":"secret"}`

	info := DefaultEventInfoDeriver(EventDeriverInput{
		Endpoint:    "/api/v1/payment-request",
		Method:      "POST",
		StatusCode:  &statusCode,
		RequestBody: &body,
	})
	if info.EventCategory != "payment-request" || info.EventName != "payment-request" {
		t.Fatalf("unexpected event identity: %#v", info)
	}
	if info.Description == "" {
		t.Fatalf("expected description to be derived")
	}
	if !strings.Contains(info.Description, "Successfully created") {
		t.Fatalf("expected create-success description, got %q", info.Description)
	}
	if strings.Contains(info.Description, "secret") {
		t.Fatalf("description leaked sensitive body value: %q", info.Description)
	}
}

func TestNewCoreLikeEventInfoDeriver_TableMatch(t *testing.T) {
	statusCode := HTTPStatusCode(200)
	deriver := NewCoreLikeEventInfoDeriver(CoreLikeEventDeriverConfig{
		BasePath:   "/api/v1",
		TableNames: []string{"members", "payment_requests"},
	})

	info := deriver(EventDeriverInput{
		Endpoint:   "/api/v1/payment-request/123",
		Method:     "GET",
		StatusCode: &statusCode,
	})
	if info.EventCategory != "PAYMENT_REQUESTS" {
		t.Fatalf("expected category PAYMENT_REQUESTS, got %q", info.EventCategory)
	}
	if info.EventName != "PAYMENT_REQUESTS_VIEW" {
		t.Fatalf("expected name PAYMENT_REQUESTS_VIEW, got %q", info.EventName)
	}
	if !strings.Contains(info.Description, "Successfully retrieved") {
		t.Fatalf("expected derived description, got %q", info.Description)
	}
}
