package util_test

import (
	"fmt"
	"os"

	"github.com/huskyci-org/huskyCI/client/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util", func() {
	Describe("CreateFile", func() {
		outputFileName := "sonarqube_test.json"
		outputFilePath := testOutputFilesPath + outputFileName
		testDataPath := "./testdata/sonarqube/sonarqube_test_example.json"
		fileString, err := os.ReadFile(testDataPath)
		if err != nil {
			Fail(fmt.Sprintf("error trying to read fixture file: %s", err.Error()))
		}
		bytesInput := []byte(fileString)
		err = util.CreateFile(bytesInput, testOutputFilesPath, outputFileName)
		if err != nil {
			Fail(fmt.Sprintf("eror trying to execute util.CreateFile: %s", err.Error()))
		}
		It("should not return error", func() {
			Expect(err).NotTo(HaveOccurred())
		})
		It("Should create a directory and file", func() {
			_, err := os.Stat(outputFilePath)
			Expect(!os.IsNotExist(err)).To(Equal(true))
		})
		It("File content should match the input string", func() {
			outputString, err := os.ReadFile(outputFilePath)
			if err != nil {
				Fail(fmt.Sprintf("error trying to read test output file: %s", err.Error()))
			}
			Expect(outputString).To(Equal(fileString))
		})
	})
})
