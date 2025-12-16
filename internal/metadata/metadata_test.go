package metadata

import (
	"testing"

	"github.com/sleuth-io/skills/internal/asset"
)

func TestParseValidMetadata(t *testing.T) {
	metadataData := []byte(`
[asset]
name = "test-skill"
version = "1.0.0"
type = "skill"
description = "A test skill"
authors = ["Test Author <test@example.com>"]
`)

	meta, err := Parse(metadataData)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if meta.Asset.Name != "test-skill" {
		t.Errorf("Expected name test-skill, got %s", meta.Asset.Name)
	}

	if meta.Asset.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", meta.Asset.Version)
	}

	if meta.Asset.Type != asset.TypeSkill {
		t.Errorf("Expected type skill, got %s", meta.Asset.Type)
	}

	if len(meta.Asset.Authors) > 0 && meta.Asset.Authors[0] != "Test Author <test@example.com>" {
		t.Errorf("Expected author 'Test Author <test@example.com>', got %s", meta.Asset.Authors[0])
	}
}

func TestValidateMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata *Metadata
		wantErr  bool
	}{
		{
			name: "valid metadata",
			metadata: &Metadata{
				Asset: Asset{
					Name:        "test-skill",
					Version:     "1.0.0",
					Type:        asset.TypeSkill,
					Description: "A test skill",
					Authors:     []string{"Test Author"},
				},
				Skill: &SkillConfig{
					PromptFile: "prompt.md",
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			metadata: &Metadata{
				Asset: Asset{
					Version:     "1.0.0",
					Type:        asset.TypeSkill,
					Description: "A test skill",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid semver",
			metadata: &Metadata{
				Asset: Asset{
					Name:        "test-skill",
					Version:     "invalid",
					Type:        asset.TypeSkill,
					Description: "A test skill",
				},
			},
			wantErr: true,
		},
		{
			name: "missing description",
			metadata: &Metadata{
				Asset: Asset{
					Name:    "test-skill",
					Version: "1.0.0",
					Type:    asset.TypeSkill,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetadataWithDependencies(t *testing.T) {
	metadataData := []byte(`
[asset]
name = "test-skill"
version = "1.0.0"
type = "skill"
description = "A test skill with dependencies"
dependencies = ["dep1", "dep2"]
`)

	meta, err := Parse(metadataData)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(meta.Asset.Dependencies) != 2 {
		t.Fatalf("Expected 2 dependencies, got %d", len(meta.Asset.Dependencies))
	}

	if meta.Asset.Dependencies[0] != "dep1" {
		t.Errorf("Expected first dependency 'dep1', got %s", meta.Asset.Dependencies[0])
	}

	if meta.Asset.Dependencies[1] != "dep2" {
		t.Errorf("Expected second dependency 'dep2', got %s", meta.Asset.Dependencies[1])
	}
}
