package sonarqube_test

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/huskyci-org/huskyCI/client/integration/sonarqube"
	"github.com/huskyci-org/huskyCI/client/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SonarQube Security Test Compatibility", func() {
	var outputPath string
	var outputFileName string

	BeforeEach(func() {
		outputPath = "./huskyCITest/"
		outputFileName = "sonarqube_test.json"
		os.MkdirAll(outputPath, os.ModePerm)
	})

	AfterEach(func() {
		os.RemoveAll(outputPath)
	})

	// Helper function to validate SonarQube output structure
	validateSonarQubeOutput := func(outputPath, outputFileName string) {
		testOutputFilePath := filepath.Join(outputPath, outputFileName)
		fileContent, err := os.ReadFile(testOutputFilePath)
		Expect(err).NotTo(HaveOccurred())

		var sonarOutput sonarqube.HuskyCISonarOutput
		err = json.Unmarshal(fileContent, &sonarOutput)
		Expect(err).NotTo(HaveOccurred())

		// Validate structure
		Expect(sonarOutput.Rules).NotTo(BeNil())
		Expect(sonarOutput.Issues).NotTo(BeNil())

		// Validate rules
		for _, rule := range sonarOutput.Rules {
			Expect(rule.ID).NotTo(BeEmpty(), "Rule ID should not be empty")
			Expect(rule.Name).NotTo(BeEmpty(), "Rule Name should not be empty")
			Expect(rule.EngineID).NotTo(BeEmpty(), "EngineID should not be empty")
			Expect(rule.EngineID).To(HavePrefix("huskyCI/"), "EngineID should start with 'huskyCI/'")
			Expect(rule.Type).To(Equal("VULNERABILITY"), "Rule Type should be VULNERABILITY")
			Expect(rule.CleanCodeAttribute).To(Equal("TRUSTWORTHY"), "CleanCodeAttribute should be TRUSTWORTHY")
			Expect(rule.Severity).To(BeElementOf("MINOR", "MAJOR", "BLOCKER", "INFO"), "Rule Severity should be valid")
			Expect(rule.Impacts).NotTo(BeEmpty(), "Rule should have impacts")
			for _, impact := range rule.Impacts {
				Expect(impact.SoftwareQuality).To(Equal("SECURITY"), "Impact SoftwareQuality should be SECURITY")
				Expect(impact.Severity).To(BeElementOf("LOW", "MEDIUM", "HIGH", "INFO"), "Impact Severity should be valid")
			}
		}

		// Validate issues
		for _, issue := range sonarOutput.Issues {
			Expect(issue.RuleID).NotTo(BeEmpty(), "Issue RuleID should not be empty")
			Expect(issue.PrimaryLocation.FilePath).NotTo(BeEmpty(), "Issue FilePath should not be empty")
			Expect(issue.PrimaryLocation.TextRange.StartLine).To(BeNumerically(">=", 1), "StartLine should be >= 1")
		}

		// Validate rule-issue consistency
		ruleIDs := make(map[string]bool)
		for _, rule := range sonarOutput.Rules {
			ruleIDs[rule.ID] = true
		}
		for _, issue := range sonarOutput.Issues {
			Expect(ruleIDs[issue.RuleID]).To(BeTrue(), "Issue RuleID should match a rule ID")
		}
	}

	Describe("Gosec (Go) Security Test", func() {
		It("should produce valid SonarQube output", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					GoResults: types.GoResults{
						HuskyCIGosecOutput: types.HuskyCISecurityTestOutput{
							LowVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Go",
									SecurityTool: "GoSec",
									Severity:     "LOW",
									Title:        "G104: Audit the use of unsafe block",
									Details:       "Unsafe block should be audited",
									File:          "/go/src/code/main.go",
									Line:          "42",
									Code:          "unsafe.Pointer(...)",
								},
							},
							MediumVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Go",
									SecurityTool: "GoSec",
									Severity:     "MEDIUM",
									Title:        "G101: Potential hardcoded credentials",
									Details:       "Potential hardcoded credentials found",
									File:          "/go/src/code/auth.go",
									Line:          "15",
									Code:          "password := \"secret\"",
								},
							},
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Go",
									SecurityTool: "GoSec",
									Severity:     "HIGH",
									Title:        "G107: Potential HTTP request made with variable url",
									Details:       "HTTP request made with variable url",
									File:          "/go/src/code/http.go",
									Line:          "30",
									Code:          "http.Get(url)",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())
			validateSonarQubeOutput(outputPath, outputFileName)
		})
	})

	Describe("Bandit (Python) Security Test", func() {
		It("should produce valid SonarQube output", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					PythonResults: types.PythonResults{
						HuskyCIBanditOutput: types.HuskyCISecurityTestOutput{
							LowVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Python",
									SecurityTool: "Bandit",
									Severity:     "LOW",
									Title:        "B101: Test for use of assert_used",
									Details:       "Use of assert detected",
									File:          "test.py",
									Line:          "10",
									Code:          "assert True",
								},
							},
							MediumVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Python",
									SecurityTool: "Bandit",
									Severity:     "MEDIUM",
									Title:        "B506: Test for use of yaml.load",
									Details:       "Use of yaml.load detected",
									File:          "config.py",
									Line:          "25",
									Code:          "yaml.load(data)",
								},
							},
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Python",
									SecurityTool: "Bandit",
									Severity:     "HIGH",
									Title:        "B104: Test for hardcoded password",
									Details:       "Hardcoded password detected",
									File:          "auth.py",
									Line:          "5",
									Code:          "password = 'secret123'",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())
			validateSonarQubeOutput(outputPath, outputFileName)
		})
	})

	Describe("Safety (Python) Security Test", func() {
		It("should produce valid SonarQube output", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					PythonResults: types.PythonResults{
						HuskyCISafetyOutput: types.HuskyCISecurityTestOutput{
							LowVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Python",
									SecurityTool: "Safety",
									Severity:     "low",
									Title:        "No requirements.txt found.",
									Details:       "It looks like your project doesn't have a requirements.txt file.",
								},
							},
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Python",
									SecurityTool: "Safety",
									Severity:     "high",
									Title:        "Vulnerable Dependency: django (<2.0.0)",
									Details:       "Django before 2.0.0 has security vulnerabilities",
									Version:      "1.11.0",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())
			validateSonarQubeOutput(outputPath, outputFileName)
		})
	})

	Describe("Brakeman (Ruby) Security Test", func() {
		It("should produce valid SonarQube output", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					RubyResults: types.RubyResults{
						HuskyCIBrakemanOutput: types.HuskyCISecurityTestOutput{
							LowVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Ruby",
									SecurityTool: "Brakeman",
									Severity:     "Low",
									Title:        "Vulnerable Dependency: SQL Injection SQL",
									Details:       "Possible SQL injection",
									File:          "app/models/user.rb",
									Line:          "20",
									Code:          "User.where(params[:query])",
								},
							},
							MediumVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Ruby",
									SecurityTool: "Brakeman",
									Severity:     "Medium",
									Title:        "Vulnerable Dependency: Cross Site Scripting XSS",
									Details:       "Unescaped user input",
									File:          "app/views/show.html.erb",
									Line:          "15",
									Code:          "<%= params[:name] %>",
								},
							},
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Ruby",
									SecurityTool: "Brakeman",
									Severity:     "High",
									Title:        "Vulnerable Dependency: Mass Assignment",
									Details:       "Mass assignment vulnerability",
									File:          "app/controllers/users_controller.rb",
									Line:          "10",
									Code:          "User.create(params[:user])",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())
			validateSonarQubeOutput(outputPath, outputFileName)
		})
	})

	Describe("NpmAudit (JavaScript) Security Test", func() {
		It("should produce valid SonarQube output", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					JavaScriptResults: types.JavaScriptResults{
						HuskyCINpmAuditOutput: types.HuskyCISecurityTestOutput{
							LowVulns: []types.HuskyCIVulnerability{
								{
									Language:     "JavaScript",
									SecurityTool: "NpmAudit",
									Severity:     "low",
									Title:        "Vulnerable Dependency: lodash (<4.17.0) (Prototype Pollution)",
									Details:       "Fix available: lodash 4.17.0",
									Version:       "Advisories and information (Via 0):\n\tSource: 1\n\tName: lodash\n",
								},
							},
							MediumVulns: []types.HuskyCIVulnerability{
								{
									Language:     "JavaScript",
									SecurityTool: "NpmAudit",
									Severity:     "medium",
									Title:        "Vulnerable Dependency: express (<4.17.0) (Path Traversal)",
									Details:       "Fix available: express 4.17.0",
									Version:       "Advisories and information (Via 0):\n\tSource: 1\n\tName: express\n",
								},
							},
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "JavaScript",
									SecurityTool: "NpmAudit",
									Severity:     "high",
									Title:        "Vulnerable Dependency: axios (<1.0.0) (Remote Code Execution)",
									Details:       "Fix available: axios 1.0.0",
									Version:       "Advisories and information (Via 0):\n\tSource: 1\n\tName: axios\n",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())
			validateSonarQubeOutput(outputPath, outputFileName)
		})
	})

	Describe("YarnAudit (JavaScript) Security Test", func() {
		It("should produce valid SonarQube output", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					JavaScriptResults: types.JavaScriptResults{
						HuskyCIYarnAuditOutput: types.HuskyCISecurityTestOutput{
							LowVulns: []types.HuskyCIVulnerability{
								{
									Language:     "JavaScript",
									SecurityTool: "YarnAudit",
									Severity:     "low",
									Title:        "Vulnerable Dependency: react (<16.8.0) (XSS)",
									Details:       "React XSS vulnerability",
									Version:       "16.7.0",
								},
							},
							MediumVulns: []types.HuskyCIVulnerability{
								{
									Language:     "JavaScript",
									SecurityTool: "YarnAudit",
									Severity:     "medium",
									Title:        "Vulnerable Dependency: webpack (<5.0.0) (Path Traversal)",
									Details:       "Webpack path traversal vulnerability",
									Version:       "4.46.0",
								},
							},
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "JavaScript",
									SecurityTool: "YarnAudit",
									Severity:     "high",
									Title:        "Vulnerable Dependency: node (<14.0.0) (Remote Code Execution)",
									Details:       "Node.js RCE vulnerability",
									Version:       "13.14.0",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())
			validateSonarQubeOutput(outputPath, outputFileName)
		})
	})

	Describe("SpotBugs (Java) Security Test", func() {
		It("should produce valid SonarQube output", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					JavaResults: types.JavaResults{
						HuskyCISpotBugsOutput: types.HuskyCISecurityTestOutput{
							LowVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Java",
									SecurityTool: "SpotBugs",
									Severity:     "LOW",
									Title:        "SQL_INJECTION_JDBC",
									Details:       "SQL_INJECTION_JDBC",
									File:          "src/main/java/UserDao.java",
									Line:          "50",
									Code:          "Code beetween Line 50 and Line 52.",
								},
							},
							MediumVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Java",
									SecurityTool: "SpotBugs",
									Severity:     "MEDIUM",
									Title:        "XSS_REQUEST_PARAMETER_TO_SEND_ERROR",
									Details:       "XSS_REQUEST_PARAMETER_TO_SEND_ERROR",
									File:          "src/main/java/ErrorHandler.java",
									Line:          "30",
									Code:          "Code beetween Line 30 and Line 32.",
								},
							},
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Java",
									SecurityTool: "SpotBugs",
									Severity:     "HIGH",
									Title:        "COMMAND_INJECTION",
									Details:       "COMMAND_INJECTION",
									File:          "src/main/java/CommandExecutor.java",
									Line:          "15",
									Code:          "Code beetween Line 15 and Line 17.",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())
			validateSonarQubeOutput(outputPath, outputFileName)
		})
	})

	Describe("SecurityCodeScan (C#) Security Test", func() {
		It("should produce valid SonarQube output", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					CSharpResults: types.CSharpResults{
						HuskyCISecurityCodeScanOutput: types.HuskyCISecurityTestOutput{
							LowVulns: []types.HuskyCIVulnerability{
								{
									Language:     "C#",
									SecurityTool: "Security Code Scan",
									Severity:     "Low",
									Title:        "SCS0005",
									Details:       "Weak random number generator",
									File:          "code/Utils.cs",
									Line:          "25",
									Code:          "Code beetween Line 25 and Line 27.",
								},
							},
							MediumVulns: []types.HuskyCIVulnerability{
								{
									Language:     "C#",
									SecurityTool: "Security Code Scan",
									Severity:     "Medium",
									Title:        "SCS0018",
									Details:       "Potential SQL injection",
									File:          "code/Database.cs",
									Line:          "40",
									Code:          "Code beetween Line 40 and Line 42.",
								},
							},
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "C#",
									SecurityTool: "Security Code Scan",
									Severity:     "High",
									Title:        "SCS0001",
									Details:       "Hardcoded password",
									File:          "code/Auth.cs",
									Line:          "10",
									Code:          "Code beetween Line 10 and Line 12.",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())
			validateSonarQubeOutput(outputPath, outputFileName)
		})
	})

	Describe("Gitleaks (Generic) Security Test", func() {
		It("should produce valid SonarQube output", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					GenericResults: types.GenericResults{
						HuskyCIGitleaksOutput: types.HuskyCISecurityTestOutput{
							LowVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Generic",
									SecurityTool: "GitLeaks",
									Severity:     "LOW",
									Title:        "Hard Coded Generic API Key in: config.json",
									Details:       "",
									File:          "config.json",
									Line:          "5",
									Code:          "api_key = \"sk_live_1234567890\"",
								},
							},
							MediumVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Generic",
									SecurityTool: "GitLeaks",
									Severity:     "MEDIUM",
									Title:        "Hard Coded AWS Secret Key in: .env",
									Details:       "",
									File:          ".env",
									Line:          "10",
									Code:          "AWS_SECRET_KEY=AKIAIOSFODNN7EXAMPLE",
								},
							},
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Generic",
									SecurityTool: "GitLeaks",
									Severity:     "HIGH",
									Title:        "Hard Coded RSA in: keys/private.pem",
									Details:       "",
									File:          "keys/private.pem",
									Line:          "1",
									Code:          "-----BEGIN RSA PRIVATE KEY-----",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())
			validateSonarQubeOutput(outputPath, outputFileName)
		})
	})

	Describe("Trivy (Generic) Security Test", func() {
		It("should produce valid SonarQube output", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					GenericResults: types.GenericResults{
						HuskyCITrivyOutput: types.HuskyCISecurityTestOutput{
							LowVulns: []types.HuskyCIVulnerability{
								{
									Language:     "generic",
									SecurityTool: "Trivy",
									Severity:     "LOW",
									Title:        "CVE-2023-1234",
									Details:       "Low severity vulnerability in package",
									File:          "./code/package.json",
								},
							},
							MediumVulns: []types.HuskyCIVulnerability{
								{
									Language:     "generic",
									SecurityTool: "Trivy",
									Severity:     "MEDIUM",
									Title:        "CVE-2023-5678",
									Details:       "Medium severity vulnerability in container image",
									File:          "./code/Dockerfile",
								},
							},
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "generic",
									SecurityTool: "Trivy",
									Severity:     "HIGH",
									Title:        "CVE-2023-9012",
									Details:       "High severity vulnerability in infrastructure",
									File:          "./code/main.tf",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())
			validateSonarQubeOutput(outputPath, outputFileName)
		})
	})

	Describe("Severity Mapping", func() {
		It("should correctly map severity levels", func() {
			testCases := []struct {
				inputSeverity    string
				expectedRuleSev  string
				expectedImpactSev string
			}{
				{"low", "MINOR", "LOW"},
				{"LOW", "MINOR", "LOW"},
				{"medium", "MAJOR", "MEDIUM"},
				{"MEDIUM", "MAJOR", "MEDIUM"},
				{"high", "BLOCKER", "HIGH"},
				{"HIGH", "BLOCKER", "HIGH"},
				{"unknown", "INFO", "INFO"},
			}

			for _, tc := range testCases {
				analysis := types.Analysis{
					HuskyCIResults: types.HuskyCIResults{
						GoResults: types.GoResults{
							HuskyCIGosecOutput: types.HuskyCISecurityTestOutput{
								HighVulns: []types.HuskyCIVulnerability{
									{
										Language:     "Go",
										SecurityTool: "GoSec",
										Severity:     tc.inputSeverity,
										Title:        "Test Vulnerability",
										Details:       "Test details",
										File:          "test.go",
										Line:          "1",
									},
								},
							},
						},
					},
				}

				err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
				Expect(err).NotTo(HaveOccurred())

				testOutputFilePath := filepath.Join(outputPath, outputFileName)
				fileContent, err := os.ReadFile(testOutputFilePath)
				Expect(err).NotTo(HaveOccurred())

				var sonarOutput sonarqube.HuskyCISonarOutput
				err = json.Unmarshal(fileContent, &sonarOutput)
				Expect(err).NotTo(HaveOccurred())

				Expect(sonarOutput.Rules).To(HaveLen(1))
				Expect(sonarOutput.Rules[0].Severity).To(Equal(tc.expectedRuleSev))
				Expect(sonarOutput.Rules[0].Impacts[0].Severity).To(Equal(tc.expectedImpactSev))
			}
		})
	})

	Describe("File Path Handling", func() {
		It("should handle Go container paths correctly", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					GoResults: types.GoResults{
						HuskyCIGosecOutput: types.HuskyCISecurityTestOutput{
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Go",
									SecurityTool: "GoSec",
									Severity:     "HIGH",
									Title:        "Test",
									Details:       "Test",
									File:          "/go/src/code/main.go",
									Line:          "10",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())

			testOutputFilePath := filepath.Join(outputPath, outputFileName)
			fileContent, err := os.ReadFile(testOutputFilePath)
			Expect(err).NotTo(HaveOccurred())

			var sonarOutput sonarqube.HuskyCISonarOutput
			err = json.Unmarshal(fileContent, &sonarOutput)
			Expect(err).NotTo(HaveOccurred())

			Expect(sonarOutput.Issues).To(HaveLen(1))
			// Go paths should have /go/src/code/ prefix removed
			Expect(sonarOutput.Issues[0].PrimaryLocation.FilePath).NotTo(ContainSubstring("/go/src/code/"))
		})

		It("should create placeholder file for vulnerabilities without file path", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					PythonResults: types.PythonResults{
						HuskyCISafetyOutput: types.HuskyCISecurityTestOutput{
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Python",
									SecurityTool: "Safety",
									Severity:     "high",
									Title:        "Vulnerable Dependency: django (<2.0.0)",
									Details:       "Django vulnerability",
									File:          "", // No file path
									Line:          "",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())

			testOutputFilePath := filepath.Join(outputPath, outputFileName)
			fileContent, err := os.ReadFile(testOutputFilePath)
			Expect(err).NotTo(HaveOccurred())

			var sonarOutput sonarqube.HuskyCISonarOutput
			err = json.Unmarshal(fileContent, &sonarOutput)
			Expect(err).NotTo(HaveOccurred())

			Expect(sonarOutput.Issues).To(HaveLen(1))
			// Should use placeholder file
			placeholderPath := filepath.Join(outputPath, "huskyCI_Placeholder_File")
			Expect(sonarOutput.Issues[0].PrimaryLocation.FilePath).To(ContainSubstring("huskyCI_Placeholder_File"))
			// Verify placeholder file was created
			_, err = os.Stat(placeholderPath)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Line Number Handling", func() {
		It("should handle invalid line numbers gracefully", func() {
			testCases := []struct {
				line         string
				expectedLine int
			}{
				{"", 1},
				{"invalid", 1},
				{"0", 1},
				{"-5", 1},
				{"42", 42},
				{"100", 100},
			}

			for _, tc := range testCases {
				analysis := types.Analysis{
					HuskyCIResults: types.HuskyCIResults{
						GoResults: types.GoResults{
							HuskyCIGosecOutput: types.HuskyCISecurityTestOutput{
								HighVulns: []types.HuskyCIVulnerability{
									{
										Language:     "Go",
										SecurityTool: "GoSec",
										Severity:     "HIGH",
										Title:        "Test",
										Details:       "Test",
										File:          "test.go",
										Line:          tc.line,
									},
								},
							},
						},
					},
				}

				err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
				Expect(err).NotTo(HaveOccurred())

				testOutputFilePath := filepath.Join(outputPath, outputFileName)
				fileContent, err := os.ReadFile(testOutputFilePath)
				Expect(err).NotTo(HaveOccurred())

				var sonarOutput sonarqube.HuskyCISonarOutput
				err = json.Unmarshal(fileContent, &sonarOutput)
				Expect(err).NotTo(HaveOccurred())

				Expect(sonarOutput.Issues).To(HaveLen(1))
				Expect(sonarOutput.Issues[0].PrimaryLocation.TextRange.StartLine).To(Equal(tc.expectedLine))
			}
		})
	})

	Describe("Multiple Security Tests Combined", func() {
		It("should handle vulnerabilities from multiple security tests", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					GoResults: types.GoResults{
						HuskyCIGosecOutput: types.HuskyCISecurityTestOutput{
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Go",
									SecurityTool: "GoSec",
									Severity:     "HIGH",
									Title:        "Go Vulnerability",
									Details:       "Go issue",
									File:          "go.go",
									Line:          "1",
								},
							},
						},
					},
					PythonResults: types.PythonResults{
						HuskyCIBanditOutput: types.HuskyCISecurityTestOutput{
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Python",
									SecurityTool: "Bandit",
									Severity:     "HIGH",
									Title:        "Python Vulnerability",
									Details:       "Python issue",
									File:          "python.py",
									Line:          "2",
								},
							},
						},
					},
					GenericResults: types.GenericResults{
						HuskyCIGitleaksOutput: types.HuskyCISecurityTestOutput{
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Generic",
									SecurityTool: "GitLeaks",
									Severity:     "HIGH",
									Title:        "Secret Vulnerability",
									Details:       "Secret found",
									File:          "secret.txt",
									Line:          "3",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())
			validateSonarQubeOutput(outputPath, outputFileName)

			// Verify all vulnerabilities are included
			testOutputFilePath := filepath.Join(outputPath, outputFileName)
			fileContent, err := os.ReadFile(testOutputFilePath)
			Expect(err).NotTo(HaveOccurred())

			var sonarOutput sonarqube.HuskyCISonarOutput
			err = json.Unmarshal(fileContent, &sonarOutput)
			Expect(err).NotTo(HaveOccurred())

			Expect(sonarOutput.Issues).To(HaveLen(3))
			Expect(sonarOutput.Rules).To(HaveLen(3))

			// Verify each security tool is represented
			engineIDs := make(map[string]bool)
			for _, rule := range sonarOutput.Rules {
				engineIDs[rule.EngineID] = true
			}
			Expect(engineIDs).To(HaveKey("huskyCI/GoSec"))
			Expect(engineIDs).To(HaveKey("huskyCI/Bandit"))
			Expect(engineIDs).To(HaveKey("huskyCI/GitLeaks"))
		})
	})

	Describe("Empty Results", func() {
		It("should handle analysis with no vulnerabilities", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())

			testOutputFilePath := filepath.Join(outputPath, outputFileName)
			fileContent, err := os.ReadFile(testOutputFilePath)
			Expect(err).NotTo(HaveOccurred())

			var sonarOutput sonarqube.HuskyCISonarOutput
			err = json.Unmarshal(fileContent, &sonarOutput)
			Expect(err).NotTo(HaveOccurred())

			Expect(sonarOutput.Rules).To(BeEmpty())
			Expect(sonarOutput.Issues).To(BeEmpty())
		})
	})

	Describe("Rule Deduplication", func() {
		It("should deduplicate rules with same ID", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					GoResults: types.GoResults{
						HuskyCIGosecOutput: types.HuskyCISecurityTestOutput{
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Go",
									SecurityTool: "GoSec",
									Severity:     "HIGH",
									Title:        "Same Vulnerability",
									Details:       "Same details",
									File:          "file1.go",
									Line:          "10",
								},
								{
									Language:     "Go",
									SecurityTool: "GoSec",
									Severity:     "HIGH",
									Title:        "Same Vulnerability",
									Details:       "Same details",
									File:          "file2.go",
									Line:          "20",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())

			testOutputFilePath := filepath.Join(outputPath, outputFileName)
			fileContent, err := os.ReadFile(testOutputFilePath)
			Expect(err).NotTo(HaveOccurred())

			var sonarOutput sonarqube.HuskyCISonarOutput
			err = json.Unmarshal(fileContent, &sonarOutput)
			Expect(err).NotTo(HaveOccurred())

			// Should have only one rule but two issues
			Expect(sonarOutput.Rules).To(HaveLen(1))
			Expect(sonarOutput.Issues).To(HaveLen(2))
		})
	})

	Describe("Message Handling", func() {
		It("should handle empty details gracefully", func() {
			analysis := types.Analysis{
				HuskyCIResults: types.HuskyCIResults{
					GenericResults: types.GenericResults{
						HuskyCIGitleaksOutput: types.HuskyCISecurityTestOutput{
							HighVulns: []types.HuskyCIVulnerability{
								{
									Language:     "Generic",
									SecurityTool: "GitLeaks",
									Severity:     "HIGH",
									Title:        "Secret Found",
									Details:       "", // Empty details
									File:          "secret.txt",
									Line:          "1",
								},
							},
						},
					},
				},
			}

			err := sonarqube.GenerateOutputFile(analysis, outputPath, outputFileName)
			Expect(err).NotTo(HaveOccurred())

			testOutputFilePath := filepath.Join(outputPath, outputFileName)
			fileContent, err := os.ReadFile(testOutputFilePath)
			Expect(err).NotTo(HaveOccurred())

			var sonarOutput sonarqube.HuskyCISonarOutput
			err = json.Unmarshal(fileContent, &sonarOutput)
			Expect(err).NotTo(HaveOccurred())

			Expect(sonarOutput.Rules).To(HaveLen(1))
			// Should have default message for empty details
			Expect(sonarOutput.Rules[0].Description).To(ContainSubstring("No details provided"))
		})
	})
})
