package properties

import (
	"bytes"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/phpdave11/gofpdf"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
)

// GetProperties fetches the properties (lead files) associated with the user that are not dropped.
func GetProperties(c *gin.Context) {
	// Get the user from the context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	// Fetch the lead files from the CRM database where customer_id matches the user's customer_number and lead_file_status_dropped is "No"
	var leadFiles []models.LeadFile
	if err := utils.CRMDB.
		Where("customer_id = ? AND lead_file_status_dropped = ?", user.CustomerNumber, "No").
		Find(&leadFiles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch properties"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"properties": leadFiles,
	})
}

func GetInstallmentSchedule(c *gin.Context) {
	// Get the user from the context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	leadFileNo := c.Param("lead_file_no")
	if leadFileNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lead file number is required"})
		return
	}

	// Verify that the lead file belongs to the user and is not dropped
	var leadFile models.LeadFile
	if err := utils.CRMDB.
		Where("lead_file_no = ? AND customer_id = ? AND lead_file_status_dropped = ?", leadFileNo, user.CustomerNumber, "No").
		First(&leadFile).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Property not found, does not belong to the user, or is dropped"})
		return
	}

	// Fetch the installment schedules from the CRM database where member_no and leadfile_no match
	var schedules []models.InstallmentSchedule
	if err := utils.CRMDB.Where("member_no = ? AND leadfile_no = ?", user.CustomerNumber, leadFileNo).Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch installment schedules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"installment_schedules": schedules,
	})
}

func GetInstallmentSchedulePDF(c *gin.Context) {
	// Get the user from the context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	leadFileNo := c.Param("lead_file_no")
	if leadFileNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lead file number is required"})
		return
	}

	// Verify that the lead file belongs to the user and is not dropped
	var leadFile models.LeadFile
	if err := utils.CRMDB.
		Where("lead_file_no = ? AND customer_id = ? AND lead_file_status_dropped = ?", leadFileNo, user.CustomerNumber, "No").
		First(&leadFile).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Property not found, does not belong to the user, or is dropped"})
		return
	}

	// Fetch the installment schedules
	var schedules []models.InstallmentSchedule
	if err := utils.CRMDB.Where("member_no = ? AND leadfile_no = ?", user.CustomerNumber, leadFileNo).Order("due_date ASC").Find(&schedules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch installment schedules"})
		return
	}

	log.Printf("Generating PDF for user: %s, property: %s", user.CustomerNumber, leadFile.PlotNumber)
	log.Printf("Number of schedules: %d", len(schedules))

	// Check if schedules are empty
	if len(schedules) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "No installment schedules found"})
		return
	}

	// Generate the PDF using gofpdf
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 20, 15)
	pdf.AddPage()

	// Add title
	pdf.SetFont("Helvetica", "B", 16)
	if pdf.Err() {
		log.Printf("Failed to set font: %v", pdf.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set font"})
		return
	}
	pdf.CellFormat(0, 10, "Payment Schedule", "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// Add property and customer information
	pdf.SetFont("Helvetica", "", 12)
	pdf.Cell(0, 10, "Property: "+leadFile.PlotNumber)
	pdf.Ln(6)
	pdf.Cell(0, 10, "Customer: "+user.CustomerNumber)
	pdf.Ln(10)

	// Table Headers
	pdf.SetFont("Helvetica", "B", 12)
	pdf.SetFillColor(200, 200, 200)
	pdf.CellFormat(10, 10, "No.", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 10, "Due Date", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 10, "Installment Amount", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 10, "Remaining Amount", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 10, "Amount Paid", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 10, "Penalties Accrued", "1", 0, "C", true, 0, "")
	pdf.CellFormat(15, 10, "Paid", "1", 1, "C", true, 0, "")
	pdf.SetFont("Helvetica", "", 12)

	// Initialize the caser for title casing
	caser := cases.Title(language.English)

	// Helper function to format amounts
	formatAmount := func(amountStr string) string {
		// Remove commas
		amountStr = strings.ReplaceAll(amountStr, ",", "")
		// Parse to float64
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return amountStr // Return original string if parsing fails
		}
		// Format with commas and two decimal places
		return humanize.CommafWithDigits(amount, 2)
	}

	// Add the data
	for _, schedule := range schedules {
		pdf.CellFormat(10, 10, strconv.Itoa(schedule.InstallmentNo), "1", 0, "C", false, 0, "")
		dueDate := ""
		if schedule.DueDate != nil {
			dueDate = schedule.DueDate.Format("2006-01-02")
		}
		pdf.CellFormat(30, 10, dueDate, "1", 0, "C", false, 0, "")
		pdf.CellFormat(35, 10, formatAmount(schedule.InstallmentAmount), "1", 0, "R", false, 0, "")
		pdf.CellFormat(35, 10, formatAmount(schedule.RemainingAmount), "1", 0, "R", false, 0, "")
		pdf.CellFormat(35, 10, formatAmount(schedule.AmountPaid), "1", 0, "R", false, 0, "")
		pdf.CellFormat(35, 10, humanize.CommafWithDigits(float64(schedule.PenaltiesAccrued), 2), "1", 0, "R", false, 0, "")
		pdf.CellFormat(15, 10, caser.String(schedule.Paid), "1", 1, "C", false, 0, "")
	}

	// Check for errors before outputting PDF
	if pdf.Err() {
		log.Printf("Error generating PDF: %v", pdf.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
		return
	}

	// Output PDF to buffer
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		log.Printf("Failed to generate PDF: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
		return
	}

	// Set headers and send the PDF
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=payment_schedule.pdf")
	c.Data(http.StatusOK, "application/pdf", buf.Bytes())
}

func GetTransactions(c *gin.Context) {
	// Get the user from the context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	leadFileNo := c.Param("lead_file_no")
	if leadFileNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lead file number is required"})
		return
	}

	// Verify that the lead file belongs to the user and is not dropped
	var leadFile models.LeadFile
	if err := utils.CRMDB.
		Where("lead_file_no = ? AND customer_id = ? AND lead_file_status_dropped = ?", leadFileNo, user.CustomerNumber, "No").
		First(&leadFile).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Property not found, does not belong to the user, or is dropped"})
		return
	}

	// Fetch receipts for the property where Type is 'Posted'
	var receipts []models.Receipt
	if err := utils.DefaultDB.Where("Lead_file_no = ? AND Customer_Id = ? AND Type = ?", leadFileNo, user.CustomerNumber, "Posted").Order("Payment_Date1 DESC").Find(&receipts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
		return
	}

	// Map the receipts data to transactions
	var transactions []map[string]interface{}
	for _, receipt := range receipts {
		dateStr := ""
		timeStr := ""

		if receipt.PaymentDate1 != "" {
			// Parse the string to time.Time
			parsedTime, err := time.Parse("2006-01-02T15:04:05Z07:00", receipt.PaymentDate1)
			if err != nil {
				log.Printf("Error parsing PaymentDate1 for receipt ID %d: %v", receipt.ID, err)
				// If parsing fails, try alternative formats
				parsedTime, err = time.Parse("2006-01-02 15:04:05", receipt.PaymentDate1)
				if err != nil {
					// If still error, use the raw string
					dateStr = receipt.PaymentDate1
					timeStr = ""
				} else {
					dateStr = parsedTime.Format("2006-01-02")
					timeStr = parsedTime.Format("15:04")
				}
			} else {
				dateStr = parsedTime.Format("2006-01-02")
				timeStr = parsedTime.Format("15:04")
			}
		}

		transaction := map[string]interface{}{
			"id":     strconv.Itoa(receipt.ID),
			"date":   dateStr,
			"type":   receipt.TransactionType, // e.g., "Installment"
			"amount": receipt.AmountLCY,
			"time":   timeStr,
			// Add other fields if necessary
		}
		transactions = append(transactions, transaction)
	}

	c.JSON(http.StatusOK, gin.H{
		"transactions": transactions,
	})
}

func GetUserProjects(c *gin.Context) {
	// Get the user from the context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	// Fetch lead files associated with the user that are not dropped to get project numbers
	var leadFiles []models.LeadFile
	if err := utils.CRMDB.
		Where("customer_id = ? AND lead_file_status_dropped = ?", user.CustomerNumber, "No").
		Find(&leadFiles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch properties"})
		return
	}

	// Extract unique project numbers
	projectNumbersMap := make(map[string]bool)
	for _, leadFile := range leadFiles {
		projectNumbersMap[leadFile.ProjectNumber] = true
	}

	var projectNumbers []string
	for projectNumber := range projectNumbersMap {
		projectNumbers = append(projectNumbers, projectNumber)
	}

	// Fetch project details from the projects table
	var projects []models.Project
	if err := utils.DefaultDB.Where("EPR_id IN ?", projectNumbers).Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"projects": projects,
	})
}

func GetUserPropertiesByProject(c *gin.Context) {
	// Get the user from the context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	projectIDStr := c.Param("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	// Convert projectID to integer
	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Project ID"})
		return
	}

	// Fetch the project to get its EPR_id
	var project models.Project
	if err := utils.DefaultDB.Where("project_id = ?", projectID).First(&project).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch project"})
		return
	}

	eprID := project.EPRID

	if eprID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Project EPR ID not found"})
		return
	}

	// Fetch properties (lead files) for the user under the given project EPR ID that are not dropped
	var leadFiles []models.LeadFile
	if err := utils.CRMDB.
		Where("customer_id = ? AND project_number = ? AND lead_file_status_dropped = ?", user.CustomerNumber, eprID, "No").
		Find(&leadFiles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch properties"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"properties": leadFiles,
	})
}

func GetReceiptsByProperty(c *gin.Context) {
	// Get the user from the context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	leadFileNo := c.Param("lead_file_no")
	if leadFileNo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Lead file number is required"})
		return
	}

	// Verify that the lead file belongs to the user and is not dropped
	var leadFile models.LeadFile
	if err := utils.CRMDB.
		Where("lead_file_no = ? AND customer_id = ? AND lead_file_status_dropped = ?", leadFileNo, user.CustomerNumber, "No").
		First(&leadFile).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Property not found, does not belong to the user, or is dropped"})
		return
	}

	// Fetch receipts for the given lead file number where Type is 'Posted'
	var receipts []models.Receipt
	if err := utils.DefaultDB.Where("Lead_file_no = ? AND Customer_Id = ? AND Type = ?", leadFileNo, user.CustomerNumber, "Posted").Find(&receipts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch receipts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"receipts": receipts,
	})
}

func GetUserTotalSpent(c *gin.Context) {
	// Get the user from the context
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	user := userInterface.(models.User)

	// Fetch active lead files for the user
	var leadFiles []models.LeadFile
	if err := utils.CRMDB.
		Where("customer_id = ? AND lead_file_status_dropped = ?", user.CustomerNumber, "No").
		Find(&leadFiles).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch properties"})
		return
	}

	// Extract lead file numbers of active properties
	var activeLeadFileNos []string
	for _, leadFile := range leadFiles {
		activeLeadFileNos = append(activeLeadFileNos, leadFile.LeadFileNo)
	}

	// Fetch receipts where customer_id = user.CustomerNumber, Type = 'Posted', and Lead_file_no in active properties
	var receipts []models.Receipt
	if err := utils.DefaultDB.
		Where("Customer_Id = ? AND Type = ? AND Lead_file_no IN ?", user.CustomerNumber, "Posted", activeLeadFileNos).
		Find(&receipts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch receipts"})
		return
	}

	totalSpent := 0.0

	for _, receipt := range receipts {
		amount := receipt.AmountLCY
		totalSpent += amount
	}

	c.JSON(http.StatusOK, gin.H{
		"total_spent": totalSpent,
	})
}

func GetFeaturedProjects(c *gin.Context) {
	var projects []models.Project
	if err := utils.DefaultDB.Where("is_featured = ?", true).Find(&projects).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch featured projects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"projects": projects,
	})
}
