package e2e

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("E2E AppSpecs", func() {

	Describe("AppSpec CRUD", func() {
		specID := "e2e-spec"

		It("creates an app spec", func() {
			body := []byte(`{
				"id": "e2e-spec",
				"name": "E2E Test Spec",
				"image": "nginx:latest"
			}`)
			resp := authPost(baseURL+"/api/appspecs", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(SatisfyAny(
				Equal(http.StatusCreated),
				Equal(http.StatusConflict),
			))
		})

		It("lists app specs", func() {
			resp := authGet(baseURL+"/api/appspecs", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			specs := decodeJSONArray(resp)
			Expect(specs).NotTo(BeNil())
		})

		It("gets an app spec by ID", func() {
			resp := authGet(baseURL+"/api/appspecs/"+specID, adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result["name"]).To(Equal("E2E Test Spec"))
		})

		It("updates an app spec", func() {
			body := []byte(`{
				"name": "E2E Updated Spec",
				"image": "nginx:alpine"
			}`)
			resp := authPut(baseURL+"/api/appspecs/"+specID, adminToken, body)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result["name"]).To(Equal("E2E Updated Spec"))
		})

		It("deletes an app spec", func() {
			resp := authDelete(baseURL+"/api/appspecs/"+specID, adminToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

			// Verify gone
			resp = authGet(baseURL+"/api/appspecs/"+specID, adminToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

	Describe("AppSpec Validation", func() {
		It("rejects creation with missing required fields", func() {
			body := []byte(`{"name": "Missing Image"}`)
			resp := authPost(baseURL+"/api/appspecs", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("rejects duplicate app spec ID", func() {
			body := []byte(`{
				"id": "e2e-dup-spec",
				"name": "Dup Spec 1",
				"image": "nginx:latest"
			}`)
			resp := authPost(baseURL+"/api/appspecs", adminToken, body)
			resp.Body.Close()

			resp = authPost(baseURL+"/api/appspecs", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusConflict))

			// Clean up
			resp = authDelete(baseURL+"/api/appspecs/e2e-dup-spec", adminToken)
			resp.Body.Close()
		})

		It("returns 404 for non-existent spec", func() {
			resp := authGet(baseURL+"/api/appspecs/nonexistent-spec", adminToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

	Describe("AppSpec RBAC", func() {
		It("rejects creation by regular user", func() {
			userID := createTestUser("e2e-spec-user", "password123", "spec@test.com", []string{"user"})
			DeferCleanup(deleteTestUser, userID)

			userToken, err := login(baseURL, "e2e-spec-user", "password123")
			Expect(err).NotTo(HaveOccurred())

			body := []byte(`{
				"id": "e2e-user-spec",
				"name": "User Spec",
				"image": "nginx:latest"
			}`)
			resp := authPost(baseURL+"/api/appspecs", userToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
		})

		It("allows creation by app-author", func() {
			authorID := createTestUser("e2e-spec-author", "password123", "spec-author@test.com", []string{"app-author"})
			DeferCleanup(deleteTestUser, authorID)

			authorToken, err := login(baseURL, "e2e-spec-author", "password123")
			Expect(err).NotTo(HaveOccurred())

			body := []byte(`{
				"id": "e2e-author-spec",
				"name": "Author Spec",
				"image": "nginx:latest"
			}`)
			resp := authPost(baseURL+"/api/appspecs", authorToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(SatisfyAny(
				Equal(http.StatusCreated),
				Equal(http.StatusConflict),
			))

			// Clean up
			resp = authDelete(baseURL+"/api/appspecs/e2e-author-spec", adminToken)
			resp.Body.Close()
		})
	})
})
