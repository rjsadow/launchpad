package e2e

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("E2E Health and Load", func() {

	Describe("Health Endpoints", func() {
		It("returns 200 for /healthz with status ok", func() {
			resp := unauthGet(baseURL + "/healthz")
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result["status"]).To(Equal("ok"))
		})

		It("returns 200 for /readyz with database health", func() {
			resp := unauthGet(baseURL + "/readyz")
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result["status"]).To(Equal("ready"))
			Expect(result).To(HaveKey("database"))
			Expect(result).To(HaveKey("sessions"))

			db := result["database"].(map[string]any)
			Expect(db["status"]).To(Equal("healthy"))
		})

		It("does not require auth for health endpoints", func() {
			endpoints := []string{"/healthz", "/readyz", "/api/load"}
			for _, ep := range endpoints {
				resp := unauthGet(baseURL + ep)
				resp.Body.Close()
				Expect(resp.StatusCode).To(Equal(http.StatusOK),
					"expected 200 for unauthenticated %s", ep)
			}
		})

		It("rejects non-GET requests", func() {
			resp := unauthPost(baseURL+"/healthz", nil)
			resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
		})
	})

	Describe("Load Status", func() {
		It("returns load status with expected fields", func() {
			resp := unauthGet(baseURL + "/api/load")
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			result := decodeJSON(resp)
			Expect(result).To(HaveKey("active_sessions"))
			Expect(result).To(HaveKey("max_sessions"))
			Expect(result).To(HaveKey("load_factor"))
			Expect(result).To(HaveKey("queue_depth"))

			loadFactor, ok := result["load_factor"].(float64)
			if ok {
				Expect(loadFactor).To(BeNumerically(">=", 0))
				Expect(loadFactor).To(BeNumerically("<=", 1))
			}
		})
	})
})
