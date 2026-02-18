// DesignDocument/8_DBDesign.dbml credit_packages
package entity

type CreditPackage struct {
	ID            string
	Name          string
	Credits       int
	StripePriceID string
	IsActive      bool
}
