package models

import "time"

type InstallmentSchedule struct {
    ISID              int        `gorm:"column:IS_id;primaryKey" json:"is_id"`
    MemberNo          string     `gorm:"column:member_no" json:"member_no"`
    LeadfileNo        string     `gorm:"column:leadfile_no" json:"leadfile_no"`
    LineNo            int        `gorm:"column:line_no" json:"line_no"`
    InstallmentNo     int        `gorm:"column:installment_no" json:"installment_no"`
    InstallmentAmount string     `gorm:"column:installment_amount" json:"installment_amount"`
    RemainingAmount   string     `gorm:"column:remaining_Amount" json:"remaining_amount"`
    DueDate           *time.Time `gorm:"column:due_date" json:"due_date"`
    Paid              string     `gorm:"column:paid" json:"paid"`
    PlotNo            string     `gorm:"column:plot_No" json:"plot_no"`
    PlotName          string     `gorm:"column:plot_Name" json:"plot_name"`
    AmountPaid        string     `gorm:"column:amount_Paid" json:"amount_paid"`
    PenaltiesAccrued  int        `gorm:"column:penalties_accrued" json:"penalties_accrued"`
}

// TableName to override the default table name
func (InstallmentSchedule) TableName() string {
    return "installment_schedule"
}
