package cmd

import (
	"fmt"
	"io"
	"net/http"

	"github.com/urfave/cli"

	"code.gitea.io/gitea/cmd/models"
	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/role_model"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/sbt/audit"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/routers/sbt/user/createuser"
)

const pass = "pass"

var microcmdTuzCreate = cli.Command{
	Name:   "create-tuz",
	Usage:  "Create a new tuz in database",
	Action: runCreateTuz,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "username",
			Usage: "Username",
		},
		cli.StringFlag{
			Name:  "health-url",
			Usage: "Healthcheck URL",
		},
		cli.BoolFlag{
			Name:  "access-token",
			Usage: "Generate access token for the user",
		},
	},
}

func runCreateTuz(c *cli.Context) error {
	auditParams := map[string]string{}
	ctx, cancel := installSignals()
	defer cancel()

	if err := initDB(ctx); err != nil {
		auditParams["error"] = "Error has occurred while init DB"
		audit.CreateAndSendEvent(audit.TuzCreateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("Fail to init DB: %w", err)
	}

	req, err := models.HandleTuzCreateRequest(c)
	if err != nil {
		auditParams["error"] = "Error has occurred while parsing request"
		audit.CreateAndSendEvent(audit.TuzCreateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("Fail to handle cli request: %w", err)
	}
	auditParams["username"] = req.Username

	resp, err := http.Get(req.HealthURL)
	if err != nil {
		auditParams["error"] = "Error has occurred while doing healthcheck request"
		audit.CreateAndSendEvent(audit.TuzCreateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("Fail to make healthcheck request: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		auditParams["error"] = "Error has occurred while reading healthcheck response"
		audit.CreateAndSendEvent(audit.TuzCreateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("Fail to read healthcheck request: %w", err)
	}
	params := struct {
		Status string `json:"status"`
	}{}
	if err = json.Unmarshal(body, &params); err != nil {
		auditParams["error"] = "Error has occurred parsing healthcheck response"
		audit.CreateAndSendEvent(audit.TuzCreateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("Fail to parse healthcheck request: %w", err)
	}
	if params.Status != pass {
		auditParams["error"] = "Error has occurred while doing healthcheck"
		audit.CreateAndSendEvent(audit.TuzCreateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("SourceControl must be running to create TUZ")
	}

	ctx, committer, err := db.TxContext(db.DefaultContext)
	if err != nil {
		auditParams["error"] = "Error has occurred while creating transaction"
		audit.CreateAndSendEvent(audit.TuzCreateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("Fail to init transaction: %w", err)
	}
	e := db.GetEngine(ctx)
	defer committer.Close()

	// default user visibility in app.ini
	visibility := setting.Service.DefaultUserVisibilityMode

	u := &user_model.User{
		Name:                         req.Username,
		IsAdmin:                      true,
		Visibility:                   visibility,
		IsRestricted:                 false,
		IsActive:                     true,
		EmailNotificationsPreference: user_model.EmailNotificationsDisabled,
	}

	if err := createuser.Create(ctx, e, u); err != nil {
		auditParams["error"] = "Error has occurred while creating user"
		audit.CreateAndSendEvent(audit.TuzCreateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("Failt to create user: %w", err)
	}

	if err := role_model.InitRoleModelDB(); err != nil {
		auditParams["error"] = "Error has occurred while initializing role model DB"
		audit.CreateAndSendEvent(audit.TuzCreateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("Fail to init role model: %w", err)
	}

	if err := role_model.GrantUserTuz(u); err != nil {
		auditParams["error"] = "Error has occurred while granting user TUZ"
		audit.CreateAndSendEvent(audit.TuzCreateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("Fail to grant user tuz: %w", err)
	}

	if req.AccessToken {
		t := &auth_model.AccessToken{
			Name: "gitea-admin",
			UID:  u.ID,
		}

		if err := auth_model.NewAccessToken(t); err != nil {
			auditParams["error"] = "Error has occurred while generating access token"
			audit.CreateAndSendEvent(audit.TuzCreateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
			return fmt.Errorf("Fail to generate access token: %w", err)
		}

		fmt.Printf("Access token was successfully created... %s\n", t.Token)
	}

	if err = committer.Commit(); err != nil {
		auditParams["error"] = "Error has occurred while closing the transaction"
		audit.CreateAndSendEvent(audit.TuzCreateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusFailure, audit.EmptyRequiredField, auditParams)
		return fmt.Errorf("Fail to commit transaction: %w", err)
	}
	committer.Close()

	audit.CreateAndSendEvent(audit.TuzCreateEvent, audit.EmptyRequiredField, audit.EmptyRequiredField, audit.StatusSuccess, audit.EmptyRequiredField, auditParams)
	fmt.Printf("New tuz '%s' has been successfully created!\n", req.Username)

	return nil
}
