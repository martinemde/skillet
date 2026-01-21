package completion

import (
	"io"
	"strings"
	"text/template"
)

var fishTemplate = `# Fish completion for skillet
# Install: skillet completion fish | source
# Or: skillet completion fish > ~/.config/fish/completions/skillet.fish

# Disable file completion by default
complete -c skillet -f

# Helper function to check if we need skill/command completion
function __skillet_needs_command
    set -l tokens (commandline -opc)
    set -l skip_next 0
    for token in $tokens[2..-1]
        if test $skip_next -eq 1
            set skip_next 0
            continue
        end
        switch $token
            case '--parse' '--prompt' '--model' '--allowed-tools' '--permission-mode' '--output-format' '--color'
                set skip_next 1
            case '-*'
                # Boolean flag, continue
            case '*'
                # Found a positional argument
                return 1
        end
    end
    return 0
end

# Helper function to get skill/command names
function __skillet_complete_names
    set -l cur (commandline -ct)
    skillet --complete "$cur" 2>/dev/null
end

# Boolean flags
complete -c skillet -l version -d 'Show version information'
complete -c skillet -l help -d 'Show help information'
complete -c skillet -l list -d 'List all available skills and commands'
complete -c skillet -l verbose -d 'Show detailed output including thinking and tool details'
complete -c skillet -l debug -d 'Print raw JSON stream to stderr'
complete -c skillet -l usage -d 'Show token usage statistics'
complete -c skillet -l dry-run -d 'Show the command that would be executed without running it'
complete -c skillet -s q -l quiet -d 'Quiet mode - suppress all output except errors'

# Flags with values
complete -c skillet -l parse -r -F -d 'Parse and format stream-json input'
complete -c skillet -l prompt -r -d 'Prompt to pass to Claude'
complete -c skillet -l model -r -f -a '{{.ModelValues}}' -d 'Override model to use'
complete -c skillet -l allowed-tools -r -f -a '{{.ToolValues}}' -d 'Override allowed tools'
complete -c skillet -l permission-mode -r -f -a '{{.PermissionModeValues}}' -d 'Override permission mode'
complete -c skillet -l output-format -r -f -a '{{.OutputFormatValues}}' -d 'Override output format'
complete -c skillet -l color -r -f -a '{{.ColorValues}}' -d 'Control color output'

# Skill and command names (only when no positional arg yet)
complete -c skillet -n '__skillet_needs_command' -a '(__skillet_complete_names)' -d 'Skill or command'
`

// GenerateFish writes the fish completion script to the writer.
func GenerateFish(w io.Writer) error {
	tmpl, err := template.New("fish").Parse(fishTemplate)
	if err != nil {
		return err
	}

	data := struct {
		ColorValues          string
		ModelValues          string
		ToolValues           string
		PermissionModeValues string
		OutputFormatValues   string
	}{
		ColorValues:          strings.Join(ColorValues, " "),
		ModelValues:          strings.Join(ModelValues, " "),
		ToolValues:           strings.Join(ToolValues, " "),
		PermissionModeValues: strings.Join(PermissionModeValues, " "),
		OutputFormatValues:   strings.Join(OutputFormatValues, " "),
	}

	return tmpl.Execute(w, data)
}
