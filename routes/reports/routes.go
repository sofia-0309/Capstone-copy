package reports

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"

	"github.com/jung-kurt/gofpdf"
	model "gitlab.msu.edu/team-corewell-2025/models"
)

type ReportData struct {
	User      model.User `json:"user"`
	Analytics struct {
		QuestionTasks        int     `json:"questionTasks"`
		LabTasks             int     `json:"labTasks"`
		PrescriptionTasks    int     `json:"prescriptionTasks"`
		TotalTimeSpent       string  `json:"totalTimeSpent"`
		AverageQuizScore     float64 `json:"averageQuizScore"`
		QuizCompletionRating float64 `json:"quizCompletionRating"`
		MaxTaskStreak        int     `json:"maxTaskStreak"`
		MaxQuizStreak        int     `json:"maxQuizStreak"`
		AchievementCount     int     `json:"achievementCount"`
	} `json:"analytics"`
}

type ReportHandler struct{}

func drawPieChart(pdf *gofpdf.Fpdf, x, y, radius float64, taskAmount int, taskType string) {
	tasksComplete := float64(min(taskAmount, 28))
	percentage := (tasksComplete / 28) * 100

	if tasksComplete == 0 {
		pdf.SetFillColor(230, 230, 230)
		pdf.Circle(x, y, radius, "F")

		pdf.SetFont("Helvetica", "B", 12)
		pdf.SetTextColor(0, 0, 0)
		pdf.SetXY(x-radius+5, y-3)
		pdf.CellFormat((radius-5)*2, 6, fmt.Sprintf("%.0f%%", percentage), "", 0, "C", false, 0, "")

	} else if tasksComplete == 28 {
		pdf.SetFillColor(2, 92, 194)
		pdf.Circle(x, y, radius, "F")

		pdf.SetFont("Helvetica", "B", 12)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetXY(x-radius+5, y-3)
		pdf.CellFormat((radius-5)*2, 6, fmt.Sprintf("%.0f%%", percentage), "", 0, "C", false, 0, "")

	} else {
		angle := 360.0 * (tasksComplete / 28)

		pdf.SetFillColor(230, 230, 230)
		pdf.Circle(x, y, radius, "F")

		pdf.SetFillColor(2, 92, 194)

		pdf.MoveTo(x, y)
		pdf.ArcTo(x, y, radius, radius, 0, 90-angle, 90)
		pdf.ClosePath()
		pdf.DrawPath("F")

		halfwayAngle := (90 - angle/2) * (math.Pi / 180.0)
		textAdjustment := radius / 2
		textX := x + textAdjustment*math.Cos(halfwayAngle)
		textY := y - textAdjustment*math.Sin(halfwayAngle)

		pdf.SetFont("Helvetica", "B", 12)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetXY(textX-10, textY-3)
		pdf.CellFormat(20, 6, fmt.Sprintf("%.0f%%", percentage), "", 0, "C", false, 0, "")

		remainingPercentage := 100 - percentage

		if remainingPercentage > 15 {
			pdf.SetTextColor(0, 0, 0)

			remainingAngle := (-90 - angle/2) * (math.Pi / 180.0)
			remainingX := x + textAdjustment*math.Cos(remainingAngle)
			remainingY := y - textAdjustment*math.Sin(remainingAngle)

			pdf.SetXY(remainingX-10, remainingY-3)
			pdf.CellFormat(20, 6, fmt.Sprintf("%.0f%%", remainingPercentage), "", 0, "C", false, 0, "")
		}
	}

	pdf.SetTextColor(0, 0, 0)
	pdf.SetDrawColor(100, 100, 100)
	pdf.SetLineWidth(0.3)
	pdf.Circle(x, y, radius, "D")

	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetXY(x-radius, y+radius+3)
	pdf.CellFormat(radius*2, 4, taskType, "", 0, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetXY(x-radius, y+radius+7)
	pdf.CellFormat(radius*2, 4, fmt.Sprintf("%d/28", min(taskAmount, 28)), "", 0, "C", false, 0, "")
}

func (h *ReportHandler) GenerateProgressReport(w http.ResponseWriter, r *http.Request) {
	var report ReportData
	err := json.NewDecoder(r.Body).Decode(&report)
	if err != nil {
		http.Error(w, "invalid rqeuest", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFillColor(2, 92, 194)
	pdf.Rect(0, 0, 210, 35, "F")
	pdf.Rect(0, 285, 210, 15, "F")
	pdf.Rect(16, 54, 2, 50, "F")
	pdf.Rect(115, 125, 2, 50, "F")

	// Path to image folder
	imagePath := "assets/"

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 24)
	pdf.SetXY(10, 12)
	pdf.Cell(0, 10, "Student Progress Report")

	pdf.ImageOptions(imagePath+"favicon.png", 140, 12, 10, 10, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

	pdf.SetXY(150, 10)
	pdf.SetFontSize(20)
	pdf.Cell(0, 10, "Corewell")
	pdf.Ln(6)
	pdf.SetX(150)
	pdf.Cell(0, 10, "Health")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetXY(15, 45)

	pdf.SetFillColor(247, 245, 243)
	pdf.Rect(10, 115, 90, 75, "F")
	pdf.Rect(110, 40, 90, 75, "F")
	pdf.Rect(10, 200, 190, 75, "F")

	// Student Details Section
	pdf.SetFont("Helvetica", "B", 14)
	pdf.Cell(0, 10, "Profile Information:")
	pdf.Ln(10)

	rotation := ""
	if report.User.Rotation != nil {
		rotation = *report.User.Rotation
	}

	pdf.SetFont("Helvetica", "", 12)

	pdf.SetX(20)
	pdf.Cell(0, 8, fmt.Sprintf("Rotation: %s", rotation))
	pdf.Ln(8)

	pdf.SetX(20)
	pdf.Cell(0, 8, fmt.Sprintf("Student Standing: %s", *report.User.StudentStanding))
	pdf.Ln(8)

	pdf.SetX(20)
	pdf.Cell(0, 8, fmt.Sprintf("Achievements Complete: %d", report.Analytics.AchievementCount))
	pdf.Ln(8)

	pdf.SetX(20)
	pdf.Cell(0, 8, fmt.Sprintf("Longest Task Streak: %d", report.Analytics.MaxTaskStreak))
	pdf.Ln(8)

	pdf.SetX(20)
	pdf.Cell(0, 8, fmt.Sprintf("Longest Quiz Streak: %d", report.Analytics.MaxQuizStreak))
	pdf.Ln(8)

	// Icon Section

	iconPath := imagePath + *report.User.Icon + ".png"
	bannerPath := imagePath + report.User.Border + ".png"
	pdf.SetLineWidth(1)
	pdf.Circle(155, 70, 15, "D")
	pdf.SetFillColor(255, 255, 255)
	pdf.Circle(155, 70, 14, "F")
	pdf.ImageOptions(iconPath, 145, 60, 20, 20, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	if report.User.Border != "" {
		pdf.ImageOptions(bannerPath, 135, 50, 40, 40, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	}

	pdf.SetFont("Helvetica", "", 12)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetXY(115, 95)

	pdf.SetFont("Helvetica", "B", 12)
	pdf.Cell(24, 6, "Nickname: ")
	pdf.SetFont("Helvetica", "", 12)
	pdf.Cell(0, 6, report.User.Nickname)
	pdf.Ln(8)

	pdf.SetX(115)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.Cell(16, 6, "UserID: ")
	pdf.SetFont("Helvetica", "", 12)
	pdf.Cell(0, 6, report.User.UserUniqueId)

	//Improvement Areas Section
	pdf.SetFont("Helvetica", "B", 14)
	pdf.Text(15, 123, "Improvement Areas:") // Using .Text cause this is being stubborn for some reason !

	if len(report.User.ImprovementAreas) > 0 {
		pdf.SetFontStyle("")
		pdf.SetFontSize(7)
		pdf.Ln(6)

		pdf.SetY(125)
		pdf.SetX(22)

		startY := pdf.GetY()

		for i, topic := range report.User.ImprovementAreas {
			if i >= 9 {
				if i == 9 {
					pdf.SetY(startY)
				}
				pdf.SetX(65)
			} else {
				pdf.SetX(22)
			}
			pdf.Cell(10, 8, "- "+topic)
			pdf.Ln(6)
		}
	}

	pdf.Ln(10)

	// Analytics Section !
	pdf.SetFont("Helvetica", "B", 14)
	pdf.Text(115, 123, "Learning Analytics:")
	pdf.Ln(10)
	pdf.SetY(127)

	pdf.SetFont("Helvetica", "", 12)
	pdf.SetX(119)
	totalTasks := report.Analytics.QuestionTasks + report.Analytics.LabTasks + report.Analytics.PrescriptionTasks
	pdf.Cell(0, 8, fmt.Sprintf("Tasks Completed: %d", totalTasks))
	pdf.Ln(8)

	pdf.SetX(119)
	completionPercentage := (float64(min(report.Analytics.QuestionTasks, 28)+min(report.Analytics.PrescriptionTasks, 28)+min(report.Analytics.LabTasks, 28)) / 84) * 100
	pdf.Cell(0, 8, fmt.Sprintf("Course Completion Percentage: %.2f%%", completionPercentage))
	pdf.Ln(8)

	pdf.SetX(119)
	pdf.Cell(0, 8, fmt.Sprintf("Average Quiz Score: %.2f%%", report.Analytics.AverageQuizScore))
	pdf.Ln(8)

	pdf.SetX(119)
	pdf.Cell(0, 8, fmt.Sprintf("Quiz Completion Rating: %.2f%%", report.Analytics.QuizCompletionRating))
	pdf.Ln(8)

	pdf.SetX(119)
	pdf.Cell(0, 8, fmt.Sprintf("Total Time Spent Learning: %s", report.Analytics.TotalTimeSpent))

	// Pie Chart Section !!
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetXY(70, 205)
	pdf.Cell(0, 8, "Task Completion By Type:")

	chartY := 240.0
	chartRadius := 18.0

	drawPieChart(pdf, 45, chartY, chartRadius, report.Analytics.QuestionTasks, "Patient Questions:")
	drawPieChart(pdf, 105, chartY, chartRadius, report.Analytics.LabTasks, "Lab Results:")
	drawPieChart(pdf, 165, chartY, chartRadius, report.Analytics.PrescriptionTasks, "Prescriptions:")

	if pdf.Error() != nil {
		log.Printf("PDF Error detected: %v", pdf.Error())
		http.Error(w, fmt.Sprintf("PDF generation error: %v", pdf.Error()), http.StatusInternalServerError)
		return
	}

	err = pdf.Output(w)
	if err != nil {
		log.Printf("Output Error: %v", err)
		http.Error(w, "Failed to generate PDF", http.StatusInternalServerError)
		return
	}
}
