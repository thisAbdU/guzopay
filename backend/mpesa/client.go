package mpesa

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Ethiopia Daraja base URLs (same as Safaricom-Ethiopia-PLC/mpesa-go-sdk).
const (
	sandboxURL    = "https://apisandbox.safaricom.et"
	productionURL = "https://api.safaricom.et"
)

// Client talks to Safaricom Ethiopia Daraja (sandbox/production).
// Official mpesa-go-sdk currently uses an invalid Go package name; this client
// mirrors those endpoints so GuzoPay is sandbox-ready.
type Client struct {
	ConsumerKey          string
	ConsumerSecret       string
	Env                  string // SANDBOX | PRODUCTION
	ShortCode            string
	PassKey              string
	InitiatorName        string
	SecurityCredential   string
	CallbackBaseURL      string
	http                 *http.Client
	mu                   sync.Mutex
	token                string
	tokenExpires         time.Time
}

var Default *Client

// InitFromEnv loads credentials. Returns nil client (mock mode) if keys missing.
func InitFromEnv() *Client {
	key := strings.TrimSpace(os.Getenv("CONSUMER_KEY"))
	secret := strings.TrimSpace(os.Getenv("CONSUMER_SECRET"))
	if key == "" || secret == "" {
		log.Println("mpesa: CONSUMER_KEY/SECRET unset — mock payments enabled")
		Default = nil
		return nil
	}
	env := strings.ToUpper(getEnv("ENVIROMENT", getEnv("ENVIRONMENT", "SANDBOX")))
	c := &Client{
		ConsumerKey:        key,
		ConsumerSecret:     secret,
		Env:                env,
		ShortCode:          os.Getenv("MPESA_SHORTCODE"),
		PassKey:            os.Getenv("MPESA_PASSKEY"),
		InitiatorName:      os.Getenv("MPESA_INITIATOR_NAME"),
		SecurityCredential: os.Getenv("MPESA_SECURITY_CREDENTIAL"),
		CallbackBaseURL:    strings.TrimRight(os.Getenv("CALLBACK_BASE_URL"), "/"),
		http:               &http.Client{Timeout: 30 * time.Second},
	}
	Default = c
	log.Printf("mpesa: Daraja client ready (%s) shortcode=%s", env, c.ShortCode)
	return c
}

func (c *Client) Enabled() bool { return c != nil && c.ConsumerKey != "" }

func (c *Client) baseURL() string {
	if c.Env == "PRODUCTION" {
		return productionURL
	}
	return sandboxURL
}

func (c *Client) Callback(path string) string {
	if c.CallbackBaseURL == "" {
		return "https://example.com" + path // validation placeholder until ngrok set
	}
	return c.CallbackBaseURL + path
}

func (c *Client) accessToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExpires) {
		return c.token, nil
	}
	url := c.baseURL() + "/v1/token/generate?grant_type=client_credentials"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.ConsumerKey, c.ConsumerSecret)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var out struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   string `json:"expires_in"`
		ResultCode  string `json:"resultCode"`
		ResultDesc  string `json:"resultDesc"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("token parse: %w body=%s", err, string(body))
	}
	if out.ResultCode != "" {
		return "", fmt.Errorf("token error: %s", out.ResultDesc)
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("empty access_token: %s", string(body))
	}
	sec, _ := strconv.Atoi(out.ExpiresIn)
	if sec == 0 {
		sec = 3599
	}
	typ := out.TokenType
	if typ == "" {
		typ = "Bearer"
	}
	c.token = typ + " " + out.AccessToken
	c.tokenExpires = time.Now().Add(time.Duration(sec-30) * time.Second)
	return c.token, nil
}

func (c *Client) postJSON(path string, payload any, authBearer bool) ([]byte, int, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL()+path, bytes.NewReader(raw))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if authBearer {
		tok, err := c.accessToken()
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Authorization", tok)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	return body, res.StatusCode, err
}

// STKPassword = Base64(ShortCode + PassKey + Timestamp)
func (c *Client) stkPassword(ts string) string {
	plain := c.ShortCode + c.PassKey + ts
	return base64.StdEncoding.EncodeToString([]byte(plain))
}

func getEnv(k, fallback string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return fallback
}

// ToMSISDN251 converts 07XXXXXXXX → 2517XXXXXXXX for Daraja PartyA/PhoneNumber.
func ToMSISDN251(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, "+", "")
	phone = strings.ReplaceAll(phone, " ", "")
	if strings.HasPrefix(phone, "07") && len(phone) == 10 {
		return "251" + phone[1:]
	}
	if strings.HasPrefix(phone, "2517") {
		return phone
	}
	return phone
}

func FromMSISDN251(msisdn string) string {
	msisdn = strings.TrimSpace(msisdn)
	if strings.HasPrefix(msisdn, "2517") && len(msisdn) == 12 {
		return "0" + msisdn[3:]
	}
	return msisdn
}
