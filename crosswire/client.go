package crosswire

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Policy struct {
	Owner                string
	Name                 string
	Entitlements         []Entitlement
	Condition            Condition
	SpecialApprover      *string
	ApprovalBehavior     *string
	UserApprovers        []string
	EntitlementApprovers []Entitlement
	Ttl                  *int64

	Id    string
	State string
}

type Condition struct {
	Quantifier    string
	Entitlements  []Entitlement
	Subconditions []Condition
}

type Entitlement struct {
	Provider, Subject, Object string
}

// HostURL - Default API endpoint
const HostURL string = "https://webhook.crosswire.io"

// Client -
type Client struct {
	HostURL    string
	HTTPClient *http.Client
	Token      string
}

// NewClient -
func NewClient(host, token *string) (*Client, error) {
	client := Client{
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		HostURL:    HostURL,
	}

	if host != nil {
		client.HostURL = *host
	}
	if token != nil {
		client.Token = *token
	}

	if success, err := client.Validate(); err != nil {
		return nil, err
	} else if !success {
		return nil, fmt.Errorf("client validation failed: ")
	}
	return &client, nil
}

func (c *Client) Validate() (bool, error) {
	if c.Token == "" {
		return false, fmt.Errorf("please enter a token")
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/integrations/crosswire_terraform/validate", c.HostURL), nil)
	if err != nil {
		return false, err
	}

	body, err := c.doRequest(req, nil)
	if err != nil {
		return false, err
	}

	if status, ok := body["success"]; ok {
		if success, ok := status.(bool); ok {
			return success, nil
		}
	} else {
		err := fmt.Sprintf("Received invalid response body: %+v\n", body)
		if trace, ok := body["traceId"]; ok {
			err = fmt.Sprintf("%sPlease use reference ID %s when requesting support.\n", err, trace)
		}
		return false, fmt.Errorf(err)
	}

	return false, nil
}

func (c *Client) createPolicy(policy Policy) (*Policy, error) {
	rb, err := json.Marshal(policy)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/integrations/crosswire_terraform/policy", c.HostURL), strings.NewReader(string(rb)))
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req, nil)
	if err != nil {
		return nil, err
	}

	return convertPolicy(body), nil
}

func (c *Client) getPolicy(label string) (*Policy, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("please enter a token")
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/integrations/crosswire_terraform/policy?label=%s", c.HostURL, url.QueryEscape(label)), nil)
	if err != nil {
		return nil, err
	}

	body, err := c.doRequest(req, nil)
	if err != nil {
		return nil, err
	}

	var foundPolicy *Policy
	if policies, ok := body["policies"]; ok {
		if policiesMap, ok := policies.(map[string]any); ok {
			if len(policiesMap) > 1 {
				return nil, fmt.Errorf("found %d policies. Expected 1 policy", len(policiesMap))
			}
			for id, policy := range policiesMap {
				if policyMap, ok := policy.(map[string]any); ok && id == policyMap["Id"].(string) {
					foundPolicy = convertPolicy(policyMap)
				}
			}
		}
	}

	return foundPolicy, nil
}

func convertPolicy(policyMap map[string]any) *Policy {
	policy := &Policy{
		Id:                   policyMap["Id"].(string),
		Owner:                policyMap["Owner"].(string),
		Name:                 policyMap["Name"].(string),
		State:                policyMap["State"].(string),
		Entitlements:         entMapSliceConverter(policyMap["Entitlements"].([]any)),
		Condition:            conditionConverter(policyMap["Condition"].(map[string]any)),
		SpecialApprover:      ToPointer(policyMap["SpecialApprover"].(string)),
		ApprovalBehavior:     ToPointer(policyMap["ApprovalBehavior"].(string)),
		UserApprovers:        interfaceSliceToStrings(policyMap["UserApprovers"].([]any)),
		EntitlementApprovers: entMapSliceConverter(policyMap["EntitlementApprovers"].([]any)),
	}
	if ttl, ok := policyMap["Ttl"].(int64); ok && ttl > 0 {
		policy.Ttl = &ttl
	}

	return policy
}

func conditionConverter(input map[string]any) (cond Condition) {
	if rawQuantifier, ok := input["Quantifier"]; ok {
		if quantifier, ok := rawQuantifier.(string); ok {
			cond.Quantifier = quantifier
		}
	}
	if rawEnts, ok := input["Entitlements"]; ok {
		if ents, ok := rawEnts.([]any); ok {
			cond.Entitlements = entMapSliceConverter(ents)
		}
	}
	if rawConds, ok := input["Subconditions"]; ok {
		if conds, ok := rawConds.([]any); ok {
			for _, rawCond := range conds {
				if subcondition, ok := rawCond.(map[string]any); ok {
					cond.Subconditions = append(cond.Subconditions, conditionConverter(subcondition))
				}
			}
		}
	}
	return
}

func entMapSliceConverter(input []any) (output []Entitlement) {
	for _, item := range input {
		output = append(output, entMapToEnt(item.(map[string]any)))
	}
	return
}

func entMapToEnt(input map[string]any) Entitlement {
	return Entitlement{
		Provider: input["Provider"].(string),
		Subject:  input["Subject"].(string),
		Object:   input["Object"].(string),
	}
}

func interfaceSliceToStrings(input []any) (output []string) {
	for _, item := range input {
		output = append(output, item.(string))
	}
	return
}

func (c *Client) doRequest(req *http.Request, authToken *string) (map[string]any, error) {
	var jsonData map[string]any

	req.Header.Set("Token", c.Token)
	if authToken != nil {
		req.Header.Set("Token", *authToken)
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		if header, ok := res.Header["X-Request-Id"]; ok && len(header) == 1 {
			return nil, fmt.Errorf("\nHTTP Response Code: %d\nTrace ID: %s\nDetails: %s", res.StatusCode, header[0], string(body))
		} else {
			return nil, fmt.Errorf("\nHTTP Response Code: %d\nDetails: %s", res.StatusCode, string(body))
		}
	}

	if err := json.Unmarshal(body, &jsonData); err != nil {
		if header, ok := res.Header["X-Request-Id"]; ok && len(header) == 1 {
			return nil, fmt.Errorf("\nHTTP Response Code: %d\nTrace ID: %s\nError: %s\nDetails: %s", res.StatusCode, header[0], err.Error(), string(body))
		} else {
			return nil, fmt.Errorf("\nHTTP Response Code: %d\nError: %s\nDetails: %s", res.StatusCode, err.Error(), string(body))
		}
	}

	if header, ok := res.Header["X-Request-Id"]; ok && len(header) == 1 {
		jsonData["traceId"] = header[0]
	}

	return jsonData, nil
}
