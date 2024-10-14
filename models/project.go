package models

type Project struct {
    ProjectID         int    `gorm:"column:project_id;primaryKey" json:"project_id"`
    Name              string `gorm:"column:name" json:"name"`
    Link              string `gorm:"column:link" json:"link"`
    Priority          string `gorm:"column:priolity" json:"priority"`
    Visibility        string `gorm:"column:visibility" json:"visibility"`
    Bank              string `gorm:"column:bank" json:"bank"`
    AccountNumber     string `gorm:"column:acc_no" json:"account_number"`
    Initials          string `gorm:"column:initials" json:"initials"`
    EPRID             string `gorm:"column:EPR_id" json:"epr_id"`
    OfferLetterExist  string `gorm:"column:offer_letter_existence" json:"offer_letter_existence"`
    PaymentModel      string `gorm:"column:Payment_model" json:"payment_model"`
    Description string `gorm:"column:description" json:"description"`
    Banner      string `gorm:"column:banner" json:"banner"`
    IsFeatured  bool   `gorm:"column:is_featured" json:"is_featured"`
}

// TableName to override the default table name
func (Project) TableName() string {
    return "Projects"
}
