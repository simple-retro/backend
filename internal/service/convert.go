package service

import (
	"api/types"
	"context"
	"fmt"
	"sort"
	"strings"
)

func (s *Service) ConvertRetrospectiveToMarkdown(ctx context.Context, retro *types.Retrospective) string {
	var sb strings.Builder

	sb.WriteString("# Simple Retro\n\n")

	sb.WriteString("## " + retro.Name + "\n\n")

	if retro.Description != "" {
		sb.WriteString(retro.Description + "\n\n")
	}

	sb.WriteString("*Created on " + retro.CreatedAt.Format("January 2, 2006 at 3:04 PM") + "*\n\n")

	sb.WriteString("---\n\n")

	for _, question := range retro.Questions {
		sb.WriteString("### " + question.Text + "\n\n")

		answers := make([]types.Answer, len(question.Answers))
		copy(answers, question.Answers)
		sort.Slice(answers, func(i, j int) bool {
			return answers[i].Position < answers[j].Position
		})

		for _, answer := range answers {
			if answer.Votes > 0 {
				sb.WriteString(fmt.Sprintf("- %s (%d Up votes)\n", answer.Text, answer.Votes))
			} else {
				sb.WriteString("- " + answer.Text + "\n")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
