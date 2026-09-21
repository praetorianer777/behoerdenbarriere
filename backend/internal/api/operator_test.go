package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func operatorRequest(t *testing.T, op Operator) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/v1/operator", nil)
	rec := httptest.NewRecorder()
	NewServer(&fakeDB{}, Options{Limits: DefaultLimits(), Operator: op}).Routes().ServeHTTP(rec, req)
	return rec
}

func TestOperatorIsServedFromConfiguration(t *testing.T) {
	rec := operatorRequest(t, Operator{
		Name: "Musterverein e. V.", Street: "Beispielweg 1", City: "12345 Musterstadt",
		Country: "Deutschland", Email: "post@example.org", Hosting: "Hetzner, Falkenstein",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := decode[map[string]any](t, rec)
	if got["name"] != "Musterverein e. V." || got["complete"] != true {
		t.Fatalf("operator = %v", got)
	}
	// Optional fields stay away instead of appearing as empty strings.
	if _, there := got["phone"]; there {
		t.Errorf("empty phone is serialised: %v", got)
	}
}

// A made-up Impressum would be worse than a missing one: the API says what is
// missing, and invents nothing.
func TestOperatorReportsIncompleteDetails(t *testing.T) {
	rec := operatorRequest(t, Operator{Name: "Nur ein Name", Country: "Deutschland"})
	got := decode[map[string]any](t, rec)
	if got["complete"] != false {
		t.Fatalf("an operator without address and e-mail counts as complete: %v", got)
	}
}

func TestOperatorCompleteNeedsTheLegalMinimum(t *testing.T) {
	full := Operator{Name: "n", Street: "s", City: "c", Email: "e", Hosting: "h"}
	if !full.Complete() {
		t.Fatal("name, street, city, e-mail and hosting should be enough")
	}
	for _, missing := range []func(*Operator){
		func(o *Operator) { o.Name = "" },
		func(o *Operator) { o.Street = "" },
		func(o *Operator) { o.City = "" },
		func(o *Operator) { o.Email = "" },
		func(o *Operator) { o.Hosting = "" },
	} {
		o := full
		missing(&o)
		if o.Complete() {
			t.Errorf("complete although a required field is empty: %+v", o)
		}
	}
	// Phone and VAT id are optional.
	if !(Operator{Name: "n", Street: "s", City: "c", Email: "e", Hosting: "h", Phone: "", VATID: ""}).Complete() {
		t.Error("phone and VAT id must not be required")
	}
}
