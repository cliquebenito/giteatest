// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package oauth2

import (
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/openidConnect"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Callout redirects request/response pair to authenticate against the provider
func (source *Source) Callout(request *http.Request, response http.ResponseWriter) error {
	// not sure if goth is thread safe (?) when using multiple providers
	request.Header.Set(ProviderHeaderKey, source.authSource.Name)

	// don't use the default gothic begin handler to prevent issues when some error occurs
	// normally the gothic library will write some custom stuff to the response instead of our own nice error page
	// gothic.BeginAuthHandler(response, request)

	gothRWMutex.RLock()
	defer gothRWMutex.RUnlock()

	url, err := gothic.GetAuthURL(response, request)
	if err == nil {
		http.Redirect(response, request, url, http.StatusTemporaryRedirect)
	}
	return err
}

// Callback handles OAuth callback, resolve to a goth user and send back to original url
// this will trigger a new authentication request, but because we save it in the session we can use that
func (source *Source) Callback(request *http.Request, response http.ResponseWriter) (goth.User, error) {
	// not sure if goth is thread safe (?) when using multiple providers
	request.Header.Set(ProviderHeaderKey, source.authSource.Name)

	gothRWMutex.RLock()
	defer gothRWMutex.RUnlock()

	user, err := gothic.CompleteUserAuth(response, request)
	if err != nil {
		return user, err
	}

	return user, nil
}

// LogOut метод, в котором отправляется POST запрос на завершение сессии пользователя в Keycloak.
// - Для выполнения этого запроса необходимы данные из провайдера: актуальный URL для завершения сессии (EndSessionEndpoint), который провайдер получает изначально.
// - Так же необходим refresh token keycloak сессии пользователя (он может быть уже протухший, в любом случае проверяется
// его валидность без срока действия). Refresh token мы получаем из сессии контекста, в которую мы предварительно положили токен во время аутентификации.
// - Нужен clientID и secret этого клиента, которые мы берем из source (на котором вызван этот метод).
func (source *Source) LogOut(request *http.Request, refToken string) error {
	request.Header.Set(ProviderHeaderKey, source.authSource.Name)

	gothRWMutex.RLock()
	defer gothRWMutex.RUnlock()

	providerName, err := gothic.GetProviderName(request)
	if err != nil {
		return err
	}

	provider, err := goth.GetProvider(providerName)
	if err != nil {
		return err
	}

	httpClient := &http.Client{Timeout: 2 * time.Second}

	data := url.Values{}
	data.Set("client_id", source.ClientID)
	data.Set("client_secret", source.ClientSecret)
	data.Set("refresh_token", refToken)

	req, err := http.NewRequest(http.MethodPost,
		provider.(*openidConnect.Provider).OpenIDConfig.EndSessionEndpoint,
		strings.NewReader(data.Encode()))

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if err != nil {
		return err
	}

	_, err = httpClient.Do(req)

	if err != nil {
		return err
	}

	return nil
}
