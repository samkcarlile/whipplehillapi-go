package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"time"
)

//H shortcut
type H map[string]interface{}

//Headers is used in my sendRequest method
type Headers map[string]string

//WhipplehillAPIClient is the full client wrapper for whipplehill.
type WhipplehillAPIClient struct {
	Client    *http.Client
	CookieJar *cookiejar.Jar
}

//Context combines schoolcontext and context elements
type Context struct {
	SchoolYear           string //This is the schoolyear label in school context in CurrentSchoolYear
	PersonaID            int    //In context in Persona array[0]
	UserID               int    //In context in UserInfo
	CurrentSectionIDList []int  //List of currentSection ids.
	CurrentDurationID    int    //Found by getting GroupTermList and then filtering by Offeringtype=1 and then currentindicater=1

}

//WhApi

//Marshal converts the map to json
func (h *H) Marshal() []byte {
	result, err := json.Marshal(h)
	if err != nil {
		panic(err)
	}
	return result
}

var username = "sam.carlile"
var password = "spicysausage"
var BaseURL = "https://fwcd.myschoolapp.com"

type whippleHillAPIPaths struct {
	SignIn               string
	SchoolContext        string
	Context              string
	TermList             string
	AcademicGroups       string
	MarkingPeriods       string
	GradebookAssignments string
}

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

func main() {

	cookieJar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	client := &http.Client{Timeout: time.Second * 15, Jar: cookieJar}

	//Sign in
	sendRequest(c, "POST", baseUrl+u)

	//My cookiejar experiment
	cookies := res.Cookies()
	println("MY COOKIES")
	for i := range cookies {
		c := cookies[i]
		println(c.Name + "=" + c.Value)
	}

	//School Context
	req, err = http.NewRequest("GET", baseUrl+Urls["SCHOOL_CONTEXT"], nil)
	if err != nil {
		panic(err)
	}
	res, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	resBody, err = ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	var schoolContext H
	schoolContext, err = unmarshal(resBody)
	if err != nil {
		panic(err)
	}
	println(string(resBody))

	//Student Context (this is probably the most useful thing on here)
	req, err = http.NewRequest("GET", baseUrl+Urls["STUDENT_CONTEXT"], nil)
	if err != nil {
		panic(err)
	}
	res, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	resBody, err = ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	var studentContext H
	studentContext, err = unmarshal(resBody)

	//StudentGradeLevelList gets a list of all the grade levels in a json array and has a "current" indicator
	req, err = http.NewRequest("GET", baseUrl+Urls["STUDENT_GRADE_LEVEL_LIST"], nil)
	if err != nil {
		panic(err)
	}
	res, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	resBody, err = ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	println(resBody)

	//StudentGroupTermList gets a json array off all the possible terms. This includes sports terms and stuff.
	req, err = http.NewRequest("GET", baseUrl+Urls["STUDENT_GROUP_TERM_LIST"], nil)
	q := req.URL.Query()
	q.Add("studentUserId", strconv.Itoa(int(studentContext["UserInfo"].(map[string]interface{})["UserId"].(float64))))
	q.Add("schoolYearLabel", schoolContext["CurrentSchoolYear"].(map[string]interface{})["SchoolYearLabel"].(string))
	q.Add("personaId", strconv.Itoa(int(studentContext["Personas"].([]interface{})[0].(map[string]interface{})["Id"].(float64))))
	req.URL.RawQuery = q.Encode()
	res, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()
	resBody, err = ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	println(resBody)

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
