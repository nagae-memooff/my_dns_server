package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/nagae-memooff/config"
	"github.com/nagae-memooff/goutils"
)

var (
	lock sync.RWMutex

	records      map[string]string
	cached_token CachedToken
)

func parse_query(m *dns.Msg) {
	for _, q := range m.Question {
		switch q.Qtype {
		case dns.TypeA:
			log.Printf("Query for %s\n", q.Name)

			lock.RLock()
			ip := records[q.Name[:len(q.Name)-1]]
			lock.RUnlock()

			if ip != "" {
				rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
				rr.Header().Ttl = 120
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		}
	}
}

func handle_dns_request(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		parse_query(m)
	default:
		log.Printf("%v", r)
	}

	w.WriteMsg(m)
}

func update_dns(w http.ResponseWriter, req *http.Request) {
	// body := []byte(`{"nagae-memooff.me": "192.168.10.10"}`)
	if req.Method != "POST" {
		w.Write([]byte("err"))
		return
	}

	body, err := ioutil.ReadAll(req.Body)
	defer req.Body.Close()
	var new_rec map[string]string

	err = json.Unmarshal(body, &new_rec)
	if err != nil {
		w.Write([]byte("err"))
		return
	}

	lock.Lock()
	defer lock.Unlock()

	old_record := records["files.nagae-memooff.me"]

	if new_rec["files.nagae-memooff.me"] == "" {
		new_rec["files.nagae-memooff.me"] = req.Header.Get("X-Real-IP")
	}

	new_record := new_rec["files.nagae-memooff.me"]

	if new_record != old_record {
		// 如果不相等则触发更新ddns逻辑
		log.Printf("ip地址变更： %s → %s", old_record, new_record)
		update_ddns(new_record)
	}

	for k, v := range new_rec {
		records[k] = v
	}

	w.Write([]byte(req.Header.Get("X-Real-IP")))
}

func replace_dns(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		return
	}

	j := []byte(req.PostFormValue("data"))

	var new_rec map[string]string

	err := json.Unmarshal(j, &new_rec)
	if err != nil {
		w.Write([]byte("err"))
		return
	}

	lock.Lock()
	defer lock.Unlock()

	records = new_rec
	w.Write([]byte("ok"))
}

func update_ddns(new_ip string) {
	access_token := GetAccessTokenFromCache()

	UpdateRecord(new_ip, access_token)
}

func main() {
	// attach request handler func

	err := config.Parse("./dns.conf")
	if err != nil {
		fmt.Println("FATAL ERROR: load config failed." + err.Error())
		os.Exit(1)
	}

	records = map[string]string{
		"files.nagae-memooff.me": config.Get("default_ip"),
		"nagae-memooff.me":       config.Get("default_ip"),
	}

	// start server
	dns.HandleFunc("nagae-memooff.me", handle_dns_request)
	listen := config.Get("dns_listen")

	server := &dns.Server{Addr: listen, Net: "udp"}
	log.Printf("Starting at %s\n", listen)

	http.HandleFunc("/update", update_dns)
	http.HandleFunc("/replace", replace_dns)

	go http.ListenAndServe(config.Get("http_listen"), nil)

	err = server.ListenAndServe()
	defer server.Shutdown()
	if err != nil {
		log.Fatalf("Failed to start server: %s\n ", err.Error())
	}
}

type TokenResponse struct {
	Access struct {
		Token struct {
			IssuedAt string `json:"issued_at"`
			Expires  string `json:"expires"`
			Id       string `json:"id"`
		} `json:"token"`
	} `json:"access"`
}

type TokenRequest struct {
	Auth struct {
		PasswordCredentials struct {
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"passwordCredentials"`

		TenantID string `json:"tenantId"`
	} `json:"auth"`
}

type CachedToken struct {
	ExpiresAt time.Time
	Id        string
}

func GetAccessTokenFromCache() (token string) {
	if cached_token.Id == "" || cached_token.ExpiresAt.Before(time.Now()) {
		token, _ = GetAccessToken()
	} else {
		token = cached_token.Id
	}

	return
}

func GetAccessToken() (token string, expires_at time.Time) {
	params := TokenRequest{}

	params.Auth.PasswordCredentials.Username = config.Get("conoha_username")
	params.Auth.PasswordCredentials.Password = config.Get("conoha_password")
	params.Auth.TenantID = config.Get("cohona_tenant_id")

	code, response, err := utils.DoHTTPJsonRequest("POST", "https://identity.tyo1.conoha.io/v2.0/tokens", params, nil, nil)

	if code != 200 || err != nil {
		// TODO
		log.Printf("获取token出错. code: %d, err: %v, response: %s", code, err, response)

	}

	token_response := TokenResponse{}
	err = json.Unmarshal([]byte(response), &token_response)
	if err != nil {
		// TODO
		log.Printf("解析json失败: %v", err)
	}

	token = token_response.Access.Token.Id
	expires_at = time.Now().Add(time.Hour * 23)

	cached_token = CachedToken{
		Id:        token,
		ExpiresAt: expires_at,
	}

	return
}

func UpdateRecord(new_ip, access_token string) {
	header := map[string]string{
		"X-Auth-Token": access_token,
	}

	body := map[string]string{
		"data": new_ip,
	}

	// 从配置读取
	domain_uuid := config.Get("conoha_domain_uuid")
	record_uuid := config.Get("conoha_record_uuid")

	url := fmt.Sprintf("https://dns-service.tyo1.conoha.io/v1/domains/%s/records/%s", domain_uuid, record_uuid)

	code, response, err := utils.DoHTTPJsonRequest("PUT", url, body, &header, nil)

	if code != 200 || err != nil {
		cached_token = CachedToken{}
		log.Printf("更新dns记录失败： code: %d, err: %v, response: %s", code, err, response)
	}
}
