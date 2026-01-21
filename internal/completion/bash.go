package completion

import (
	"io"
	"strings"
	"text/template"
)

var bashTemplate = `# Bash completion for skillet
# Install: source <(skillet completion bash)
# Or: skillet completion bash > /etc/bash_completion.d/skillet

_skillet_completions() {
    local cur prev words cword
    _init_completion || return

    local flags="--version --help --list --verbose --debug --usage --dry-run -q --quiet --parse --prompt --model --allowed-tools --permission-mode --output-format --color"
    local bool_flags="--version --help --list --verbose --debug --usage --dry-run -q --quiet"

    case "${prev}" in
        --color)
            COMPREPLY=($(compgen -W "{{.ColorValues}}" -- "${cur}"))
            return 0
            ;;
        --model)
            COMPREPLY=($(compgen -W "{{.ModelValues}}" -- "${cur}"))
            return 0
            ;;
        --allowed-tools)
            COMPREPLY=($(compgen -W "{{.ToolValues}}" -- "${cur}"))
            return 0
            ;;
        --permission-mode)
            COMPREPLY=($(compgen -W "{{.PermissionModeValues}}" -- "${cur}"))
            return 0
            ;;
        --output-format)
            COMPREPLY=($(compgen -W "{{.OutputFormatValues}}" -- "${cur}"))
            return 0
            ;;
        --parse)
            _filedir
            return 0
            ;;
        --prompt)
            # Free text, no completion
            return 0
            ;;
    esac

    # Check if we're completing a flag
    if [[ "${cur}" == -* ]]; then
        COMPREPLY=($(compgen -W "${flags}" -- "${cur}"))
        return 0
    fi

    # Check if we've already seen a positional argument (skill/command name)
    local has_positional=0
    for ((i=1; i < cword; i++)); do
        local word="${words[i]}"
        # Skip flags and their values
        if [[ "${word}" == -* ]]; then
            # If this is a non-bool flag, skip its value too
            if [[ ! " ${bool_flags} " =~ " ${word} " ]]; then
                ((i++))
            fi
            continue
        fi
        has_positional=1
        break
    done

    # If no positional argument yet, complete skill/command names
    if [[ ${has_positional} -eq 0 ]]; then
        local names
        names=$(skillet --complete "${cur}" 2>/dev/null)
        if [[ -n "${names}" ]]; then
            COMPREPLY=($(compgen -W "${names}" -- "${cur}"))
        fi
    fi
}

complete -F _skillet_completions skillet
`

// GenerateBash writes the bash completion script to the writer.
func GenerateBash(w io.Writer) error {
	tmpl, err := template.New("bash").Parse(bashTemplate)
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
