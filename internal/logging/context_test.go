package logging

import (
	"context"
	"strings"
	"testing"
)

func TestContextWithCorrelationID(t *testing.T) {
	ctx := context.Background()
	correlationID := "test-correlation-123"

	// Add correlation ID to context
	ctx = ContextWithCorrelationID(ctx, correlationID)

	// Retrieve correlation ID
	retrieved, ok := CorrelationIDFromContext(ctx)
	if !ok {
		t.Fatal("Expected correlation ID to be present in context")
	}
	if retrieved != correlationID {
		t.Errorf("Expected correlation ID %s, got %s", correlationID, retrieved)
	}
}

func TestContextWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "test-request-456"

	// Add request ID to context
	ctx = ContextWithRequestID(ctx, requestID)

	// Retrieve request ID
	retrieved, ok := RequestIDFromContext(ctx)
	if !ok {
		t.Fatal("Expected request ID to be present in context")
	}
	if retrieved != requestID {
		t.Errorf("Expected request ID %s, got %s", requestID, retrieved)
	}
}

func TestGenerateCorrelationID(t *testing.T) {
	id1 := GenerateCorrelationID()
	id2 := GenerateCorrelationID()

	// Check prefix
	if !strings.HasPrefix(id1, "corr_") {
		t.Errorf("Expected correlation ID to start with 'corr_', got %s", id1)
	}

	// Check uniqueness
	if id1 == id2 {
		t.Error("Expected unique correlation IDs, but got duplicates")
	}

	// Check format
	parts := strings.Split(id1, "_")
	if len(parts) != 2 {
		t.Errorf("Expected correlation ID format 'corr_<id>', got %s", id1)
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	// Check prefix
	if !strings.HasPrefix(id1, "req_") {
		t.Errorf("Expected request ID to start with 'req_', got %s", id1)
	}

	// Check uniqueness
	if id1 == id2 {
		t.Error("Expected unique request IDs, but got duplicates")
	}

	// Check format
	parts := strings.Split(id1, "_")
	if len(parts) != 2 {
		t.Errorf("Expected request ID format 'req_<id>', got %s", id1)
	}
}

func TestMissingContextValues(t *testing.T) {
	ctx := context.Background()

	// Test missing correlation ID
	_, ok := CorrelationIDFromContext(ctx)
	if ok {
		t.Error("Expected no correlation ID in empty context")
	}

	// Test missing request ID
	_, ok = RequestIDFromContext(ctx)
	if ok {
		t.Error("Expected no request ID in empty context")
	}
}

func TestContextChaining(t *testing.T) {
	ctx := context.Background()
	correlationID := "corr-123"
	requestID := "req-456"

	// Add both IDs to context
	ctx = ContextWithCorrelationID(ctx, correlationID)
	ctx = ContextWithRequestID(ctx, requestID)

	// Verify both are present
	retrievedCorr, ok := CorrelationIDFromContext(ctx)
	if !ok || retrievedCorr != correlationID {
		t.Errorf("Expected correlation ID %s, got %s", correlationID, retrievedCorr)
	}

	retrievedReq, ok := RequestIDFromContext(ctx)
	if !ok || retrievedReq != requestID {
		t.Errorf("Expected request ID %s, got %s", requestID, retrievedReq)
	}
}
