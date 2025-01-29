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

	"github.com/mrlhansen/idrac_exporter/internal/config"
	"github.com/mrlhansen/idrac_exporter/internal/log"
)

type Redfish struct {
	http     *http.Client
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

func NewRedfish(hostname, username, password string) *Redfish {
	return &Redfish{
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
	url := fmt.Sprintf("https://%s/redfish/v1/SessionService/Sessions", r.hostname)
	session := Session{
		Username: r.username,
		Password: r.password,
	}
	body, _ := json.Marshal(&session)

	resp, err := r.http.Post(url, "application/json", bytes.NewBuffer(body))
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Error("Failed to query %q: %v", url, err)
		return false
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

	url := fmt.Sprintf("https://%s%s", r.hostname, r.session.id)
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

	if len(r.session.token) == 0 {
		ok := r.CreateSession()
		if !ok {
			r.session.disabled = true
		}
		return ok
	}

	url := fmt.Sprintf("https://%s%s", r.hostname, r.session.id)
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

	url := fmt.Sprintf("https://%s%s", r.hostname, path)
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

	url := fmt.Sprintf("https://%s%s", r.hostname, path)
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

	if resp.StatusCode == http.StatusNotFound {
		return false
	}

	return true
}
