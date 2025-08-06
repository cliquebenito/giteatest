package license

// ResponseInfoLicense формат ответы для отображения информации о лицензии в файле с лицензией
type ResponseInfoLicense struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Conditions  []string `json:"conditions"`
	Limitations []string `json:"limitations"`
}

// InfoLicenses структура для получения информации о лицензии
type InfoLicenses struct {
	SpdxID      string `json:"spdx_id"`
	Title       string `json:"name"`
	Description string `json:"description"`
	Permissions string `json:"permissions"`
	Conditions  string `json:"conditions"`
	Limitations string `json:"limitations"`
}

// LicenseInfoJson получение информации о лицензии из бд
type LicenseInfoJson struct {
	SpdxID      string   `json:"spdx_id"`
	Title       string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Conditions  []string `json:"conditions"`
	Limitations []string `json:"limitations"`
	Body        string   `json:"body"`
}

// RepoLicenses основная информация из таблицы sc_repo_licenses
type RepoLicenses struct {
	RepositoryID int64
	SpdxID       string
	NameLicense  string
	BranchName   string
	PathFile     string
}
