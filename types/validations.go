package types

import "fmt"

const (
	NAME_LIMIT   = 100
	DESC_LIMIT   = 300
	ANSWER_LIMIT = 300
)

func (r *RetrospectiveCreateRequest) ValidateCreate() error {
	if len(r.Name) == 0 {
		return fmt.Errorf("retrospective name cannot be empty")
	}

	if len(r.Name) > NAME_LIMIT {
		return fmt.Errorf("retrospective name too big. Limit is %d", NAME_LIMIT)
	}

	if len(r.Description) > DESC_LIMIT {
		return fmt.Errorf("retrospective description too big. Limit is %d", DESC_LIMIT)
	}

	return nil
}

func (r *RetrospectiveCreateRequest) ValidateUpdate() error {
	if len(r.Name) == 0 && len(r.Description) == 0 {
		return fmt.Errorf("nothing to do")
	}

	if len(r.Name) > NAME_LIMIT {
		return fmt.Errorf("retrospective name too big. Limit is %d", NAME_LIMIT)
	}

	if len(r.Description) > DESC_LIMIT {
		return fmt.Errorf("retrospective description too big. Limit is %d", DESC_LIMIT)
	}

	return nil
}

func (r *QuestionCreateRequest) ValidateCreate() error {
	if len(r.Text) == 0 {
		return fmt.Errorf("question text cannot be empty")
	}

	if len(r.Text) > DESC_LIMIT {
		return fmt.Errorf("question name too big. Limit is %d", NAME_LIMIT)
	}

	return nil
}

func (a *AnswerCreateRequest) ValidateCreate() error {
	if len(a.Text) > ANSWER_LIMIT {
		return fmt.Errorf("answer text too big. Limit is %d", ANSWER_LIMIT)
	}

	if len(a.QuestionID.String()) == 0 {
		return fmt.Errorf("question id cannot be empty")
	}

	return nil
}
