package gitnames

type EmptyUnitCodesListError struct{}

func NewEmptyUnitCodesListError() *EmptyUnitCodesListError {
	return &EmptyUnitCodesListError{}
}

func (e *EmptyUnitCodesListError) Error() string {
	return "empty unit codes list"
}
