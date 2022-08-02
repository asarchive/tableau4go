// Copyright 2013 Matthew Baird
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tableau4go

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const contentTypeHeader = "Content-Type"
const contentLengthHeader = "Content-Length"
const authHeader = "X-Tableau-Auth"
const applicationXmlContentType = "application/xml"
const POST = "POST"
const GET = "GET"
const DELETE = "DELETE"
const PAGESIZE = 100

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Sign_In%3FTocPath%3DAPI%2520Reference%7C_____51
func (api *API) Signin(username, password string, contentUrl string, userIdToImpersonate string) error {
	requestUrl := fmt.Sprintf("%s/api/%s/auth/signin", api.Server, api.Version)
	credentials := Credentials{Name: username, Password: password}
	if len(userIdToImpersonate) > 0 {
		credentials.Impersonate = &User{ID: userIdToImpersonate}
	}
	siteName := contentUrl
	// this seems to have changed. If you are looking for the default site, you must pass
	// blank
	if api.OmitDefaultSiteName {
		if contentUrl == api.DefaultSiteName {
			siteName = ""
		}
	}
	credentials.Site = &Site{ContentUrl: siteName}
	request := SigninRequest{Request: credentials}
	signInXML, err := request.XML()
	if err != nil {
		return err
	}
	payload := string(signInXML)
	headers := make(map[string]string)
	headers[contentTypeHeader] = applicationXmlContentType
	retval := AuthResponse{}
	err = api.makeRequest(requestUrl, POST, []byte(payload), &retval, headers)
	if err == nil {
		api.AuthToken = retval.Credentials.Token
	}
	return err
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Sign_Out%3FTocPath%3DAPI%2520Reference%7C_____52
func (api *API) Signout() error {
	requestUrl := fmt.Sprintf("%s/api/%s/auth/signout", api.Server, api.Version)
	headers := make(map[string]string)
	headers[contentTypeHeader] = applicationXmlContentType
	err := api.makeRequest(requestUrl, POST, nil, nil, headers)
	return err
}

// helper method to convert to contentUrl as most api methods use this
func ConvertSiteNameToContentURL(siteName string) string {
	return strings.ReplaceAll(siteName, " ", "")
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Server_Info%3FTocPath%3DAPI%2520Reference%7C__
func (api *API) ServerInfo() (ServerInfo, error) {
	// this call only works on apiVersion 2.4 and up
	requestUrl := fmt.Sprintf("%s/api/%s/serverinfo", api.Server, "2.4")
	headers := make(map[string]string)
	retval := ServerInfoResponse{}
	err := api.makeRequest(requestUrl, GET, nil, &retval, headers)
	return retval.ServerInfo, err
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Sites%3FTocPath%3DAPI%2520Reference%7C_____40
func (api *API) QuerySites() ([]Site, error) {
	requestUrl := fmt.Sprintf("%s/api/%s/sites/", api.Server, api.Version)
	headers := make(map[string]string)
	retval := QuerySitesResponse{}
	err := api.makeRequest(requestUrl, GET, nil, &retval, headers)
	return retval.Sites.Sites, err
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Sites%3FTocPath%3DAPI%2520Reference%7C_____40
func (api *API) QuerySite(siteID string, includeStorage bool) (Site, error) {
	requestUrl := fmt.Sprintf("%s/api/%s/sites/%s", api.Server, api.Version, siteID)
	if includeStorage {
		requestUrl += fmt.Sprintf("?includeStorage=%v", includeStorage)
	}
	return api.executeQuerySite(requestUrl)
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Sites%3FTocPath%3DAPI%2520Reference%7C_____40
func (api *API) QuerySiteByName(name string, includeStorage bool) (Site, error) {
	return api.querySiteByKey("name", name, includeStorage)
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Sites%3FTocPath%3DAPI%2520Reference%7C_____40
func (api *API) QuerySiteByContentURL(contentURL string, includeStorage bool) (Site, error) {
	return api.querySiteByKey("contentUrl", contentURL, includeStorage)
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Sites%3FTocPath%3DAPI%2520Reference%7C_____40
func (api *API) querySiteByKey(key, value string, includeStorage bool) (Site, error) {
	requestUrl := fmt.Sprintf("%s/api/%s/sites/%s?key=%s", api.Server, api.Version, value, key)
	if includeStorage {
		requestUrl += fmt.Sprintf("&includeStorage=%v", includeStorage)
	}
	return api.executeQuerySite(requestUrl)
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Sites%3FTocPath%3DAPI%2520Reference%7C_____40
func (api *API) executeQuerySite(requestUrl string) (Site, error) {
	headers := make(map[string]string)
	retval := QuerySiteResponse{}
	err := api.makeRequest(requestUrl, GET, nil, &retval, headers)
	return retval.Site, err
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_User_On_Site%3FTocPath%3DAPI%2520Reference%7C_____47
func (api *API) QueryUserOnSite(siteId, userId string) (User, error) {
	requestUrl := fmt.Sprintf("%s/api/%s/sites/%s/users/%s", api.Server, api.Version, siteId, userId)
	headers := make(map[string]string)
	retval := QueryUserOnSiteResponse{}
	err := api.makeRequest(requestUrl, GET, nil, &retval, headers)
	return retval.User, err
}

func (api *API) QueryProjects(siteId string) ([]Project, error) {
	totalAvailable := 1
	projects := []Project{}
	for i := 1; len(projects) < totalAvailable; i++ {
		projectsResponse, err := api.QueryProjectsByPage(siteId, i)
		if err != nil {
			return projects, err
		}
		projects = append(projects, projectsResponse.Projects.Projects...)
		// bjenkins: projects may be added or deleted while we are requesting them from the server.
		// so it's best to keep resetting the total
		totalAvailable = projectsResponse.Pagination.TotalAvailable
	}
	return projects, nil
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Projects%3FTocPath%3DAPI%2520Reference%7C_____38
func (api *API) QueryProjectsByPage(siteId string, pageNum int) (QueryProjectsResponse, error) {
	requestUrl := fmt.Sprintf("%s/api/%s/sites/%s/projects?pageSize=%v&pageNumber=%v", api.Server, api.Version, siteId, PAGESIZE, pageNum)
	headers := make(map[string]string)
	response := QueryProjectsResponse{}
	err := api.makeRequest(requestUrl, GET, nil, &response, headers)
	return response, err
}

func (api *API) GetProjectByName(siteId, name string) (Project, error) {
	projects, err := api.QueryProjects(siteId)
	if err != nil {
		return Project{}, err
	}
	for _, project := range projects {
		if project.Name == name {
			return project, nil
		}
	}
	return Project{}, fmt.Errorf("Project Named '%s' Not Found", name)
}

func (api *API) GetProjectByID(siteId, id string) (Project, error) {
	projects, err := api.QueryProjects(siteId)
	if err != nil {
		return Project{}, err
	}
	for _, project := range projects {
		if project.ID == id {
			return project, nil
		}
	}
	return Project{}, fmt.Errorf("Project with ID '%s' Not Found", id)
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Query_Datasources%3FTocPath%3DAPI%2520Reference%7C_____33
func (api *API) QueryDatasources(siteId string, datasourceName string) ([]Datasource, error) {
	// jbarefoot: We don't do any paging here, but setting the pageSize to the max of 1000 + filter by name should work
	var requestUrl string
	if datasourceName != "" {
		requestUrl = fmt.Sprintf("%s/api/%s/sites/%s/datasources?pageSize=1000&filter=name:eq:%s", api.Server, api.Version, siteId, url.QueryEscape(datasourceName))
	} else {
		requestUrl = fmt.Sprintf("%s/api/%s/sites/%s/datasources?pageSize=1000", api.Server, api.Version, siteId)
	}

	headers := make(map[string]string)
	retval := QueryDatasourcesResponse{}
	err := api.makeRequest(requestUrl, GET, nil, &retval, headers)
	if api.Debug {
		fmt.Printf("Found %d datasources for siteId %s \n", len(retval.Datasources.Datasources), siteId)
	}
	return retval.Datasources.Datasources, err
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Download_Datasource%3FTocPath%3DAPI%2520Reference%7C_____34
// NOTE: that even though this is under the /datasources path, the docs list it under "Download Datasource" and not e.g. "Query Datasource Content".
func (api *API) getDatasourceContent(siteId, datasourceId string) (string, error) {
	requestUrl := fmt.Sprintf("%s/api/%s/sites/%s/datasources/%s/content?includeExtract=false", api.Server, api.Version, siteId, datasourceId)
	headers := make(map[string]string)

	body, err := api.makeRequestGetBody(requestUrl, GET, nil, nil, headers)
	if err != nil {
		return "", err
	}

	extractedXml, err := extractXmlFromZip(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		if api.Debug {
			fmt.Printf("For datasource with id %s: Got an error treating datasource like a zip (.tdsx), assuming it's plain xml (.tds) instead. \n", datasourceId)
		}
		extractedXml = string(body)
	}

	return extractedXml, nil
}

// assumption is that the intersection of site, project, and datasource name is unique
func (api *API) GetDatasourceContentXML(siteId, tableauProjectId, datasourceName string) (string, error) {
	if api.Debug {
		fmt.Printf("\n Getting data source raw xml for siteId %s, tableauProjectId %s, and datasourceName %s \n", siteId, tableauProjectId, datasourceName)
	}

	var datasource *Datasource
	datasources, err := api.QueryDatasources(siteId, datasourceName)
	if err != nil {
		return "", err
	}

	for _, d := range datasources {
		if d.Project.ID == tableauProjectId && d.Name == datasourceName {
			d := d
			datasource = &d
			break
		}
	}

	if datasource == nil {
		if api.Debug {
			fmt.Printf("Could not find datasource for siteId %s, tableauProjectId %s, and datasourceName %s \n", siteId, tableauProjectId, datasourceName)
		}
		return "", nil
	}

	datasourceXML, err := api.getDatasourceContent(siteId, datasource.ID)

	if err != nil {
		return "", err
	}

	if api.Debug {
		fmt.Printf("Got raw xml for datasource with id %s, raw xml is: \n %s \n", datasource.ID, datasourceXML)
	}

	return datasourceXML, nil
}

// A .tdsx is really just a zip file containing the .tds XML
func extractXmlFromZip(in io.ReaderAt, size int64) (string, error) {
	r, err := zip.NewReader(in, size)

	if err != nil {
		return "", err
	}

	var datasourceFile *zip.File
	if len(r.File) != 1 {
		return "", errors.New("A .tdsx file is expect to be a zip file containing exactly one file, the .tds datasource")
	}
	for _, f := range r.File {
		datasourceFile = f
		break
	}

	readerCloser, err := datasourceFile.Open()
	if err != nil {
		return "", err
	}
	defer readerCloser.Close()

	buf := new(bytes.Buffer)
	if _, err = buf.ReadFrom(readerCloser); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (api *API) GetSiteID(siteName string) (string, error) {
	site, err := api.QuerySiteByName(siteName, false)
	if err != nil {
		return "", err
	}
	return site.ID, err
}

// use this method to easily get the site by name
func (api *API) GetSite(siteName string) (Site, error) {
	if siteName == api.DefaultSiteName {
		site, err := api.QuerySiteByName(siteName, false)
		if err != nil {
			return site, err
		}
		return site, err
	}

	contentUrl := ConvertSiteNameToContentURL(siteName)
	site, err := api.QuerySiteByContentURL(contentUrl, false)
	if err != nil {
		return site, err
	}

	return site, err
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Create_Project%3FTocPath%3DAPI%2520Reference%7C_____14
// POST /api/api-version/sites/site-id/projects
func (api *API) CreateProject(siteId string, project Project) (*Project, error) {
	requestUrl := fmt.Sprintf("%s/api/%s/sites/%s/projects", api.Server, api.Version, siteId)
	createProjectRequest := CreateProjectRequest{Request: project}
	xmlRep, err := createProjectRequest.XML()
	if err != nil {
		return nil, err
	}
	headers := make(map[string]string)
	headers[contentTypeHeader] = applicationXmlContentType
	createProjectResponse := CreateProjectResponse{}
	err = api.makeRequest(requestUrl, POST, xmlRep, &createProjectResponse, headers)
	return &createProjectResponse.Project, err
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Publish_Datasource%3FTocPath%3DAPI%2520Reference%7C_____31
func (api *API) PublishTDS(siteId string, tdsMetadata Datasource, fullTds string, overwrite bool) (*Datasource, error) {
	return api.publishDatasource(siteId, tdsMetadata, fullTds, "tds", overwrite)
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Publish_Datasource%3FTocPath%3DAPI%2520Reference%7C_____31
func (api *API) publishDatasource(siteId string, tdsMetadata Datasource, datasource string, datasourceType string, overwrite bool) (*Datasource, error) {
	requestUrl := fmt.Sprintf("%s/api/%s/sites/%s/datasources?datasourceType=%s&overwrite=%v", api.Server, api.Version, siteId, datasourceType, overwrite)
	payload := fmt.Sprintf("--%s\r\n", api.Boundary)
	payload += "Content-Disposition: name=\"request_payload\"\r\n"
	payload += "Content-Type: text/xml\r\n"
	payload += "\r\n"
	tdsRequest := DatasourceCreateRequest{Request: tdsMetadata}
	xmlRepresentation, err := tdsRequest.XML()
	if err != nil {
		return nil, err
	}

	payload += string(xmlRepresentation)
	payload += fmt.Sprintf("\r\n--%s\r\n", api.Boundary)
	payload += fmt.Sprintf("Content-Disposition: name=\"tableau_datasource\"; filename=\"%s.tds\"\r\n", tdsMetadata.Name)
	payload += "Content-Type: application/octet-stream\r\n"
	payload += "\r\n"
	payload += datasource
	payload += fmt.Sprintf("\r\n--%s--\r\n", api.Boundary)
	headers := make(map[string]string)
	headers[contentTypeHeader] = fmt.Sprintf("multipart/mixed; boundary=%s", api.Boundary)

	var retDatasource *Datasource
	err = api.makeRequest(requestUrl, POST, []byte(payload), retDatasource, headers)
	return retDatasource, err
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Delete_Datasource%3FTocPath%3DAPI%2520Reference%7C_____15
func (api *API) DeleteDatasource(siteId string, datasourceId string) error {
	requestUrl := fmt.Sprintf("%s/api/%s/sites/%s/datasources/%s", api.Server, api.Version, siteId, datasourceId)
	return api.delete(requestUrl)
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Delete_Project%3FTocPath%3DAPI%2520Reference%7C_____17
func (api *API) DeleteProject(siteId string, projectId string) error {
	requestUrl := fmt.Sprintf("%s/api/%s/sites/%s/projects/%s", api.Server, api.Version, siteId, projectId)
	return api.delete(requestUrl)
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Delete_Project%3FTocPath%3DAPI%2520Reference%7C_____17
func (api *API) DeleteSite(siteId string) error {
	requestUrl := fmt.Sprintf("%s/api/%s/sites/%s", api.Server, api.Version, siteId)
	return api.delete(requestUrl)
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Delete_Site%3FTocPath%3DAPI%2520Reference%7C_____19
func (api *API) DeleteSiteByName(name string) error {
	return api.deleteSiteByKey("name", name)
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Delete_Site%3FTocPath%3DAPI%2520Reference%7C_____19
func (api *API) DeleteSiteByContentUrl(contentUrl string) error {
	return api.deleteSiteByKey("contentUrl", contentUrl)
}

// http://onlinehelp.tableau.com/current/api/rest_api/en-us/help.htm#REST/rest_api_ref.htm#Delete_Site%3FTocPath%3DAPI%2520Reference%7C_____19
func (api *API) deleteSiteByKey(key string, value string) error {
	requestUrl := fmt.Sprintf("%s/api/%s/sites/%s?key=%s", api.Server, api.Version, value, key)
	return api.delete(requestUrl)
}

func (api *API) delete(requestUrl string) error {
	headers := make(map[string]string)
	return api.makeRequest(requestUrl, DELETE, nil, nil, headers)
}

func (api *API) makeRequest(requestUrl string, method string, payload []byte, result interface{}, headers map[string]string) error {
	_, err := api.makeRequestGetBody(requestUrl, method, payload, result, headers)
	return err
}

//nolint:gocognit // TODO: refactor to smaller functions
func (api *API) makeRequestGetBody(requestUrl string, method string, payload []byte, result interface{}, headers map[string]string) ([]byte, error) {
	if api.Debug {
		fmt.Printf("%s:%v\n", method, requestUrl)
		if payload != nil {
			fmt.Printf("%v\n", string(payload))
		}
	}

	client := NewTimeoutClient(api.ConnectTimeout, api.ReadTimeout, true)
	var req *http.Request
	if len(payload) > 0 {
		var httpErr error
		req, httpErr = http.NewRequest(strings.TrimSpace(method), strings.TrimSpace(requestUrl), bytes.NewBuffer(payload))
		if httpErr != nil {
			return nil, httpErr
		}
		req.Header.Add(contentLengthHeader, strconv.Itoa(len(payload)))
	} else {
		var httpErr error
		req, httpErr = http.NewRequest(strings.TrimSpace(method), strings.TrimSpace(requestUrl), nil)
		if httpErr != nil {
			return nil, httpErr
		}
	}

	for header, headerValue := range headers {
		req.Header.Add(header, headerValue)
	}

	if len(api.AuthToken) > 0 {
		if api.Debug {
			fmt.Printf("%s:%s\n", authHeader, api.AuthToken)
		}
		req.Header.Add(authHeader, api.AuthToken)
	}

	var httpErr error
	resp, httpErr := client.Do(req)
	if httpErr != nil {
		return nil, httpErr
	}
	defer resp.Body.Close()
	body, readBodyError := ioutil.ReadAll(resp.Body)

	if api.Debug {
		fmt.Printf("t4g Response:%v\n", body)
	}

	if readBodyError != nil {
		return nil, readBodyError
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &StatusError{Code: http.StatusNotFound, Msg: "Resource not found", URL: requestUrl}
	}

	if resp.StatusCode >= http.StatusMultipleChoices {
		tErrorResponse := ErrorResponse{}
		err := xml.Unmarshal(body, &tErrorResponse)
		if err != nil {
			return body, err
		}
		return body, tErrorResponse.Error
	}
	if result != nil {
		// else unmarshall to the result type specified by caller
		err := xml.Unmarshal(body, &result)
		if err != nil {
			return body, err
		}
	}
	return body, nil
}
