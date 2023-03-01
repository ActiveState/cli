package rtusage

import (
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/graphql"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/model"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

const cacheKey = "runtime-usage-"

// Checker is the struct that we use to do checks with
type Checker struct {
	config configurable
	cache  *cache.Cache
	auth   *authentication.Auth
}

// configurable defines the configuration function used by the functions in this package
type configurable interface {
	ConfigPath() string
	GetTime(key string) time.Time
	Set(key string, value interface{}) error
	Close() error
}

// NewChecker returns a new instance of the Checker struct
func NewChecker(configuration configurable, auth *authentication.Auth) *Checker {
	checker := &Checker{
		configuration,
		cache.New(1*time.Hour, 1*time.Hour),
		auth,
	}

	return checker
}

// Check will check the runtime usage for the given organization, it may return a cached result
func (c *Checker) Check(organizationName string) (*model.RuntimeUsage, error) {
	if cached, ok := c.cache.Get(cacheKey + organizationName); ok {
		return cached.(*model.RuntimeUsage), nil
	}

	if err := c.auth.Refresh(); err != nil {
		return nil, errs.Wrap(err, "Could not refresh authentication")
	}

	if !c.auth.Authenticated() {
		// Usage information can only be given to authenticated users, and the API doesn't support authentication errors
		// so we just don't even attempt it if not authenticated.
		return nil, nil
	}

	client := graphql.New()

	orgsResponse := model.Organizations{}
	if err := client.Run(request.OrganizationsByName(organizationName), &orgsResponse); err != nil {
		return nil, errs.Wrap(err, "Could not fetch organization: %s", organizationName)
	}
	if len(orgsResponse.Organizations) == 0 {
		return nil, errs.New("Could not find organization: %s", organizationName)
	}
	org := orgsResponse.Organizations[0]

	usageResponse := model.RuntimeUsageResponse{}
	if err := client.Run(request.RuntimeUsage(org.ID), &usageResponse); err != nil {
		return nil, errs.Wrap(err, "Could not fetch runtime usage information")
	}

	if len(usageResponse.Usage) == 0 {
		logging.Debug("No runtime usage information found for organization: %s", organizationName)
		return nil, nil
	}

	c.cache.Set(cacheKey+organizationName, &usageResponse.Usage[0], 0)

	return &usageResponse.Usage[0], nil
}
