package mozzle

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

type appNotFoundError string

func (e appNotFoundError) Error() string {
	return fmt.Sprintf("application %s not found", string(e))
}

type operator string

type filter string

const (
	// OperatorEqual specifies that the Filter and the Value operands should
	// be equal.
	OperatorEqual operator = ":"
	// OperatorEqual specifies that the Filter operand should be greater than
	// the Value operand.
	OperatorGreater = ">"
)

const (
	// FilterActee specifies that the filter should be applied to the actee
	// field.
	FilterActee filter = "actee"
	// FilterTimestamp specifies that the filter should be applied to the
	// timestamp field.
	FilterTimestamp = "timestamp"
)

// query represents a query regarding a resource.
type query struct {
	// Filter specifies the resource's field to filter on.
	Filter filter
	// Operator specifies the operator applied when filtering.
	Operator operator
	// Value specifies the filter's value argument.
	Value string
}

// String returns the string representation of the query.
// It is of the form <filter><operator><value>.
func (q query) String() string {
	return string(q.Filter) + string(q.Operator) + q.Value
}

type cloudFoundry struct {
	client *cfclient.Client
}

// DopplerEndpoint returns the Doppler endpoint.
func (c *cloudFoundry) DopplerEndpoint() string {
	return c.client.Endpoint.DopplerEndpoint
}

// RefreshAuthToken returns new OAuth token.
func (c *cloudFoundry) RefreshAuthToken() (string, error) {
	return c.client.GetToken()
}

func (c *cloudFoundry) AppSummary(guid string) (appSummary, error) {
	path := fmt.Sprintf("/v2/apps/%s/summary", guid)
	req := c.client.NewRequest("GET", path)
	resp, err := c.client.DoRequest(req)
	if err != nil {
		return appSummary{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return appSummary{}, appNotFoundError(guid)
	}

	var s appSummary
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return appSummary{}, err
	}
	return s, nil
}

// Apps returns all applications under the specified target.
func (c *cloudFoundry) Apps(org, space string) ([]application, error) {
	targetSpace, err := c.space(org, space)
	if err != nil {
		return nil, err
	}

	apps, err := c.spaceApps(targetSpace.Guid)
	if err != nil {
		return nil, err
	}

	var result []application
	for _, app := range apps {
		result = append(result, application{
			Org:   org,
			Space: space,
			Guid:  app.Guid,
			Name:  app.Name,
		})
	}
	return result, nil
}

func (c *cloudFoundry) Events(queries ...query) ([]appEvent, error) {
	var query url.Values = make(map[string][]string)
	for _, q := range queries {
		query.Add("q", q.String())
	}
	path := fmt.Sprintf("/v2/events?%s", query.Encode())
	req := c.client.NewRequest("GET", path)
	resp, err := c.client.DoRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var events []appEvent
	err = json.NewDecoder(resp.Body).Decode(&events)
	return events, err
}

func (c *cloudFoundry) spaceApps(spaceGuid string) ([]cfclient.App, error) {
	// TODO(ivan): If the result is paginated, this will return only the
	// first page.
	spaceApps := make([]cfclient.App, 0)
	apps, err := c.client.ListApps()
	if err != nil {
		return nil, err
	}
	for _, app := range apps {
		if app.SpaceData.Meta.Guid == spaceGuid {
			spaceApps = append(spaceApps, app)
		}
	}
	return spaceApps, nil
}

func (c *cloudFoundry) space(orgName, spaceName string) (cfclient.Space, error) {
	org, err := c.org(orgName)
	if err != nil {
		return cfclient.Space{}, err
	}

	spaces, err := c.client.OrgSpaces(org.Guid)
	if err != nil {
		return cfclient.Space{}, err
	}

	for _, space := range spaces {
		if space.Name == spaceName {
			return space, nil
		}
	}
	return cfclient.Space{}, fmt.Errorf("space %s not found", spaceName)
}

func (c *cloudFoundry) org(name string) (cfclient.Org, error) {
	path := fmt.Sprintf("/v2/organizations?q=name:%s", name)
	req := c.client.NewRequest("GET", path)
	resp, err := c.client.DoRequest(req)
	if err != nil {
		return cfclient.Org{}, fmt.Errorf("error requesting organizations: %v", err)
	}
	defer resp.Body.Close()

	var orgResp cfclient.OrgResponse
	d := json.NewDecoder(resp.Body)
	if err := d.Decode(&orgResp); err != nil {
		return cfclient.Org{}, fmt.Errorf("error decoding response: %v", err)
	}

	if orgResp.Count == 0 {
		return cfclient.Org{}, fmt.Errorf("org %q not found", name)
	}
	if orgResp.Count > 1 {
		return cfclient.Org{}, fmt.Errorf("name %q does not refer to a single org", name)
	}

	return cfclient.Org{
		Guid: orgResp.Resources[0].Meta.Guid,
		Name: orgResp.Resources[0].Entity.Name,
	}, nil
}
