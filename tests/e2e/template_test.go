package e2e

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("E2E Templates", func() {

	Describe("Public Template List", func() {
		It("lists templates without auth", func() {
			resp := unauthGet(baseURL + "/api/templates")
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSONArray(resp)
			Expect(result).NotTo(BeNil())
		})
	})

	Describe("Admin Template CRUD", func() {
		templateID := "e2e-template"

		It("creates a template", func() {
			body := []byte(`{
				"template_id": "e2e-template",
				"name": "E2E Test Template",
				"category": "testing",
				"template_category": "development",
				"image": "nginx:latest",
				"launch_type": "container"
			}`)
			resp := authPost(baseURL+"/api/admin/templates", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(SatisfyAny(
				Equal(http.StatusCreated),
				Equal(http.StatusConflict),
			))
		})

		It("reads the template back", func() {
			resp := authGet(baseURL+"/api/admin/templates/"+templateID, adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result["name"]).To(Equal("E2E Test Template"))
		})

		It("reads the template via public endpoint", func() {
			resp := unauthGet(baseURL + "/api/templates/" + templateID)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result["template_id"]).To(Equal(templateID))
		})

		It("updates the template", func() {
			body := []byte(`{
				"name": "E2E Updated Template",
				"category": "testing",
				"template_category": "development",
				"image": "nginx:alpine",
				"launch_type": "container"
			}`)
			resp := authPut(baseURL+"/api/admin/templates/"+templateID, adminToken, body)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result["name"]).To(Equal("E2E Updated Template"))
		})

		It("deletes the template", func() {
			resp := authDelete(baseURL+"/api/admin/templates/"+templateID, adminToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

			// Verify it's gone
			resp = authGet(baseURL+"/api/admin/templates/"+templateID, adminToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("requires all fields for creation", func() {
			body := []byte(`{"name": "Missing Fields"}`)
			resp := authPost(baseURL+"/api/admin/templates", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("returns 404 for non-existent template", func() {
			resp := authGet(baseURL+"/api/admin/templates/nonexistent", adminToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

	Describe("Non-admin Template Access", func() {
		var userToken string

		BeforeEach(func() {
			userID := createTestUser("e2e-tmpl-user", "password123", "tmpl@test.com", []string{"user"})
			DeferCleanup(deleteTestUser, userID)

			var err error
			userToken, err = login(baseURL, "e2e-tmpl-user", "password123")
			Expect(err).NotTo(HaveOccurred())
		})

		It("rejects template creation by non-admin", func() {
			body := []byte(`{
				"template_id": "e2e-forbidden-tmpl",
				"name": "Forbidden Template",
				"category": "testing",
				"template_category": "development",
				"image": "nginx:latest"
			}`)
			resp := authPost(baseURL+"/api/admin/templates", userToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
		})
	})
})
