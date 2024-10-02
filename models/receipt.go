package models

type Receipt struct {
    ID              int     `gorm:"column:id;primaryKey" json:"id"`
    ReceiptNo       string  `gorm:"column:Receipt_No" json:"receipt_no"`
    DatePosted      string  `gorm:"column:Date_Posted" json:"date_posted"`
    PaymentDate     string  `gorm:"column:Payment_date" json:"payment_date"`
    BankName        string  `gorm:"column:Bank_Name" json:"bank_name"`
    BankAccount     string  `gorm:"column:Bank_Account" json:"bank_account"`
    CustomerID      string  `gorm:"column:Customer_Id" json:"customer_id"`
    CustomerName    string  `gorm:"column:Customer_Name" json:"customer_name"`
    PayMode         string  `gorm:"column:Pay_mode" json:"pay_mode"`
    LeadFileNo      string  `gorm:"column:Lead_file_no" json:"lead_file_no"`
    ProjectName     string  `gorm:"column:Project_Name" json:"project_name"`
    PlotNo          string  `gorm:"column:Plot_NO" json:"plot_no"`
    Marketer        string  `gorm:"column:Marketer" json:"marketer"`
    Teams           string  `gorm:"column:Teams" json:"teams"`
    Regions         string  `gorm:"column:Regions" json:"regions"`
    DepositThreshold string `gorm:"column:Deposit_Threshold" json:"deposit_threshold"`
    TransactionType string  `gorm:"column:Transaction_type" json:"transaction_type"`
    TransferReceipt string  `gorm:"column:transfer_receipt" json:"transfer_receipt"`
    AmountLCY       float64 `gorm:"column:Amount_LCY" json:"amount_lcy"`
    BalanceLCY      float64 `gorm:"column:Balance_LCY" json:"balance_lcy"`
    Type            string  `gorm:"column:Type" json:"type"`
    Status          string  `gorm:"column:status" json:"status"`
    PostedDate1     string  `gorm:"column:POSTED_DATE1" json:"posted_date1"`
    PaymentDate1    string  `gorm:"column:PAYMENT_DATE1" json:"payment_date1"`
}

// TableName to override the default table name
func (Receipt) TableName() string {
    return "Recipts" // Note the misspelling to match your actual table name
}
