package response

import "time"

type PayloadCommit struct {
	ID           string                     `json:"id"`
	Message      string                     `json:"message"`
	Author       *PayloadUser               `json:"author"`
	Committer    *PayloadUser               `json:"committer"`
	Verification *PayloadCommitVerification `json:"verification"`
	Timestamp    time.Time                  `json:"timestamp"`
	Added        []string                   `json:"added"`
	Removed      []string                   `json:"removed"`
	Modified     []string                   `json:"modified"`
}

type PayloadUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	UserName string `json:"username"`
}

type PayloadCommitVerification struct {
	Verified  bool         `json:"verified"`
	Reason    string       `json:"reason"`
	Signature string       `json:"signature"`
	Signer    *PayloadUser `json:"signer"`
	Payload   string       `json:"payload"`
}
