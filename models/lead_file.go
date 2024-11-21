package models

import "time"

type LeadFile struct {
    LeadFileNo            string     `gorm:"column:lead_file_no;primaryKey" json:"lead_file_no"`
    LeadFileStatusDropped string     `gorm:"column:lead_file_status_dropped" json:"lead_file_status_dropped"`
    PlotNumber            string     `gorm:"column:plot_number" json:"plot_number"`
    ProjectNumber         string     `gorm:"column:project_number" json:"project_number"`
    Marketer              string     `gorm:"column:marketer" json:"marketer"`
    PurchasePrice         float64    `gorm:"column:purchase_price" json:"purchase_price"`
    SellingPrice          string     `gorm:"column:selling_price" json:"selling_price"`
    BalanceLCY            float64    `gorm:"column:\"balance(LCY)\"" json:"balance_lcy"`
    CustomerID            string     `gorm:"column:customer_id" json:"customer_id"`
    CustomerName          string     `gorm:"column:customer_name" json:"customer_name"`
    PurchaseType          string     `gorm:"column:purchase_type" json:"purchase_type"`
    CommissionThreshold   string     `gorm:"column:commission_threshold" json:"commission_threshold"`
    DepositThreshold      float64    `gorm:"column:deposit_threshold" json:"deposit_threshold"`
    Discount              string     `gorm:"column:discount" json:"discount"`
    CompletionDate        string     `gorm:"column:completion_date" json:"completion_date"`
    NoOfInstallments      string     `gorm:"column:no_of_installments" json:"no_of_installments"`
    InstallmentAmount     string     `gorm:"column:installment_amount" json:"installment_amount"`
    SaleAgreementSent     string     `gorm:"column:sale_agreement_sent" json:"sale_agreement_sent"`
    SaleAgreementSigned   string     `gorm:"column:sale_agreement_signed" json:"sale_agreement_signed"`
    TotalPaid             float64    `gorm:"column:total_paid" json:"total_paid"`
    TransferCostCharged   string     `gorm:"column:transfer_cost_charged" json:"transfer_cost_charged"`
    TransferCostPaid      string     `gorm:"column:transfer_cost_paid" json:"transfer_cost_paid"`
    Overpayments          string     `gorm:"column:overpayments" json:"overpayments"`
    Refunds               string     `gorm:"column:refunds" json:"refunds"`
    RefundableAmount      string     `gorm:"column:refundable_amount" json:"refundable_amount"`
    PenaltiesAccrued      string     `gorm:"column:penalties_accrued" json:"penalties_accrued"`
    CustomerLeadSource    string     `gorm:"column:customer_lead_source" json:"customer_lead_source"`
    CatLeadSource         string     `gorm:"column:cat_lead_source" json:"cat_lead_source"`
    BookingID             string     `gorm:"column:booking_id" json:"booking_id"`
    BookingDate           *time.Time `gorm:"column:Booking_date" json:"booking_date"`
    AdditionalDepositDate *time.Time `gorm:"column:Additional_deposit_date" json:"additional_deposit_date"`
    TitleStatus           string     `gorm:"column:title_status" json:"title_status"`
}

// TableName to override the default table name
func (LeadFile) TableName() string {
    return "lead_files"
}
