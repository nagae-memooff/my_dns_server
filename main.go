package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/miekg/dns"
	"github.com/nagae-memooff/config"
)

var (
	lock sync.RWMutex

	records map[string]string
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

	for k, v := range new_rec {
		records[k] = v
	}

	w.Write([]byte("ok"))
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
