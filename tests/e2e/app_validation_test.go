package e2e

import (
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("E2E App Validation", func() {

	Describe("App Creation Validation", func() {
		It("rejects app with missing name", func() {
			body := []byte(`{
				"id": "e2e-no-name",
				"launch_type": "url",
				"url": "https://example.com"
			}`)
			resp := authPost(baseURL+"/api/apps", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("rejects app with missing ID", func() {
			body := []byte(`{
				"name": "Missing ID App",
				"launch_type": "url",
				"url": "https://example.com"
			}`)
			resp := authPost(baseURL+"/api/apps", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("rejects URL app without url field", func() {
			body := []byte(`{
				"id": "e2e-no-url",
				"name": "No URL App",
				"launch_type": "url"
			}`)
			resp := authPost(baseURL+"/api/apps", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("rejects container app without container_image", func() {
			body := []byte(`{
				"id": "e2e-no-image",
				"name": "No Image App",
				"launch_type": "container"
			}`)
			resp := authPost(baseURL+"/api/apps", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("rejects web_proxy app without container_image", func() {
			body := []byte(`{
				"id": "e2e-no-proxy-image",
				"name": "No Proxy Image App",
				"launch_type": "web_proxy"
			}`)
			resp := authPost(baseURL+"/api/apps", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("rejects duplicate app ID", func() {
			body := []byte(`{
				"id": "e2e-dup-app",
				"name": "Dup App 1",
				"launch_type": "url",
				"url": "https://example.com"
			}`)
			resp := authPost(baseURL+"/api/apps", adminToken, body)
			resp.Body.Close()
			DeferCleanup(deleteE2EApp, "e2e-dup-app")

			resp = authPost(baseURL+"/api/apps", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusConflict))
		})

		It("rejects invalid JSON", func() {
			resp := authPost(baseURL+"/api/apps", adminToken, []byte(`not json`))
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("App Read Operations", func() {
		It("returns 404 for non-existent app", func() {
			resp := authGet(baseURL+"/api/apps/nonexistent-app-id", adminToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("lists all apps", func() {
			resp := authGet(baseURL+"/api/apps", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSONArray(resp)
			Expect(result).NotTo(BeNil())
		})
	})

	Describe("App Update Validation", func() {
		It("rejects update with missing name", func() {
			// Create the app first
			body := []byte(`{
				"id": "e2e-update-val",
				"name": "Update Val App",
				"launch_type": "url",
				"url": "https://example.com"
			}`)
			resp := authPost(baseURL+"/api/apps", adminToken, body)
			resp.Body.Close()
			DeferCleanup(deleteE2EApp, "e2e-update-val")

			// Update with missing name
			updateBody := []byte(`{"launch_type": "url", "url": "https://example.com"}`)
			resp = authPut(baseURL+"/api/apps/e2e-update-val", adminToken, updateBody)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("Container App Types", func() {
		It("creates a container app with image", func() {
			body := []byte(`{
				"id": "e2e-container-type",
				"name": "Container Type App",
				"launch_type": "container",
				"container_image": "nginx:latest"
			}`)
			resp := authPost(baseURL+"/api/apps", adminToken, body)
			defer resp.Body.Close()
			DeferCleanup(deleteE2EApp, "e2e-container-type")

			Expect(resp.StatusCode).To(SatisfyAny(
				Equal(http.StatusCreated),
				Equal(http.StatusConflict),
			))

			if resp.StatusCode == http.StatusCreated {
				result := decodeJSON(resp)
				Expect(result["launch_type"]).To(Equal("container"))
				Expect(result["container_image"]).To(Equal("nginx:latest"))
			}
		})

		It("creates a web_proxy app", func() {
			body := []byte(`{
				"id": "e2e-webproxy-type",
				"name": "WebProxy Type App",
				"launch_type": "web_proxy",
				"container_image": "nginx:latest"
			}`)
			resp := authPost(baseURL+"/api/apps", adminToken, body)
			defer resp.Body.Close()
			DeferCleanup(deleteE2EApp, "e2e-webproxy-type")

			Expect(resp.StatusCode).To(SatisfyAny(
				Equal(http.StatusCreated),
				Equal(http.StatusConflict),
			))

			if resp.StatusCode == http.StatusCreated {
				result := decodeJSON(resp)
				Expect(result["launch_type"]).To(Equal("web_proxy"))
			}
		})
	})

	Describe("Session Validation", func() {
		It("rejects session creation with missing app_id", func() {
			body := []byte(`{"user_id":"e2e-user"}`)
			resp := authPost(baseURL+"/api/sessions", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("rejects session creation for non-existent app", func() {
			body := []byte(`{"app_id":"nonexistent-app","user_id":"e2e-user"}`)
			resp := authPost(baseURL+"/api/sessions", adminToken, body)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("returns 404 for non-existent session", func() {
			resp := authGet(baseURL+"/api/sessions/nonexistent-session-id", adminToken)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("lists sessions with filter", func() {
			resp := authGet(baseURL+"/api/sessions?user_id=e2e-user", adminToken)
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSONArray(resp)
			Expect(result).NotTo(BeNil())
		})
	})

	Describe("Session State Transitions", func() {
		It("rejects restart on non-existent session", func() {
			resp := authPost(baseURL+"/api/sessions/nonexistent/restart", adminToken, nil)
			resp.Body.Close()
			Expect(resp.StatusCode).To(SatisfyAny(
				Equal(http.StatusNotFound),
				Equal(http.StatusBadRequest),
			))
		})

		It("rejects stop on non-existent session", func() {
			resp := authPost(baseURL+"/api/sessions/nonexistent/stop", adminToken, nil)
			resp.Body.Close()
			Expect(resp.StatusCode).To(SatisfyAny(
				Equal(http.StatusNotFound),
				Equal(http.StatusBadRequest),
			))
		})
	})

	Describe("Method Enforcement", func() {
		It("rejects non-GET/POST for /api/apps", func() {
			req, err := http.NewRequest(http.MethodPatch, baseURL+"/api/apps", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Authorization", "Bearer "+adminToken)
			resp, err := http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
		})
	})

	Describe("Legacy apps.json", func() {
		It("returns apps list in legacy format", func() {
			resp := unauthGet(baseURL + "/apps.json")
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var result map[string]json.RawMessage
			json.NewDecoder(resp.Body).Decode(&result)
			Expect(result).To(HaveKey("applications"))
		})
	})
})
