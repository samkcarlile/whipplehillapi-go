package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

//Notes:
// * There is a constant called MemberLevel that should always be equal to 3

type whippleHillAPIPaths struct {
	SignIn               string
	SchoolContext        string
	Context              string
	TermList             string
	AcademicGroups       string
	MarkingPeriods       string
	GradebookAssignments string
}

//BaseURL is le baseurl...for now. I'm gonna fix the whole url thing nicely soon.
var BaseURL = "https://fwcd.myschoolapp.com"

//WhippleHillAPIPaths is the struct to serve as a container for all the (necessary) API paths
var WhippleHillAPIPaths = whippleHillAPIPaths{
	SignIn:               BaseURL + "/api/SignIn",
	SchoolContext:        BaseURL + "/api/webapp/schoolcontext",
	Context:              BaseURL + "/api/webapp/context",
	TermList:             BaseURL + "/api/DataDirect/StudentGroupTermList/",
	AcademicGroups:       BaseURL + "/api/datadirect/ParentStudentUserAcademicGroupsGet",
	MarkingPeriods:       BaseURL + "/api/gradebook/GradeBookMyDayMarkingPeriods",
	GradebookAssignments: BaseURL + "/api/datadirect/GradeBookPerformanceAssignmentStudentList/",
}

//H shortcut
type H map[string]interface{}

//Headers is used in my sendRequest method
type Headers map[string]string

//UserInfo has the username and password of the user along with some other stuff like the UserID
//and things related to the user that don't change with every request.
type UserInfo struct {
	Username  string
	Password  string
	UserID    int64
	PersonaID int
}

//Context has things that aren't directly related to user's characterstics but rather the actual classes and sections
// that the user is in
type Context struct {
	SchoolYearLabel        string
	CurrentDurationID      int64
	CurrentMarkingPeriodID int64
}

//AcademicGroup is synonymous with a class that the user is enrolled in. I just use AcademicGroup because there are
// different types of groups and if I ever want to fully wrap the api it's important to refer to things how they are
// in the actual API
type AcademicGroup struct {
	SectionID     int64 // Ok...I have no idea what the difference between these is, but I am just doing it how their API does
	LeadSectionID int64
	Title         string
	Grade         int
}

//Assignment is the struct for an assignment returned from when you get the gradebook of a class.
type Assignment struct {
	ShortDescription string
	Type             string
	MaxPoints        int
	Points           float32
}

//WhipplehillAPIClient is the full client wrapper for whipplehill.
type WhipplehillAPIClient struct {
	BaseURL   string
	Client    *http.Client
	CookieJar *cookiejar.Jar
	UserInfo  *UserInfo
	Context   *Context
}

//NewWhipplehillAPIClient returns a new client with some things
func NewWhipplehillAPIClient(baseurl string) *WhipplehillAPIClient {
	j, _ := cookiejar.New(nil)
	c := &http.Client{Timeout: time.Second * 15, Jar: j}
	return &WhipplehillAPIClient{
		BaseURL:   baseurl,
		Client:    c,
		CookieJar: j,
		UserInfo:  &UserInfo{},
		Context:   &Context{},
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
	if headers != nil {
		for k, v := range headers {
			req.Header.Add(k, v)
		}
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

	payload := &H{
		"Username": username,
		"Password": password,
	}

	body, err := wac.request("POST", WhippleHillAPIPaths.SignIn, payload.Marshal(), wac.defaultHeaders())
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

//LoadContexts does the shit
func (wac *WhipplehillAPIClient) LoadContexts() error {

}

//Marshal converts the map to json
func (h *H) Marshal() []byte {
	result, err := json.Marshal(h)
	if err != nil {
		panic(err)
	}
	return result
}

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

func sendRequest(c *http.Client, method string, u string, body []byte, headers Headers) ([]byte, error) {
	req, err := http.NewRequest(method, u, bytes.NewBufferString(string(body)))
	if err != nil {
		return nil, err
	}

	//Add headers
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	//Execute request
	res, err := c.Do(req)
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

func unmarshal(data []byte) (H, error) {
	var result map[string]interface{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}