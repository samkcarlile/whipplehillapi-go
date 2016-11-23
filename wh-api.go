package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"
)

//Notes:
// * There is a constant called MemberLevel that should always be equal to 3
// * I need to actually check for the t cookie or sign in before I make some of these requests

//WhippleHillAPIPaths holds the api paths
type WhippleHillAPIPaths struct {
	SignIn               string
	SchoolContext        string
	Context              string
	TermList             string
	AcademicGroups       string
	MarkingPeriods       string
	GradebookAssignments string
}

//GetAPIPaths returns a WhippleHillAPIPaths struct with the paths set to the provided baseURL.
func GetAPIPaths(baseURL string) *WhippleHillAPIPaths {
	return &WhippleHillAPIPaths{
		SignIn:               baseURL + "/api/SignIn",
		SchoolContext:        baseURL + "/api/webapp/schoolcontext",
		Context:              baseURL + "/api/webapp/context",
		TermList:             baseURL + "/api/DataDirect/StudentGroupTermList/",
		AcademicGroups:       baseURL + "/api/datadirect/ParentStudentUserAcademicGroupsGet",
		MarkingPeriods:       baseURL + "/api/gradebook/GradeBookMyDayMarkingPeriods",
		GradebookAssignments: baseURL + "/api/datadirect/GradeBookPerformanceAssignmentStudentList/",
	}
}

//Headers is used in the wac.request method
type Headers map[string]string

//UserInfo has the username and password of the user along with some other stuff like the UserID
//and things related to the user that don't change with every request. Anything in the student context will be on this struct.
type UserInfo struct {
	Username  string
	Password  string
	UserID    int
	PersonaID int
}

//Context has things that aren't directly related to user's characterstics but rather the actual classes and sections
// that the user is in. Anything in the school context or other necessary information will be stored on this struct.
type Context struct {
	SchoolName             string
	SchoolYearLabel        string
	CurrentDurationID      int //this is a semester identifier
	CurrentMarkingPeriodID int //this is a quarter identifier. You need both
}

//AcademicGroup is synonymous with a class that the user is enrolled in. I just use AcademicGroup because there are
// different types of groups and if I ever want to fully wrap the api it's important to refer to things how they are
// in the actual API
type AcademicGroup struct {
	DurationID               int    `json:"DurationId"`
	OwnerID                  int    `json:"OwnerId"`
	AssignmentsActiveToday   int    `json:"assignmentactivetoday"`
	AssignmentsAssignedToday int    `json:"assignmentassignedtoday"`
	AssignmentsDueToday      int    `json:"assignmentduetoday"`
	Description              string `json:"coursedescription"`
	CumGrade                 string `json:"cumgrade"`
	GroupOwnerName           string `json:"groupownername"`
	GroupOwnerEmail          string `json:"groupowneremail"`
	LeadSectionID            int    `json:"leadsectionid"`
	MarkingPeriodID          int    `json:"markingperiodid"`
	SectionID                int    `json:"sectionid"`
	SectionTitle             string `json:"sectionidentifier"`
}

//Assignment is the struct for an assignment returned from when you get the gradebook of a class.
type Assignment struct {
	ShortDescription string `json:"AssignmentShortDescription"`
	Type             string `json:"AssignmentType"`
	MaxPoints        int    `json:"MaxPoints"`
	Points           string `json:"Points"`
}

//GetGrade calculates the grade of the assignment
func (a *Assignment) GetGrade() float64 {
	points, _ := strconv.ParseFloat(a.Points, 32)
	return points / float64(a.MaxPoints)
}

//Term holds the terms data that comes back in an array from getting studentusergrouptermlist
type Term struct {
	CurrentIndicator int    `json:"CurrentInd"`
	Description      string `json:"DurationDescription"`
	DurationID       int    `json:"DurationId"`
	OfferingType     int    `json:"OfferingType"`
}

//WhipplehillAPIClient is the full client wrapper for whipplehill.
type WhipplehillAPIClient struct {
	BaseURL   string
	Client    *http.Client
	CookieJar *cookiejar.Jar
	UserInfo  *UserInfo
	Context   *Context
	APIPaths  *WhippleHillAPIPaths
}

//NewWhipplehillAPIClient returns a new client with some things
func NewWhipplehillAPIClient(baseURL string) *WhipplehillAPIClient {
	j, _ := cookiejar.New(nil)
	c := &http.Client{Timeout: time.Second * 15, Jar: j}
	return &WhipplehillAPIClient{
		BaseURL:   baseURL,
		Client:    c,
		CookieJar: j,
		UserInfo:  &UserInfo{},
		Context:   &Context{},
		APIPaths:  GetAPIPaths(baseURL),
	}
}

func (wac *WhipplehillAPIClient) request(method string, u string, body []byte, headers Headers) ([]byte, error) {
	var req *http.Request
	var err error
	if body == nil {
		req, err = http.NewRequest(method, u, nil)
	} else {
		req, err = http.NewRequest(method, u, bytes.NewBufferString(string(body)))
	}
	if err != nil {
		return nil, err
	}

	//Add headers
	if headers == nil {
		headers = wac.defaultHeaders()
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	//Execute request
	res, err := wac.Client.Do(req)
	if err != nil {
		return nil, err
	}
	//VERY IMPORTANT
	defer res.Body.Close()
	//Read the body into a []byte and return that.
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return resBody, nil
}

func (wac *WhipplehillAPIClient) defaultHeaders() Headers {
	return Headers{
		"Content-Type": "application/json",
	}
}

//SignIn uses the provided username and password to attempt a sign in request.
func (wac *WhipplehillAPIClient) SignIn(username string, password string) error {

	payload := map[string]string{
		"Username": username,
		"Password": password,
	}

	jsonPayload, err := json.Marshal(payload)

	body, err := wac.request("POST", wac.APIPaths.SignIn, jsonPayload, nil)
	if err != nil {
		return err
	}
	jsonBody, err := unmarshal(body)
	if err != nil {
		return err
	}
	isSuccess := jsonBody["LoginSuccessful"].(bool)
	if !isSuccess {
		return errors.New("Invalid login credentials")
	}
	wac.UserInfo.Username = username
	wac.UserInfo.Password = password
	return nil
}

//GetUserContext returns the user context json in a map[string]interface{}
func (wac *WhipplehillAPIClient) GetUserContext() (map[string]interface{}, error) {
	body, err := wac.request("GET", wac.APIPaths.Context, nil, nil)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(body, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

//GetSchoolContext returns the school context json in a map[string]interface{}
func (wac *WhipplehillAPIClient) GetSchoolContext() (map[string]interface{}, error) {
	body, err := wac.request("GET", wac.APIPaths.SchoolContext, nil, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

//GetTermList returns a term list for the current user. Again, I need to check if the login is succesfull...
func (wac *WhipplehillAPIClient) GetTermList(userID int, personaID int, schoolYearLabel string) ([]Term, error) {
	urlWithQueries := addQueries(wac.APIPaths.TermList, map[string]string{
		"studentUserId":   string(userID),
		"personaId":       string(personaID),
		"schoolYearLabel": schoolYearLabel,
	})
	body, err := wac.request("GET", urlWithQueries, nil, nil)
	if err != nil {
		return nil, err
	}

	terms := make([]Term, 0)
	err = json.Unmarshal(body, terms)
	if err != nil {
		return nil, err
	}

	return terms, nil
}

//GetCurrentAcademicTerm uses the current academic term id for the durationList parameter.
func (wac *WhipplehillAPIClient) GetCurrentAcademicTerm(terms []Term) *Term {
	var currentTerm Term
	found := false

	for i := 0; i < len(terms) && !found; i++ {
		currentTerm = terms[i]
		if currentTerm.OfferingType == 1 && currentTerm.CurrentIndicator == 1 {
			found = true
		}
	}

	return &currentTerm
}

//GetAcademicGroups returns an array of the user's academic groups provided the term
func (wac *WhipplehillAPIClient) GetAcademicGroups(userID int, schoolYearLabel string, personaID int, durationID int) ([]AcademicGroup, error) {
	urlWithQueries := addQueries(wac.APIPaths.AcademicGroups, map[string]string{
		"userId":          string(userID),
		"schoolYearLabel": schoolYearLabel,
		"memberLevel":     string(3),
		"persona":         string(personaID),
		"durationList":    string(durationID),
		"markingPeriodId": "", //idk why but you have to have this or you'll get an error
	})

	body, err := wac.request("GET", urlWithQueries, nil, nil)
	if err != nil {
		return nil, err
	}

	var result = make([]AcademicGroup, 0)
	err = json.Unmarshal(body, result)
	if err != nil {
		return nil, err
	}

	return result, nil

}

//GetGradebookAssignments returns a list of gradebook assignments for the given class (determined by the sectionID parameter)
func (wac *WhipplehillAPIClient) GetGradebookAssignments(userID int, sectionID int, markingPeriodID int) ([]Assignment, error) {
	urlWithQueries := addQueries(wac.APIPaths.GradebookAssignments, map[string]string{
		"sectionId":       string(sectionID),
		"markingPeriodId": string(markingPeriodID),
		"studentUserId":   string(userID),
	})

	body, err := wac.request("GET", urlWithQueries, nil, nil)
	if err != nil {
		return nil, err
	}

	assignments := make([]Assignment, 0)
	err = json.Unmarshal(body, assignments)
	if err != nil {
		return nil, err
	}

	return assignments, nil
}

//LoadUserContext gets the user context and fills the user struct. NOTE: needs to be renamed based on the new api design
// func (wac *WhipplehillAPIClient) LoadUserContext() error {
// 	body, err := wac.request("GET", wac.APIPaths.Context, nil, nil)
// 	if err != nil {
// 		return err
// 	}
// 	jsonBody, err := unmarshal(body)
// 	if err != nil {
// 		return err
// 	}

// 	//This looks sort of gross but it's just type assertions
// 	wac.UserInfo.UserID = jsonBody["UserInfo"].(H)["UserId"].(int)
// 	wac.UserInfo.PersonaID = jsonBody["Personas"].([]H)[0]["Id"].(int)

// 	return nil

// }

// //LoadSchoolContext gets the school context and fills the schoolcontext struct. Warning: this makes multiple requests.
// func (wac *WhipplehillAPIClient) LoadSchoolContext() error {
// 	body, err := wac.request("GET", wac.APIPaths.SchoolContext, nil, nil)
// 	if err != nil {
// 		return err
// 	}
// 	jsonBody, err := unmarshal(body)
// 	if err != nil {
// 		return err
// 	}
// 	wac.Context.SchoolYearLabel = jsonBody["CurrentSchoolYear"].(H)["SchoolYearLabel"].(string)
// 	wac.Context.SchoolName = jsonBody["SchoolInfo"].(H)["SchoolName"].(string)

// 	err = wac.getCurrentDurationID()
// 	if err != nil {
// 		return err
// 	}

// }

func main() {

	wac := NewWhipplehillAPIClient("https://fwcd.myschoolapp.com")

	err := wac.SignIn("sam.carlile", "spicysausage")
	if err != nil {
		panic(err)
	}
	println(wac.UserInfo.Username)

	//StudentGroupTermList gets a json array off all the possible terms. This includes sports terms and stuff.
	// req, err = http.NewRequest("GET", baseUrl+Urls["STUDENT_GROUP_TERM_LIST"], nil)
	// q := req.URL.Query()
	// q.Add("studentUserId", strconv.Itoa(int(studentContext["UserInfo"].(map[string]interface{})["UserId"].(float64))))
	// q.Add("schoolYearLabel", schoolContext["CurrentSchoolYear"].(map[string]interface{})["SchoolYearLabel"].(string))
	// q.Add("personaId", strconv.Itoa(int(studentContext["Personas"].([]interface{})[0].(map[string]interface{})["Id"].(float64))))
	// req.URL.RawQuery = q.Encode()
}

func addQueries(us string, qs map[string]string) string {
	u, err := url.Parse(us)
	if err != nil {
		panic(err)
	}
	q := u.Query()
	for k, v := range qs {
		q.Add(k, v)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

func unmarshal(data []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
