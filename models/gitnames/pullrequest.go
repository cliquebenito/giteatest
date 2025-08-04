package gitnames

type PullRequestLinks struct {
	CommitsLinks CommitsLinks
	BranchLinks  BranchLinks

	Base
}

func (p PullRequestLinks) IsEmpty() bool {
	return p.Base.IsEmpty() && p.BranchLinks.IsEmpty() && p.CommitsLinks.IsEmpty()
}

func (p PullRequestLinks) GetUniqCodes() ([]UnitCode, error) {
	var codes LinkedUnits

	for _, code := range p.Base.LinkedUnits {
		codes = append(codes, code)
	}

	for _, commits := range p.CommitsLinks {
		for _, code := range commits.LinkedUnits {
			codes = append(codes, code)
		}
	}

	for _, code := range p.BranchLinks.LinkedUnits {
		codes = append(codes, code)
	}

	if len(codes) == 0 || codes == nil {
		return nil, NewEmptyUnitCodesListError()
	}

	return codes.getUniqCodes()
}
