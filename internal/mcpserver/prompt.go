package mcpserver

import "fmt"

// Question represents a single question from AskUserQuestion (local parsing type)
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
