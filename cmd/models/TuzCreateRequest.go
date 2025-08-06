package models

import (
	"fmt"

	"github.com/urfave/cli"
)

// TuzCreateRequest структура для параметров при создании ТУЗа из cli
type TuzCreateRequest struct {
	Username    string
	AccessToken bool
	HealthURL   string
}

// HandleTuzCreateRequest функция для парсинга параметров при запросе из cli на создание ТУЗа
func HandleTuzCreateRequest(c *cli.Context) (*TuzCreateRequest, error) {
	req := &TuzCreateRequest{
		Username:    c.String("username"),
		AccessToken: c.Bool("access-token"),
		HealthURL:   c.String("health-url"),
	}

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("Fail to validate create tuz request: %w", err)
	}

	return req, nil
}

func (r *TuzCreateRequest) Validate() error {
	if r.Username == "" {
		return fmt.Errorf("username must be set")
	}

	if r.HealthURL == "" {
		return fmt.Errorf("health-url must be set")
	}

	return nil
}
