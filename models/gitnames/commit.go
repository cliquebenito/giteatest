package gitnames

type CommitLinks struct {
	Base
}

type CommitsLinks []CommitLinks

func (c CommitsLinks) IsEmpty() bool {
	return len(c) == 0
}

func (b CommitLinks) GetUniqCodes() ([]UnitCode, error) {
	if len(b.LinkedUnits) == 0 || b.LinkedUnits == nil {
		return nil, NewEmptyUnitCodesListError()
	}

	return b.LinkedUnits.getUniqCodes()
}
