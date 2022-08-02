// Copyright (C) 2022 Specter Ops, Inc.
//
// This file is part of AzureHound.
//
// AzureHound is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// AzureHound is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package rest

//go:generate go run github.com/golang/mock/mockgen -destination=./mocks/client.go -package=mocks . RestClient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/bloodhoundad/azurehound/client/config"
	"github.com/bloodhoundad/azurehound/constants"
)

type RestClient interface {
	Authenticate() error
	Delete(ctx context.Context, path string, body interface{}, params, headers map[string]string) (*http.Response, error)
	Get(ctx context.Context, path string, params, headers map[string]string) (*http.Response, error)
	Patch(ctx context.Context, path string, body interface{}, params, headers map[string]string) (*http.Response, error)
	Post(ctx context.Context, path string, body interface{}, params, headers map[string]string) (*http.Response, error)
	Put(ctx context.Context, path string, body interface{}, params, headers map[string]string) (*http.Response, error)
	Send(req *http.Request) (*http.Response, error)
}

func NewRestClient(apiUrl string, config config.Config) (RestClient, error) {
	if auth, err := url.Parse(config.AuthorityUrl()); err != nil {
		return nil, err
	} else if api, err := url.Parse(apiUrl); err != nil {
		return nil, err
	} else {
		client := &restClient{
			*api,
			*auth,
			config.JWT,
			config.ApplicationId,
			config.ClientSecret,
			config.ClientCert,
			config.ClientKey,
			config.ClientKeyPass,
			config.Username,
			config.Password,
			NewHTTPClient(),
			sync.RWMutex{},
			config.RefreshToken,
			config.Tenant,
			Token{},
			config.SubscriptionId,
			config.MgmtGroupId,
		}
		return client, nil
	}
}

type restClient struct {
	api           url.URL
	authUrl       url.URL
	jwt           string
	clientId      string
	clientSecret  string
	clientCert    string
	clientKey     string
	clientKeyPass string
	username      string
	password      string
	http          *http.Client
	mutex         sync.RWMutex
	refreshToken  string
	tenant        string
	token         Token
	subId         []string
	mgmtGroupId   []string
}

func (s *restClient) Authenticate() error {
	var (
		path         = url.URL{Path: fmt.Sprintf("/%s/oauth2/v2.0/token", s.tenant)}
		endpoint     = s.authUrl.ResolveReference(&path)
		defaultScope = url.URL{Path: "/.default"}
		scope        = s.api.ResolveReference(&defaultScope)
		body         = url.Values{}
	)

	if s.clientId == "" {
		body.Add("client_id", constants.AzPowerShellClientID)
	} else {
		body.Add("client_id", s.clientId)
	}

	body.Add("scope", scope.ResolveReference(&defaultScope).String())

	if s.refreshToken != "" {
		body.Add("grant_type", "refresh_token")
		body.Add("refresh_token", s.refreshToken)
	} else if s.clientSecret != "" {
		body.Add("grant_type", "client_credentials")
		body.Add("client_secret", s.clientSecret)
	} else if s.clientCert != "" && s.clientKey != "" {
		if clientAssertion, err := NewClientAssertion(endpoint.String(), s.clientId, s.clientCert, s.clientKey, s.clientKeyPass); err != nil {
			return err
		} else {
			body.Add("grant_type", "client_credentials")
			body.Add("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
			body.Add("client_assertion", clientAssertion)
		}
	} else if s.username != "" && s.password != "" {
		body.Add("grant_type", "password")
		body.Add("username", s.username)
		body.Add("password", s.password)
	} else {
		return fmt.Errorf("unable to authenticate. no valid credential provided")
	}

	if req, err := NewRequest(context.Background(), "POST", endpoint, body, nil, nil); err != nil {
		return err
	} else if res, err := s.send(req); err != nil {
		return err
	} else {
		defer res.Body.Close()
		s.mutex.Lock()
		defer s.mutex.Unlock()
		if err := json.NewDecoder(res.Body).Decode(&s.token); err != nil {
			return err
		} else {
			return nil
		}
	}
}

func (s *restClient) Delete(ctx context.Context, path string, body interface{}, params, headers map[string]string) (*http.Response, error) {
	endpoint := s.api.ResolveReference(&url.URL{Path: path})
	if req, err := NewRequest(ctx, http.MethodDelete, endpoint, body, params, headers); err != nil {
		return nil, err
	} else {
		return s.Send(req)
	}
}

func (s *restClient) Get(ctx context.Context, path string, params, headers map[string]string) (*http.Response, error) {
	endpoint := s.api.ResolveReference(&url.URL{Path: path})
	if req, err := NewRequest(ctx, http.MethodGet, endpoint, nil, params, headers); err != nil {
		return nil, err
	} else {
		return s.Send(req)
	}
}

func (s *restClient) Patch(ctx context.Context, path string, body interface{}, params, headers map[string]string) (*http.Response, error) {
	endpoint := s.api.ResolveReference(&url.URL{Path: path})
	if req, err := NewRequest(ctx, http.MethodPatch, endpoint, body, params, headers); err != nil {
		return nil, err
	} else {
		return s.Send(req)
	}
}

func (s *restClient) Post(ctx context.Context, path string, body interface{}, params, headers map[string]string) (*http.Response, error) {
	endpoint := s.api.ResolveReference(&url.URL{Path: path})
	if req, err := NewRequest(ctx, http.MethodPost, endpoint, body, params, headers); err != nil {
		return nil, err
	} else {
		return s.Send(req)
	}
}

func (s *restClient) Put(ctx context.Context, path string, body interface{}, params, headers map[string]string) (*http.Response, error) {
	endpoint := s.api.ResolveReference(&url.URL{Path: path})
	if req, err := NewRequest(ctx, http.MethodPost, endpoint, body, params, headers); err != nil {
		return nil, err
	} else {
		return s.Send(req)
	}
}

func (s *restClient) Send(req *http.Request) (*http.Response, error) {
	if s.jwt != "" {
		if aud, err := ParseAud(s.jwt); err != nil {
			return nil, err
		} else if aud != s.api.String() {
			return nil, fmt.Errorf("invalid audience")
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.jwt))
	} else {
		if s.token.IsExpired() {
			if err := s.Authenticate(); err != nil {
				return nil, err
			}
		}
		req.Header.Set("Authorization", s.token.String())
	}
	return s.send(req)
}

func (s *restClient) send(req *http.Request) (*http.Response, error) {
	res, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusBadRequest {
		var errRes map[string]interface{}
		if err := Decode(res.Body, &errRes); err != nil {
			return nil, fmt.Errorf("malformed error response, status code: %d", res.StatusCode)
		} else {
			return nil, fmt.Errorf("Error: %v", errRes)
		}
	} else {
		return res, nil
	}
}
