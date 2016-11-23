package whapi

import (
	"encoding/json"
	"strconv"
	"testing"
)

var wac *WhipplehillAPIClient

func TestSignIn(t *testing.T) {
	wac = NewWhipplehillAPIClient("https://fwcd.myschoolapp.com")

	err := wac.SignIn("sam.carlile", "spicysausage")
	if err != nil {
		t.Error(err)
	}

	println("Username: " + wac.UserInfo.Username + ", Password: " + wac.UserInfo.Password)
}

func TestContexts(t *testing.T) {
	err := wac.GetContexts()
	if err != nil {
		t.Error(err)
	}

	println("wac.UserInfo")
	println("UserID: " + strconv.Itoa(wac.UserInfo.UserID) + ", PersonaID: " + strconv.Itoa(wac.UserInfo.PersonaID))
	println("wac.Context")
	println("School Year: " + wac.Context.SchoolYearLabel + ", School Name: " + wac.Context.SchoolName)
}

func TestTerms(t *testing.T) {
	terms, err := wac.GetTermList()
	if err != nil {
		t.Error(err)
	}
	wtf, _ := json.Marshal(terms)
	println(string(wtf))
	//term := wac.GetCurrentAcademicTerm(terms)
	println("Current Academic Term")
	// println("Desc: " + term.Description)
	println("CurrentDurationID (from wac.Context): " + strconv.Itoa(wac.Context.CurrentDurationID))
}
