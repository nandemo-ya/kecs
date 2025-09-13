package api

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// NewELBv2RouterWrapper creates a new wrapper for the ELBv2 router
func NewELBv2RouterWrapper(api generated_elbv2.ElasticLoadBalancing_v10API) *ELBv2RouterWrapper {
	return &ELBv2RouterWrapper{
		innerRouter: generated_elbv2.NewRouter(api),
		api:         api,
	}
}

// Route handles the HTTP request, converting form data to JSON when necessary
func (w *ELBv2RouterWrapper) Route(resp http.ResponseWriter, req *http.Request) {
	logging.Debug("ELBv2RouterWrapper.Route called",
		"method", req.Method,
		"path", req.URL.Path,
		"content-type", req.Header.Get("Content-Type"))

	// If it's form data, convert it to JSON format expected by the generated code
	if req.Method == "POST" && strings.Contains(req.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
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

		default:
			// For other actions, try to route with the action in the header
			// This allows the generated router to handle it
			req.Header.Set("X-Amz-Target", "ElasticLoadBalancing."+action)
			w.innerRouter.Route(resp, req)
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
