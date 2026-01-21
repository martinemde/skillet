package completion

import (
	"io"
	"strings"
	"text/template"
)

var zshTemplate = `#compdef skillet

# Zsh completion for skillet
# Install: source <(skillet completion zsh)
# Or: skillet completion zsh > "${fpath[1]}/_skillet"

_skillet() {
    _arguments -C \
        '--version[Show version information]' \
        '--help[Show help information]' \
        '--list[List all available skills and commands]' \
        '--verbose[Show detailed output including thinking and tool details]' \
        '--debug[Print raw JSON stream to stderr]' \
        '--usage[Show token usage statistics]' \
        '--dry-run[Show the command that would be executed without running it]' \
        {-q,--quiet}'[Quiet mode - suppress all output except errors]' \
        '--parse[Parse and format stream-json input]:file:_files' \
        '--prompt[Prompt to pass to Claude]:prompt:' \
        '--model[Override model to use]:model:({{.ModelValues}})' \
        '--allowed-tools[Override allowed tools]:tools:({{.ToolValues}})' \
        '--permission-mode[Override permission mode]:mode:({{.PermissionModeValues}})' \
        '--output-format[Override output format]:format:({{.OutputFormatValues}})' \
        '--color[Control color output]:color:({{.ColorValues}})' \
        '1:skill or command:_skillet_names' \
        '*::arguments:_files'
}

_skillet_names() {
    local -a names
    names=(${(f)"$(skillet --complete "${words[CURRENT]}" 2>/dev/null)"})
    if [[ ${#names[@]} -gt 0 ]]; then
        _describe -t skills 'skills and commands' names
    fi
}

# Register completion function (works when sourced directly)
if [[ -n ${_comps+1} ]]; then
    compdef _skillet skillet
fi
`

// GenerateZsh writes the zsh completion script to the writer.
func GenerateZsh(w io.Writer) error {
	tmpl, err := template.New("zsh").Parse(zshTemplate)
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
