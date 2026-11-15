package handler

import (
	"net/http/httptest"
	"strings"
	"testing"
)

type decodePayload struct {
	Name string `json:"name"`
}

func TestDecodeBodyRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"blueprint","extra":true}`))
	recorder := httptest.NewRecorder()

	var payload decodePayload
	if err := decodeBody(recorder, req, &payload); err == nil {
		t.Fatal("decodeBody() accepted an unknown field")
	}
}

func TestDecodeBodyRejectsTrailingJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"blueprint"}{"name":"second"}`))
	recorder := httptest.NewRecorder()

	var payload decodePayload
	if err := decodeBody(recorder, req, &payload); err == nil {
		t.Fatal("decodeBody() accepted trailing JSON")
	}
}
