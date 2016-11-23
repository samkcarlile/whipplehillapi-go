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
// * There must be a method to manually load the context and userinfo values

type (
	//WhippleHillAPIPaths holds the api paths
	WhippleHillAPIPaths struct {
		SignIn               string
		SchoolContext        string
		Context              string
		TermList             string
		AcademicGroups       string
		MarkingPeriods       string
		GradebookAssignments string
	}

	//UserInfo has the username and password of the user along with some other stuff like the UserID
	//and things related to the user that don't change with every request. Anything in the student context will be on this struct.
	UserInfo struct {
		Username  string
		Password  string
		UserID    int
		PersonaID int
	}

	//Context has things that aren't directly related to user's characterstics but rather the actual classes and sections
	// that the user is in. Anything in the school context or other necessary information will be stored on this struct.
	Context struct {
		SchoolName             string
		SchoolYearLabel        string
		CurrentDurationID      int //this is a semester identifier
		CurrentMarkingPeriodID int //this is a quarter identifier. You need both
	}

	//AcademicGroup is synonymous with a class that the user is enrolled in. I just use AcademicGroup because there are
	// different types of groups and if I ever want to fully wrap the api it's important to refer to things how they are
	// in the actual API
	AcademicGroup struct {
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
	Assignment struct {
		ShortDescription string `json:"AssignmentShortDescription"`
		Type             string `json:"AssignmentType"`
		MaxPoints        int    `json:"MaxPoints"`
		Points           string `json:"Points"`
	}

	//Term holds the terms data that comes back in an array from getting studentusergrouptermlist
	Term struct {
		CurrentIndicator int    `json:"CurrentInd"`
		Description      string `json:"DurationDescription"`
		DurationID       int    `json:"DurationId"`
		OfferingType     int    `json:"OfferingType"`
	}

	//Headers is used in the wac.request method
	Headers map[string]string

	//WhipplehillAPIClient is the full client wrapper for whipplehill.
	WhipplehillAPIClient struct {
		BaseURL   string
		Client    *http.Client
		CookieJar *cookiejar.Jar
		UserInfo  *UserInfo
		Context   *Context
		APIPaths  *WhippleHillAPIPaths
		SignedIn  bool
	}
)

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

//GetGrade calculates the grade of the assignment
func (a *Assignment) GetGrade() float64 {
	points, _ := strconv.ParseFloat(a.Points, 32)
	return points / float64(a.MaxPoints)
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
		SignedIn:  false,
	}
}

//LoadUserInfo is my pointless lazy version of loading existing userinfo into the client
func (wac *WhipplehillAPIClient) LoadUserInfo(ui *UserInfo) {
	//probably should do some validation here
	wac.UserInfo = ui
}

//LoadContext is again my pointless lazy version of loading existing context into the client.
//This is more of a reminder for me to actually implement this in the future.
func (wac *WhipplehillAPIClient) LoadContext(ctx *Context) {
	wac.Context = ctx
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

//SignIn uses the provided username and password to attempt a sign in request. If succesfull, it gets the contexts and fills that too.
// The api is ready to be used after this is called.
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
	wac.SignedIn = true
	return nil
}

//GetUserContext returns the user context json in a map[string]interface{}
func (wac *WhipplehillAPIClient) GetUserContext() (map[string]interface{}, error) {
	err := wac.checkSignIn()
	if err != nil {
		return nil, err
	}

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

func (wac *WhipplehillAPIClient) checkSignIn() error {
	if !wac.SignedIn {
		return errors.New("Not signed in. Must call wac.SignIn before subsequent API calls.")
	}
	return nil
}

func (wac *WhipplehillAPIClient) checkContexts() error {
	if len(wac.Context.SchoolName) < 1 || len(wac.Context.SchoolYearLabel) < 1 {
		return errors.New("Context hasn't been loaded. Must call wac.LoadContexts before subsequent API calls not including wac.SignIn")
	}
	return nil
}

func (wac *WhipplehillAPIClient) checkReady() error {
	err1 := wac.checkSignIn()
	err2 := wac.checkContexts()
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

//GetSchoolContext returns the school context json in a map[string]interface{}
func (wac *WhipplehillAPIClient) GetSchoolContext() (map[string]interface{}, error) {
	err := wac.checkSignIn()
	if err != nil {
		return nil, err
	}

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

//LoadContexts gets the user and school context and fills some important variables
func (wac *WhipplehillAPIClient) GetContexts() error {
	err := wac.checkSignIn()
	if err != nil {
		return err
	}

	userCtx, err := wac.GetUserContext()
	if err != nil {
		return err
	}

	//todo: add error checking to these type assertions
	wac.UserInfo.UserID = userCtx["UserInfo"].(map[string]interface{})["UserId"].(int)
	wac.UserInfo.PersonaID = userCtx["Personas"].([]map[string]interface{})[0]["Id"].(int)

	schoolCtx, err := wac.GetSchoolContext()
	if err != nil {
		return err
	}

	//todo: add error checking to these type assertions
	wac.Context.SchoolName = schoolCtx["SchoolInfo"].(map[string]interface{})["SchoolName"].(string)
	wac.Context.SchoolYearLabel = schoolCtx["CurrentSchoolYear"].(map[string]interface{})["SchoolYearLabel"].(string)

	return nil
}

//GetTermList returns a term list for the current user. Again, I need to check if the login is succesfull...
func (wac *WhipplehillAPIClient) GetTermList() ([]Term, error) {
	err := wac.checkReady()
	if err != nil {
		return nil, err
	}

	urlWithQueries := addQueries(wac.APIPaths.TermList, map[string]string{
		"studentUserId":   string(wac.UserInfo.UserID),
		"personaId":       string(wac.UserInfo.PersonaID),
		"schoolYearLabel": wac.Context.SchoolYearLabel,
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

//GetAcademicGroups returns an array of the user's academic groups provided the durationID of the term (the marking period is automatically set as the current marking period.)
// Note: I have left out entirley the request that returns the marking periods for a given term because of this exact reason ^
// Also, this is how the Client gets the markingid...It's assuming you are going to call this at some point before you call anything that relies on that.
// This may be a bad design but it can be easily fixed.
func (wac *WhipplehillAPIClient) GetAcademicGroups(durationID int) ([]AcademicGroup, error) {
	err := wac.checkReady()
	if err != nil {
		return nil, err
	}

	urlWithQueries := addQueries(wac.APIPaths.AcademicGroups, map[string]string{
		"userId":          string(wac.UserInfo.UserID),
		"schoolYearLabel": wac.Context.SchoolYearLabel,
		"memberLevel":     string(3), //this is an apparent constant (for students at least)
		"persona":         string(wac.UserInfo.PersonaID),
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

	//This is where we hackily snatch the CurrentMarkingPeriodID (instead of sending a request for it and finding it, which is possible but not in this codebase.)
	wac.Context.CurrentMarkingPeriodID = result[0].MarkingPeriodID //bc they are all the same. It's a list of the current classees the student is taking..its assuming youu're taking classes.

	return result, nil

}

//GetAssignments returns a list of gradebook assignments for the given class (determined by the sectionID parameter)
func (wac *WhipplehillAPIClient) GetAssignments(sectionID int) ([]Assignment, error) {
	err := wac.checkSignIn()
	if err != nil {
		return nil, err
	}

	urlWithQueries := addQueries(wac.APIPaths.GradebookAssignments, map[string]string{
		"sectionId":       string(sectionID),
		"markingPeriodId": string(wac.Context.CurrentMarkingPeriodID),
		"studentUserId":   string(wac.UserInfo.UserID),
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
