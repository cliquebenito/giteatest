//go:build !correct

package user

import (
	"github.com/Nerzal/gocloak/v13"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TestAccessTokenExpired проверка токена на протухание
func TestAccessTokenExpired(t *testing.T) {
	jwt := gocloak.JWT{ExpiresIn: 300}
	token := accessTokenParams{
		jwt:     &jwt,
		created: time.Now(),
	}

	assert.False(t, accessTokenExpired(token)) // токен не протух

	token.jwt.ExpiresIn = timeSkew            //время жизни токена равно дельте
	assert.True(t, accessTokenExpired(token)) // поэтому он протух

	token.jwt.ExpiresIn = timeSkew + 1         //время жизни токена равно дельте + 1 секунду
	assert.False(t, accessTokenExpired(token)) // токен не протух
	time.Sleep(time.Second)                    // ждем секунду, что бы токен протух
	assert.True(t, accessTokenExpired(token))  //токен протух
}
