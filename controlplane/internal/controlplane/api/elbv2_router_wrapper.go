package api

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// ELBv2RouterWrapper wraps the generated router to handle form data
type ELBv2RouterWrapper struct {
	innerRouter *generated_elbv2.Router
	api         generated_elbv2.ElasticLoadBalancing_v10API
}

// XML response structures for ELBv2 API
type DescribeLoadBalancersResponse struct {
	XMLName          xml.Name                    `xml:"DescribeLoadBalancersResponse"`
	XMLNS            string                      `xml:"xmlns,attr"`
	Result           DescribeLoadBalancersResult `xml:"DescribeLoadBalancersResult"`
	ResponseMetadata ResponseMetadata            `xml:"ResponseMetadata"`
}

type DescribeLoadBalancersResult struct {
	LoadBalancers []LoadBalancer `xml:"LoadBalancers>member"`
}

type CreateLoadBalancerResponse struct {
	XMLName          xml.Name                 `xml:"CreateLoadBalancerResponse"`
	XMLNS            string                   `xml:"xmlns,attr"`
	Result           CreateLoadBalancerResult `xml:"CreateLoadBalancerResult"`
	ResponseMetadata ResponseMetadata         `xml:"ResponseMetadata"`
}

type CreateLoadBalancerResult struct {
	LoadBalancers []LoadBalancer `xml:"LoadBalancers>member"`
}

type LoadBalancer struct {
	LoadBalancerArn       string `xml:"LoadBalancerArn"`
	DNSName               string `xml:"DNSName"`
	CanonicalHostedZoneId string `xml:"CanonicalHostedZoneId"`
	CreatedTime           string `xml:"CreatedTime"`
	LoadBalancerName      string `xml:"LoadBalancerName"`
	Scheme                string `xml:"Scheme"`
	VpcId                 string `xml:"VpcId"`
	State                 State  `xml:"State"`
	Type                  string `xml:"Type"`
	IpAddressType         string `xml:"IpAddressType"`
}

type State struct {
	Code string `xml:"Code"`
}

type ResponseMetadata struct {
	RequestId string `xml:"RequestId"`
}

// Target Group response structures
type CreateTargetGroupResponse struct {
	XMLName          xml.Name                `xml:"CreateTargetGroupResponse"`
	XMLNS            string                  `xml:"xmlns,attr"`
	Result           CreateTargetGroupResult `xml:"CreateTargetGroupResult"`
	ResponseMetadata ResponseMetadata        `xml:"ResponseMetadata"`
}

type CreateTargetGroupResult struct {
	TargetGroups []TargetGroup `xml:"TargetGroups>member"`
}

type DescribeTargetGroupsResponse struct {
	XMLName          xml.Name                   `xml:"DescribeTargetGroupsResponse"`
	XMLNS            string                     `xml:"xmlns,attr"`
	Result           DescribeTargetGroupsResult `xml:"DescribeTargetGroupsResult"`
	ResponseMetadata ResponseMetadata           `xml:"ResponseMetadata"`
}

type DescribeTargetGroupsResult struct {
	TargetGroups []TargetGroup `xml:"TargetGroups>member"`
}

type TargetGroup struct {
	TargetGroupArn             string `xml:"TargetGroupArn"`
	TargetGroupName            string `xml:"TargetGroupName"`
	Protocol                   string `xml:"Protocol"`
	Port                       int32  `xml:"Port"`
	VpcId                      string `xml:"VpcId"`
	HealthCheckEnabled         bool   `xml:"HealthCheckEnabled"`
	HealthCheckIntervalSeconds int32  `xml:"HealthCheckIntervalSeconds"`
	HealthCheckPath            string `xml:"HealthCheckPath"`
	HealthCheckPort            string `xml:"HealthCheckPort"`
	HealthCheckProtocol        string `xml:"HealthCheckProtocol"`
	HealthCheckTimeoutSeconds  int32  `xml:"HealthCheckTimeoutSeconds"`
	HealthyThresholdCount      int32  `xml:"HealthyThresholdCount"`
	UnhealthyThresholdCount    int32  `xml:"UnhealthyThresholdCount"`
	TargetType                 string `xml:"TargetType"`
}

// Listener response structures
type CreateListenerResponse struct {
	XMLName          xml.Name             `xml:"CreateListenerResponse"`
	XMLNS            string               `xml:"xmlns,attr"`
	Result           CreateListenerResult `xml:"CreateListenerResult"`
	ResponseMetadata ResponseMetadata     `xml:"ResponseMetadata"`
}

type CreateListenerResult struct {
	Listeners []Listener `xml:"Listeners>member"`
}

type DescribeListenersResponse struct {
	XMLName          xml.Name                `xml:"DescribeListenersResponse"`
	XMLNS            string                  `xml:"xmlns,attr"`
	Result           DescribeListenersResult `xml:"DescribeListenersResult"`
	ResponseMetadata ResponseMetadata        `xml:"ResponseMetadata"`
}

type DescribeListenersResult struct {
	Listeners []Listener `xml:"Listeners>member"`
}

type Listener struct {
	ListenerArn     string   `xml:"ListenerArn"`
	LoadBalancerArn string   `xml:"LoadBalancerArn"`
	Port            int32    `xml:"Port"`
	Protocol        string   `xml:"Protocol"`
	DefaultActions  []Action `xml:"DefaultActions>member"`
}

type Action struct {
	Type           string `xml:"Type"`
	TargetGroupArn string `xml:"TargetGroupArn"`
	Order          int32  `xml:"Order,omitempty"`
}

// RegisterTargets response structures
type RegisterTargetsResponse struct {
	XMLName          xml.Name         `xml:"RegisterTargetsResponse"`
	XMLNS            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// DeregisterTargets response structures
type DeregisterTargetsResponse struct {
	XMLName          xml.Name         `xml:"DeregisterTargetsResponse"`
	XMLNS            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// DescribeTargetHealth response structures
type DescribeTargetHealthResponse struct {
	XMLName          xml.Name                   `xml:"DescribeTargetHealthResponse"`
	XMLNS            string                     `xml:"xmlns,attr"`
	Result           DescribeTargetHealthResult `xml:"DescribeTargetHealthResult"`
	ResponseMetadata ResponseMetadata           `xml:"ResponseMetadata"`
}

type DescribeTargetHealthResult struct {
	TargetHealthDescriptions []TargetHealthDescription `xml:"TargetHealthDescriptions>member"`
}

type TargetHealthDescription struct {
	Target          Target       `xml:"Target"`
	HealthCheckPort string       `xml:"HealthCheckPort"`
	TargetHealth    TargetHealth `xml:"TargetHealth"`
}

type Target struct {
	Id               string `xml:"Id"`
	Port             int32  `xml:"Port,omitempty"`
	AvailabilityZone string `xml:"AvailabilityZone,omitempty"`
}

type TargetHealth struct {
	State       string `xml:"State"`
	Reason      string `xml:"Reason,omitempty"`
	Description string `xml:"Description,omitempty"`
}

// ModifyTargetGroup response structures
type ModifyTargetGroupResponse struct {
	XMLName          xml.Name                `xml:"ModifyTargetGroupResponse"`
	XMLNS            string                  `xml:"xmlns,attr"`
	Result           ModifyTargetGroupResult `xml:"ModifyTargetGroupResult"`
	ResponseMetadata ResponseMetadata        `xml:"ResponseMetadata"`
}

type ModifyTargetGroupResult struct {
	TargetGroups []TargetGroup `xml:"TargetGroups>member"`
}

// DeleteTargetGroup response structures
type DeleteTargetGroupResponse struct {
	XMLName          xml.Name         `xml:"DeleteTargetGroupResponse"`
	XMLNS            string           `xml:"xmlns,attr"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

// CreateRule response structures
type CreateRuleResponse struct {
	XMLName          xml.Name         `xml:"CreateRuleResponse"`
	XMLNS            string           `xml:"xmlns,attr"`
	Result           CreateRuleResult `xml:"CreateRuleResult"`
	ResponseMetadata ResponseMetadata `xml:"ResponseMetadata"`
}

type CreateRuleResult struct {
	Rules []Rule `xml:"Rules>member"`
}

type Rule struct {
	Actions    []Action        `xml:"Actions>member"`
	Conditions []RuleCondition `xml:"Conditions>member"`
	IsDefault  bool            `xml:"IsDefault"`
	Priority   string          `xml:"Priority"`
	RuleArn    string          `xml:"RuleArn"`
}

type RuleCondition struct {
	Field                   string                   `xml:"Field,omitempty"`
	HostHeaderConfig        *HostHeaderConfig        `xml:"HostHeaderConfig,omitempty"`
	HttpHeaderConfig        *HttpHeaderConfig        `xml:"HttpHeaderConfig,omitempty"`
	HttpRequestMethodConfig *HttpRequestMethodConfig `xml:"HttpRequestMethodConfig,omitempty"`
	PathPatternConfig       *PathPatternConfig       `xml:"PathPatternConfig,omitempty"`
	QueryStringConfig       *QueryStringConfig       `xml:"QueryStringConfig,omitempty"`
	SourceIpConfig          *SourceIpConfig          `xml:"SourceIpConfig,omitempty"`
	Values                  []string                 `xml:"Values>member,omitempty"`
}

type PathPatternConfig struct {
	Values []string `xml:"Values>member"`
}

type HostHeaderConfig struct {
	Values []string `xml:"Values>member"`
}

type HttpHeaderConfig struct {
	Values []string `xml:"Values>member"`
}

type HttpRequestMethodConfig struct {
	Values []string `xml:"Values>member"`
}

type QueryStringConfig struct {
	Values []QueryStringKeyValuePair `xml:"Values>member"`
}

type QueryStringKeyValuePair struct {
	Key   string `xml:"Key,omitempty"`
	Value string `xml:"Value,omitempty"`
}

type SourceIpConfig struct {
	Values []string `xml:"Values>member"`
}

// DescribeRules response structures
type DescribeRulesResponse struct {
	XMLName          xml.Name            `xml:"DescribeRulesResponse"`
	XMLNS            string              `xml:"xmlns,attr"`
	Result           DescribeRulesResult `xml:"DescribeRulesResult"`
	ResponseMetadata ResponseMetadata    `xml:"ResponseMetadata"`
}

type DescribeRulesResult struct {
	Rules      []Rule  `xml:"Rules>member"`
	NextMarker *string `xml:"NextMarker,omitempty"`
}

// DescribeLoadBalancerAttributes response structures
type DescribeLoadBalancerAttributesResponse struct {
	XMLName          xml.Name                             `xml:"DescribeLoadBalancerAttributesResponse"`
	XMLNS            string                               `xml:"xmlns,attr"`
	Result           DescribeLoadBalancerAttributesResult `xml:"DescribeLoadBalancerAttributesResult"`
	ResponseMetadata ResponseMetadata                     `xml:"ResponseMetadata"`
}

type DescribeLoadBalancerAttributesResult struct {
	Attributes []Attribute `xml:"Attributes>member"`
}

// DescribeTargetGroupAttributes response structures
type DescribeTargetGroupAttributesResponse struct {
	XMLName          xml.Name                            `xml:"DescribeTargetGroupAttributesResponse"`
	XMLNS            string                              `xml:"xmlns,attr"`
	Result           DescribeTargetGroupAttributesResult `xml:"DescribeTargetGroupAttributesResult"`
	ResponseMetadata ResponseMetadata                    `xml:"ResponseMetadata"`
}

type DescribeTargetGroupAttributesResult struct {
	Attributes []Attribute `xml:"Attributes>member"`
}

// ModifyTargetGroupAttributes response structures
type ModifyTargetGroupAttributesResponse struct {
	XMLName          xml.Name                          `xml:"ModifyTargetGroupAttributesResponse"`
	XMLNS            string                            `xml:"xmlns,attr"`
	Result           ModifyTargetGroupAttributesResult `xml:"ModifyTargetGroupAttributesResult"`
	ResponseMetadata ResponseMetadata                  `xml:"ResponseMetadata"`
}

type ModifyTargetGroupAttributesResult struct {
	Attributes []Attribute `xml:"Attributes>member"`
}

// Attribute represents a load balancer or target group attribute
type Attribute struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

// NewELBv2RouterWrapper creates a new wrapper for the ELBv2 router
func NewELBv2RouterWrapper(api generated_elbv2.ElasticLoadBalancing_v10API) *ELBv2RouterWrapper {
	return &ELBv2RouterWrapper{
		innerRouter: generated_elbv2.NewRouter(api),
		api:         api,
	}
}

// Route handles the HTTP request, converting form data to JSON when necessary
func (w *ELBv2RouterWrapper) Route(resp http.ResponseWriter, req *http.Request) {
	// Recover from any panics
	defer func() {
		if r := recover(); r != nil {
			logging.Error("Panic in ELBv2RouterWrapper.Route", "error", r)
			w.writeError(resp, http.StatusInternalServerError, "InternalError", fmt.Sprintf("Internal error: %v", r))
		}
	}()

	contentType := req.Header.Get("Content-Type")
	logging.Info("ELBv2RouterWrapper.Route called",
		"method", req.Method,
		"path", req.URL.Path,
		"content-type", contentType)

	// If it's form data, convert it to JSON format expected by the generated code
	isFormData := req.Method == "POST" && strings.Contains(contentType, "application/x-www-form-urlencoded")
	logging.Info("Checking if form data", "isFormData", isFormData, "method", req.Method, "contentType", contentType)

	if isFormData {
		// Parse form data
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			w.writeError(resp, http.StatusBadRequest, "InvalidRequest", fmt.Sprintf("Failed to read body: %v", err))
			return
		}
		req.Body.Close()

		values, err := url.ParseQuery(string(bodyBytes))
		if err != nil {
			w.writeError(resp, http.StatusBadRequest, "InvalidRequest", fmt.Sprintf("Failed to parse form data: %v", err))
			return
		}

		action := values.Get("Action")
		logging.Info("Processing ELBv2 form data request",
			"action", action,
			"body", string(bodyBytes))

		// Handle specific actions
		logging.Info("About to switch on action", "action", action, "length", len(action))
		switch action {
		case "DescribeLoadBalancers":
			// Convert form data to DescribeLoadBalancersInput
			input := &generated_elbv2.DescribeLoadBalancersInput{}

			// Parse LoadBalancerArns if present
			if arns := values["LoadBalancerArns.member.1"]; len(arns) > 0 {
				input.LoadBalancerArns = []string{}
				for i := 1; ; i++ {
					key := fmt.Sprintf("LoadBalancerArns.member.%d", i)
					if val := values.Get(key); val != "" {
						input.LoadBalancerArns = append(input.LoadBalancerArns, val)
					} else {
						break
					}
				}
			}

			// Parse Names if present
			if names := values["Names.member.1"]; len(names) > 0 {
				input.Names = []string{}
				for i := 1; ; i++ {
					key := fmt.Sprintf("Names.member.%d", i)
					if val := values.Get(key); val != "" {
						input.Names = append(input.Names, val)
					} else {
						break
					}
				}
			}

			// Parse Marker if present
			if marker := values.Get("Marker"); marker != "" {
				input.Marker = &marker
			}

			// Parse PageSize if present
			if pageSize := values.Get("PageSize"); pageSize != "" {
				// Note: Would need to convert string to int32
			}

			// Call the API directly
			output, err := w.api.DescribeLoadBalancers(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			logging.Info("Converting DescribeLoadBalancers response to XML")
			xmlResp := w.convertDescribeLoadBalancersToXML(output)
			logging.Info("Writing XML response")
			w.writeXML(resp, xmlResp)
			return

		case "CreateLoadBalancer":
			// Convert form data to CreateLoadBalancerInput
			input := &generated_elbv2.CreateLoadBalancerInput{}

			// Parse Name
			input.Name = values.Get("Name")

			// Parse Subnets
			if subnets := values["Subnets.member.1"]; len(subnets) > 0 {
				input.Subnets = []string{}
				for i := 1; ; i++ {
					key := fmt.Sprintf("Subnets.member.%d", i)
					if val := values.Get(key); val != "" {
						input.Subnets = append(input.Subnets, val)
					} else {
						break
					}
				}
			}

			// Parse SecurityGroups
			if sgs := values["SecurityGroups.member.1"]; len(sgs) > 0 {
				input.SecurityGroups = []string{}
				for i := 1; ; i++ {
					key := fmt.Sprintf("SecurityGroups.member.%d", i)
					if val := values.Get(key); val != "" {
						input.SecurityGroups = append(input.SecurityGroups, val)
					} else {
						break
					}
				}
			}

			// Parse Scheme
			if scheme := values.Get("Scheme"); scheme != "" {
				schemeEnum := generated_elbv2.LoadBalancerSchemeEnum(scheme)
				input.Scheme = &schemeEnum
			}

			// Parse Type
			if lbType := values.Get("Type"); lbType != "" {
				typeEnum := generated_elbv2.LoadBalancerTypeEnum(lbType)
				input.Type = &typeEnum
			}

			// Parse IpAddressType
			if ipType := values.Get("IpAddressType"); ipType != "" {
				ipTypeEnum := generated_elbv2.IpAddressType(ipType)
				input.IpAddressType = &ipTypeEnum
			}

			// Call the API directly
			output, err := w.api.CreateLoadBalancer(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			xmlResp := w.convertCreateLoadBalancerToXML(output)
			w.writeXML(resp, xmlResp)
			return

		case "CreateTargetGroup":
			// Convert form data to CreateTargetGroupInput
			input := &generated_elbv2.CreateTargetGroupInput{}

			// Parse Name (required)
			input.Name = values.Get("Name")

			// Parse Protocol
			if protocol := values.Get("Protocol"); protocol != "" {
				protocolEnum := generated_elbv2.ProtocolEnum(protocol)
				input.Protocol = &protocolEnum
			}

			// Parse Port
			if portStr := values.Get("Port"); portStr != "" {
				port, _ := strconv.Atoi(portStr)
				port32 := int32(port)
				input.Port = &port32
			}

			// Parse VpcId
			if vpcId := values.Get("VpcId"); vpcId != "" {
				input.VpcId = &vpcId
			}

			// Parse TargetType
			if targetType := values.Get("TargetType"); targetType != "" {
				targetTypeEnum := generated_elbv2.TargetTypeEnum(targetType)
				input.TargetType = &targetTypeEnum
			}

			// Parse HealthCheckEnabled
			if healthCheckEnabled := values.Get("HealthCheckEnabled"); healthCheckEnabled != "" {
				enabled := healthCheckEnabled == "true"
				input.HealthCheckEnabled = &enabled
			}

			// Parse HealthCheckPath
			if healthCheckPath := values.Get("HealthCheckPath"); healthCheckPath != "" {
				input.HealthCheckPath = &healthCheckPath
			}

			// Parse HealthCheckIntervalSeconds
			if intervalStr := values.Get("HealthCheckIntervalSeconds"); intervalStr != "" {
				interval, _ := strconv.Atoi(intervalStr)
				interval32 := int32(interval)
				input.HealthCheckIntervalSeconds = &interval32
			}

			// Parse HealthCheckTimeoutSeconds
			if timeoutStr := values.Get("HealthCheckTimeoutSeconds"); timeoutStr != "" {
				timeout, _ := strconv.Atoi(timeoutStr)
				timeout32 := int32(timeout)
				input.HealthCheckTimeoutSeconds = &timeout32
			}

			// Parse HealthyThresholdCount
			if healthyStr := values.Get("HealthyThresholdCount"); healthyStr != "" {
				healthy, _ := strconv.Atoi(healthyStr)
				healthy32 := int32(healthy)
				input.HealthyThresholdCount = &healthy32
			}

			// Parse UnhealthyThresholdCount
			if unhealthyStr := values.Get("UnhealthyThresholdCount"); unhealthyStr != "" {
				unhealthy, _ := strconv.Atoi(unhealthyStr)
				unhealthy32 := int32(unhealthy)
				input.UnhealthyThresholdCount = &unhealthy32
			}

			// Parse Matcher (HttpCode)
			if matcherCode := values.Get("Matcher.HttpCode"); matcherCode != "" {
				matcher := &generated_elbv2.Matcher{
					HttpCode: &matcherCode,
				}
				input.Matcher = matcher
			}

			// Parse Tags
			tags := []generated_elbv2.Tag{}
			for i := 1; ; i++ {
				keyParam := fmt.Sprintf("Tags.member.%d.Key", i)
				valueParam := fmt.Sprintf("Tags.member.%d.Value", i)
				key := values.Get(keyParam)
				value := values.Get(valueParam)
				if key == "" {
					break
				}
				tags = append(tags, generated_elbv2.Tag{
					Key:   key,
					Value: &value,
				})
			}
			if len(tags) > 0 {
				input.Tags = tags
			}

			// Call the API
			logging.Info("Calling CreateTargetGroup API", "input", input)
			output, err := w.api.CreateTargetGroup(req.Context(), input)
			if err != nil {
				logging.Error("CreateTargetGroup API failed", "error", err)
				w.writeAPIError(resp, err)
				return
			}
			logging.Info("CreateTargetGroup API succeeded", "output", output)

			// Convert to XML response
			xmlResp := w.convertCreateTargetGroupToXML(output)
			w.writeXML(resp, xmlResp)
			return

		case "DescribeTargetGroups":
			// Convert form data to DescribeTargetGroupsInput
			input := &generated_elbv2.DescribeTargetGroupsInput{}

			// Parse TargetGroupArns if present
			if arns := values["TargetGroupArns.member.1"]; len(arns) > 0 {
				input.TargetGroupArns = []string{}
				for i := 1; ; i++ {
					key := fmt.Sprintf("TargetGroupArns.member.%d", i)
					if val := values.Get(key); val != "" {
						input.TargetGroupArns = append(input.TargetGroupArns, val)
					} else {
						break
					}
				}
			}

			// Parse Names if present
			if names := values["Names.member.1"]; len(names) > 0 {
				input.Names = []string{}
				for i := 1; ; i++ {
					key := fmt.Sprintf("Names.member.%d", i)
					if val := values.Get(key); val != "" {
						input.Names = append(input.Names, val)
					} else {
						break
					}
				}
			}

			// Call the API
			output, err := w.api.DescribeTargetGroups(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			xmlResp := w.convertDescribeTargetGroupsToXML(output)
			w.writeXML(resp, xmlResp)
			return

		case "CreateListener":
			// Convert form data to CreateListenerInput
			input := &generated_elbv2.CreateListenerInput{}

			// Parse LoadBalancerArn (required)
			input.LoadBalancerArn = values.Get("LoadBalancerArn")

			// Parse Protocol (required)
			if protocol := values.Get("Protocol"); protocol != "" {
				protocolEnum := generated_elbv2.ProtocolEnum(protocol)
				input.Protocol = &protocolEnum
			}

			// Parse Port (required)
			if portStr := values.Get("Port"); portStr != "" {
				port, _ := strconv.Atoi(portStr)
				port32 := int32(port)
				input.Port = &port32
			}

			// Parse DefaultActions
			if actionType := values.Get("DefaultActions.member.1.Type"); actionType != "" {
				actions := []generated_elbv2.Action{}
				for i := 1; ; i++ {
					typeKey := fmt.Sprintf("DefaultActions.member.%d.Type", i)
					if actionType := values.Get(typeKey); actionType == "" {
						break
					} else {
						action := generated_elbv2.Action{}
						actionTypeEnum := generated_elbv2.ActionTypeEnum(actionType)
						action.Type = actionTypeEnum

						// Parse TargetGroupArn for forward action
						if tgArn := values.Get(fmt.Sprintf("DefaultActions.member.%d.TargetGroupArn", i)); tgArn != "" {
							action.TargetGroupArn = &tgArn
						}

						actions = append(actions, action)
					}
				}
				input.DefaultActions = actions
			}

			// Call the API
			output, err := w.api.CreateListener(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			xmlResp := w.convertCreateListenerToXML(output)
			w.writeXML(resp, xmlResp)
			return

		case "DescribeListeners":
			// Convert form data to DescribeListenersInput
			input := &generated_elbv2.DescribeListenersInput{}

			// Parse LoadBalancerArn
			if lbArn := values.Get("LoadBalancerArn"); lbArn != "" {
				input.LoadBalancerArn = &lbArn
			}

			// Parse ListenerArns if present
			if arns := values["ListenerArns.member.1"]; len(arns) > 0 {
				input.ListenerArns = []string{}
				for i := 1; ; i++ {
					key := fmt.Sprintf("ListenerArns.member.%d", i)
					if val := values.Get(key); val != "" {
						input.ListenerArns = append(input.ListenerArns, val)
					} else {
						break
					}
				}
			}

			// Call the API
			output, err := w.api.DescribeListeners(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			xmlResp := w.convertDescribeListenersToXML(output)
			w.writeXML(resp, xmlResp)
			return

		case "RegisterTargets":
			// Convert form data to RegisterTargetsInput
			input := &generated_elbv2.RegisterTargetsInput{}

			// Parse TargetGroupArn (required)
			input.TargetGroupArn = values.Get("TargetGroupArn")

			// Parse Targets using helper function
			input.Targets = w.parseTargets(values)

			// Call the API
			output, err := w.api.RegisterTargets(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			xmlResp := w.convertRegisterTargetsToXML(output)
			w.writeXML(resp, xmlResp)
			return

		case "DeregisterTargets":
			// Convert form data to DeregisterTargetsInput
			input := &generated_elbv2.DeregisterTargetsInput{}

			// Parse TargetGroupArn (required)
			input.TargetGroupArn = values.Get("TargetGroupArn")

			// Parse Targets using helper function
			input.Targets = w.parseTargets(values)

			// Call the API
			output, err := w.api.DeregisterTargets(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			xmlResp := w.convertDeregisterTargetsToXML(output)
			w.writeXML(resp, xmlResp)
			return

		case "DescribeTargetHealth":
			// Convert form data to DescribeTargetHealthInput
			input := &generated_elbv2.DescribeTargetHealthInput{}

			// Parse TargetGroupArn (required)
			input.TargetGroupArn = values.Get("TargetGroupArn")

			// Parse Targets if present using helper function
			if values.Get("Targets.member.1.Id") != "" {
				input.Targets = w.parseTargets(values)
			}

			// Call the API
			output, err := w.api.DescribeTargetHealth(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			xmlResp := w.convertDescribeTargetHealthToXML(output)
			w.writeXML(resp, xmlResp)
			return

		case "ModifyTargetGroup":
			// Convert form data to ModifyTargetGroupInput
			input := &generated_elbv2.ModifyTargetGroupInput{}

			// Parse TargetGroupArn (required)
			input.TargetGroupArn = values.Get("TargetGroupArn")

			// Parse HealthCheckEnabled
			if enabled := values.Get("HealthCheckEnabled"); enabled != "" {
				boolVal := enabled == "true"
				input.HealthCheckEnabled = &boolVal
			}

			// Parse HealthCheckIntervalSeconds
			if intervalStr := values.Get("HealthCheckIntervalSeconds"); intervalStr != "" {
				interval, _ := strconv.Atoi(intervalStr)
				interval32 := int32(interval)
				input.HealthCheckIntervalSeconds = &interval32
			}

			// Parse HealthCheckPath
			if path := values.Get("HealthCheckPath"); path != "" {
				input.HealthCheckPath = &path
			}

			// Parse HealthCheckPort
			if port := values.Get("HealthCheckPort"); port != "" {
				input.HealthCheckPort = &port
			}

			// Parse HealthCheckProtocol
			if protocol := values.Get("HealthCheckProtocol"); protocol != "" {
				protocolEnum := generated_elbv2.ProtocolEnum(protocol)
				input.HealthCheckProtocol = &protocolEnum
			}

			// Parse HealthCheckTimeoutSeconds
			if timeoutStr := values.Get("HealthCheckTimeoutSeconds"); timeoutStr != "" {
				timeout, _ := strconv.Atoi(timeoutStr)
				timeout32 := int32(timeout)
				input.HealthCheckTimeoutSeconds = &timeout32
			}

			// Parse HealthyThresholdCount
			if thresholdStr := values.Get("HealthyThresholdCount"); thresholdStr != "" {
				threshold, _ := strconv.Atoi(thresholdStr)
				threshold32 := int32(threshold)
				input.HealthyThresholdCount = &threshold32
			}

			// Parse UnhealthyThresholdCount
			if thresholdStr := values.Get("UnhealthyThresholdCount"); thresholdStr != "" {
				threshold, _ := strconv.Atoi(thresholdStr)
				threshold32 := int32(threshold)
				input.UnhealthyThresholdCount = &threshold32
			}

			// Call the API
			output, err := w.api.ModifyTargetGroup(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			xmlResp := w.convertModifyTargetGroupToXML(output)
			w.writeXML(resp, xmlResp)
			return

		case "DeleteTargetGroup":
			// Convert form data to DeleteTargetGroupInput
			input := &generated_elbv2.DeleteTargetGroupInput{}

			// Parse TargetGroupArn (required)
			input.TargetGroupArn = values.Get("TargetGroupArn")

			// Call the API
			output, err := w.api.DeleteTargetGroup(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			xmlResp := w.convertDeleteTargetGroupToXML(output)
			w.writeXML(resp, xmlResp)
			return

		case "CreateRule":
			// Convert form data to CreateRuleInput
			input := &generated_elbv2.CreateRuleInput{}

			// Parse ListenerArn (required)
			input.ListenerArn = values.Get("ListenerArn")

			// Parse Priority (required)
			if priorityStr := values.Get("Priority"); priorityStr != "" {
				priority, _ := strconv.Atoi(priorityStr)
				input.Priority = int32(priority)
			}

			// Parse Conditions
			conditions := []generated_elbv2.RuleCondition{}
			for i := 1; ; i++ {
				fieldKey := fmt.Sprintf("Conditions.member.%d.Field", i)
				if field := values.Get(fieldKey); field != "" {
					condition := generated_elbv2.RuleCondition{
						Field: &field,
					}

					// Parse Values for this condition
					conditionValues := []string{}
					for j := 1; ; j++ {
						valueKey := fmt.Sprintf("Conditions.member.%d.Values.member.%d", i, j)
						if value := values.Get(valueKey); value != "" {
							conditionValues = append(conditionValues, value)
						} else {
							break
						}
					}
					condition.Values = conditionValues

					// Parse PathPatternConfig if present
					if pathValues := values.Get(fmt.Sprintf("Conditions.member.%d.PathPatternConfig.Values.member.1", i)); pathValues != "" {
						pathPatternValues := []string{}
						for j := 1; ; j++ {
							pathKey := fmt.Sprintf("Conditions.member.%d.PathPatternConfig.Values.member.%d", i, j)
							if value := values.Get(pathKey); value != "" {
								pathPatternValues = append(pathPatternValues, value)
							} else {
								break
							}
						}
						condition.PathPatternConfig = &generated_elbv2.PathPatternConditionConfig{
							Values: pathPatternValues,
						}
					}

					conditions = append(conditions, condition)
				} else {
					break
				}
			}
			input.Conditions = conditions

			// Parse Actions
			actions := []generated_elbv2.Action{}
			for i := 1; ; i++ {
				typeKey := fmt.Sprintf("Actions.member.%d.Type", i)
				if actionType := values.Get(typeKey); actionType != "" {
					action := generated_elbv2.Action{}
					actionTypeEnum := generated_elbv2.ActionTypeEnum(actionType)
					action.Type = actionTypeEnum

					// Parse TargetGroupArn for forward action
					if tgArn := values.Get(fmt.Sprintf("Actions.member.%d.TargetGroupArn", i)); tgArn != "" {
						action.TargetGroupArn = &tgArn
					}

					// Parse Order if present
					if orderStr := values.Get(fmt.Sprintf("Actions.member.%d.Order", i)); orderStr != "" {
						order, _ := strconv.Atoi(orderStr)
						order32 := int32(order)
						action.Order = &order32
					}

					actions = append(actions, action)
				} else {
					break
				}
			}
			input.Actions = actions

			// Call the API
			output, err := w.api.CreateRule(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			xmlResp := w.convertCreateRuleToXML(output)
			w.writeXML(resp, xmlResp)
			return

		case "DescribeRules":
			// Convert form data to DescribeRulesInput
			input := &generated_elbv2.DescribeRulesInput{}

			// Parse ListenerArn
			if listenerArn := values.Get("ListenerArn"); listenerArn != "" {
				input.ListenerArn = &listenerArn
			}

			// Parse RuleArns if present
			if arns := values["RuleArns.member.1"]; len(arns) > 0 {
				input.RuleArns = []string{}
				for i := 1; ; i++ {
					key := fmt.Sprintf("RuleArns.member.%d", i)
					if val := values.Get(key); val != "" {
						input.RuleArns = append(input.RuleArns, val)
					} else {
						break
					}
				}
			}

			// Parse PageSize if present
			if pageSizeStr := values.Get("PageSize"); pageSizeStr != "" {
				pageSize, _ := strconv.Atoi(pageSizeStr)
				pageSize32 := int32(pageSize)
				input.PageSize = &pageSize32
			}

			// Parse Marker if present
			if marker := values.Get("Marker"); marker != "" {
				input.Marker = &marker
			}

			// Call the API
			output, err := w.api.DescribeRules(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			xmlResp := w.convertDescribeRulesToXML(output)
			w.writeXML(resp, xmlResp)
			return

		case "DescribeLoadBalancerAttributes":
			// Convert form data to DescribeLoadBalancerAttributesInput
			input := &generated_elbv2.DescribeLoadBalancerAttributesInput{}

			// Parse LoadBalancerArn (required)
			input.LoadBalancerArn = values.Get("LoadBalancerArn")

			// Call the API
			output, err := w.api.DescribeLoadBalancerAttributes(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			xmlResp := w.convertDescribeLoadBalancerAttributesToXML(output)
			w.writeXML(resp, xmlResp)
			return

		case "DescribeTargetGroupAttributes":
			// Convert form data to DescribeTargetGroupAttributesInput
			input := &generated_elbv2.DescribeTargetGroupAttributesInput{}

			// Parse TargetGroupArn (required)
			input.TargetGroupArn = values.Get("TargetGroupArn")

			// Call the API
			output, err := w.api.DescribeTargetGroupAttributes(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			xmlResp := w.convertDescribeTargetGroupAttributesToXML(output)
			w.writeXML(resp, xmlResp)
			return

		case "ModifyTargetGroupAttributes":
			// Convert form data to ModifyTargetGroupAttributesInput
			input := &generated_elbv2.ModifyTargetGroupAttributesInput{}

			// Parse TargetGroupArn (required)
			input.TargetGroupArn = values.Get("TargetGroupArn")

			// Parse Attributes
			attributes := []generated_elbv2.TargetGroupAttribute{}
			for i := 1; ; i++ {
				keyParam := fmt.Sprintf("Attributes.member.%d.Key", i)
				valueParam := fmt.Sprintf("Attributes.member.%d.Value", i)
				key := values.Get(keyParam)
				value := values.Get(valueParam)
				if key == "" {
					break
				}
				attributes = append(attributes, generated_elbv2.TargetGroupAttribute{
					Key:   &key,
					Value: &value,
				})
			}
			input.Attributes = attributes

			// Call the API
			output, err := w.api.ModifyTargetGroupAttributes(req.Context(), input)
			if err != nil {
				w.writeAPIError(resp, err)
				return
			}

			// Convert to XML response
			xmlResp := w.convertModifyTargetGroupAttributesToXML(output)
			w.writeXML(resp, xmlResp)
			return

		default:
			logging.Info("Handling default case for action", "action", action)
			// For other actions, return an error for now
			w.writeError(resp, http.StatusNotImplemented, "NotImplemented", fmt.Sprintf("Action %s not yet implemented", action))
			return
		}
	} else {
		// For non-form data requests, delegate to the inner router
		w.innerRouter.Route(resp, req)
	}
}

// writeError writes an error response
func (w *ELBv2RouterWrapper) writeError(resp http.ResponseWriter, statusCode int, errorCode, message string) {
	resp.Header().Set("Content-Type", "application/x-amz-json-1.1")
	resp.WriteHeader(statusCode)
	json.NewEncoder(resp).Encode(map[string]string{
		"__type":  errorCode,
		"message": message,
	})
}

// writeAPIError writes an API error response
func (w *ELBv2RouterWrapper) writeAPIError(resp http.ResponseWriter, err error) {
	// Default to internal server error
	w.writeError(resp, http.StatusInternalServerError, "InternalError", err.Error())
}

// writeJSON writes a JSON response
func (w *ELBv2RouterWrapper) writeJSON(resp http.ResponseWriter, data interface{}) {
	resp.Header().Set("Content-Type", "application/x-amz-json-1.1")
	resp.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(resp).Encode(data); err != nil {
		logging.Error("Failed to encode response", "error", err)
	}
}

// writeXML writes an XML response
func (w *ELBv2RouterWrapper) writeXML(resp http.ResponseWriter, data interface{}) {
	resp.Header().Set("Content-Type", "text/xml")
	resp.WriteHeader(http.StatusOK)

	// Write XML declaration
	resp.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"))

	// Encode the response
	encoder := xml.NewEncoder(resp)
	encoder.Indent("", "  ")
	if err := encoder.Encode(data); err != nil {
		logging.Error("Failed to encode XML response", "error", err)
	}
}

// parseTargets parses target descriptions from form values
func (w *ELBv2RouterWrapper) parseTargets(values url.Values) []generated_elbv2.TargetDescription {
	targets := []generated_elbv2.TargetDescription{}
	for i := 1; ; i++ {
		idKey := fmt.Sprintf("Targets.member.%d.Id", i)
		if id := values.Get(idKey); id != "" {
			target := generated_elbv2.TargetDescription{
				Id: id,
			}

			// Parse Port if present
			if portStr := values.Get(fmt.Sprintf("Targets.member.%d.Port", i)); portStr != "" {
				if port, err := strconv.Atoi(portStr); err == nil {
					port32 := int32(port)
					target.Port = &port32
				}
			}

			// Parse AvailabilityZone if present
			if az := values.Get(fmt.Sprintf("Targets.member.%d.AvailabilityZone", i)); az != "" {
				target.AvailabilityZone = &az
			}

			targets = append(targets, target)
		} else {
			break
		}
	}
	return targets
}

// convertDescribeLoadBalancersToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertDescribeLoadBalancersToXML(output *generated_elbv2.DescribeLoadBalancersOutput) *DescribeLoadBalancersResponse {
	resp := &DescribeLoadBalancersResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	if output != nil && output.LoadBalancers != nil {
		for _, lb := range output.LoadBalancers {
			xmlLB := LoadBalancer{}
			if lb.LoadBalancerArn != nil {
				xmlLB.LoadBalancerArn = *lb.LoadBalancerArn
			}
			if lb.DNSName != nil {
				xmlLB.DNSName = *lb.DNSName
			}
			if lb.CanonicalHostedZoneId != nil {
				xmlLB.CanonicalHostedZoneId = *lb.CanonicalHostedZoneId
			}
			if lb.CreatedTime != nil {
				xmlLB.CreatedTime = lb.CreatedTime.Format(time.RFC3339)
			}
			if lb.LoadBalancerName != nil {
				xmlLB.LoadBalancerName = *lb.LoadBalancerName
			}
			if lb.Scheme != nil {
				xmlLB.Scheme = string(*lb.Scheme)
			}
			if lb.VpcId != nil {
				xmlLB.VpcId = *lb.VpcId
			}
			if lb.State != nil && lb.State.Code != nil {
				xmlLB.State = State{Code: string(*lb.State.Code)}
			}
			if lb.Type != nil {
				xmlLB.Type = string(*lb.Type)
			}
			if lb.IpAddressType != nil {
				xmlLB.IpAddressType = string(*lb.IpAddressType)
			}
			resp.Result.LoadBalancers = append(resp.Result.LoadBalancers, xmlLB)
		}
	}

	return resp
}

// convertCreateLoadBalancerToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertCreateLoadBalancerToXML(output *generated_elbv2.CreateLoadBalancerOutput) *CreateLoadBalancerResponse {
	resp := &CreateLoadBalancerResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	if output != nil && output.LoadBalancers != nil {
		for _, lb := range output.LoadBalancers {
			xmlLB := LoadBalancer{}
			if lb.LoadBalancerArn != nil {
				xmlLB.LoadBalancerArn = *lb.LoadBalancerArn
			}
			if lb.DNSName != nil {
				xmlLB.DNSName = *lb.DNSName
			}
			if lb.CanonicalHostedZoneId != nil {
				xmlLB.CanonicalHostedZoneId = *lb.CanonicalHostedZoneId
			}
			if lb.CreatedTime != nil {
				xmlLB.CreatedTime = lb.CreatedTime.Format(time.RFC3339)
			}
			if lb.LoadBalancerName != nil {
				xmlLB.LoadBalancerName = *lb.LoadBalancerName
			}
			if lb.Scheme != nil {
				xmlLB.Scheme = string(*lb.Scheme)
			}
			if lb.VpcId != nil {
				xmlLB.VpcId = *lb.VpcId
			}
			if lb.State != nil && lb.State.Code != nil {
				xmlLB.State = State{Code: string(*lb.State.Code)}
			}
			if lb.Type != nil {
				xmlLB.Type = string(*lb.Type)
			}
			if lb.IpAddressType != nil {
				xmlLB.IpAddressType = string(*lb.IpAddressType)
			}
			resp.Result.LoadBalancers = append(resp.Result.LoadBalancers, xmlLB)
		}
	}

	return resp
}

// convertCreateTargetGroupToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertCreateTargetGroupToXML(output *generated_elbv2.CreateTargetGroupOutput) *CreateTargetGroupResponse {
	resp := &CreateTargetGroupResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	if output != nil && output.TargetGroups != nil {
		for _, tg := range output.TargetGroups {
			xmlTG := TargetGroup{}
			if tg.TargetGroupArn != nil {
				xmlTG.TargetGroupArn = *tg.TargetGroupArn
			}
			if tg.TargetGroupName != nil {
				xmlTG.TargetGroupName = *tg.TargetGroupName
			}
			if tg.Protocol != nil {
				xmlTG.Protocol = string(*tg.Protocol)
			}
			if tg.Port != nil {
				xmlTG.Port = int32(*tg.Port)
			}
			if tg.VpcId != nil {
				xmlTG.VpcId = *tg.VpcId
			}
			if tg.HealthCheckEnabled != nil {
				xmlTG.HealthCheckEnabled = *tg.HealthCheckEnabled
			}
			if tg.HealthCheckIntervalSeconds != nil {
				xmlTG.HealthCheckIntervalSeconds = int32(*tg.HealthCheckIntervalSeconds)
			}
			if tg.HealthCheckTimeoutSeconds != nil {
				xmlTG.HealthCheckTimeoutSeconds = int32(*tg.HealthCheckTimeoutSeconds)
			}
			if tg.HealthyThresholdCount != nil {
				xmlTG.HealthyThresholdCount = int32(*tg.HealthyThresholdCount)
			}
			if tg.UnhealthyThresholdCount != nil {
				xmlTG.UnhealthyThresholdCount = int32(*tg.UnhealthyThresholdCount)
			}
			if tg.HealthCheckPath != nil {
				xmlTG.HealthCheckPath = *tg.HealthCheckPath
			}
			if tg.HealthCheckPort != nil {
				xmlTG.HealthCheckPort = *tg.HealthCheckPort
			}
			if tg.HealthCheckProtocol != nil {
				xmlTG.HealthCheckProtocol = string(*tg.HealthCheckProtocol)
			}
			if tg.TargetType != nil {
				xmlTG.TargetType = string(*tg.TargetType)
			}
			resp.Result.TargetGroups = append(resp.Result.TargetGroups, xmlTG)
		}
	}

	return resp
}

// convertDescribeTargetGroupsToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertDescribeTargetGroupsToXML(output *generated_elbv2.DescribeTargetGroupsOutput) *DescribeTargetGroupsResponse {
	resp := &DescribeTargetGroupsResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	if output != nil && output.TargetGroups != nil {
		for _, tg := range output.TargetGroups {
			xmlTG := TargetGroup{}
			if tg.TargetGroupArn != nil {
				xmlTG.TargetGroupArn = *tg.TargetGroupArn
			}
			if tg.TargetGroupName != nil {
				xmlTG.TargetGroupName = *tg.TargetGroupName
			}
			if tg.Protocol != nil {
				xmlTG.Protocol = string(*tg.Protocol)
			}
			if tg.Port != nil {
				xmlTG.Port = int32(*tg.Port)
			}
			if tg.VpcId != nil {
				xmlTG.VpcId = *tg.VpcId
			}
			if tg.HealthCheckEnabled != nil {
				xmlTG.HealthCheckEnabled = *tg.HealthCheckEnabled
			}
			if tg.HealthCheckIntervalSeconds != nil {
				xmlTG.HealthCheckIntervalSeconds = int32(*tg.HealthCheckIntervalSeconds)
			}
			if tg.HealthCheckTimeoutSeconds != nil {
				xmlTG.HealthCheckTimeoutSeconds = int32(*tg.HealthCheckTimeoutSeconds)
			}
			if tg.HealthyThresholdCount != nil {
				xmlTG.HealthyThresholdCount = int32(*tg.HealthyThresholdCount)
			}
			if tg.UnhealthyThresholdCount != nil {
				xmlTG.UnhealthyThresholdCount = int32(*tg.UnhealthyThresholdCount)
			}
			if tg.HealthCheckPath != nil {
				xmlTG.HealthCheckPath = *tg.HealthCheckPath
			}
			if tg.HealthCheckPort != nil {
				xmlTG.HealthCheckPort = *tg.HealthCheckPort
			}
			if tg.HealthCheckProtocol != nil {
				xmlTG.HealthCheckProtocol = string(*tg.HealthCheckProtocol)
			}
			if tg.TargetType != nil {
				xmlTG.TargetType = string(*tg.TargetType)
			}
			resp.Result.TargetGroups = append(resp.Result.TargetGroups, xmlTG)
		}
	}

	return resp
}

// convertCreateListenerToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertCreateListenerToXML(output *generated_elbv2.CreateListenerOutput) *CreateListenerResponse {
	resp := &CreateListenerResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	if output != nil && output.Listeners != nil {
		for _, l := range output.Listeners {
			xmlListener := Listener{}
			if l.ListenerArn != nil {
				xmlListener.ListenerArn = *l.ListenerArn
			}
			if l.LoadBalancerArn != nil {
				xmlListener.LoadBalancerArn = *l.LoadBalancerArn
			}
			if l.Port != nil {
				xmlListener.Port = int32(*l.Port)
			}
			if l.Protocol != nil {
				xmlListener.Protocol = string(*l.Protocol)
			}
			if l.DefaultActions != nil {
				for _, action := range l.DefaultActions {
					xmlAction := Action{}
					xmlAction.Type = string(action.Type)
					if action.TargetGroupArn != nil {
						xmlAction.TargetGroupArn = *action.TargetGroupArn
					}
					xmlListener.DefaultActions = append(xmlListener.DefaultActions, xmlAction)
				}
			}
			resp.Result.Listeners = append(resp.Result.Listeners, xmlListener)
		}
	}

	return resp
}

// convertDescribeListenersToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertDescribeListenersToXML(output *generated_elbv2.DescribeListenersOutput) *DescribeListenersResponse {
	resp := &DescribeListenersResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	if output != nil && output.Listeners != nil {
		for _, l := range output.Listeners {
			xmlListener := Listener{}
			if l.ListenerArn != nil {
				xmlListener.ListenerArn = *l.ListenerArn
			}
			if l.LoadBalancerArn != nil {
				xmlListener.LoadBalancerArn = *l.LoadBalancerArn
			}
			if l.Port != nil {
				xmlListener.Port = int32(*l.Port)
			}
			if l.Protocol != nil {
				xmlListener.Protocol = string(*l.Protocol)
			}
			if l.DefaultActions != nil {
				for _, action := range l.DefaultActions {
					xmlAction := Action{}
					xmlAction.Type = string(action.Type)
					if action.TargetGroupArn != nil {
						xmlAction.TargetGroupArn = *action.TargetGroupArn
					}
					xmlListener.DefaultActions = append(xmlListener.DefaultActions, xmlAction)
				}
			}
			resp.Result.Listeners = append(resp.Result.Listeners, xmlListener)
		}
	}

	return resp
}

// convertRegisterTargetsToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertRegisterTargetsToXML(output *generated_elbv2.RegisterTargetsOutput) *RegisterTargetsResponse {
	resp := &RegisterTargetsResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}
	return resp
}

// convertDeregisterTargetsToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertDeregisterTargetsToXML(output *generated_elbv2.DeregisterTargetsOutput) *DeregisterTargetsResponse {
	resp := &DeregisterTargetsResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}
	return resp
}

// convertDescribeTargetHealthToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertDescribeTargetHealthToXML(output *generated_elbv2.DescribeTargetHealthOutput) *DescribeTargetHealthResponse {
	resp := &DescribeTargetHealthResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	if output != nil && output.TargetHealthDescriptions != nil {
		for _, thd := range output.TargetHealthDescriptions {
			xmlTHD := TargetHealthDescription{}

			// Target
			if thd.Target != nil {
				xmlTHD.Target.Id = thd.Target.Id
				if thd.Target.Port != nil {
					xmlTHD.Target.Port = *thd.Target.Port
				}
				if thd.Target.AvailabilityZone != nil {
					xmlTHD.Target.AvailabilityZone = *thd.Target.AvailabilityZone
				}
			}

			// HealthCheckPort
			if thd.HealthCheckPort != nil {
				xmlTHD.HealthCheckPort = *thd.HealthCheckPort
			}

			// TargetHealth
			if thd.TargetHealth != nil {
				if thd.TargetHealth.State != nil {
					xmlTHD.TargetHealth.State = string(*thd.TargetHealth.State)
				}
				if thd.TargetHealth.Reason != nil {
					xmlTHD.TargetHealth.Reason = string(*thd.TargetHealth.Reason)
				}
				if thd.TargetHealth.Description != nil {
					xmlTHD.TargetHealth.Description = *thd.TargetHealth.Description
				}
			}

			resp.Result.TargetHealthDescriptions = append(resp.Result.TargetHealthDescriptions, xmlTHD)
		}
	}

	return resp
}

// convertModifyTargetGroupToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertModifyTargetGroupToXML(output *generated_elbv2.ModifyTargetGroupOutput) *ModifyTargetGroupResponse {
	resp := &ModifyTargetGroupResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	if output != nil && output.TargetGroups != nil {
		for _, tg := range output.TargetGroups {
			xmlTG := TargetGroup{}
			if tg.TargetGroupArn != nil {
				xmlTG.TargetGroupArn = *tg.TargetGroupArn
			}
			if tg.TargetGroupName != nil {
				xmlTG.TargetGroupName = *tg.TargetGroupName
			}
			if tg.Protocol != nil {
				xmlTG.Protocol = string(*tg.Protocol)
			}
			if tg.Port != nil {
				xmlTG.Port = int32(*tg.Port)
			}
			if tg.VpcId != nil {
				xmlTG.VpcId = *tg.VpcId
			}
			if tg.HealthCheckEnabled != nil {
				xmlTG.HealthCheckEnabled = *tg.HealthCheckEnabled
			}
			if tg.HealthCheckIntervalSeconds != nil {
				xmlTG.HealthCheckIntervalSeconds = int32(*tg.HealthCheckIntervalSeconds)
			}
			if tg.HealthCheckTimeoutSeconds != nil {
				xmlTG.HealthCheckTimeoutSeconds = int32(*tg.HealthCheckTimeoutSeconds)
			}
			if tg.HealthyThresholdCount != nil {
				xmlTG.HealthyThresholdCount = int32(*tg.HealthyThresholdCount)
			}
			if tg.UnhealthyThresholdCount != nil {
				xmlTG.UnhealthyThresholdCount = int32(*tg.UnhealthyThresholdCount)
			}
			if tg.HealthCheckPath != nil {
				xmlTG.HealthCheckPath = *tg.HealthCheckPath
			}
			if tg.HealthCheckPort != nil {
				xmlTG.HealthCheckPort = *tg.HealthCheckPort
			}
			if tg.HealthCheckProtocol != nil {
				xmlTG.HealthCheckProtocol = string(*tg.HealthCheckProtocol)
			}
			if tg.TargetType != nil {
				xmlTG.TargetType = string(*tg.TargetType)
			}
			resp.Result.TargetGroups = append(resp.Result.TargetGroups, xmlTG)
		}
	}

	return resp
}

// convertDeleteTargetGroupToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertDeleteTargetGroupToXML(output *generated_elbv2.DeleteTargetGroupOutput) *DeleteTargetGroupResponse {
	resp := &DeleteTargetGroupResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}
	return resp
}

// convertCreateRuleToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertCreateRuleToXML(output *generated_elbv2.CreateRuleOutput) *CreateRuleResponse {
	resp := &CreateRuleResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	if output != nil && output.Rules != nil {
		for _, rule := range output.Rules {
			xmlRule := Rule{}
			if rule.RuleArn != nil {
				xmlRule.RuleArn = *rule.RuleArn
			}
			if rule.Priority != nil {
				xmlRule.Priority = *rule.Priority
			}
			if rule.IsDefault != nil {
				xmlRule.IsDefault = *rule.IsDefault
			}

			// Convert actions
			for _, action := range rule.Actions {
				xmlAction := Action{
					Type: string(action.Type),
				}
				if action.TargetGroupArn != nil {
					xmlAction.TargetGroupArn = *action.TargetGroupArn
				}
				if action.Order != nil {
					xmlAction.Order = *action.Order
				}
				xmlRule.Actions = append(xmlRule.Actions, xmlAction)
			}

			// Convert conditions
			for _, condition := range rule.Conditions {
				xmlCondition := RuleCondition{}
				if condition.Field != nil {
					xmlCondition.Field = *condition.Field
				}
				if condition.Values != nil {
					xmlCondition.Values = condition.Values
				}

				// Convert PathPatternConfig
				if condition.PathPatternConfig != nil && condition.PathPatternConfig.Values != nil {
					xmlCondition.PathPatternConfig = &PathPatternConfig{
						Values: condition.PathPatternConfig.Values,
					}
				}

				// Convert HostHeaderConfig
				if condition.HostHeaderConfig != nil && condition.HostHeaderConfig.Values != nil {
					xmlCondition.HostHeaderConfig = &HostHeaderConfig{
						Values: condition.HostHeaderConfig.Values,
					}
				}

				xmlRule.Conditions = append(xmlRule.Conditions, xmlCondition)
			}

			resp.Result.Rules = append(resp.Result.Rules, xmlRule)
		}
	}

	return resp
}

// convertDescribeRulesToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertDescribeRulesToXML(output *generated_elbv2.DescribeRulesOutput) *DescribeRulesResponse {
	resp := &DescribeRulesResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	if output != nil {
		if output.NextMarker != nil {
			resp.Result.NextMarker = output.NextMarker
		}

		if output.Rules != nil {
			for _, rule := range output.Rules {
				xmlRule := Rule{}
				if rule.RuleArn != nil {
					xmlRule.RuleArn = *rule.RuleArn
				}
				if rule.Priority != nil {
					xmlRule.Priority = *rule.Priority
				}
				if rule.IsDefault != nil {
					xmlRule.IsDefault = *rule.IsDefault
				}

				// Convert actions
				for _, action := range rule.Actions {
					xmlAction := Action{
						Type: string(action.Type),
					}
					if action.TargetGroupArn != nil {
						xmlAction.TargetGroupArn = *action.TargetGroupArn
					}
					if action.Order != nil {
						xmlAction.Order = *action.Order
					}
					xmlRule.Actions = append(xmlRule.Actions, xmlAction)
				}

				// Convert conditions
				for _, condition := range rule.Conditions {
					xmlCondition := RuleCondition{}
					if condition.Field != nil {
						xmlCondition.Field = *condition.Field
					}
					if condition.Values != nil {
						xmlCondition.Values = condition.Values
					}

					// Convert PathPatternConfig
					if condition.PathPatternConfig != nil && condition.PathPatternConfig.Values != nil {
						xmlCondition.PathPatternConfig = &PathPatternConfig{
							Values: condition.PathPatternConfig.Values,
						}
					}

					// Convert HostHeaderConfig
					if condition.HostHeaderConfig != nil && condition.HostHeaderConfig.Values != nil {
						xmlCondition.HostHeaderConfig = &HostHeaderConfig{
							Values: condition.HostHeaderConfig.Values,
						}
					}

					xmlRule.Conditions = append(xmlRule.Conditions, xmlCondition)
				}

				resp.Result.Rules = append(resp.Result.Rules, xmlRule)
			}
		}
	}

	return resp
}

// convertDescribeLoadBalancerAttributesToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertDescribeLoadBalancerAttributesToXML(output *generated_elbv2.DescribeLoadBalancerAttributesOutput) *DescribeLoadBalancerAttributesResponse {
	resp := &DescribeLoadBalancerAttributesResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	if output != nil && output.Attributes != nil {
		for _, attr := range output.Attributes {
			xmlAttr := Attribute{}
			if attr.Key != nil {
				xmlAttr.Key = *attr.Key
			}
			if attr.Value != nil {
				xmlAttr.Value = *attr.Value
			}
			resp.Result.Attributes = append(resp.Result.Attributes, xmlAttr)
		}
	}

	return resp
}

// convertDescribeTargetGroupAttributesToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertDescribeTargetGroupAttributesToXML(output *generated_elbv2.DescribeTargetGroupAttributesOutput) *DescribeTargetGroupAttributesResponse {
	resp := &DescribeTargetGroupAttributesResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	if output != nil && output.Attributes != nil {
		for _, attr := range output.Attributes {
			xmlAttr := Attribute{}
			if attr.Key != nil {
				xmlAttr.Key = *attr.Key
			}
			if attr.Value != nil {
				xmlAttr.Value = *attr.Value
			}
			resp.Result.Attributes = append(resp.Result.Attributes, xmlAttr)
		}
	}

	return resp
}

// convertModifyTargetGroupAttributesToXML converts the API output to XML format
func (w *ELBv2RouterWrapper) convertModifyTargetGroupAttributesToXML(output *generated_elbv2.ModifyTargetGroupAttributesOutput) *ModifyTargetGroupAttributesResponse {
	resp := &ModifyTargetGroupAttributesResponse{
		XMLNS: "http://elasticloadbalancing.amazonaws.com/doc/2015-12-01/",
		ResponseMetadata: ResponseMetadata{
			RequestId: "generated-" + fmt.Sprintf("%d", time.Now().Unix()),
		},
	}

	if output != nil && output.Attributes != nil {
		for _, attr := range output.Attributes {
			xmlAttr := Attribute{}
			if attr.Key != nil {
				xmlAttr.Key = *attr.Key
			}
			if attr.Value != nil {
				xmlAttr.Value = *attr.Value
			}
			resp.Result.Attributes = append(resp.Result.Attributes, xmlAttr)
		}
	}

	return resp
}
