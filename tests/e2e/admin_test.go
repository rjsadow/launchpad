package e2e

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("E2E Admin", func() {

	Describe("Settings", func() {
		It("retrieves admin settings", func() {
			resp := authGet(baseURL+"/api/admin/settings", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result).To(HaveKey("allow_registration"))
			Expect(result).To(HaveKey("max_sessions_per_user"))
			Expect(result).To(HaveKey("max_global_sessions"))
		})

		It("updates and reads back a setting", func() {
			// Get current value
			resp := authGet(baseURL+"/api/admin/settings", adminToken)
			original := decodeJSON(resp)
			resp.Body.Close()

			// Update
			resp = authPut(baseURL+"/api/admin/settings", adminToken,
				[]byte(`{"e2e_test_setting":"e2e_value"}`))
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

			// Verify
			resp = authGet(baseURL+"/api/admin/settings", adminToken)
			updated := decodeJSON(resp)
			resp.Body.Close()
			Expect(updated["e2e_test_setting"]).To(Equal("e2e_value"))

			// original settings are preserved
			Expect(updated).To(HaveKey("max_sessions_per_user"))
			_ = original
		})
	})

	Describe("Health", func() {
		It("returns detailed health info", func() {
			resp := authGet(baseURL+"/api/admin/health", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result).To(HaveKey("status"))
			Expect(result).To(HaveKey("components"))
			Expect(result).To(HaveKey("stats"))

			stats := result["stats"].(map[string]any)
			Expect(stats).To(HaveKey("active_sessions"))
			Expect(stats).To(HaveKey("total_apps"))
			Expect(stats).To(HaveKey("total_users"))
			Expect(stats).To(HaveKey("goroutines"))
			Expect(stats).To(HaveKey("memory_alloc_mb"))
		})
	})

	Describe("Diagnostics", func() {
		It("returns a diagnostics bundle as JSON", func() {
			resp := authGet(baseURL+"/api/admin/diagnostics", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("Content-Type")).To(ContainSubstring("application/json"))
		})
	})

	Describe("Support Info", func() {
		It("returns support information", func() {
			resp := authGet(baseURL+"/api/admin/support/info", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result["application"]).To(Equal("launchpad"))
			Expect(result).To(HaveKey("go_version"))
			Expect(result).To(HaveKey("os"))
			Expect(result).To(HaveKey("arch"))
			Expect(result).To(HaveKey("config"))

			cfg := result["config"].(map[string]any)
			Expect(cfg).To(HaveKey("namespace"))
			Expect(cfg).To(HaveKey("max_sessions_per_user"))
		})
	})

	Describe("Admin Sessions", func() {
		It("lists all sessions", func() {
			resp := authGet(baseURL+"/api/admin/sessions", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			// Response should be an array
			result := decodeJSONArray(resp)
			Expect(result).NotTo(BeNil())
		})
	})

	Describe("Admin Users", func() {
		It("lists users with admin user present", func() {
			resp := authGet(baseURL+"/api/admin/users", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			users := decodeJSONArray(resp)
			var adminFound bool
			for _, u := range users {
				user := u.(map[string]any)
				if user["username"] == "admin" {
					adminFound = true
					Expect(user).NotTo(HaveKey("password_hash"))
					break
				}
			}
			Expect(adminFound).To(BeTrue(), "admin user should be in the list")
		})

		It("prevents admin from deleting self", func() {
			// Get admin's user ID
			resp := authGet(baseURL+"/api/auth/me", adminToken)
			me := decodeJSON(resp)
			resp.Body.Close()
			adminID := me["id"].(string)

			resp = authDelete(baseURL+"/api/admin/users/"+adminID, adminToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("returns 404 for non-existent user", func() {
			resp := authDelete(baseURL+"/api/admin/users/nonexistent-id", adminToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})
})
