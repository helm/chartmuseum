package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
)

var (
	Version = "v1"
	Kind    = "Webhook"
)

type (
	Webhook struct {
		Version  string          `json:"version"`
		Kind     string          `json:"kind"`
		Metadata WebhookMetadata `json:"metadata"`
		Spec     WebhookSpec     `json:"spec"`
	}

	WebhookMetadata struct {
		Action      string                     `json:"action"`
		Labels      map[string]string          `json:"labels"`
		Timestamp   string                     `json:"timestamp"`
		Integration WebhookMetadataIntegration `json:"integration"`
	}

	WebhookSpec struct {
		Chart    string              `json:"chart"`
		Resource WebhookSpecResource `json:"resource"`
	}

	WebhookSpecResource struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}

	WebhookMetadataIntegration struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
)

func (wh *Webhook) SendHook(url string, secret string, logger cm_logger.LoggingFn) {
	message := fmt.Sprintf("Preparing to send hook")
	logger(cm_logger.DebugLevel, message,
		"URL", url,
	)
	mJSON, _ := json.Marshal(wh)
	contentReader := bytes.NewReader(mJSON)
	req, _ := http.NewRequest("POST", url, contentReader)
	req.Header.Set("X-Chartmuseum-HMAC", getHMACHeaderToRequest(req, secret, mJSON, logger))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, _ := client.Do(req)
	logger(cm_logger.DebugLevel, "Webhook been sent", "Status_Code", resp.StatusCode)
}

// Calc HMAC using Sha-256 using the secret string from the integration
// Set the result as header of the request
func getHMACHeaderToRequest(req *http.Request, secret string, payload []byte, logger cm_logger.LoggingFn) string {
	if secret != "" {
		message := fmt.Sprintf("Singing payload with secret")
		logger(cm_logger.DebugLevel, message,
			"Secret", secret,
		)
		key := []byte(secret)
		mac := hmac.New(sha256.New, key)
		mac.Write(payload)
		hmac := base64.URLEncoding.EncodeToString(mac.Sum(nil))
		return hmac
	}
	return ""
}
