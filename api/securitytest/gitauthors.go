package securitytest

import (
	"encoding/json"
	"strings"

	"github.com/huskyci-org/huskyCI/api/log"
	"github.com/huskyci-org/huskyCI/api/util"
)

// GitAuthorsOutput is the struct that holds all commit authors from a branch.
type GitAuthorsOutput struct {
	Authors []string `json:"authors"`
}

func analyzeGitAuthors(gitAuthorsScan *SecTestScanInfo) error {

	gitAuthorsOutput := GitAuthorsOutput{}
	gitAuthorsScan.FinalOutput = gitAuthorsOutput

	output := strings.TrimSpace(gitAuthorsScan.Container.COutput)
	// Empty or invalid JSON (e.g. from file:// with no git history): treat as no authors, do not fail analysis
	if output == "" {
		gitAuthorsScan.CommitAuthorsNotFound = true
		gitAuthorsScan.CommitAuthors = gitAuthorsOutput
		gitAuthorsScan.prepareContainerAfterScan()
		return nil
	}
	if err := json.Unmarshal([]byte(gitAuthorsScan.Container.COutput), &gitAuthorsOutput); err != nil {
		// For empty/invalid JSON, treat as no authors so analysis can complete (e.g. file:// has no git)
		if strings.Contains(err.Error(), "unexpected end of JSON input") || strings.Contains(err.Error(), "EOF") {
			log.Info("analyzeGitAuthors", "GITAUTHORS", 16, "GitAuthors output empty or invalid JSON, treating as no authors")
			gitAuthorsScan.CommitAuthorsNotFound = true
			gitAuthorsScan.CommitAuthors = gitAuthorsOutput
			gitAuthorsScan.prepareContainerAfterScan()
			return nil
		}
		log.Error("analyzeGitAuthors", "GITAUTHORS", 1035, gitAuthorsScan.Container.COutput, err)
		gitAuthorsScan.ErrorFound = util.HandleScanError(gitAuthorsScan.Container.COutput, err)
		gitAuthorsScan.prepareContainerAfterScan()
		return gitAuthorsScan.ErrorFound
	}
	gitAuthorsScan.FinalOutput = gitAuthorsOutput

	// check if authors is empty (master branch was probably sent)
	if len(gitAuthorsOutput.Authors) == 0 {
		gitAuthorsScan.CommitAuthorsNotFound = true
		gitAuthorsScan.prepareContainerAfterScan()
	}

	gitAuthorsScan.CommitAuthors = gitAuthorsOutput
	gitAuthorsScan.prepareContainerAfterScan()
	return nil
}
