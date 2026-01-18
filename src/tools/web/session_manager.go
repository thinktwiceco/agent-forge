package web

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	agentforge "github.com/thinktwice/agentForge/src"
)

const (
	// idleSessionTimeout is the duration after which an idle session will be automatically closed
	idleSessionTimeout = 5 * time.Minute
	// cleanupInterval is how often the cleanup goroutine checks for idle sessions
	cleanupInterval = 1 * time.Minute
)

// BrowserSession holds a browser context and its cancel function
type browserSession struct {
	ctx      context.Context
	cancel   context.CancelFunc
	created  time.Time
	lastUsed time.Time
}

// SessionManager manages browser sessions for multiple agents
type SessionManager struct {
	sessions      map[string]*browserSession
	mutex         sync.RWMutex
	metrics       *SessionMetrics
	stopCleanup   chan struct{}
	cleanupDone   sync.WaitGroup
	cleanupActive bool
}

// SessionMetrics tracks session statistics
type SessionMetrics struct {
	TotalSessionsCreated int64
	TotalSessionsClosed  int64
	ActiveSessions       int64
	TotalOperations      int64
	FailedOperations     int64
	mutex                sync.RWMutex
}

// NewSessionManager creates a new session manager and starts the background cleanup goroutine
func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*browserSession),
		metrics: &SessionMetrics{
			TotalSessionsCreated: 0,
			TotalSessionsClosed:  0,
			ActiveSessions:       0,
			TotalOperations:      0,
			FailedOperations:     0,
		},
		stopCleanup: make(chan struct{}),
	}

	// Start background cleanup goroutine
	sm.startCleanupGoroutine()

	return sm
}

// startCleanupGoroutine starts a background goroutine that periodically cleans up idle sessions
func (sm *SessionManager) startCleanupGoroutine() {
	sm.mutex.Lock()
	if sm.cleanupActive {
		sm.mutex.Unlock()
		return
	}
	sm.cleanupActive = true
	sm.mutex.Unlock()

	sm.cleanupDone.Add(1)
	go func() {
		defer sm.cleanupDone.Done()
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				sm.cleanupIdleSessions()
			case <-sm.stopCleanup:
				agentforge.Info("Stopping browser session cleanup goroutine")
				return
			}
		}
	}()
}

// cleanupIdleSessions closes sessions that have been idle for more than idleSessionTimeout
func (sm *SessionManager) cleanupIdleSessions() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	var sessionsToClose []struct {
		key     string
		idleFor time.Duration
	}

	// Find idle sessions
	for key, session := range sm.sessions {
		// Check if session is idle (not used for more than idleSessionTimeout)
		idleDuration := now.Sub(session.lastUsed)
		if idleDuration > idleSessionTimeout {
			sessionsToClose = append(sessionsToClose, struct {
				key     string
				idleFor time.Duration
			}{key: key, idleFor: idleDuration})
		}
	}

	// Close idle sessions
	for _, sessionInfo := range sessionsToClose {
		agentforge.Info("Closing idle browser session %s (idle for %v)", sessionInfo.key, sessionInfo.idleFor)
		sm.closeSessionLocked(sessionInfo.key)
	}

	if len(sessionsToClose) > 0 {
		agentforge.Info("Cleaned up %d idle browser session(s)", len(sessionsToClose))
	}
}

// StopCleanup stops the background cleanup goroutine
// This should be called during graceful shutdown
// Safe to call multiple times
func (sm *SessionManager) StopCleanup() {
	sm.mutex.Lock()
	wasActive := sm.cleanupActive
	sm.cleanupActive = false
	sm.mutex.Unlock()

	if wasActive {
		close(sm.stopCleanup)
		sm.cleanupDone.Wait()
	}
}

// GetMetrics returns a copy of the current metrics
func (sm *SessionManager) GetMetrics() SessionMetrics {
	sm.metrics.mutex.RLock()
	defer sm.metrics.mutex.RUnlock()

	sm.mutex.RLock()
	activeCount := int64(len(sm.sessions))
	sm.mutex.RUnlock()

	return SessionMetrics{
		TotalSessionsCreated: sm.metrics.TotalSessionsCreated,
		TotalSessionsClosed:  sm.metrics.TotalSessionsClosed,
		ActiveSessions:       activeCount,
		TotalOperations:      sm.metrics.TotalOperations,
		FailedOperations:     sm.metrics.FailedOperations,
	}
}

// getSessionKey generates a unique key for browser sessions
func (sm *SessionManager) getSessionKey(agentContext map[string]any) string {
	agentName, ok := agentContext["agentName"].(string)
	if !ok || agentName == "" {
		agentName = "default"
	}
	return fmt.Sprintf("browser_%s", agentName)
}

// GetOrCreateBrowser gets an existing browser context or creates a new one.
// headless controls whether the browser runs in headless mode (default: false).
// The browser context persists across tool calls.
func (sm *SessionManager) GetOrCreateBrowser(agentContext map[string]any, headless ...bool) (context.Context, error) {
	sessionKey := sm.getSessionKey(agentContext)

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Check if browser session already exists and is still valid
	if session, exists := sm.sessions[sessionKey]; exists {
		// Check if context is still valid (not cancelled or expired)
		select {
		case <-session.ctx.Done():
			// Context is invalid, clean it up and create a new one
			agentforge.Info("Browser context %s is closed, creating new one", sessionKey)
			sm.closeSessionLocked(sessionKey)
		default:
			// Context is still valid, reuse it
			agentforge.Info("Reusing existing browser context %s", sessionKey)
			session.lastUsed = time.Now()
			sm.recordOperation(true)
			return session.ctx, nil
		}
	}

	// Determine headless mode (default to false for better debugging)
	headlessMode := false
	if len(headless) > 0 {
		headlessMode = headless[0]
	}

	// Create new browser context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headlessMode),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancelCtx := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(format string, v ...interface{}) {
		// Log browser messages if needed
	}))

	// Initialize the browser by running a simple task
	// This ensures the browser is actually started and ready
	initErr := chromedp.Run(ctx)
	if initErr != nil {
		cancelCtx()
		cancelAlloc()
		sm.recordOperation(false)
		return nil, fmt.Errorf("failed to initialize browser: %v", initErr)
	}

	agentforge.Info("Created new browser context %s (headless=%v)", sessionKey, headlessMode)

	// Store in sessions map
	cancelFunc := func() {
		cancelCtx()
		cancelAlloc()
	}
	now := time.Now()
	sm.sessions[sessionKey] = &browserSession{
		ctx:      ctx,
		cancel:   cancelFunc,
		created:  now,
		lastUsed: now,
	}

	sm.metrics.mutex.Lock()
	sm.metrics.TotalSessionsCreated++
	sm.metrics.mutex.Unlock()

	sm.recordOperation(true)
	return ctx, nil
}

// CloseBrowser cleans up browser resources for a specific agent.
func (sm *SessionManager) CloseBrowser(agentContext map[string]any) error {
	sessionKey := sm.getSessionKey(agentContext)

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	return sm.closeSessionLocked(sessionKey)
}

// closeSessionLocked closes a session. Must be called with mutex locked.
func (sm *SessionManager) closeSessionLocked(sessionKey string) error {
	if session, exists := sm.sessions[sessionKey]; exists {
		session.cancel()
		delete(sm.sessions, sessionKey)

		sm.metrics.mutex.Lock()
		sm.metrics.TotalSessionsClosed++
		sm.metrics.mutex.Unlock()

		agentforge.Info("Closed browser session %s", sessionKey)
	}
	return nil
}

// CloseAllBrowsers cleans up all browser sessions and stops the cleanup goroutine.
// Useful for graceful shutdown.
func (sm *SessionManager) CloseAllBrowsers() {
	// Stop the cleanup goroutine first
	sm.StopCleanup()

	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for key := range sm.sessions {
		sm.closeSessionLocked(key)
	}
}

// GetBrowserContext returns the existing browser context without creating a new one.
// Returns nil if no browser context exists.
func (sm *SessionManager) GetBrowserContext(agentContext map[string]any) context.Context {
	sessionKey := sm.getSessionKey(agentContext)

	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if session, exists := sm.sessions[sessionKey]; exists {
		// Check if context is still valid
		select {
		case <-session.ctx.Done():
			return nil
		default:
			return session.ctx
		}
	}
	return nil
}

// recordOperation records an operation for metrics
func (sm *SessionManager) recordOperation(success bool) {
	sm.metrics.mutex.Lock()
	defer sm.metrics.mutex.Unlock()

	sm.metrics.TotalOperations++
	if !success {
		sm.metrics.FailedOperations++
	}
}

// RecordOperation records an operation for metrics (public method)
func (sm *SessionManager) RecordOperation(success bool) {
	sm.recordOperation(success)
}
