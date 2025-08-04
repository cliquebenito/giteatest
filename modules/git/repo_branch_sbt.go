package git

// GetBranchesNames возвращает список имен веток репозитория
func (repo *Repository) GetBranchesNames() ([]string, error) {
	brs, _, err := repo.GetBranchNames(0, 0)

	return brs, err
}
