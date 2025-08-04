package repo_marks_db

// ErrMarkAlreadyExists кастомная ошибка, если метка CodeHub уже существует
type ErrMarkAlreadyExists struct {
	MarkKey string
}

func (e ErrMarkAlreadyExists) Error() string {
	return "repo mark with key " + e.MarkKey + " already exists"
}
