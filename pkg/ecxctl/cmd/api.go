// Copyright © 2018 Juan Manuel Irigaray <jirigaray@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"log"
	"net/http"

	"crypto/tls"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	apiclient "github.com/jxoir/go-ecxfabric/client"
	"github.com/jxoir/go-ecxfabric/client/access_token"
	"github.com/jxoir/go-ecxfabric/models"
)

// EquinixAPIParams struct for generic Equinix params
type EquinixAPIParams struct {
	AppID           string
	AppSecret       string
	GrantType       string
	UserName        string
	UserPassword    string
	Endpoint        string
	PlaygroundToken string
}

// EquinixAPIClient containing structure for Client, params and apitoken
// TODO: Implement token refresh
type EquinixAPIClient struct {
	Client   *apiclient.GoEcxfabric
	Params   *EquinixAPIParams
	apiToken runtime.ClientAuthInfoWriter
}

const (
	// ECX declare ECX API type
	ECX = iota
)

// APIHandler implements common Equinix API handlers commands
type APIHandler interface {
	Authenticate() error
	GetToken() (runtime.ClientAuthInfoWriter, error)
}

var defaultGrantType = "client_credentials"

// NewEcxAPIClient returns an instantiated ECX client with token
func NewEcxAPIClient(params *EquinixAPIParams, endpoint string, ignoreSSL bool) *EquinixAPIClient {
	var equinixAPIClient *EquinixAPIClient
	if ignoreSSL != false {
		log.Println(" - Insecure mode, ingoring SSL certificate")

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		transport := httptransport.NewWithClient(endpoint, "", nil, client)
		ecxAPIClient := apiclient.New(transport, strfmt.Default)

		equinixAPIClient = &EquinixAPIClient{
			Params: params,
			Client: ecxAPIClient,
		}

	} else {
		// create the transport
		transport := httptransport.New(endpoint, "", nil)
		// create the API client, with the transport
		ecxAPIClient := apiclient.New(transport, strfmt.Default)

		equinixAPIClient = &EquinixAPIClient{
			Params: params,
			Client: ecxAPIClient,
		}
	}

	return equinixAPIClient
}

// GetToken returns local token, if token doesn't exists tries to authenticate and retrieve token
func (ec *EquinixAPIClient) GetToken() (runtime.ClientAuthInfoWriter, error) {
	if ec.apiToken == nil {

		err := ec.Authenticate()
		if err != nil {
			return nil, err
		}
	}

	return ec.apiToken, nil

}

// Authenticate tries to authenticate and stores token from remote endpoint
func (ec *EquinixAPIClient) Authenticate() error {
	// set default parameters
	if ec.Params.PlaygroundToken != "" {
		// we are going to use playground mode, that means fixed token for each request
		log.Println("Playground mode enabled - token:" + ec.Params.PlaygroundToken)
		bearerTokenAuth := httptransport.BearerToken(ec.Params.PlaygroundToken)
		fmt.Println(bearerTokenAuth)
		ec.apiToken = bearerTokenAuth
		return nil
	}
	if ec.Params.AppID == "" {
		log.Fatal("EQUINIX_API_ID not set")
	}
	if ec.Params.AppSecret == "" {
		log.Fatal("EQUINIX_API_SECRET not set")
	}
	if ec.Params.Endpoint == "" {
		log.Fatal("ECX_API_HOST not specified")
	}
	if ec.Params.GrantType == "" {
		ec.Params.AppSecret = defaultGrantType
	}

	accessTokenParams := access_token.NewGetAccessTokenParams()
	accessTokenRequest := models.OAuthRequest{
		ClientID:     ec.Params.AppID,
		ClientSecret: ec.Params.AppSecret,
		GrantType:    ec.Params.GrantType,
		UserName:     ec.Params.UserName,
		UserPassword: ec.Params.UserPassword,
	}

	accessTokenParams.SetRequest(&accessTokenRequest)
	accessTokenParams.Authorization = "Bearer"

	accessToken, err := ec.Client.AccessToken.GetAccessToken(accessTokenParams, nil)
	if err != nil {
		if globalFlags.Debug {
			log.Println("Failed to retrieve token...")
		}
		return err
	}

	if globalFlags.Debug {
		log.Println("Token acquired...")
	}

	bearerTokenAuth := httptransport.BearerToken(accessToken.Payload.AccessToken)

	ec.apiToken = bearerTokenAuth

	if globalFlags.Debug {
		log.Println("User:" + ec.Params.UserName)
		log.Println("Endpoint:" + ec.Params.Endpoint)
		log.Println("AppId:" + ec.Params.AppID)
		log.Println("Token:" + accessToken.Payload.AccessToken)
		log.Println("Grant Type:" + ec.Params.GrantType)
	}

	return nil
}
