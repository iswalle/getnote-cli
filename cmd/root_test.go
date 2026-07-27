package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/iswalle/getnote-cli/internal/client"
)

func TestWriteErrorOutputsStructuredJSON(t *testing.T) {
	var output bytes.Buffer
	writeError(&output, &client.RequestError{
		APIError: client.APIError{
			Code:         10000,
			Message:      "参数错误",
			Reason:       "invalid_request",
			Field:        "parent_id",
			Constraint:   "non_negative_decimal_integer",
			ExpectedType: "decimal string or JSON integer",
		},
		RequestID: "req_test",
	}, "json")

	var payload struct {
		Success   bool            `json:"success"`
		Error     client.APIError `json:"error"`
		RequestID string          `json:"request_id"`
	}
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Success ||
		payload.Error.Field != "parent_id" ||
		payload.Error.Constraint != "non_negative_decimal_integer" ||
		payload.RequestID != "req_test" {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestWriteErrorAddsCLIMembershipPurchaseURL(t *testing.T) {
	requestErr := &client.RequestError{
		APIError: client.APIError{
			Code:    10201,
			Message: "OpenAPI 仅对会员开放",
			Reason:  "not_member",
		},
	}

	var jsonOutput bytes.Buffer
	writeError(&jsonOutput, requestErr, "json")
	var payload struct {
		Error client.APIError `json:"error"`
	}
	if err := json.Unmarshal(jsonOutput.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Error.MembershipURL != client.MembershipPurchaseURL {
		t.Fatalf("membership URL = %q", payload.Error.MembershipURL)
	}

	var textOutput bytes.Buffer
	writeError(&textOutput, requestErr, "table")
	if !strings.Contains(textOutput.String(), client.MembershipPurchaseURL) {
		t.Fatalf("text error missing membership URL: %s", textOutput.String())
	}
}
