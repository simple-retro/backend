package types

import "fmt"

const (
	NAME_LIMIT   = 100
	DESC_LIMIT   = 300
	ANSWER_LIMIT = 600
)

type ApiLimits struct {
	Retrospective limits `json:"retrospective,omitempty"`
	Question      limits `json:"question,omitempty"`
	Answer        limits `json:"answer,omitempty"`
}

type limits struct {
	Name        int `json:"name,omitempty"`
	Text        int `json:"text,omitempty"`
	Description int `json:"description,omitempty"`
}

func GetApiLimits() *ApiLimits {
	return &ApiLimits{
		Retrospective: limits{
			Name:        NAME_LIMIT,
			Description: DESC_LIMIT,
		},
		Question: limits{
			Text: DESC_LIMIT,
		},
		Answer: limits{
			Text: ANSWER_LIMIT,
		},
	}
}

func (r *RetrospectiveCreateRequest) ValidateCreate() error {
	if len(r.Name) == 0 {
		return fmt.Errorf("retrospective name cannot be empty")
	}

	retroLimits := GetApiLimits().Retrospective

	if len(r.Name) > retroLimits.Name {
		return fmt.Errorf("retrospective name too big. Limit is %d", retroLimits.Name)
	}

	if len(r.Description) > retroLimits.Description {
		return fmt.Errorf("retrospective description too big. Limit is %d", retroLimits.Description)
	}

	return nil
}

func (r *RetrospectiveCreateRequest) ValidateUpdate() error {
	if len(r.Name) == 0 && len(r.Description) == 0 {
		return fmt.Errorf("nothing to do")
	}

	retroLimits := GetApiLimits().Retrospective

	if len(r.Name) > retroLimits.Name {
		return fmt.Errorf("retrospective name too big. Limit is %d", retroLimits.Description)
	}

	if len(r.Description) > retroLimits.Description {
		return fmt.Errorf("retrospective description too big. Limit is %d", DESC_LIMIT)
	}

	return nil
}

func (r *QuestionCreateRequest) ValidateCreate() error {
	if len(r.Text) == 0 {
		return fmt.Errorf("question text cannot be empty")
	}

	questionLimits := GetApiLimits().Question

	if len(r.Text) > questionLimits.Text {
		return fmt.Errorf("question too big. Limit is %d", questionLimits.Text)
	}

	return nil
}

func (a *AnswerCreateRequest) ValidateCreate() error {
	answerLimits := GetApiLimits().Answer
	if len(a.Text) > answerLimits.Text {
		return fmt.Errorf("answer text too big. Limit is %d", answerLimits.Text)
	}

	if len(a.QuestionID.String()) == 0 {
		return fmt.Errorf("question id cannot be empty")
	}

	return nil
}
