package surface

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

const (
	SchemaV1                      = "tetra.surface.runtime.v1"
	ReleaseSchemaV1               = "tetra.surface.release.v1"
	RendererFeatureSchemaV1       = "tetra.surface.renderer-feature.v1"
	TextInputSchemaV1             = "tetra.surface.text-input.v1"
	LinuxAppShellSchemaV1         = "tetra.surface.linux-app-shell.v1"
	BrowserSurfaceSchemaV1        = "tetra.surface.browser-surface.v1"
	SecurityPermissionSchemaV1    = "tetra.surface.security-permission.v1"
	PerformanceBudgetSchemaV1     = "tetra.surface.performance-budget.v1"
	TargetHostStatusSchemaV1      = "tetra.surface.target-host-status.v1"
	ReleaseScopeSurfaceV1LinuxWeb = "surface-v1-linux-web"
)

func decodeSchema(raw []byte) (string, error) {
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return "", err
	}
	if strings.TrimSpace(header.Schema) == "" {
		return "", errors.New("schema is required")
	}
	return header.Schema, nil
}

func decodeStrict(raw []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if dec.More() {
		return errors.New("unexpected trailing JSON")
	}
	return nil
}
