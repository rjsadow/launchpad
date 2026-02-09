package e2e

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("E2E Tenants", func() {

	Describe("Tenant List", func() {
		It("lists tenants including the default tenant", func() {
			resp := authGet(baseURL+"/api/admin/tenants", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			tenants := decodeJSONArray(resp)
			Expect(len(tenants)).To(BeNumerically(">=", 1))

			var defaultFound bool
			for _, t := range tenants {
				tenant := t.(map[string]any)
				if tenant["slug"] == "default" {
					defaultFound = true
					break
				}
			}
			Expect(defaultFound).To(BeTrue(), "default tenant should exist")
		})
	})

	Describe("Tenant CRUD", func() {
		var tenantID string

		It("creates a new tenant", func() {
			body := []byte(`{
				"name": "E2E Test Tenant",
				"slug": "e2e-test-tenant"
			}`)
			resp := authPost(baseURL+"/api/admin/tenants", adminToken, body)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(SatisfyAny(
				Equal(http.StatusCreated),
				Equal(http.StatusConflict),
			))

			if resp.StatusCode == http.StatusCreated {
				result := decodeJSON(resp)
				tenantID = result["id"].(string)
			} else {
				// Find the existing tenant
				listResp := authGet(baseURL+"/api/admin/tenants", adminToken)
				tenants := decodeJSONArray(listResp)
				listResp.Body.Close()
				for _, t := range tenants {
					tenant := t.(map[string]any)
					if tenant["slug"] == "e2e-test-tenant" {
						tenantID = tenant["id"].(string)
						break
					}
				}
			}
			Expect(tenantID).NotTo(BeEmpty())
		})

		It("reads the tenant back", func() {
			if tenantID == "" {
				Skip("no tenant ID")
			}
			resp := authGet(baseURL+"/api/admin/tenants/"+tenantID, adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result["name"]).To(Equal("E2E Test Tenant"))
		})

		It("updates the tenant", func() {
			if tenantID == "" {
				Skip("no tenant ID")
			}
			body := []byte(`{
				"name": "E2E Updated Tenant",
				"slug": "e2e-test-tenant"
			}`)
			resp := authPut(baseURL+"/api/admin/tenants/"+tenantID, adminToken, body)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result["name"]).To(Equal("E2E Updated Tenant"))
		})

		It("deletes the tenant", func() {
			if tenantID == "" {
				Skip("no tenant ID")
			}
			resp := authDelete(baseURL+"/api/admin/tenants/"+tenantID, adminToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

			// Verify it's gone
			resp = authGet(baseURL+"/api/admin/tenants/"+tenantID, adminToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

	Describe("Default Tenant Protection", func() {
		It("prevents deletion of the default tenant", func() {
			// Find the default tenant ID
			resp := authGet(baseURL+"/api/admin/tenants", adminToken)
			tenants := decodeJSONArray(resp)
			resp.Body.Close()

			var defaultTenantID string
			for _, t := range tenants {
				tenant := t.(map[string]any)
				if tenant["slug"] == "default" {
					defaultTenantID = tenant["id"].(string)
					break
				}
			}

			Expect(defaultTenantID).NotTo(BeEmpty(), "default tenant should exist")

			resp = authDelete(baseURL+"/api/admin/tenants/"+defaultTenantID, adminToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
		})
	})

	Describe("Duplicate Slug", func() {
		It("rejects tenant creation with duplicate slug", func() {
			body := []byte(`{
				"name": "Dup Tenant 1",
				"slug": "e2e-dup-slug"
			}`)
			resp := authPost(baseURL+"/api/admin/tenants", adminToken, body)
			resp.Body.Close()

			// Create again with same slug
			body = []byte(`{
				"name": "Dup Tenant 2",
				"slug": "e2e-dup-slug"
			}`)
			resp = authPost(baseURL+"/api/admin/tenants", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusConflict))

			// Clean up
			listResp := authGet(baseURL+"/api/admin/tenants", adminToken)
			tenants := decodeJSONArray(listResp)
			listResp.Body.Close()
			for _, t := range tenants {
				tenant := t.(map[string]any)
				if tenant["slug"] == "e2e-dup-slug" {
					delResp := authDelete(baseURL+"/api/admin/tenants/"+tenant["id"].(string), adminToken)
					delResp.Body.Close()
					break
				}
			}
		})
	})

	Describe("Invalid Tenant Header", func() {
		It("returns 404 for requests with invalid X-Tenant-ID", func() {
			req, err := http.NewRequest(http.MethodGet, baseURL+"/api/apps", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Authorization", "Bearer "+adminToken)
			req.Header.Set("X-Tenant-ID", "nonexistent-tenant")

			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})
})
