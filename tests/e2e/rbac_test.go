package e2e

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("E2E RBAC", func() {
	var (
		regularUserID    string
		regularUserToken string
		appAuthorID      string
		appAuthorToken   string
	)

	BeforeEach(func() {
		regularUserID = createTestUser("e2e-rbac-user", "password123", "rbac-user@test.com", []string{"user"})
		var err error
		regularUserToken, err = login(baseURL, "e2e-rbac-user", "password123")
		Expect(err).NotTo(HaveOccurred())

		appAuthorID = createTestUser("e2e-rbac-author", "password123", "rbac-author@test.com", []string{"app-author"})
		appAuthorToken, err = login(baseURL, "e2e-rbac-author", "password123")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		deleteTestUser(regularUserID)
		deleteTestUser(appAuthorID)
	})

	Describe("Regular User Permissions", func() {
		It("can list apps", func() {
			resp := authGet(baseURL+"/api/apps", regularUserToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("cannot create apps", func() {
			body := []byte(`{
				"id": "e2e-rbac-test-app",
				"name": "RBAC Test",
				"launch_type": "url",
				"url": "https://example.com"
			}`)
			resp := authPost(baseURL+"/api/apps", regularUserToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
		})

		It("cannot access admin routes", func() {
			adminEndpoints := []string{
				"/api/admin/settings",
				"/api/admin/users",
				"/api/admin/sessions",
				"/api/admin/health",
				"/api/admin/tenants",
				"/api/admin/templates",
			}
			for _, ep := range adminEndpoints {
				resp := authGet(baseURL+ep, regularUserToken)
				resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusForbidden),
					"expected 403 for regular user on %s", ep)
			}
		})

		It("can list sessions", func() {
			resp := authGet(baseURL+"/api/sessions", regularUserToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("can check quotas", func() {
			resp := authGet(baseURL+"/api/quotas", regularUserToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	Describe("App Author Permissions", func() {
		It("can create apps", func() {
			body := []byte(`{
				"id": "e2e-author-app",
				"name": "Author Test App",
				"launch_type": "url",
				"url": "https://example.com"
			}`)
			resp := authPost(baseURL+"/api/apps", appAuthorToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(SatisfyAny(
				Equal(http.StatusCreated),
				Equal(http.StatusConflict),
			))
			DeferCleanup(deleteE2EApp, "e2e-author-app")
		})

		It("can update apps", func() {
			// Create first
			createBody := []byte(`{
				"id": "e2e-author-update",
				"name": "Author Update Test",
				"launch_type": "url",
				"url": "https://example.com"
			}`)
			resp := authPost(baseURL+"/api/apps", appAuthorToken, createBody)
			resp.Body.Close()
			DeferCleanup(deleteE2EApp, "e2e-author-update")

			// Update
			updateBody := []byte(`{
				"name": "Updated by Author",
				"launch_type": "url",
				"url": "https://example.com"
			}`)
			resp = authPut(baseURL+"/api/apps/e2e-author-update", appAuthorToken, updateBody)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("can delete apps", func() {
			createBody := []byte(`{
				"id": "e2e-author-delete",
				"name": "Author Delete Test",
				"launch_type": "url",
				"url": "https://example.com"
			}`)
			resp := authPost(baseURL+"/api/apps", appAuthorToken, createBody)
			resp.Body.Close()

			resp = authDelete(baseURL+"/api/apps/e2e-author-delete", appAuthorToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
		})

		It("cannot access admin routes", func() {
			resp := authGet(baseURL+"/api/admin/settings", appAuthorToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
		})
	})

	Describe("Admin Permissions", func() {
		It("can access all admin routes", func() {
			adminEndpoints := []string{
				"/api/admin/settings",
				"/api/admin/users",
				"/api/admin/sessions",
				"/api/admin/health",
				"/api/admin/tenants",
				"/api/admin/templates",
			}
			for _, ep := range adminEndpoints {
				resp := authGet(baseURL+ep, adminToken)
				resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusOK),
					"expected 200 for admin on %s", ep)
			}
		})

		It("can create and manage apps", func() {
			body := []byte(`{
				"id": "e2e-admin-managed",
				"name": "Admin Managed App",
				"launch_type": "url",
				"url": "https://example.com"
			}`)
			resp := authPost(baseURL+"/api/apps", adminToken, body)
			resp.Body.Close()
			DeferCleanup(deleteE2EApp, "e2e-admin-managed")

			Expect(resp.StatusCode).To(SatisfyAny(
				Equal(http.StatusCreated),
				Equal(http.StatusConflict),
			))
		})
	})
})
