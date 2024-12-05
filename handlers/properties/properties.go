package properties

import (
	"bytes"
	"fmt"
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

// Helper function to add a header to the PDF
func addHeader(pdf *gofpdf.Fpdf) {
	pdf.SetHeaderFunc(func() {
			// Add company name
			pdf.SetFont("Helvetica", "B", 20)
			pdf.CellFormat(0, 15, "Optiven Limited", "", 0, "C", false, 0, "")
			pdf.Ln(20)
	})
}


// Helper function to add a footer to the PDF
func addFooter(pdf *gofpdf.Fpdf) {
    pdf.SetFooterFunc(func() {
        pdf.SetY(-15)
        pdf.SetFont("Helvetica", "I", 8)
        pdf.CellFormat(0, 10, fmt.Sprintf("Page %d of {nb}", pdf.PageNo()), "", 0, "C", false, 0, "")
    })
    pdf.AliasNbPages("")
}

// Helper function to format amounts
func formatAmount(amount float64) string {
    return humanize.CommafWithDigits(amount, 2)
}

// Helper function to parse string to float64
func parseFloat(amountStr string) float64 {
	amountStr = strings.TrimSpace(amountStr)
	if amountStr == "" {
			return 0.0
	}
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
			log.Printf("Error parsing amount '%s': %v", amountStr, err)
			return 0.0
	}
	return amount
}

// Helper function to sum widths for drawing lines
func sum(widths []float64) float64 {
    total := 0.0
    for _, w := range widths {
        total += w
    }
    return total
}

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

    // Fetch the installment schedules from the CRM database where member_no and leadfile_no match, ordered by due_date
    var schedules []models.InstallmentSchedule
    if err := utils.CRMDB.
        Where("member_no = ? AND leadfile_no = ?", user.CustomerNumber, leadFileNo).
        Order("due_date ASC").
        Find(&schedules).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch installment schedules"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "installment_schedules": schedules,
    })
}


func GetInstallmentSchedulePDF(c *gin.Context) {
    // Retrieve user from context
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

    // Verify ownership and existence of the lead file
    var leadFile models.LeadFile
    if err := utils.CRMDB.
        Where("lead_file_no = ? AND customer_id = ? AND lead_file_status_dropped = ?", leadFileNo, user.CustomerNumber, "No").
        First(&leadFile).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Property not found, does not belong to the user, or is dropped"})
        return
    }

    // Fetch installment schedules
    var schedules []models.InstallmentSchedule
    if err := utils.CRMDB.
        Where("member_no = ? AND leadfile_no = ?", user.CustomerNumber, leadFileNo).
        Order("due_date ASC").
        Find(&schedules).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch installment schedules"})
        return
    }

    if len(schedules) == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "No installment schedules found"})
        return
    }

    // Generate the PDF
    pdf := gofpdf.New("P", "mm", "A4", "")
    pdf.SetMargins(15, 20, 15)
    pdf.SetAutoPageBreak(true, 20)
    pdf.AddPage()

    // Add header and footer
    addHeader(pdf)
    addFooter(pdf)

    pdf.SetFont("Helvetica", "B", 16)
    pdf.CellFormat(0, 10, "Payment Schedule", "", 1, "C", false, 0, "")
    pdf.Ln(5)

    // Customer and property information
    pdf.SetFont("Helvetica", "", 12)
    pdf.CellFormat(0, 8, "Customer Number: "+user.CustomerNumber, "", 1, "", false, 0, "")
    pdf.CellFormat(0, 8, "Property: "+leadFile.PlotNumber, "", 1, "", false, 0, "")
    pdf.CellFormat(0, 8, "Date: "+time.Now().Format("02 January 2006"), "", 1, "", false, 0, "")
    pdf.Ln(5)

    // Table Headers
    headers := []string{"No.", "Due Date", "Installment Amount", "Remaining Amount", "Amount Paid", "Penalties Accrued", "Paid"}
    widths := []float64{10, 25, 35, 35, 35, 35, 15}

    pdf.SetFont("Helvetica", "B", 11)
    pdf.SetFillColor(230, 230, 230)
    for i, header := range headers {
        pdf.CellFormat(widths[i], 10, header, "1", 0, "C", true, 0, "")
    }
    pdf.Ln(-1)

    // Table Body
    pdf.SetFont("Helvetica", "", 10)
    fill := false
    for _, schedule := range schedules {
        if fill {
            pdf.SetFillColor(255, 255, 255)
        } else {
            pdf.SetFillColor(245, 245, 245)
        }
        fill = !fill

        // Installment Number
        pdf.CellFormat(widths[0], 8, strconv.Itoa(schedule.InstallmentNo), "1", 0, "C", true, 0, "")

        // Due Date
        dueDate := ""
        if schedule.DueDate != nil {
            dueDate = schedule.DueDate.Format("02 Jan 2006")
        }
        pdf.CellFormat(widths[1], 8, dueDate, "1", 0, "C", true, 0, "")

        // Installment Amount
        installmentAmount := parseFloat(schedule.InstallmentAmount)
        pdf.CellFormat(widths[2], 8, formatAmount(installmentAmount), "1", 0, "R", true, 0, "")

        // Remaining Amount
        remainingAmount := parseFloat(schedule.RemainingAmount)
        pdf.CellFormat(widths[3], 8, formatAmount(remainingAmount), "1", 0, "R", true, 0, "")

        // Amount Paid
        amountPaid := parseFloat(schedule.AmountPaid)
        pdf.CellFormat(widths[4], 8, formatAmount(amountPaid), "1", 0, "R", true, 0, "")

        // Penalties Accrued
        penaltiesAccrued := float64(schedule.PenaltiesAccrued)
        pdf.CellFormat(widths[5], 8, formatAmount(penaltiesAccrued), "1", 0, "R", true, 0, "")

        // Paid Status
        paidStatus := cases.Title(language.English).String(schedule.Paid)
        pdf.CellFormat(widths[6], 8, paidStatus, "1", 0, "C", true, 0, "")

        pdf.Ln(-1)
    }

    // Draw bottom line
    pdf.SetLineWidth(0.5)
    pdf.Line(15, pdf.GetY(), 15+sum(widths), pdf.GetY())

    // Output PDF to buffer
    var buf bytes.Buffer
    err := pdf.Output(&buf)
    if err != nil {
        log.Printf("Failed to generate PDF: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
        return
    }

    // Send the PDF file
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
    if err := utils.DefaultDB.Where("Lead_file_no = ? AND Customer_Id = ? AND Type = ? AND Transaction_Type = ?", leadFileNo, user.CustomerNumber, "Posted", "Installment").Order("Payment_Date1 DESC").Find(&receipts).Error; err != nil {
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
            parsedTime, err := time.Parse(time.RFC3339, receipt.PaymentDate1)
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
            "type":   receipt.TransactionType,
            "amount": receipt.AmountLCY,
            "time":   timeStr,
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

func GetAllVisibleProjects(c *gin.Context) {
    var featuredProjects []models.Project
    var otherProjects []models.Project

    // Fetch featured projects with visibility = 'SHOW', order by name
    if err := utils.DefaultDB.Where("is_featured = ? AND visibility = ?", true, "SHOW").Order("name ASC").Find(&featuredProjects).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch featured projects"})
        return
    }

    // Fetch other projects (not featured) with visibility = 'SHOW', order by name
    if err := utils.DefaultDB.Where("is_featured = ? AND visibility = ?", false, "SHOW").Order("name ASC").Find(&otherProjects).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch other projects"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "featured_projects": featuredProjects,
        "other_projects":    otherProjects,
    })
}

func GetTitleStatus(c *gin.Context) {
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

    // Respond with the title status
    c.JSON(http.StatusOK, gin.H{
        "title_status": leadFile.TitleStatus,
    })
}

func GetReceiptPDF(c *gin.Context) {
    // Retrieve user from context
    userInterface, exists := c.Get("user")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
        return
    }
    user := userInterface.(models.User)

    leadFileNo := c.Param("lead_file_no")
    receiptIDStr := c.Param("receipt_id")

    if leadFileNo == "" || receiptIDStr == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Lead file number and receipt ID are required"})
        return
    }

    // Convert receipt ID to integer
    receiptID, err := strconv.Atoi(receiptIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid receipt ID"})
        return
    }

    // Verify ownership and existence of the lead file
    var leadFile models.LeadFile
    if err := utils.CRMDB.
        Where("lead_file_no = ? AND customer_id = ? AND lead_file_status_dropped = ?", leadFileNo, user.CustomerNumber, "No").
        First(&leadFile).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Property not found, does not belong to the user, or is dropped"})
        return
    }

    // Fetch the receipt
    var receipt models.Receipt
    if err := utils.DefaultDB.
        Where("ID = ? AND Lead_file_no = ? AND Customer_Id = ? AND Type = ?", receiptID, leadFileNo, user.CustomerNumber, "Posted").
        First(&receipt).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Receipt not found, does not belong to the user or the property, or is not posted"})
        return
    }

    // Generate the PDF
    pdf := gofpdf.New("P", "mm", "A4", "")
    pdf.SetMargins(15, 20, 15)
    pdf.SetAutoPageBreak(true, 20)
    pdf.AddPage()

    // Add header and footer
    addHeader(pdf)
    addFooter(pdf)

    // Add receipt title
    pdf.SetFont("Helvetica", "B", 16)
    pdf.CellFormat(0, 10, "Receipt", "", 1, "C", false, 0, "")
    pdf.Ln(10)

    // Receipt details
    pdf.SetFont("Helvetica", "", 12)

    // Format date
    datePosted := receipt.DatePosted
    if datePosted == "" {
        datePosted = time.Now().Format("02 January 2006")
    } else {
        // Parse the date and format it
        parsedDate, err := time.Parse("2006-01-02", datePosted)
        if err == nil {
            datePosted = parsedDate.Format("02 January 2006")
        }
    }

    // Data to display
    data := [][]string{
        {"Receipt No:", receipt.ReceiptNo},
        {"Date:", datePosted},
        {"Customer:", user.CustomerNumber},
        {"Property:", leadFile.PlotNumber},
        {"Amount:", "KES " + formatAmount(receipt.AmountLCY)},
    }

    // Set column widths
    colWidths := []float64{50, 120} // Adjust as needed

    // Starting position
    startY := pdf.GetY()

    // Draw rectangle border
    pdf.Rect(15, startY, 180, float64(len(data)*10+10), "D")

    pdf.Ln(5)
    for _, row := range data {
        pdf.SetFont("Helvetica", "B", 12)
        pdf.CellFormat(colWidths[0], 10, row[0], "", 0, "L", false, 0, "")
        pdf.SetFont("Helvetica", "", 12)
        pdf.CellFormat(colWidths[1], 10, row[1], "", 1, "L", false, 0, "")
    }
    pdf.Ln(10)

    // Thank you message
    pdf.SetFont("Helvetica", "I", 12)
    pdf.MultiCell(0, 8, "Thank you for your payment. If you have any questions, please contact our customer service.", "", "C", false)

    // Footer with company contact info (optional)
    pdf.SetY(-30)
    pdf.SetFont("Helvetica", "", 10)
    pdf.CellFormat(0, 5, "Optiven Limited", "", 1, "C", false, 0, "")
    pdf.CellFormat(0, 5, "Phone: +254790300300 | Email: info@optiven.co.ke", "", 1, "C", false, 0, "")
    pdf.CellFormat(0, 5, "Head Office: Absa Towers, Loita Street , 2nd Floor,", "", 1, "C", false, 0, "")

    // Output PDF to buffer
    var buf bytes.Buffer
    err = pdf.Output(&buf)
    if err != nil {
        log.Printf("Failed to generate PDF: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
        return
    }

    // Send the PDF file
    c.Header("Content-Type", "application/pdf")
    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=receipt_%s.pdf", receipt.ReceiptNo))
    c.Data(http.StatusOK, "application/pdf", buf.Bytes())
}