package provider

import (
	"encoding/json"
	"os"
	"testing"
)

// The Terraform Registry reads terraform-registry-manifest.json to learn which
// plugin protocol versions the provider serves. The provider uses the framework
// with protocol v6, so the manifest must declare exactly that.
func TestRegistryManifestDeclaresProtocolV6(t *testing.T) {
	raw, err := os.ReadFile("../../terraform-registry-manifest.json")
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var manifest struct {
		Version  int `json:"version"`
		Metadata struct {
			ProtocolVersions []string `json:"protocol_versions"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if manifest.Version != 1 {
		t.Fatalf("manifest version = %d, want 1", manifest.Version)
	}
	if len(manifest.Metadata.ProtocolVersions) != 1 || manifest.Metadata.ProtocolVersions[0] != "6.0" {
		t.Fatalf("protocol_versions = %v, want [6.0]", manifest.Metadata.ProtocolVersions)
	}
}
