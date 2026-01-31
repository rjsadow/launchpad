package sessions

import (
	"context"
	"testing"
	"time"

	"github.com/rjsadow/launchpad/internal/db"
)

func TestInMemoryCache_GetSet(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	session := &db.Session{
		ID:        "test-session-1",
		UserID:    "user-1",
		AppID:     "app-1",
		PodName:   "pod-1",
		PodIP:     "10.0.0.1",
		Status:    db.SessionStatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Test cache miss
	_, err := cache.Get(ctx, session.ID)
	if err != ErrCacheMiss {
		t.Errorf("expected ErrCacheMiss, got %v", err)
	}

	// Test set
	if err := cache.Set(ctx, session, time.Hour); err != nil {
		t.Errorf("unexpected error on Set: %v", err)
	}

	// Test get
	retrieved, err := cache.Get(ctx, session.ID)
	if err != nil {
		t.Errorf("unexpected error on Get: %v", err)
	}
	if retrieved.ID != session.ID {
		t.Errorf("expected ID %s, got %s", session.ID, retrieved.ID)
	}
	if retrieved.UserID != session.UserID {
		t.Errorf("expected UserID %s, got %s", session.UserID, retrieved.UserID)
	}
	if retrieved.Status != session.Status {
		t.Errorf("expected Status %s, got %s", session.Status, retrieved.Status)
	}
}

func TestInMemoryCache_Delete(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	session := &db.Session{
		ID:     "test-session-2",
		UserID: "user-1",
		Status: db.SessionStatusRunning,
	}

	// Set session
	if err := cache.Set(ctx, session, time.Hour); err != nil {
		t.Errorf("unexpected error on Set: %v", err)
	}

	// Verify it exists
	if _, err := cache.Get(ctx, session.ID); err != nil {
		t.Errorf("session should exist after Set: %v", err)
	}

	// Delete session
	if err := cache.Delete(ctx, session.ID); err != nil {
		t.Errorf("unexpected error on Delete: %v", err)
	}

	// Verify it's gone
	_, err := cache.Get(ctx, session.ID)
	if err != ErrCacheMiss {
		t.Errorf("expected ErrCacheMiss after Delete, got %v", err)
	}
}

func TestInMemoryCache_Close(t *testing.T) {
	cache := NewInMemoryCache()

	// Close should not error for in-memory cache
	if err := cache.Close(); err != nil {
		t.Errorf("unexpected error on Close: %v", err)
	}
}

func TestInMemoryCache_UpdateSession(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	session := &db.Session{
		ID:     "test-session-3",
		UserID: "user-1",
		Status: db.SessionStatusCreating,
	}

	// Set initial session
	if err := cache.Set(ctx, session, time.Hour); err != nil {
		t.Errorf("unexpected error on Set: %v", err)
	}

	// Update session
	session.Status = db.SessionStatusRunning
	session.PodIP = "10.0.0.5"
	if err := cache.Set(ctx, session, time.Hour); err != nil {
		t.Errorf("unexpected error on Set: %v", err)
	}

	// Verify update
	retrieved, err := cache.Get(ctx, session.ID)
	if err != nil {
		t.Errorf("unexpected error on Get: %v", err)
	}
	if retrieved.Status != db.SessionStatusRunning {
		t.Errorf("expected Status %s, got %s", db.SessionStatusRunning, retrieved.Status)
	}
	if retrieved.PodIP != "10.0.0.5" {
		t.Errorf("expected PodIP 10.0.0.5, got %s", retrieved.PodIP)
	}
}

func TestInMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewInMemoryCache()
	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			session := &db.Session{
				ID:     "concurrent-session",
				UserID: "user-1",
				Status: db.SessionStatusRunning,
			}
			for j := 0; j < 100; j++ {
				cache.Set(ctx, session, time.Hour)
				cache.Get(ctx, session.ID)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestSessionCacheInterface verifies that InMemoryCache implements SessionCache
func TestSessionCacheInterface(t *testing.T) {
	var _ SessionCache = (*InMemoryCache)(nil)
	var _ SessionCache = (*RedisCache)(nil)
}
