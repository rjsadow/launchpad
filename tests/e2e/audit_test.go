package e2e

import (
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("E2E Audit Logs", func() {

	Describe("Audit Log Query", func() {
		It("returns audit logs", func() {
			resp := authGet(baseURL+"/api/audit", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result).To(HaveKey("logs"))
		})

		It("filters by action", func() {
			resp := authGet(baseURL+"/api/audit?action=LOGIN", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			logs, ok := result["logs"].([]any)
			if ok && len(logs) > 0 {
				for _, l := range logs {
					log := l.(map[string]any)
					Expect(log["action"]).To(Equal("LOGIN"))
				}
			}
		})

		It("filters by user", func() {
			resp := authGet(baseURL+"/api/audit?user=admin", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			logs, ok := result["logs"].([]any)
			if ok && len(logs) > 0 {
				for _, l := range logs {
					log := l.(map[string]any)
					Expect(log["user"]).To(Equal("admin"))
				}
			}
		})

		It("supports pagination with limit and offset", func() {
			resp := authGet(baseURL+"/api/audit?limit=2&offset=0", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			logs, ok := result["logs"].([]any)
			if ok {
				Expect(len(logs)).To(BeNumerically("<=", 2))
			}
		})
	})

	Describe("Audit Filters", func() {
		It("returns available filter options", func() {
			resp := authGet(baseURL+"/api/audit/filters", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result).To(HaveKey("actions"))
			Expect(result).To(HaveKey("users"))

			actions, ok := result["actions"].([]any)
			if ok {
				Expect(len(actions)).To(BeNumerically(">=", 1),
					"at least LOGIN action should exist from BeforeSuite")
			}
		})
	})

	Describe("Audit Export", func() {
		It("exports as JSON", func() {
			resp := authGet(baseURL+"/api/audit/export?format=json", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("Content-Type")).To(ContainSubstring("application/json"))
			Expect(resp.Header.Get("Content-Disposition")).To(ContainSubstring("audit_log.json"))
		})

		It("exports as CSV", func() {
			resp := authGet(baseURL+"/api/audit/export?format=csv", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("Content-Type")).To(ContainSubstring("text/csv"))
			Expect(resp.Header.Get("Content-Disposition")).To(ContainSubstring("audit_log.csv"))
		})
	})

	Describe("Audit Logging of Actions", func() {
		It("records LOGIN events from BeforeSuite", func() {
			resp := authGet(baseURL+"/api/audit?action=LOGIN&limit=10", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			logs, ok := result["logs"].([]any)
			Expect(ok).To(BeTrue())
			Expect(len(logs)).To(BeNumerically(">=", 1),
				"at least one LOGIN event should be recorded")
		})

		It("records app CRUD events", func() {
			// Create an app to generate an audit event
			body := []byte(`{
				"id": "e2e-audit-app",
				"name": "Audit Test App",
				"launch_type": "url",
				"url": "https://example.com"
			}`)
			resp := authPost(baseURL+"/api/apps", adminToken, body)
			resp.Body.Close()
			DeferCleanup(deleteE2EApp, "e2e-audit-app")

			// Check audit log for CREATE_APP
			resp = authGet(baseURL+"/api/audit?action=CREATE_APP&limit=10", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			logs, ok := result["logs"].([]any)
			Expect(ok).To(BeTrue())

			var found bool
			for _, l := range logs {
				log := l.(map[string]any)
				details, _ := log["details"].(string)
				if strings.Contains(details, "e2e-audit-app") {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "CREATE_APP for e2e-audit-app should be logged")
		})
	})

	Describe("Audit Access Control", func() {
		It("rejects non-admin access to audit logs", func() {
			userID := createTestUser("e2e-audit-user", "password123", "audit@test.com", []string{"user"})
			DeferCleanup(deleteTestUser, userID)

			userToken, err := login(baseURL, "e2e-audit-user", "password123")
			Expect(err).NotTo(HaveOccurred())

			resp := authGet(baseURL+"/api/audit", userToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
		})
	})
})
