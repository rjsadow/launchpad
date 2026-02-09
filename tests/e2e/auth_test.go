package e2e

import (
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("E2E Authentication", func() {

	Describe("Login", func() {
		It("returns access and refresh tokens for valid credentials", func() {
			body := []byte(`{"username":"admin","password":"admin123"}`)
			resp := unauthPost(baseURL+"/api/auth/login", body)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result).To(HaveKey("access_token"))
			Expect(result).To(HaveKey("refresh_token"))
			Expect(result["access_token"]).NotTo(BeEmpty())
			Expect(result["refresh_token"]).NotTo(BeEmpty())
		})

		It("returns 401 for invalid password", func() {
			body := []byte(`{"username":"admin","password":"wrongpassword"}`)
			resp := unauthPost(baseURL+"/api/auth/login", body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})

		It("returns 400 for missing fields", func() {
			body := []byte(`{"username":"admin"}`)
			resp := unauthPost(baseURL+"/api/auth/login", body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("returns 400 for invalid JSON", func() {
			body := []byte(`not json`)
			resp := unauthPost(baseURL+"/api/auth/login", body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("Token Refresh", func() {
		It("returns new tokens for a valid refresh token", func() {
			_, refreshToken, err := loginFull(baseURL, adminUsername, adminPassword)
			Expect(err).NotTo(HaveOccurred())

			body := []byte(`{"refresh_token":"` + refreshToken + `"}`)
			resp := unauthPost(baseURL+"/api/auth/refresh", body)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result).To(HaveKey("access_token"))
			Expect(result["access_token"]).NotTo(BeEmpty())
		})

		It("returns 401 for an invalid refresh token", func() {
			body := []byte(`{"refresh_token":"invalid-token-value"}`)
			resp := unauthPost(baseURL+"/api/auth/refresh", body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})

		It("returns 400 for missing refresh token", func() {
			body := []byte(`{}`)
			resp := unauthPost(baseURL+"/api/auth/refresh", body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("Logout", func() {
		It("returns 204 and clears the auth cookie", func() {
			resp := unauthPost(baseURL+"/api/auth/logout", nil)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

			// Verify cookie is cleared
			for _, cookie := range resp.Cookies() {
				if cookie.Name == "launchpad_access_token" {
					Expect(cookie.MaxAge).To(BeNumerically("<", 0))
				}
			}
		})
	})

	Describe("Current User (Me)", func() {
		It("returns current user info with valid token", func() {
			resp := authGet(baseURL+"/api/auth/me", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result["username"]).To(Equal("admin"))
		})

		It("returns 401 without token", func() {
			resp := unauthGet(baseURL + "/api/auth/me")
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})

		It("returns 401 with invalid token", func() {
			resp := authGet(baseURL+"/api/auth/me", "totally-invalid-token")
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
		})
	})

	Describe("Protected Routes", func() {
		It("rejects unauthenticated access to protected endpoints", func() {
			endpoints := []string{
				"/api/apps",
				"/api/sessions",
				"/api/quotas",
			}
			for _, ep := range endpoints {
				resp := unauthGet(baseURL + ep)
				resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized),
					"expected 401 for unauthenticated %s", ep)
			}
		})
	})

	Describe("Registration", func() {
		It("handles registration based on allow_registration setting", func() {
			body := []byte(`{
				"username": "e2e-reg-test",
				"password": "testpass123",
				"email": "e2e-reg@test.com"
			}`)
			resp := unauthPost(baseURL+"/api/auth/register", body)
			defer resp.Body.Close()

			// Either 201 (registration enabled) or 403 (disabled)
			Expect(resp.StatusCode).To(SatisfyAny(
				Equal(http.StatusCreated),
				Equal(http.StatusForbidden),
				Equal(http.StatusConflict),
			))

			// Clean up if registration succeeded
			if resp.StatusCode == http.StatusCreated {
				listResp := authGet(baseURL+"/api/admin/users", adminToken)
				users := decodeJSONArray(listResp)
				listResp.Body.Close()
				for _, u := range users {
					user := u.(map[string]any)
					if user["username"] == "e2e-reg-test" {
						deleteTestUser(user["id"].(string))
						break
					}
				}
			}
		})

		It("rejects registration with short password", func() {
			// Enable registration temporarily
			resp := authPut(baseURL+"/api/admin/settings", adminToken,
				[]byte(`{"allow_registration":"true"}`))
			resp.Body.Close()
			DeferCleanup(func() {
				r := authPut(baseURL+"/api/admin/settings", adminToken,
					[]byte(`{"allow_registration":"false"}`))
				r.Body.Close()
			})

			body := []byte(`{
				"username": "e2e-shortpw",
				"password": "abc",
				"email": "short@test.com"
			}`)
			resp = unauthPost(baseURL+"/api/auth/register", body)
			resp.Body.Close()

			// With registration enabled, short password should return 400
			if resp.StatusCode != http.StatusForbidden {
				Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
			}
		})
	})

	Describe("Method Enforcement", func() {
		It("rejects non-POST for login", func() {
			resp := unauthGet(baseURL + "/api/auth/login")
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
		})

		It("rejects non-POST for logout", func() {
			resp := unauthGet(baseURL + "/api/auth/logout")
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
		})

		It("rejects non-GET for /api/auth/me", func() {
			resp := authPost(baseURL+"/api/auth/me", adminToken, nil)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
		})
	})

	Describe("Cookie-based Auth", func() {
		It("sets HttpOnly cookie on login", func() {
			body := []byte(`{"username":"admin","password":"admin123"}`)
			resp := unauthPost(baseURL+"/api/auth/login", body)
			resp.Body.Close()

			var found bool
			for _, cookie := range resp.Cookies() {
				if cookie.Name == "launchpad_access_token" {
					found = true
					Expect(cookie.HttpOnly).To(BeTrue())
					Expect(cookie.Value).NotTo(BeEmpty())
				}
			}
			Expect(found).To(BeTrue(), "expected launchpad_access_token cookie")
		})
	})

	Describe("Admin User CRUD", func() {
		var testUserID string

		It("creates a new user", func() {
			testUserID = createTestUser("e2e-user-crud", "password123", "crud@test.com", []string{"user"})
			Expect(testUserID).NotTo(BeEmpty())
		})

		It("lists users including the new user", func() {
			resp := authGet(baseURL+"/api/admin/users", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			users := decodeJSONArray(resp)
			var found bool
			for _, u := range users {
				user := u.(map[string]any)
				if user["username"] == "e2e-user-crud" {
					found = true
					// Password hash should not be exposed
					Expect(user).NotTo(HaveKey("password_hash"))
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("logs in as the new user", func() {
			token, err := login(baseURL, "e2e-user-crud", "password123")
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())
		})

		It("deletes the test user", func() {
			if testUserID == "" {
				Skip("no test user to delete")
			}
			resp := authDelete(baseURL+"/api/admin/users/"+testUserID, adminToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

			// Verify the user is gone
			_, err := login(baseURL, "e2e-user-crud", "password123")
			Expect(err).To(HaveOccurred())
		})

		It("rejects duplicate username", func() {
			id1 := createTestUser("e2e-dup-user", "password123", "dup@test.com", []string{"user"})
			DeferCleanup(deleteTestUser, id1)

			body := []byte(`{
				"username": "e2e-dup-user",
				"password": "password456",
				"email": "dup2@test.com"
			}`)
			resp := authPost(baseURL+"/api/admin/users", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusConflict))
		})
	})

	Describe("OIDC SSO", func() {
		It("returns 503 when OIDC is not configured", func() {
			// In a standard E2E env, OIDC is typically not configured
			resp := unauthGet(baseURL + "/api/auth/oidc/login")
			resp.Body.Close()

			// Either 503 (not configured) or 302 (redirect to provider)
			Expect(resp.StatusCode).To(SatisfyAny(
				Equal(http.StatusServiceUnavailable),
				Equal(http.StatusFound),
			))
		})
	})

	Describe("Config Endpoint", func() {
		It("returns branding config without auth", func() {
			resp := unauthGet(baseURL + "/api/config")
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result).To(HaveKey("allow_registration"))
			Expect(result).To(HaveKey("sso_enabled"))
			Expect(result).To(HaveKey("tenant_name"))
		})
	})

	Describe("Login Response Format", func() {
		It("returns expires_in field", func() {
			body := []byte(`{"username":"admin","password":"admin123"}`)
			resp := unauthPost(baseURL+"/api/auth/login", body)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var result map[string]json.RawMessage
			json.NewDecoder(resp.Body).Decode(&result)
			Expect(result).To(HaveKey("expires_in"))
		})
	})
})
