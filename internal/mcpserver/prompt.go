package mcpserver

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

// Question represents a single question from AskUserQuestion
type Question struct {
	Question    string   `json:"question"`
	Header      string   `json:"header"`
	Options     []Option `json:"options"`
	MultiSelect bool     `json:"multiSelect"`
}

// Option represents a selectable option for a question
type Option struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// parseQuestions extracts questions from the AskUserQuestion tool input
func parseQuestions(toolInput map[string]any) ([]Question, error) {
	questionsRaw, ok := toolInput["questions"]
	if !ok {
		return nil, fmt.Errorf("missing 'questions' field in input")
	}

	questionsSlice, ok := questionsRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("'questions' is not an array")
	}

	questions := make([]Question, 0, len(questionsSlice))
	for i, qRaw := range questionsSlice {
		qMap, ok := qRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("question %d is not an object", i)
		}

		q := Question{}

		if v, ok := qMap["question"].(string); ok {
			q.Question = v
		} else {
			return nil, fmt.Errorf("question %d missing 'question' field", i)
		}

		if v, ok := qMap["header"].(string); ok {
			q.Header = v
		}

		if v, ok := qMap["multiSelect"].(bool); ok {
			q.MultiSelect = v
		}

		if optionsRaw, ok := qMap["options"].([]any); ok {
			for j, optRaw := range optionsRaw {
				optMap, ok := optRaw.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("question %d option %d is not an object", i, j)
				}

				opt := Option{}
				if v, ok := optMap["label"].(string); ok {
					opt.Label = v
				}
				if v, ok := optMap["description"].(string); ok {
					opt.Description = v
				}
				q.Options = append(q.Options, opt)
			}
		}

		questions = append(questions, q)
	}

	return questions, nil
}

// promptUser displays questions using huh and collects answers
func promptUser(questions []Question) (map[string]string, error) {
	answers := make(map[string]string)

	for _, q := range questions {
		var answer string

		if q.MultiSelect {
			answer = promptMultiSelect(q)
		} else {
			answer = promptSelect(q)
		}

		answers[q.Question] = answer
	}

	return answers, nil
}

// promptSelect handles single-select questions
func promptSelect(q Question) string {
	options := make([]huh.Option[string], 0, len(q.Options)+1)
	for _, opt := range q.Options {
		options = append(options, huh.NewOption(opt.Label, opt.Label))
	}
	// Add "Other" option for free text input
	options = append(options, huh.NewOption("Other...", "__other__"))

	var answer string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(q.Header).
				Description(q.Question).
				Options(options...).
				Value(&answer),
		),
	)

	if err := form.Run(); err != nil {
		return ""
	}

	if answer == "__other__" {
		return promptFreeText(q.Header)
	}

	return answer
}

// promptMultiSelect handles multi-select questions
func promptMultiSelect(q Question) string {
	options := make([]huh.Option[string], 0, len(q.Options)+1)
	for _, opt := range q.Options {
		options = append(options, huh.NewOption(opt.Label, opt.Label))
	}
	// Add "Other" option for free text input
	options = append(options, huh.NewOption("Other...", "__other__"))

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title(q.Header).
				Description(q.Question).
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return ""
	}

	// Check if "Other" was selected
	hasOther := false
	filtered := make([]string, 0, len(selected))
	for _, s := range selected {
		if s == "__other__" {
			hasOther = true
		} else {
			filtered = append(filtered, s)
		}
	}

	if hasOther {
		other := promptFreeText(q.Header)
		if other != "" {
			filtered = append(filtered, other)
		}
	}

	return strings.Join(filtered, ", ")
}

// promptFreeText prompts for free text input
func promptFreeText(title string) string {
	var text string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Description("Enter your answer").
				Value(&text),
		),
	)

	if err := form.Run(); err != nil {
		return ""
	}

	return text
}
