package main

import (
	"bytes"
	"encoding/json"
	utils "github.com/nagae-memooff/goutils"
	// "errors"
	// "fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"time"
)

var (
	regex    *regexp.Regexp
	last_ip  string
	interval = time.Minute * 1

	registe_url = "http://nagae-memooff.me/dns/update"

//   registe_url = "http://localhost:8081/update"
)

func main() {
	regex = regexp.MustCompile(`[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}`)

	for {
		RegisteToServer("")
		time.Sleep(interval)
	}

}

// func GetIP() (string, error) {
// 	client := &http.Client{}
//
// 	req, err := http.NewRequest("GET", "http://ip.cn", nil)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	req.Header.Set("User-Agent", "curl/7.47.0")
//
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	defer resp.Body.Close()
//
// 	body, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		return "", err
// 	}
//
// 	if !regex.Match(body) {
// 		return "", errors.New(fmt.Sprintf("can't find ip. resp body: %s.", body))
// 	}
//
// 	iplist := regex.FindAllString(string(body), 1)
//
// 	return iplist[0], nil
// }

// func CompareIP() {
// 	new_ip, err := GetIP()
// 	if err != nil {
// 		log.Println(err)
// 	}
//
// 	if new_ip == last_ip {
// 		// log.Printf("IP 相等： %s", new_ip)
// 	} else {
// 		log.Printf("IP changed: %s -> %s.", last_ip, new_ip)
// 		RegisteToServer(new_ip)
//
// 		last_ip = new_ip
// 	}
// }

func RegisteToServer(ip string) {
	data := map[string]string{
		"files.nagae-memooff.me": ip,
	}

	data_json, _ := json.Marshal(data)

	resp, err := http.Post(registe_url, "", bytes.NewReader(data_json))
	if err != nil {
		log.Println(err)
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}

	now_ip := string(body)

	if now_ip != last_ip {
		// ip变化，注册iptables规则
		log.Printf("ip变化，注册iptables规则。")
		utils.Sysexec("/home/nagae-memooff/dns/ext_ip_nat.sh", now_ip)
	}

	last_ip = now_ip

	log.Printf("report ip %s.", last_ip)
}
