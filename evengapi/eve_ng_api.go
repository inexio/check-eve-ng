package evengapi

import (
	"encoding/json"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"net"
	"net/url"
	"regexp"
	"strings"
)

/*
EveNgApi is used to communicate with eve ng api.
*/
type EveNgApi struct {
	*eveNgApi
}

type eveNgApi struct {
	hostname string
	username string
	password string
	client   *resty.Client
	http     bool
}

/*
NotValidError is returned when an EveNgApi object was not initialized properly with the NewEveNgApi function
*/
type NotValidError struct{}

func (m *NotValidError) Error() string {
	return "EveNgApi was not created properly with the func NewEveNgApi()"
}

/*
NewEveNgApi generates a new eve ng api object and validates the input parameters
*/
func NewEveNgApi(hostname string, username string, password string) (*EveNgApi, error) {
	err := validateInputParams(hostname, username, password)
	if err != nil {
		return nil, errors.Wrap(err, "invalid input parameter")
	}
	eveNgApi := eveNgApi{hostname, username, password, resty.New(), false}
	return &EveNgApi{&eveNgApi}, nil
}

func (e *EveNgApi) isValid() bool {
	return e.eveNgApi != nil
}

/*
ForceHttp can be used to force http instead of https
*/
func (e *EveNgApi) ForceHttp(useHttp bool) error {
	if !e.isValid() {
		return &NotValidError{}
	}
	e.http = true
	return nil
}

func validateInputParams(hostname string, username string, password string) error {
	//validate hostname, check if hostname is a valid hostname or a valid ip
	ip := net.ParseIP(hostname)
	if ip == nil {
		_, err := net.LookupHost(hostname)
		if err != nil {
			return errors.New("given hostname is neither a valid ip address nor a hostname (dns lookup failed)")
		}
	}
	if username == "" {
		return errors.New("invalid username")
	}
	if password == "" {
		return errors.New("invalid password")
	}
	return nil
}

func (e *EveNgApi) get(path string, body string) (*resty.Response, error) {
	request := e.client.R()
	if body != "" {
		request.SetBody(body)
	}
	response, err := request.Get(e.getProtocol() + "://" + e.hostname + urlEscape(path))
	if err != nil {
		return nil, errors.Wrap(err, "error during http request")
	}
	if response.StatusCode() != 200 {
		return nil, errors.Wrap(getHttpError(response), "http status code != 200")
	}
	return response, nil
}

func (e *EveNgApi) post(path string, body string) (*resty.Response, error) {
	request := e.client.R()
	if body != "" {
		request.SetBody(body)
	}
	response, err := request.Post(e.getProtocol() + "://" + e.hostname + urlEscape(path))
	if err != nil {
		return nil, errors.Wrap(err, "error during http request")
	}
	if response.StatusCode() != 200 {
		return nil, errors.Wrap(getHttpError(response), "http status code != 200")
	}
	return response, nil
}

func (e *EveNgApi) getProtocol() string {
	if e.http {
		return "http"
	}
	return "https"
}

/*
Login does a login with the given username and password
*/
func (e *EveNgApi) Login() error {
	if !e.isValid() {
		return &NotValidError{}
	}
	escapedUsername, err := jsonEscape(e.username)
	if err != nil {
		return errors.Wrap(err, "error during json escaping username")
	}

	escapedPassword, err := jsonEscape(e.password)
	if err != nil {
		return errors.Wrap(err, "error during json escaping password")
	}
	_, err = e.post("/api/auth/login", `{"username":"`+escapedUsername+`","password":"`+escapedPassword+`"}`)
	if err != nil {
		return errors.Wrap(err, "error during http login request")
	}
	return nil
}

/*
Logout closes the connection to the api. It should always be called directly after Login in an defer statement
*/
func (e *EveNgApi) Logout() error {
	if !e.isValid() {
		return &NotValidError{}
	}
	_, err := e.get("/api/auth/logout", "")
	if err != nil {
		return errors.Wrap(err, "error during http loqout request")
	}
	return nil
}

func getHttpError(response *resty.Response) error {
	data, err := jsonDecode(response.Body())
	if err != nil {
		return errors.New("Status != 200")
	}
	return errors.New(data["message"].(string))
}

/*
GetSystemStatus returns the system status of eve ng
*/
func (e *EveNgApi) GetSystemStatus() (SystemStatus, error) {
	if !e.isValid() {
		return SystemStatus{}, &NotValidError{}
	}
	response, err := e.get("/api/status", "")
	if err != nil {
		return SystemStatus{}, errors.Wrap(err, "error during http get system status request")
	}
	var systemStatusResponse systemStatusResponse
	err = json.Unmarshal(response.Body(), &systemStatusResponse)
	if err != nil {
		return SystemStatus{}, errors.Wrap(err, "error during unmarshal")
	}
	return systemStatusResponse.Data, nil
}

/*
GetAllNodesForLab returns all nodes that exist for one lab.
*/
func (e *EveNgApi) GetAllNodesForLab(lab string) (map[string]Nodes, error) {
	if !e.isValid() {
		return nil, &NotValidError{}
	}
	response, err := e.get("/api/labs/"+lab+".unl/nodes", "")
	if err != nil {
		return nil, errors.Wrap(err, "error during http get request")
	}
	var nodes nodesResponse
	err = json.Unmarshal(response.Body(), &nodes)
	if err != nil {
		return nil, err
	}
	return nodes.Data, nil
}

/*
GetAllLabs returns all labs.
*/
func (e *EveNgApi) GetAllLabs() ([]string, error) {
	if !e.isValid() {
		return nil, &NotValidError{}
	}
	return e.getAllLabsForFolder("/")
}

func (e *EveNgApi) getAllLabsForFolder(folder string) ([]string, error) {
	var labs []string
	response, err := e.get("/api/folders"+folder, "")
	if err != nil {
		return nil, errors.Wrap(err, "error during http get request")
	}
	var res folderResponse
	err = json.Unmarshal(response.Body(), &res)
	if err != nil {
		return nil, errors.Wrap(err, "error during unmarshal")
	}

	for _, folder := range res.Data.Folders {
		if folder.Name == ".." {
			continue
		}

		subLabs, err := e.getAllLabsForFolder(folder.Path)
		if err != nil {
			return nil, errors.Wrap(err, "error while getting labs for path "+folder.Path)
		}
		labs = SliceMerge(labs, subLabs)
	}

	for _, lab := range res.Data.Labs {
		regexFirstSlash := regexp.MustCompile(`^/(.+)$`)
		labName := regexFirstSlash.ReplaceAllString(lab.Path, "$1")

		regexUnl := regexp.MustCompile(`^(.+)\.unl$`)
		labName = regexUnl.ReplaceAllString(labName, "$1")

		labs = append(labs, labName)
	}
	return labs, nil
}

func jsonDecode(byteArr []byte) (map[string]interface{}, error) {
	var data map[string]interface{}
	err := json.Unmarshal(byteArr, &data)
	if err != nil {
		return nil, errors.Wrap(err, "error during json decode")
	}
	return data, nil
}

func urlEscape(unescaped string) string {
	arr := strings.Split(unescaped, "/")
	for i, partString := range strings.Split(unescaped, "/") {
		arr[i] = url.QueryEscape(partString)
	}
	return strings.Join(arr, "/")
}

func jsonEscape(unescaped string) (string, error) {
	escaped, err := json.Marshal(unescaped)
	if err != nil {
		return "", errors.Wrap(err, "json marshal failed")
	}
	return string(escaped)[1 : len(escaped)-1], nil
}

/*
SliceMerge is used to merge two slices into one and removes all duplicate entries.
*/
func SliceMerge(slice1 []string, slice2 []string) []string {
	slice1 = append(slice1, slice2...)
	tempMap := make(map[string]struct{}, len(slice1))
	i := 0
	for _, s := range slice1 {
		if _, ok := tempMap[s]; ok {
			continue
		}
		tempMap[s] = struct{}{}

		slice1[i] = s
		i++
	}
	return slice1[:i]
}
