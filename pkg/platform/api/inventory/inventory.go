package inventory

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	iop "github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_client/inventory_operations"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

// persist contains the active API Client connection
var persist inventory_operations.ClientService

var transport http.RoundTripper

type Client struct {
	client    inventory_operations.ClientService
	transport *httptransport.Runtime
}

func InitClient(auth *authentication.Auth) *Client {
	return NewClient(api.GetServiceURL(api.ServiceInventory), auth.ClientAuth())
}

func NewClient(serviceURL *url.URL, auth runtime.ClientAuthInfoWriter) *Client {
	logging.Debug("apiURL: %s", serviceURL.String())
	transportRuntime := httptransport.New(serviceURL.Host, serviceURL.Path, []string{serviceURL.Scheme})
	transportRuntime.Transport = api.NewRoundTripper(http.DefaultTransport)

	// transportRuntime.SetDebug(true)

	if auth != nil {
		transportRuntime.DefaultAuthentication = auth
	}

	return &Client{
		client:    inventory_operations.New(transportRuntime, strfmt.Default),
		transport: transportRuntime,
	}
}

// func (c *Client) ResolveRecipes(params *iop.ResolveRecipesParams) (*inventory_operations.ResolveRecipesOK, error) {
// 	if params == nil {
// 		params = iop.NewResolveRecipesParams()
// 	}

// 	result, err := c.transport.Submit(&runtime.ClientOperation{
// 		ID:                 "resolveRecipes",
// 		Method:             "POST",
// 		PathPattern:        "/v1/recipes",
// 		ProducesMediaTypes: []string{"application/json"},
// 		ConsumesMediaTypes: []string{"application/json"},
// 		Schemes:            []string{"http"},
// 		Params:             params,
// 		Reader:             &RawResponder{},
// 		AuthInfo:           c.transport.DefaultAuthentication,
// 		Context:            params.Context,
// 		Client:             params.HTTPClient,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	platformIDs, err := model.FilterPlatformIDs(*hostPlatform, runtime.GOARCH, params.Order.Platforms)
// 	if err != nil {
// 		return nil, errs.Wrap(err, "filterPlatformIDs failed")
// 	}
// 	if len(platformIDs) == 0 {
// 		return nil, locale.NewInputError("err_recipe_no_platform")
// 	} else if len(platformIDs) > 1 {
// 		logging.Debug("Received multiple platform IDs.  Picking the first one.")
// 	}
// 	platformID := platformIDs[0]

// 	for _, recipe := range response.Payload.Recipes {
// 		if recipe.Platform != nil && recipe.Platform.PlatformID != nil && *recipe.Platform.PlatformID == platformID {
// 			return recipe, nil
// 		}
// 	}
// 	return result.(*inventory_operations.ResolveRecipesOK), nil
// }

// Init will create a new API client using default settings
func Init(auth *authentication.Auth) (inventory_operations.ClientService, runtime.ClientTransport) {
	return New(api.GetServiceURL(api.ServiceInventory), auth.ClientAuth())
}

// New initializes a new api client
func New(serviceURL *url.URL, auth runtime.ClientAuthInfoWriter) (inventory_operations.ClientService, runtime.ClientTransport) {
	transportRuntime := httptransport.New(serviceURL.Host, serviceURL.Path, []string{serviceURL.Scheme})
	transportRuntime.Transport = api.NewRoundTripper(http.DefaultTransport)

	// transportRuntime.SetDebug(true)

	if auth != nil {
		transportRuntime.DefaultAuthentication = auth
	}

	return inventory_operations.New(transportRuntime, strfmt.Default), transportRuntime
}

// Get returns a cached version of the default api client
func Get() inventory_operations.ClientService {
	if persist == nil {
		persist, _ = Init(authentication.LegacyGet())
	}
	return persist
}

type RecipesResponse struct {
	Recipes []interface{}
}

func ResolveRecipes(transport runtime.ClientTransport, params *iop.ResolveRecipesParams, authInfo runtime.ClientAuthInfoWriter) (string, error) {
	if params == nil {
		params = iop.NewResolveRecipesParams()
	}

	result, err := transport.Submit(&runtime.ClientOperation{
		ID:                 "resolveRecipes",
		Method:             "POST",
		PathPattern:        "/v1/recipes",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http"},
		Params:             params,
		Reader:             &RawResponder{},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	})
	if err != nil {
		return "", err
	}

	return string(result.([]byte)), nil
}

type RawResponder struct{}

func (r *RawResponder) ReadResponse(res runtime.ClientResponse, cons runtime.Consumer) (interface{}, error) {
	defer res.Body().Close()
	bytes, err := ioutil.ReadAll(res.Body())
	if err != nil {
		return nil, err
	}

	var umRecipe RecipesResponse
	err = json.Unmarshal(bytes, &umRecipe)
	if err != nil {
		return nil, err
	}

	if len(umRecipe.Recipes) == 0 {
		return nil, locale.NewError(locale.T("err_no_recipes"))
	}

	return json.Marshal(umRecipe.Recipes[0])
}
