package properties

import (
	"mobile-customer-portal-server/models"
	"mobile-customer-portal-server/utils"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetProperties(c *gin.Context) {
	// Get the user from the context
	userInterface, exists := c.Get("user")
	if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
			return
	}
	user := userInterface.(models.User)

	// Fetch the lead files from the CRM database where customer_id matches the user's customer_number
	var leadFiles []models.LeadFile
	if err := utils.CRMDB.Where("customer_id = ?", user.CustomerNumber).Find(&leadFiles).Error; err != nil {
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

	// Verify that the lead file belongs to the user
	var leadFile models.LeadFile
	if err := utils.CRMDB.Where("lead_file_no = ? AND customer_id = ?", leadFileNo, user.CustomerNumber).First(&leadFile).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Property not found or does not belong to the user"})
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

	// Verify that the lead file belongs to the user
	var leadFile models.LeadFile
	if err := utils.CRMDB.Where("lead_file_no = ? AND customer_id = ?", leadFileNo, user.CustomerNumber).First(&leadFile).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Property not found or does not belong to the user"})
			return
	}

	// Fetch the installment schedules for the property
	var schedules []models.InstallmentSchedule
	if err := utils.CRMDB.Where("member_no = ? AND leadfile_no = ?", user.CustomerNumber, leadFileNo).Find(&schedules).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
			return
	}

	// Map the InstallmentSchedule data to transactions
	var transactions []map[string]interface{}
	for _, schedule := range schedules {
			// Parse the amount_paid string to float64
			amountPaidStr := strings.ReplaceAll(schedule.AmountPaid, ",", "")
			amountPaid, err := strconv.ParseFloat(amountPaidStr, 64)
			if err != nil {
					amountPaid = 0
			}

			// Format the date and time
			dateStr := ""
			timeStr := ""
			if schedule.DueDate != nil {
					dateStr = schedule.DueDate.Format("2006-01-02")
					timeStr = schedule.DueDate.Format("15:04")
			}

			transaction := map[string]interface{}{
					"id":     strconv.Itoa(schedule.ISID),
					"date":   dateStr,
					"type":   "Installment",
					"amount": amountPaid,
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

	// Fetch lead files associated with the user to get project numbers
	var leadFiles []models.LeadFile
	if err := utils.CRMDB.Where("customer_id = ?", user.CustomerNumber).Find(&leadFiles).Error; err != nil {
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

	// Fetch properties (lead files) for the user under the given project EPR ID
	var leadFiles []models.LeadFile
	if err := utils.CRMDB.Where("customer_id = ? AND project_number = ?", user.CustomerNumber, eprID).Find(&leadFiles).Error; err != nil {
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

	// Verify that the lead file belongs to the user
	var leadFile models.LeadFile
	if err := utils.CRMDB.Where("lead_file_no = ? AND customer_id = ?", leadFileNo, user.CustomerNumber).First(&leadFile).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Property not found or does not belong to the user"})
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
