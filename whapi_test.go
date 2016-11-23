package whapi

import (
	"fmt"
	"testing"
)

var wac *WhipplehillAPIClient
var currentTerm *Term
var groups []AcademicGroup

func TestSignIn(t *testing.T) {
	wac = NewWhipplehillAPIClient("https://fwcd.myschoolapp.com")

	err := wac.SignIn("sam.carlile", "spicysausage")
	if err != nil {
		t.Error(err)
	}

}

func TestContexts(t *testing.T) {
	err := wac.GetContexts()
	if err != nil {
		t.Error(err)
	}

	println(wac.UserInfo.String())
	println(wac.Context.String())

}

func TestTerms(t *testing.T) {
	terms, err := wac.GetTermList()
	if err != nil {
		t.Error(err)
	}
	println("Printing Term List!")
	for i := range terms {
		println(terms[i].String() + ",")
	}

	currentTerm = wac.GetCurrentAcademicTerm(terms)
}

func TestGetClasses(t *testing.T) {
	var err error
	groups, err = wac.GetAcademicGroups(currentTerm.DurationID)
	if err != nil {
		t.Error(err)
	}

}

func TestGetAsssignments(t *testing.T) {
	assignments, err := wac.GetAssignments(groups[0].SectionID)
	if err != nil {
		t.Error(err)
	}
	println("ASSIGNMENTS FOR CLASS: " + groups[0].SectionTitle)
	for i := range assignments {
		fmt.Printf("%v - %f", assignments[i].ShortDescription, assignments[i].GetGrade()*100)
		println()
	}
}
