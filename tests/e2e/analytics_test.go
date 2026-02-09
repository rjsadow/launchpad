package e2e

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("E2E Analytics", func() {

	Describe("Launch Recording", func() {
		It("records a launch event", func() {
			// Create an app to record a launch for
			createBody := []byte(`{
				"id": "e2e-analytics-app",
				"name": "Analytics Test App",
				"launch_type": "url",
				"url": "https://example.com"
			}`)
			resp := authPost(baseURL+"/api/apps", adminToken, createBody)
			resp.Body.Close()
			DeferCleanup(deleteE2EApp, "e2e-analytics-app")

			// Record a launch
			body := []byte(`{"app_id":"e2e-analytics-app"}`)
			resp = authPost(baseURL+"/api/analytics/launch", adminToken, body)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusCreated))

			result := decodeJSON(resp)
			Expect(result["status"]).To(Equal("recorded"))
		})

		It("returns 400 for missing app_id", func() {
			body := []byte(`{}`)
			resp := authPost(baseURL+"/api/analytics/launch", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("Analytics Stats", func() {
		It("returns stats for admin", func() {
			resp := authGet(baseURL+"/api/analytics/stats", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("rejects non-admin access to stats", func() {
			userID := createTestUser("e2e-analytics-user", "password123", "analytics@test.com", []string{"user"})
			DeferCleanup(deleteTestUser, userID)

			userToken, err := login(baseURL, "e2e-analytics-user", "password123")
			Expect(err).NotTo(HaveOccurred())

			resp := authGet(baseURL+"/api/analytics/stats", userToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
		})
	})
})
