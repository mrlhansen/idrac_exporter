package collector

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"path"
	"strings"
	"time"

	"github.com/mrlhansen/idrac_exporter/v2/internal/config"
	"github.com/mrlhansen/idrac_exporter/v2/internal/log"
)

type Redfish struct {
	http     *http.Client
	baseurl  string
	hostname string
	username string
	password string
	session  struct {
		disabled bool
		id       string
		token    string
	}
}

const redfishRootPath = "/redfish/v1"

func NewRedfish(scheme, hostname, username, password string) *Redfish {
	return &Redfish{
		baseurl:  fmt.Sprintf("%s://%s", scheme, hostname),
		hostname: hostname,
		username: username,
		password: password,
		http: &http.Client{
			Transport: &http.Transport{
				Proxy:           http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: time.Duration(config.Config.Timeout) * time.Second,
		},
	}
}

func (r *Redfish) CreateSession() bool {
	url := fmt.Sprintf("%s/redfish/v1/SessionService/Sessions", r.baseurl)
	session := Session{
		Username: r.username,
		Password: r.password,
	}
	body, _ := json.Marshal(&session)

	resp, err := r.http.Post(url, "application/json", bytes.NewBuffer(body))
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()
	if err != nil {
		log.Error("Failed to query %q: %v", url, err)
		return false
	}

	// iDRAC 8
	// https://dl.dell.com/topicspdf/idrac9-lifecycle-controller-v4x-series_api-guide_en-us.pdf
	// mentions that old URL for session management was /redfish/v1/Sessions and
	// the new URL is /redfish/v1/SessionService/Sessions which implies earlier iDRAC
	// versions used the former.
	if resp.StatusCode == http.StatusMethodNotAllowed {
		if resp != nil {
			resp.Body.Close()
		}

		url = fmt.Sprintf("%s/redfish/v1/Sessions", r.baseurl)
		resp, err = r.http.Post(url, "application/json", bytes.NewBuffer(body))
		if err != nil {
			r.session.disabled = true
			return false
		}
	}

	if resp.StatusCode != http.StatusCreated {
		log.Error("Unexpected status code from %q: %s", url, resp.Status)
		return false
	}

	err = json.NewDecoder(resp.Body).Decode(&session)
	if err != nil {
		log.Error("Error decoding response from %q: %v", url, err)
		return false
	}

	r.session.id = session.OdataId
	r.session.token = resp.Header.Get("X-Auth-Token")

	// iLO 4
	if len(r.session.id) == 0 {
		u, err := neturl.Parse(resp.Header.Get("Location"))
		if err == nil {
			r.session.id = u.Path
		}
	}

	log.Debug("Succesfully created session: %s", path.Base(r.session.id))
	return true
}

func (r *Redfish) DeleteSession() bool {
	if len(r.session.token) == 0 {
		return true
	}

	url := fmt.Sprintf("%s%s", r.baseurl, r.session.id)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return false
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Set("X-Auth-Token", r.session.token)

	resp, err := r.http.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		log.Error("Failed to query %q: %v", url, err)
		return false
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		log.Error("Unexpected status code from %q: %s", url, resp.Status)
		return false
	}

	log.Debug("Succesfully deleted session: %s", path.Base(r.session.id))
	r.session.id = ""
	r.session.token = ""

	return true
}

func (r *Redfish) RefreshSession() bool {
	if r.session.disabled {
		return false
	}

	defer func() {
		if r.session.disabled {
			log.Info("Session authentication disabled for %s due to failed refresh", r.hostname)
		}
	}()

	if len(r.session.token) == 0 {
		ok := r.CreateSession()
		if !ok {
			r.session.disabled = true
		}
		return ok
	}

	url := fmt.Sprintf("%s%s", r.baseurl, r.session.id)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Set("X-Auth-Token", r.session.token)

	resp, err := r.http.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return false
	}

	if resp.StatusCode == http.StatusUnauthorized {
		if r.CreateSession() {
			return true
		} else {
			r.session.disabled = true
			r.session.token = ""
			r.session.id = ""
			return false
		}
	}

	return true
}

func (r *Redfish) Get(path string, res any) bool {
	if !strings.HasPrefix(path, redfishRootPath) {
		return false
	}

	url := fmt.Sprintf("%s%s", r.baseurl, path)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}

	req.Header.Add("Accept", "application/json")
	if len(r.session.token) > 0 {
		req.Header.Set("X-Auth-Token", r.session.token)
	} else {
		req.SetBasicAuth(r.username, r.password)
	}

	log.Debug("Querying %q", url)
	resp, err := r.http.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Error("Failed to query %q: %v", url, err)
		return false
	}

	if resp.StatusCode != http.StatusOK {
		log.Error("Unexpected status code from %q: %s", url, resp.Status)
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("Error reading response from %q: %v", url, err)
		return false
	}

	if config.Debug {
		log.Debug("Response from %q: %s", url, body)
	}

	err = json.Unmarshal(body, res)
	if err != nil {
		log.Error("Error decoding response from %q: %v", url, err)
		return false
	}

	return true
}

func (r *Redfish) Exists(path string) bool {
	if !strings.HasPrefix(path, redfishRootPath) {
		return false
	}

	url := fmt.Sprintf("%s%s", r.baseurl, path)
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false
	}

	req.Header.Add("Accept", "application/json")
	if len(r.session.token) > 0 {
		req.Header.Set("X-Auth-Token", r.session.token)
	} else {
		req.SetBasicAuth(r.username, r.password)
	}

	resp, err := r.http.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	if err != nil {
		return false
	}

	if resp.StatusCode >= 400 && resp.StatusCode <= 499 {
		return false
	}

	return true
}
