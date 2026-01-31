package service

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"api/types"

	"github.com/go-pdf/fpdf"
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

func (s *Service) ConvertRetrospectiveToPDF(ctx context.Context, retro *types.Retrospective) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	// Title
	pdf.SetFont("Arial", "B", 24)
	pdf.Cell(0, 12, "Simple Retro")
	pdf.Ln(16)

	// Subtitle (retrospective name)
	pdf.SetFont("Arial", "B", 18)
	pdf.Cell(0, 10, retro.Name)
	pdf.Ln(12)

	// Description
	if retro.Description != "" {
		pdf.SetFont("Arial", "", 12)
		pdf.MultiCell(0, 6, retro.Description, "", "", false)
		pdf.Ln(4)
	}

	// Created at
	pdf.SetFont("Arial", "I", 10)
	pdf.SetTextColor(128, 128, 128)
	pdf.Cell(0, 6, "Created on "+retro.CreatedAt.Format("January 2, 2006 at 3:04 PM"))
	pdf.Ln(10)
	pdf.SetTextColor(0, 0, 0)

	// Separator line
	pdf.Line(20, pdf.GetY(), 190, pdf.GetY())
	pdf.Ln(8)

	// Questions and answers
	for _, question := range retro.Questions {
		// Question title
		pdf.SetFont("Arial", "B", 14)
		pdf.MultiCell(0, 8, question.Text, "", "", false)
		pdf.Ln(4)

		// Sort answers by position
		answers := make([]types.Answer, len(question.Answers))
		copy(answers, question.Answers)
		sort.Slice(answers, func(i, j int) bool {
			return answers[i].Position < answers[j].Position
		})

		pdf.SetFont("Arial", "", 11)
		for _, answer := range answers {
			var text string
			if answer.Votes > 0 {
				text = fmt.Sprintf("  - %s (%d votes)", answer.Text, answer.Votes)
			} else {
				text = fmt.Sprintf("  - %s", answer.Text)
			}
			pdf.MultiCell(0, 6, text, "", "", false)
			pdf.Ln(1)
		}
		pdf.Ln(6)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
