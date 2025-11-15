package models_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"event-api/internal/models"
)

func TestUpdateRoleRequestTags(t *testing.T) {
	typ := reflect.TypeOf(models.UpdateRoleRequest{})

	userField, ok := typ.FieldByName("UserID")
	if !ok {
		t.Fatalf("UserID field missing")
	}
	if tag := userField.Tag.Get("json"); tag != "user_id" {
		t.Fatalf("unexpected json tag for UserID: %s", tag)
	}
	if binding := userField.Tag.Get("binding"); !strings.Contains(binding, "required") {
		t.Fatalf("expected UserID binding tag to contain required, got %s", binding)
	}

	roleField, ok := typ.FieldByName("Role")
	if !ok {
		t.Fatalf("Role field missing")
	}
	if tag := roleField.Tag.Get("json"); tag != "role" {
		t.Fatalf("unexpected json tag for Role: %s", tag)
	}
	if binding := roleField.Tag.Get("binding"); !strings.Contains(binding, "required") {
		t.Fatalf("expected Role binding tag to contain required, got %s", binding)
	}
}

func TestUpdateRoleRequestJSONRoundTrip(t *testing.T) {
	original := models.UpdateRoleRequest{UserID: "user-123", Role: models.RoleAdmin}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded models.UpdateRoleRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded != original {
		t.Fatalf("round trip mismatch: %+v vs %+v", decoded, original)
	}
}
