package gitnames

type BranchLinks struct {
	Base
}

func (b BranchLinks) IsEmpty() bool {
	return len(b.LinkedUnits) == 0
}

func (b BranchLinks) GetUniqCodes() ([]UnitCode, error) {
	if len(b.LinkedUnits) == 0 || b.LinkedUnits == nil {
		return nil, NewEmptyUnitCodesListError()
	}

	return b.LinkedUnits.getUniqCodes()
}
