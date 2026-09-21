package api

import "net/http"

// Operator is who runs this installation: the Impressum and the privacy notice are
// built from it. It comes from the environment, not from the source -- whoever runs
// the published images cannot touch the code, and must not have to for their own
// Impressum.
type Operator struct {
	Name    string `json:"name"`
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
	Email   string `json:"email"`
	Phone   string `json:"phone,omitempty"`
	VATID   string `json:"vat_id,omitempty"`
	Hosting string `json:"hosting"`
	// CheckedAt is the date of the operator's own accessibility check, ISO format.
	CheckedAt string `json:"accessibility_checked_at,omitempty"`
}

// Complete says whether everything § 5 DDG requires is there. Phone and VAT id are
// optional; country defaults. The frontend shows a warning as long as this is false --
// a made-up Impressum would be worse than a missing one, so nothing is invented.
func (o Operator) Complete() bool {
	return o.Name != "" && o.Street != "" && o.City != "" && o.Email != "" && o.Hosting != ""
}

type operatorDTO struct {
	Operator
	Complete bool `json:"complete"`
}

func (s *Server) handleOperator(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, operatorDTO{Operator: s.operator, Complete: s.operator.Complete()})
}
