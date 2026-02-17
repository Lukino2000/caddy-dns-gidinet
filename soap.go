package gidinet

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

const (
	defaultAPIEndpoint = "https://api.quickservicebox.com/API/Beta/DNSAPI.asmx"
	soapNamespace      = "https://api.quickservicebox.com/DNS/DNSAPI"
)

// soap12Envelope is the SOAP 1.2 envelope wrapper.
type soap12Envelope struct {
	XMLName xml.Name    `xml:"soap12:Envelope"`
	XSI     string      `xml:"xmlns:xsi,attr"`
	XSD     string      `xml:"xmlns:xsd,attr"`
	Soap12  string      `xml:"xmlns:soap12,attr"`
	Body    soap12Body  `xml:"soap12:Body"`
}

type soap12Body struct {
	Content []byte `xml:",innerxml"`
}

// soap12ResponseEnvelope is used to parse SOAP 1.2 responses.
type soap12ResponseEnvelope struct {
	XMLName xml.Name             `xml:"Envelope"`
	Body    soap12ResponseBody   `xml:"Body"`
}

type soap12ResponseBody struct {
	Content []byte `xml:",innerxml"`
}

// dnsRecord represents the DNSRecord type used in SOAP requests.
type dnsRecord struct {
	XMLName    xml.Name `xml:"record"`
	DomainName string   `xml:"DomainName"`
	HostName   string   `xml:"HostName"`
	RecordType string   `xml:"RecordType"`
	Data       string   `xml:"Data"`
	TTL        uint     `xml:"TTL"`
	Priority   uint     `xml:"Priority"`
}

// dnsOldRecord represents the oldRecord element used in recordUpdate.
type dnsOldRecord struct {
	XMLName    xml.Name `xml:"oldRecord"`
	DomainName string   `xml:"DomainName"`
	HostName   string   `xml:"HostName"`
	RecordType string   `xml:"RecordType"`
	Data       string   `xml:"Data"`
	TTL        uint     `xml:"TTL"`
	Priority   uint     `xml:"Priority"`
}

// dnsNewRecord represents the newRecord element used in recordUpdate.
type dnsNewRecord struct {
	XMLName    xml.Name `xml:"newRecord"`
	DomainName string   `xml:"DomainName"`
	HostName   string   `xml:"HostName"`
	RecordType string   `xml:"RecordType"`
	Data       string   `xml:"Data"`
	TTL        uint     `xml:"TTL"`
	Priority   uint     `xml:"Priority"`
}

// apiResult represents the common result structure returned by all API methods.
type apiResult struct {
	ResultText    string `xml:"resultText"`
	ResultCode    int    `xml:"resultCode"`
	ResultSubCode int    `xml:"resultSubCode"`
}

// dnsRecordListItem represents a single DNS record returned by recordGetList.
type dnsRecordListItem struct {
	DomainName       string `xml:"DomainName"`
	HostName         string `xml:"HostName"`
	RecordType       string `xml:"RecordType"`
	Data             string `xml:"Data"`
	TTL              int    `xml:"TTL"`
	Priority         int    `xml:"Priority"`
	ReadOnly         bool   `xml:"ReadOnly"`
	Suspended        bool   `xml:"Suspended"`
	SuspensionReason string `xml:"SuspensionReason"`
}

// --- SOAP request bodies ---

type recordAddRequest struct {
	XMLName            xml.Name  `xml:"recordAdd"`
	Namespace          string    `xml:"xmlns,attr"`
	AccountUsername    string    `xml:"accountUsername"`
	AccountPasswordB64 string    `xml:"accountPasswordB64"`
	Record             dnsRecord `xml:"record"`
}

type recordDeleteRequest struct {
	XMLName            xml.Name  `xml:"recordDelete"`
	Namespace          string    `xml:"xmlns,attr"`
	AccountUsername    string    `xml:"accountUsername"`
	AccountPasswordB64 string    `xml:"accountPasswordB64"`
	Record             dnsRecord `xml:"record"`
}

type recordUpdateRequest struct {
	XMLName            xml.Name     `xml:"recordUpdate"`
	Namespace          string       `xml:"xmlns,attr"`
	AccountUsername    string       `xml:"accountUsername"`
	AccountPasswordB64 string       `xml:"accountPasswordB64"`
	OldRecord          dnsOldRecord `xml:"oldRecord"`
	NewRecord          dnsNewRecord `xml:"newRecord"`
}

type recordGetListRequest struct {
	XMLName            xml.Name `xml:"recordGetList"`
	Namespace          string   `xml:"xmlns,attr"`
	AccountUsername    string   `xml:"accountUsername"`
	AccountPasswordB64 string   `xml:"accountPasswordB64"`
	DomainName         string   `xml:"domainName"`
}

// --- SOAP response bodies ---

type recordAddResponse struct {
	XMLName xml.Name  `xml:"recordAddResponse"`
	Result  apiResult `xml:"recordAddResult"`
}

type recordDeleteResponse struct {
	XMLName xml.Name  `xml:"recordDeleteResponse"`
	Result  apiResult `xml:"recordDeleteResult"`
}

type recordUpdateResponse struct {
	XMLName xml.Name  `xml:"recordUpdateResponse"`
	Result  apiResult `xml:"recordUpdateResult"`
}

type recordGetListResponse struct {
	XMLName xml.Name           `xml:"recordGetListResponse"`
	Result  recordGetListResult `xml:"recordGetListResult"`
}

type recordGetListResult struct {
	ResultText     string              `xml:"resultText"`
	ResultCode     int                 `xml:"resultCode"`
	ResultSubCode  int                 `xml:"resultSubCode"`
	ResultItems    []dnsRecordListItem `xml:"resultItems>DNSRecordListItem"`
	ResultItemCount int               `xml:"resultItemCount"`
}

// soapClient handles SOAP communication with the Gidinet API.
type soapClient struct {
	endpoint   string
	username   string
	passwordB64 string
	httpClient *http.Client
}

// newSOAPClient creates a new SOAP client for the Gidinet DNS API.
func newSOAPClient(username, passwordB64, endpoint string) *soapClient {
	if endpoint == "" {
		endpoint = defaultAPIEndpoint
	}
	return &soapClient{
		endpoint:    endpoint,
		username:    username,
		passwordB64: passwordB64,
		httpClient:  &http.Client{},
	}
}

// doRequest sends a SOAP 1.2 request and returns the raw body content inside the SOAP Body element.
func (c *soapClient) doRequest(ctx context.Context, bodyContent interface{}) ([]byte, error) {
	// Marshal the inner body content
	innerXML, err := xml.Marshal(bodyContent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SOAP body content: %w", err)
	}

	envelope := soap12Envelope{
		XSI:    "http://www.w3.org/2001/XMLSchema-instance",
		XSD:    "http://www.w3.org/2001/XMLSchema",
		Soap12: "http://www.w3.org/2003/05/soap-envelope",
		Body: soap12Body{
			Content: innerXML,
		},
	}

	envelopeXML, err := xml.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SOAP envelope: %w", err)
	}

	// Prepend XML declaration
	fullPayload := []byte(xml.Header + string(envelopeXML))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(fullPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse the SOAP envelope to extract the body content
	var respEnvelope soap12ResponseEnvelope
	if err := xml.Unmarshal(respBody, &respEnvelope); err != nil {
		return nil, fmt.Errorf("failed to parse SOAP response envelope: %w", err)
	}

	return respEnvelope.Body.Content, nil
}

// recordAdd calls the recordAdd SOAP method.
func (c *soapClient) recordAdd(ctx context.Context, rec dnsRecord) (*apiResult, error) {
	reqBody := recordAddRequest{
		Namespace:          soapNamespace,
		AccountUsername:    c.username,
		AccountPasswordB64: c.passwordB64,
		Record:             rec,
	}

	respContent, err := c.doRequest(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("recordAdd request failed: %w", err)
	}

	var resp recordAddResponse
	if err := xml.Unmarshal(respContent, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse recordAdd response: %w", err)
	}

	return &resp.Result, nil
}

// recordDelete calls the recordDelete SOAP method.
func (c *soapClient) recordDelete(ctx context.Context, rec dnsRecord) (*apiResult, error) {
	reqBody := recordDeleteRequest{
		Namespace:          soapNamespace,
		AccountUsername:    c.username,
		AccountPasswordB64: c.passwordB64,
		Record:             rec,
	}

	respContent, err := c.doRequest(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("recordDelete request failed: %w", err)
	}

	var resp recordDeleteResponse
	if err := xml.Unmarshal(respContent, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse recordDelete response: %w", err)
	}

	return &resp.Result, nil
}

// recordUpdate calls the recordUpdate SOAP method.
func (c *soapClient) recordUpdate(ctx context.Context, oldRec dnsOldRecord, newRec dnsNewRecord) (*apiResult, error) {
	reqBody := recordUpdateRequest{
		Namespace:          soapNamespace,
		AccountUsername:    c.username,
		AccountPasswordB64: c.passwordB64,
		OldRecord:          oldRec,
		NewRecord:          newRec,
	}

	respContent, err := c.doRequest(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("recordUpdate request failed: %w", err)
	}

	var resp recordUpdateResponse
	if err := xml.Unmarshal(respContent, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse recordUpdate response: %w", err)
	}

	return &resp.Result, nil
}

// recordGetList calls the recordGetList SOAP method.
func (c *soapClient) recordGetList(ctx context.Context, domainName string) (*recordGetListResult, error) {
	reqBody := recordGetListRequest{
		Namespace:          soapNamespace,
		AccountUsername:    c.username,
		AccountPasswordB64: c.passwordB64,
		DomainName:         domainName,
	}

	respContent, err := c.doRequest(ctx, reqBody)
	if err != nil {
		return nil, fmt.Errorf("recordGetList request failed: %w", err)
	}

	var resp recordGetListResponse
	if err := xml.Unmarshal(respContent, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse recordGetList response: %w", err)
	}

	return &resp.Result, nil
}

// checkResult verifies that the API returned a success code (0).
// Returns a descriptive error if the operation failed.
func checkResult(result *apiResult, operation string) error {
	if result.ResultCode == 0 {
		return nil
	}
	return fmt.Errorf("%s failed: code=%d subcode=%d text=%q",
		operation, result.ResultCode, result.ResultSubCode, result.ResultText)
}
