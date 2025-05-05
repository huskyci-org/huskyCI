package routes_test

import (
	apiContext "github.com/huskyci-org/huskyCI/api/context"
	"github.com/ghuskyci-org/huskyCI/api/routes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("getRequestResult", func() {

	expected := map[string]string{
		"version": apiContext.DefaultConf.GetAPIVersion(),
		"date":    apiContext.DefaultConf.GetAPIReleaseDate(),
	}

	apiContext.DefaultConf.SetOnceConfig()
	config := apiContext.APIConfiguration

	Context("When version and date are requested", func() {
		It("Should return a map with API version and date", func() {
			Expect(routes.GetRequestResult(config)).To(Equal(expected))
		})
	})

})
