package export

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf/v2"
	"github.com/xuri/excelize/v2"

	"apiPos/internal/accounting/report"
)

// Service renders statements to PDF / XLSX.
type Service struct {
	report *report.Service
}

// NewService creates an export Service.
func NewService(r *report.Service) *Service { return &Service{report: r} }

// Export renders the requested statement in the requested format.
// report is one of: trial-balance, general-ledger, general-journal,
// income-statement, balance-sheet, changes-in-equity, cash-flow, validate.
// format is pdf | xlsx.
func (s *Service) Export(ctx context.Context, reportName, format, startDate, endDate string) ([]byte, string, string, error) {
	stmt, err := s.fetch(ctx, reportName, startDate, endDate)
	if err != nil {
		return nil, "", "", err
	}
	switch strings.ToLower(format) {
	case "pdf":
		return s.renderPDF(stmt)
	case "xlsx":
		return s.renderXLSX(stmt)
	default:
		return nil, "", "", fmt.Errorf("unsupported format: %s", format)
	}
}

func (s *Service) fetch(ctx context.Context, reportName, startDate, endDate string) (*report.StatementResult, error) {
	switch reportName {
	case "trial-balance":
		return s.report.TrialBalance(ctx, startDate, endDate)
	case "income-statement":
		return s.report.IncomeStatement(ctx, startDate, endDate)
	case "balance-sheet":
		return s.report.BalanceSheet(ctx, endDate)
	case "changes-in-equity":
		return s.report.ChangesInEquity(ctx, startDate, endDate)
	case "cash-flow":
		return s.report.CashFlow(ctx, startDate, endDate)
	default:
		// General journal / ledger / validate fall through to trial balance
		// for export purposes (rows are capped at service level).
		return s.report.TrialBalance(ctx, startDate, endDate)
	}
}

func (s *Service) renderPDF(stmt *report.StatementResult) ([]byte, string, string, error) {
	// Portrait A4, basic latin (Indonesian-safe: no diacritics).
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.Cell(0, 10, stmt.Title)
	pdf.Ln(8)
	pdf.SetFont("Helvetica", "", 10)
	pdf.Cell(0, 6, fmt.Sprintf("Period: %s to %s", stmt.StartDate, stmt.EndDate))
	pdf.Ln(6)
	pdf.Cell(0, 6, "Generated: "+time.Now().Format("2006-01-02 15:04"))
	pdf.Ln(12)

	// Table header.
	pdf.SetFont("Helvetica", "B", 9)
	colW := []float64{25, 70, 20, 30, 30, 30}
	headers := []string{"Code", "Account", "Type", "Debit", "Credit", "Balance"}
	for i, h := range headers {
		pdf.CellFormat(colW[i], 8, h, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetFont("Helvetica", "", 9)
	for _, l := range stmt.Lines {
		pdf.CellFormat(colW[0], 7, l.AccountCode, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colW[1], 7, safeText(l.AccountName), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colW[2], 7, safeText(l.AccountType), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[3], 7, l.Debit.StringFixed(2), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colW[4], 7, l.Credit.StringFixed(2), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colW[5], 7, l.Balance.StringFixed(2), "1", 0, "R", false, 0, "")
		pdf.Ln(-1)
	}
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(colW[0]+colW[1]+colW[2], 8, "Total", "1", 0, "R", false, 0, "")
	pdf.CellFormat(colW[3], 8, stmt.TotalDebit.StringFixed(2), "1", 0, "R", false, 0, "")
	pdf.CellFormat(colW[4], 8, stmt.TotalCredit.StringFixed(2), "1", 0, "R", false, 0, "")
	pdf.CellFormat(colW[5], 8, stmt.TotalBalance.StringFixed(2), "1", 0, "R", false, 0, "")
	pdf.Ln(-1)

	if len(stmt.Warnings) > 0 {
		pdf.Ln(4)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.Cell(0, 8, "Validation Warnings")
		pdf.Ln(8)
		pdf.SetFont("Helvetica", "", 9)
		for _, w := range stmt.Warnings {
			pdf.Cell(0, 6, fmt.Sprintf("[%s] %s (%s)", w.Severity, safeText(w.Message), w.Code))
			pdf.Ln(-1)
		}
	}

	var buf strings.Builder
	// gofpdf v2 writes to io.Writer; capture via bytes buffer.
	pdfBuf, err := writePDFToBytes(pdf)
	if err != nil {
		return nil, "", "", err
	}
	_ = buf
	filename := fmt.Sprintf("%s_%s.pdf", strings.ToLower(strings.ReplaceAll(stmt.Title, " ", "-")), time.Now().Format("20060102"))
	return pdfBuf, filename, "application/pdf", nil
}

func (s *Service) renderXLSX(stmt *report.StatementResult) ([]byte, string, string, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	sheet := "Statement"
	f.SetSheetName("Sheet1", sheet)

	// Title row.
	_ = f.SetCellValue(sheet, "A1", stmt.Title)
	_ = f.SetCellValue(sheet, "A2", fmt.Sprintf("Period: %s to %s", stmt.StartDate, stmt.EndDate))
	_ = f.SetCellValue(sheet, "A3", "Generated: "+time.Now().Format("2006-01-02 15:04"))

	headers := []string{"Code", "Account", "Type", "Debit", "Credit", "Balance", "Prior", "Delta"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 5)
		_ = f.SetCellValue(sheet, cell, h)
	}
	row := 6
	for _, l := range stmt.Lines {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), l.AccountCode)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), l.AccountName)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), l.AccountType)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", row), l.Debit.StringFixed(2))
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), l.Credit.StringFixed(2))
		_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", row), l.Balance.StringFixed(2))
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), l.PriorBalance.StringFixed(2))
		_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", row), l.Delta.StringFixed(2))
		row++
	}
	// Totals.
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), "Total")
	_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", row), stmt.TotalDebit.StringFixed(2))
	_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), stmt.TotalCredit.StringFixed(2))
	_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", row), stmt.TotalBalance.StringFixed(2))

	// Freeze header row (best-effort) + write buffer.
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", "", err
	}
	filename := fmt.Sprintf("%s_%s.xlsx", strings.ToLower(strings.ReplaceAll(stmt.Title, " ", "-")), time.Now().Format("20060102"))
	return buf.Bytes(), filename, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
}

func safeText(s string) string {
	// gofpdf Helvetica is Latin-1; replace non-printable.
	var b strings.Builder
	for _, r := range s {
		if r < 32 || r > 255 {
			b.WriteRune('?')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// writePDFToBytes captures a gofpdf document into bytes.
func writePDFToBytes(pdf *gofpdf.Fpdf) ([]byte, error) {
	var buf strings.Builder
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}
