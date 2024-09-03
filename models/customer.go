package models

// Customer struct to map the fields in the customer table in the CRM database
type Customer struct {
    CustomerNo         string `gorm:"column:customer_no;primaryKey"`
    CustomerName       string `gorm:"column:customer_name"`
    NationalID         string `gorm:"column:national_id"`
    PassportNo         string `gorm:"column:passport_no"`
    KRAPin             string `gorm:"column:kra_pin"`
    DOB                string `gorm:"column:dob"`
    Gender             string `gorm:"column:gender"`
    MaritalStatus      string `gorm:"column:marital_status"`
    Phone              string `gorm:"column:phone"`
    PrimaryEmail       string `gorm:"column:primary_email"`
    AlternativeEmail   string `gorm:"column:alternative_email"`
    Address            string `gorm:"column:address"`
    CustomerType       string `gorm:"column:customer_type"`
    LeadSource         string `gorm:"column:lead_source"`
    SubCatID           string `gorm:"column:sub_cat_id"`
    Marketer           string `gorm:"column:marketer"`
    CountryOfResidence string `gorm:"column:country_of_residence"`
    DateOfRegistration string `gorm:"column:date_of_registration"`
    AlternativePhone   string `gorm:"column:alternative_phone"`
    OTP                string `gorm:"-"` // Not stored in DB
    OTPGeneratedAt     string `gorm:"-"` // Not stored in DB
}

// TableName overrides the default table name to "customer"
func (Customer) TableName() string {
    return "customer"
}
