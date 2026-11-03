package client

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

var operationPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

// RecoveryError preserves the original operation when issuance cannot be confirmed.
// Error text never includes credentials; machine data is intended for local tooling.
type RecoveryError struct {
	Err                      error
	Kind, OperationID, State string
}

func (e *RecoveryError) Error() string {
	if e.State == "object_upload_unconfirmed" {
		return "图片传输结果未确认；核对原对象，勿重新上传"
	}
	return "upload credential result unconfirmed; query original operation " + e.OperationID
}
func (e *RecoveryError) Unwrap() error { return e.Err }
func (e *RecoveryError) ErrorData() any {
	action := "query_original_operation"
	if e.State == "object_upload_unconfirmed" || e.OperationID == "" {
		action = "inspect_existing_upload"
	}
	return map[string]any{"upload_recovery": map[string]any{"kind": e.Kind, "operation_id": e.OperationID, "state": e.State, "retry_action": action, "retryable": false}}
}

func (c *Client) ImageConfig() (*ResourceResponse, error) {
	return doGet[ResourceResponse](c, "/open/api/v1/resource/image/config", nil)
}
func (c *Client) UploadOperation(kind, id string) (*ResourceResponse, error) {
	if !operationPattern.MatchString(id) {
		return nil, errors.New("operation_id must be 1–128 letters, digits or ._:-")
	}
	path, err := operationPath(kind)
	if err != nil {
		return nil, err
	}
	supported, err := c.uploadRecoverySupported(kind)
	if err != nil {
		return nil, err
	}
	if !supported {
		return nil, errors.New("server does not advertise upload operation recovery")
	}
	result, err := doGet[ResourceResponse](c, path, url.Values{"operation_id": {id}})
	if err != nil {
		return nil, err
	}
	var data map[string]json.RawMessage
	if json.Unmarshal(result.Data, &data) != nil {
		return nil, errors.New("invalid operation status")
	}
	var returnedID, state, until string
	json.Unmarshal(data["operation_id"], &returnedID)
	json.Unmarshal(data["state"], &state)
	json.Unmarshal(data["resume_until"], &until)
	if returnedID != id {
		return nil, errors.New("operation response ID mismatch")
	}
	deadline, parseErr := time.Parse(time.RFC3339, until)
	if state != "issued" || parseErr != nil || !time.Now().Before(deadline) {
		delete(data, "credential")
		if state == "issued" && parseErr == nil && !time.Now().Before(deadline) {
			data["state"] = json.RawMessage(`"expired"`)
		}
	}
	if cred := data["credential"]; len(cred) > 0 && string(cred) != "null" {
		safe, err := validateCredential(cred, id, deadline)
		if err != nil {
			return nil, &RecoveryError{Err: err, Kind: kind, OperationID: id, State: "invalid_credential"}
		}
		data["credential"] = safe
	}
	result.Data, _ = json.Marshal(data)
	return result, nil
}
func operationPath(kind string) (string, error) {
	switch kind {
	case "image":
		return "/open/api/v1/resource/image/upload_operation", nil
	case "file":
		return "/open/api/v1/resource/knowledge/file/upload_operation", nil
	}
	return "", errors.New("unknown upload kind")
}
func (c *Client) ImageToken(ext, id string) (*ResourceResponse, error) {
	return c.credentialToken("image", ext, id)
}
func (c *Client) FileTokenWithOperation(ext, id string) (*ResourceResponse, error) {
	return c.credentialToken("file", ext, id)
}
func (c *Client) credentialToken(kind, ext, id string) (*ResourceResponse, error) {
	if id != "" && !operationPattern.MatchString(id) {
		return nil, errors.New("operation_id must be 1–128 letters, digits or ._:-")
	}
	enabled, err := c.uploadRecoverySupported(kind)
	if err != nil {
		return nil, err
	}
	if !enabled && id != "" {
		return nil, errors.New("server does not advertise upload operation recovery")
	}
	if enabled && id == "" {
		b := make([]byte, 16)
		if _, err = rand.Read(b); err != nil {
			return nil, err
		}
		id = hex.EncodeToString(b)
	}
	path := "/open/api/v1/resource/image/upload_token"
	if kind == "file" {
		path = "/open/api/v1/resource/knowledge/file/upload_token"
	}
	q := url.Values{"mime_type": {ext}}
	if kind == "image" {
		q.Set("count", "1")
	}
	// A non-replayable body prevents net/http silently reissuing a credential GET
	// on a stale keep-alive connection. Application retries require explicit refusal.
	req, err := c.newRequest(http.MethodGet, path+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}

	if enabled {
		req.Header.Set("Idempotency-Key", id)
	}
	result, issueErr := doRequest[ResourceResponse](c, req)
	if issueErr == nil {
		if enabled {
			result.Data, err = validateCredential(result.Data, id, time.Time{})
			if err != nil {
				return nil, &RecoveryError{Err: err, Kind: kind, OperationID: id, State: "invalid_credential"}
			}
		}
		return result, nil
	}
	var re *RequestError
	uncertain := !errors.As(issueErr, &re) || re.Reason == "credential_operation_unconfirmed" || re.Reason == "credential_operation_unavailable"
	if !enabled {
		return nil, issueErr
	}
	if !uncertain {
		return nil, &RecoveryError{Err: issueErr, Kind: kind, OperationID: id, State: re.Reason}
	}
	view, lookupErr := c.UploadOperation(kind, id)
	if lookupErr != nil {
		return nil, &RecoveryError{Err: errors.Join(issueErr, lookupErr), Kind: kind, OperationID: id, State: "unknown"}
	}
	var op struct {
		OperationID string          `json:"operation_id"`
		State       string          `json:"state"`
		ResumeUntil string          `json:"resume_until"`
		Credential  json.RawMessage `json:"credential"`
	}
	if json.Unmarshal(view.Data, &op) != nil || op.OperationID != id {
		return nil, &RecoveryError{Err: issueErr, Kind: kind, OperationID: id, State: "unknown"}
	}
	until, parseErr := time.Parse(time.RFC3339, op.ResumeUntil)
	if op.State == "issued" && parseErr == nil && time.Now().Before(until) && len(op.Credential) > 0 && string(op.Credential) != "null" {
		safe, err := validateCredential(op.Credential, id, until)
		if err != nil {
			return nil, &RecoveryError{Err: err, Kind: kind, OperationID: id, State: "invalid_credential"}
		}
		return &ResourceResponse{Success: true, Data: safe, RequestID: view.RequestID}, nil
	}
	return nil, &RecoveryError{Err: issueErr, Kind: kind, OperationID: id, State: op.State}
}

func decodeImageToken(raw *ResourceResponse) (*ImageUploadTokenResponse, error) {
	var token ImageUploadToken
	if err := json.Unmarshal(raw.Data, &token); err != nil {
		return nil, fmt.Errorf("invalid image credential: %w", err)
	}
	return &ImageUploadTokenResponse{Success: raw.Success, Data: token, RequestID: raw.RequestID}, nil
}

func validateCredential(raw json.RawMessage, id string, outer time.Time) (json.RawMessage, error) {
	var data map[string]json.RawMessage
	if json.Unmarshal(raw, &data) != nil || data == nil {
		return nil, errors.New("invalid credential")
	}
	if value, ok := data["operation_id"]; ok {
		var returned string
		if json.Unmarshal(value, &returned) != nil || returned != id {
			return nil, errors.New("credential operation ID mismatch")
		}
	}
	deadline := outer
	if value, ok := data["resume_until"]; ok {
		var until string
		if json.Unmarshal(value, &until) != nil {
			return nil, errors.New("invalid credential deadline")
		}
		parsed, err := time.Parse(time.RFC3339, until)
		if err != nil {
			return nil, errors.New("invalid credential deadline")
		}
		if deadline.IsZero() || parsed.Before(deadline) {
			deadline = parsed
		}
	}
	if value, ok := data["expire"]; ok {
		var sec int64
		if json.Unmarshal(value, &sec) != nil || sec <= 0 {
			return nil, errors.New("invalid credential expiry")
		}
		parsed := time.Unix(sec, 0)
		if deadline.IsZero() || parsed.Before(deadline) {
			deadline = parsed
		}
	}
	if deadline.IsZero() || !time.Now().Before(deadline) {
		return nil, errors.New("credential expiry missing or elapsed")
	}
	data["operation_id"], _ = json.Marshal(id)
	data["resume_until"], _ = json.Marshal(deadline.UTC().Format(time.RFC3339))
	return json.Marshal(data)
}

func (c *Client) uploadRecoverySupported(kind string) (bool, error) {
	var caps *ResourceResponse
	var err error
	if kind == "image" {
		caps, err = c.ImageConfig()
	} else {
		caps, err = c.FileCapabilities()
	}
	enabled := false
	if err != nil {
		var re *RequestError
		if !errors.As(err, &re) || (re.StatusCode != 404 && !(kind == "file" && re.StatusCode == 403)) {
			return false, err
		}
	} else {
		var d struct {
			Runtime struct {
				Governance bool   `json:"governance_enabled"`
				Endpoint   string `json:"operation_status_endpoint"`
				Replay     bool   `json:"operation_replay"`
				Header     string `json:"idempotency_header"`
			} `json:"runtime"`
		}
		if json.Unmarshal(caps.Data, &d) != nil {
			return false, errors.New("invalid upload runtime")
		}
		expected, _ := operationPath(kind)
		enabled = d.Runtime.Governance && d.Runtime.Replay && d.Runtime.Header == "Idempotency-Key" && d.Runtime.Endpoint == expected
	}
	return enabled, nil
}
