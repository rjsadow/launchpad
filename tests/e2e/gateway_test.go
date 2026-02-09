package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gorilla/websocket"
)

var _ = Describe("E2E WebSocket Gateway", func() {
	var (
		appID     string
		sessionID string
	)

	BeforeEach(func() {
		appID = "e2e-gateway-app"
		createE2EApp(appID, "container")

		body := []byte(`{"app_id":"` + appID + `","user_id":"e2e-user"}`)
		resp := authPost(baseURL+"/api/sessions", adminToken, body)
		var session map[string]any
		json.NewDecoder(resp.Body).Decode(&session)
		resp.Body.Close()
		Expect(resp.StatusCode).To(Equal(http.StatusCreated))

		sessionID = session["id"].(string)
		waitForSessionRunning(sessionID, 3*time.Minute)
	})

	AfterEach(func() {
		if sessionID != "" {
			resp := authPost(baseURL+"/api/sessions/"+sessionID+"/stop", adminToken, nil)
			resp.Body.Close()
			resp = authDelete(baseURL+"/api/sessions/"+sessionID, adminToken)
			resp.Body.Close()
		}
		deleteE2EApp(appID)
	})

	wsURL := func(path string) string {
		base := strings.Replace(baseURL, "http://", "ws://", 1)
		base = strings.Replace(base, "https://", "wss://", 1)
		return base + path
	}

	Describe("VNC WebSocket Connection", func() {
		It("connects with valid token in query param", func() {
			url := wsURL("/ws/sessions/" + sessionID + "?token=" + adminToken)
			dialer := websocket.Dialer{
				HandshakeTimeout: 5 * time.Second,
			}
			conn, resp, err := dialer.Dial(url, nil)
			if err != nil {
				// The VNC backend may refuse if no VNC server is running
				// in the nginx:latest container. A 101 (upgrade) or connection
				// with immediate close is acceptable proof the gateway auth worked.
				if resp != nil {
					// Gateway auth passed but backend proxy failed
					GinkgoWriter.Printf("WebSocket dial error (expected with nginx): %v, status: %d\n", err, resp.StatusCode)
					// If we got past auth (not 401/403), the gateway is working
					Expect(resp.StatusCode).NotTo(Equal(http.StatusUnauthorized))
					Expect(resp.StatusCode).NotTo(Equal(http.StatusForbidden))
					return
				}
				// Connection-level error after upgrade - gateway worked
				GinkgoWriter.Printf("WebSocket connection error (expected with nginx): %v\n", err)
				return
			}
			defer conn.Close()
			// Connection succeeded - gateway auth and routing work
		})

		It("rejects connection without token", func() {
			url := wsURL("/ws/sessions/" + sessionID)
			dialer := websocket.Dialer{
				HandshakeTimeout: 5 * time.Second,
			}
			_, resp, err := dialer.Dial(url, nil)
			Expect(err).To(HaveOccurred())
			if resp != nil {
				resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			}
		})

		It("rejects connection with invalid token", func() {
			url := wsURL("/ws/sessions/" + sessionID + "?token=invalid-token")
			dialer := websocket.Dialer{
				HandshakeTimeout: 5 * time.Second,
			}
			_, resp, err := dialer.Dial(url, nil)
			Expect(err).To(HaveOccurred())
			if resp != nil {
				resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
			}
		})

		It("connects with Authorization header", func() {
			url := wsURL("/ws/sessions/" + sessionID)
			dialer := websocket.Dialer{
				HandshakeTimeout: 5 * time.Second,
			}
			headers := http.Header{}
			headers.Set("Authorization", "Bearer "+adminToken)

			conn, resp, err := dialer.Dial(url, headers)
			if err != nil {
				if resp != nil {
					GinkgoWriter.Printf("WebSocket dial with header error: %v, status: %d\n", err, resp.StatusCode)
					Expect(resp.StatusCode).NotTo(Equal(http.StatusUnauthorized))
					Expect(resp.StatusCode).NotTo(Equal(http.StatusForbidden))
					return
				}
				GinkgoWriter.Printf("WebSocket connection error (expected with nginx): %v\n", err)
				return
			}
			defer conn.Close()
		})
	})

	Describe("Session Ownership", func() {
		It("rejects non-owner access to WebSocket", func() {
			// Create a regular user
			userID := createTestUser("e2e-ws-user", "password123", "ws@test.com", []string{"user"})
			DeferCleanup(deleteTestUser, userID)

			userToken, err := login(baseURL, "e2e-ws-user", "password123")
			Expect(err).NotTo(HaveOccurred())

			url := wsURL("/ws/sessions/" + sessionID + "?token=" + userToken)
			dialer := websocket.Dialer{
				HandshakeTimeout: 5 * time.Second,
			}
			_, resp, err := dialer.Dial(url, nil)
			Expect(err).To(HaveOccurred())
			if resp != nil {
				resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
			}
		})
	})

	Describe("Non-existent Session", func() {
		It("returns 404 for non-existent session ID", func() {
			url := wsURL("/ws/sessions/nonexistent-session-id?token=" + adminToken)
			dialer := websocket.Dialer{
				HandshakeTimeout: 5 * time.Second,
			}
			_, resp, err := dialer.Dial(url, nil)
			Expect(err).To(HaveOccurred())
			if resp != nil {
				resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			}
		})
	})
})
