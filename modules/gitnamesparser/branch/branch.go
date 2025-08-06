package branch

import (
	"fmt"
	"regexp"

	"code.gitea.io/gitea/models/gitnames"
	"code.gitea.io/gitea/modules/gitnamesparser"
)

type branchParser struct{}

func NewParser() branchParser {
	return branchParser{}
}

var branchCodeRE = regexp.MustCompile("[^/_a-zа-я-][A-Z_0-9]{1,30}-[0-9]{1,30}")

// Parse метод ищет коды юнитов TaskTracker в названии ветки
func (b branchParser) Parse(branchName string) (gitnames.BranchLinks, error) {
	codes, desc, err := gitnamesparser.ParseCodesAndDescription(branchName, branchCodeRE)
	if err != nil {
		return gitnames.BranchLinks{}, fmt.Errorf("parse codes and description: %w", err)
	}

	branch := gitnames.BranchLinks{
		Base: gitnames.Base{Description: desc},
	}

	if len(codes) == 0 {
		return gitnames.BranchLinks{}, gitnames.NewEmptyUnitCodesListError()
	}

	for _, code := range codes {
		branch.LinkedUnits = append(branch.LinkedUnits, gitnames.UnitCode{Code: code})
	}

	return branch, nil
}
