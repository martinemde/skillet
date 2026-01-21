package completion

import (
	"bytes"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name      string
		shell     string
		wantErr   bool
		contains  []string
		errSubstr string
	}{
		{
			name:  "bash generates valid script",
			shell: "bash",
			contains: []string{
				"_skillet_completions",
				"complete -F",
				"--color",
				"--model",
			},
		},
		{
			name:  "zsh generates valid script",
			shell: "zsh",
			contains: []string{
				"#compdef skillet",
				"_skillet",
				"_arguments",
			},
		},
		{
			name:  "fish generates valid script",
			shell: "fish",
			contains: []string{
				"complete -c skillet",
				"__skillet_needs_command",
			},
		},
		{
			name:      "unsupported shell returns error",
			shell:     "powershell",
			wantErr:   true,
			errSubstr: "unsupported shell",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := Generate(&buf, tt.shell)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Generate() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("Generate() error = %v, want error containing %q", err, tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := buf.String()
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("Generate() output missing %q", want)
				}
			}
		})
	}
}

func TestSupportedShells(t *testing.T) {
	shells := SupportedShells()

	// Should return at least bash, zsh, fish
	want := []string{"bash", "fish", "zsh"}
	if len(shells) != len(want) {
		t.Errorf("SupportedShells() = %v, want %v", shells, want)
	}

	for i, shell := range want {
		if shells[i] != shell {
			t.Errorf("SupportedShells()[%d] = %v, want %v", i, shells[i], shell)
		}
	}
}

func TestPrintCompletions(t *testing.T) {
	var buf bytes.Buffer
	PrintCompletions(&buf, "")

	// Output should have names separated by newlines
	output := buf.String()
	if output != "" {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				t.Error("PrintCompletions() output contains empty line")
			}
		}
	}
}

func TestFlagValues(t *testing.T) {
	// Verify that flag value lists are non-empty and reasonable
	if len(ColorValues) == 0 {
		t.Error("ColorValues is empty")
	}
	if len(ModelValues) == 0 {
		t.Error("ModelValues is empty")
	}
	if len(ToolValues) == 0 {
		t.Error("ToolValues is empty")
	}
	if len(PermissionModeValues) == 0 {
		t.Error("PermissionModeValues is empty")
	}
	if len(OutputFormatValues) == 0 {
		t.Error("OutputFormatValues is empty")
	}

	// Verify color values
	expected := []string{"auto", "always", "never"}
	if len(ColorValues) != len(expected) {
		t.Errorf("ColorValues = %v, want %v", ColorValues, expected)
	}
}
