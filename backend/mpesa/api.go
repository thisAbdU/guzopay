package mpesa

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type SyncResponse struct {
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
	ErrorCode                string `json:"errorCode"`
	ErrorMessage             string `json:"errorMessage"`
	RequestID                string `json:"requestId"`
}

// RegisterC2BURLs registers validation + confirmation webhooks.
// Docs: POST /v1/c2b-register-url/register?apikey={consumer_key}
func (c *Client) RegisterC2BURLs() (map[string]any, error) {
	if c.ShortCode == "" {
		return nil, fmt.Errorf("MPESA_SHORTCODE required")
	}
	// Ethiopia sandbox expects camelCase (docs samples use PascalCase — those fail with
	// "Missing shortCode value"). Optional Bearer helps some gateways.
	payload := map[string]string{
		"shortCode":       c.ShortCode,
		"responseType":    "Completed",
		"commandID":       "RegisterURL",
		"confirmationURL": c.Callback("/api/v1/mpesa/c2b-callback"),
		"validationURL":   c.Callback("/api/v1/mpesa/c2b-validation"),
	}
	path := "/v1/c2b-register-url/register?apikey=" + url.QueryEscape(c.ConsumerKey)
	body, status, err := c.postJSON(path, payload, true)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	_ = json.Unmarshal(body, &out)
	if status >= 400 {
		return out, fmt.Errorf("register URL HTTP %d: %s", status, string(body))
	}
	// Safaricom often returns HTTP 200 with header.responseCode 400.
	if h, ok := out["header"].(map[string]any); ok {
		code := fmt.Sprint(h["responseCode"])
		msg := fmt.Sprint(h["responseMessage"])
		// 400 + already registered is OK to proceed.
		if code != "200" && code != "0" && !strings.Contains(strings.ToLower(msg), "already registered") {
			return out, fmt.Errorf("register URL: %s", msg)
		}
	}
	return out, nil
}

// SimulateCustomerPayment sandbox C2B (money → merchant).
// Docs: POST /mpesa/b2c/simulatetransaction/v1/request
func (c *Client) SimulateCustomerPayment(amount float64, msisdn, billRef string) (*SyncResponse, error) {
	if c.ShortCode == "" {
		return nil, fmt.Errorf("MPESA_SHORTCODE required")
	}
	payload := map[string]string{
		"CommandID":     "CustomerPayBillOnline",
		"Amount":        formatAmount(amount),
		"Msisdn":        ToMSISDN251(msisdn),
		"BillRefNumber": billRef,
		"ShortCode":     c.ShortCode,
	}
	body, status, err := c.postJSON("/mpesa/b2c/simulatetransaction/v1/request", payload, true)
	if err != nil {
		return nil, err
	}
	var out SyncResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("simulate parse: %w body=%s", err, string(body))
	}
	if status >= 400 || (out.ResponseCode != "" && out.ResponseCode != "0") {
		msg := out.ResponseDescription
		if out.ErrorMessage != "" {
			msg = out.ErrorMessage
		}
		if msg == "" {
			msg = string(body)
		}
		return &out, fmt.Errorf("simulate failed: %s", msg)
	}
	return &out, nil
}

// B2CPayOut merchant → driver M-Pesa.
// Docs: POST /mpesa/b2c/v2/paymentrequest
func (c *Client) B2CPayOut(originatorID string, amount float64, partyB, remarks string) (*SyncResponse, error) {
	if c.ShortCode == "" || c.InitiatorName == "" || c.SecurityCredential == "" {
		return nil, fmt.Errorf("MPESA_SHORTCODE, MPESA_INITIATOR_NAME, MPESA_SECURITY_CREDENTIAL required for B2C")
	}
	partyA, _ := strconv.Atoi(c.ShortCode)
	payload := map[string]any{
		"OriginatorConversationID": originatorID,
		"InitiatorName":            c.InitiatorName,
		"SecurityCredential":       c.SecurityCredential,
		"CommandID":                "BusinessPayment",
		"PartyA":                   partyA,
		"PartyB":                   ToMSISDN251(partyB),
		"Amount":                   amount,
		"Remarks":                  remarks,
		"Occassion":                "GuzoPayPayout",
		"QueueTimeOutURL":          c.Callback("/api/v1/mpesa/b2c-timeout"),
		"ResultURL":                c.Callback("/api/v1/mpesa/b2c-result"),
	}
	body, status, err := c.postJSON("/mpesa/b2c/v2/paymentrequest", payload, true)
	if err != nil {
		return nil, err
	}
	var out SyncResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("b2c parse: %w body=%s", err, string(body))
	}
	if status >= 400 || (out.ResponseCode != "" && out.ResponseCode != "0") {
		msg := out.ResponseDescription
		if out.ErrorMessage != "" {
			msg = out.ErrorMessage
		}
		if msg == "" {
			msg = string(body)
		}
		return &out, fmt.Errorf("b2c failed: %s", msg)
	}
	return &out, nil
}

func formatAmount(a float64) string {
	if a == float64(int64(a)) {
		return strconv.FormatInt(int64(a), 10)
	}
	return strconv.FormatFloat(a, 'f', 2, 64)
}
