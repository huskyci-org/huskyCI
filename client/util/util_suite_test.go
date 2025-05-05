package util_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUtil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Util Suite")
}

var _ = AfterSuite(func() {
	cleanTestOutputFiles()
})

const testOutputFilesPath = "./huskyCITest/"

func cleanTestOutputFiles() {
	os.RemoveAll(testOutputFilesPath)
}
