package customers

import "time"

// Response is the API view of a customer.
type Response struct {
	ID            string    `json:"id"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Phone         string    `json:"phone"`
	Email         string    `json:"email"`
	KRAPin        string    `json:"kra_pin"`
	CreditLimit   *string   `json:"credit_limit"`
	CreditBalance string    `json:"credit_balance"`
	Notes         *string   `json:"notes,omitempty"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

// ToResponse maps a customer for the API.
func ToResponse(c Customer) Response {
	r := Response{
		ID: c.PublicID, Code: c.Code, Name: c.Name, Phone: c.Phone, Email: c.Email,
		KRAPin: c.KRAPin, Notes: c.Notes, Status: c.Status, CreatedAt: c.CreatedAt,
	}
	if c.CreditLimit != nil {
		s := c.CreditLimit.String()
		r.CreditLimit = &s
	}
	if c.CreditBalance != nil {
		r.CreditBalance = c.CreditBalance.String()
	} else {
		r.CreditBalance = "0"
	}
	return r
}
