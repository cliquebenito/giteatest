package gitnames

import "sort"

type UnitCode struct {
	Code string
}

type LinkedUnits []UnitCode

type Base struct {
	Description string
	LinkedUnits
}

func (b Base) IsEmpty() bool {
	return len(b.LinkedUnits) == 0
}

func (l LinkedUnits) getUniqCodes() ([]UnitCode, error) {
	uniqCodesMap := map[UnitCode]struct{}{}

	for _, code := range l {
		uniqCodesMap[code] = struct{}{}
	}

	var uniqCodes []UnitCode

	for code := range uniqCodesMap {
		uniqCodes = append(uniqCodes, code)
	}

	sort.Slice(uniqCodes, func(i, j int) bool {
		return uniqCodes[i].Code < uniqCodes[j].Code
	})

	return uniqCodes, nil
}
