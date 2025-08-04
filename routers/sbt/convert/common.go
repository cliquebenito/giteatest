package convert

import (
	asymkey_model "code.gitea.io/gitea/models/asymkey"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/routers/sbt/response"
	"context"
)

// ToVerification конвертирует git.Commit.Signature в response.PayloadCommitVerification
func ToVerification(ctx context.Context, c *git.Commit) *response.PayloadCommitVerification {
	verif := asymkey_model.ParseCommitWithSignature(ctx, c)
	commitVerification := &response.PayloadCommitVerification{
		Verified: verif.Verified,
		Reason:   verif.Reason,
	}
	if c.Signature != nil {
		commitVerification.Signature = c.Signature.Signature
		commitVerification.Payload = c.Signature.Payload
	}
	if verif.SigningUser != nil {
		commitVerification.Signer = &response.PayloadUser{
			Name:  verif.SigningUser.Name,
			Email: verif.SigningUser.Email,
		}
	}
	return commitVerification
}
