package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type VServer struct {
	Host       string `json:"host"`
	Path       string `json:"path"`
	TLS        string `json:"tls"`
	VerifyCert bool   `json:"verify_cert"`
	Add        string `json:"add"`
	Port       int    `json:"port"`
	Aid        int    `json:"aid"`
	Net        string `json:"net"`
	HeaderType string `json:"headerType"`
	V          string `json:"v"`
	Type       string `json:"type"`
	Ps         string `json:"ps"`
	Remark     string `json:"remark"`
	ID         string `json:"id"`
	Class      int    `json:"class"`
}

func main() {
	var urlStr string
	var fpath string
	var outFile string
	var verbose bool
	var timeout time.Duration
	flag.StringVar(&urlStr, "u", "", "the subscription url")
	flag.StringVar(&fpath, "f", "", "the subscription file")
	flag.StringVar(&outFile, "o", "", "also output to file")
	flag.BoolVar(&verbose, "v", false, "verbose mode, print node detail")
	flag.DurationVar(&timeout, "t", time.Duration(2)*time.Second, "timeout of tcp connection")
	flag.Parse()
	var content string
	if fpath != "" {
		b, err := ioutil.ReadFile(fpath)
		if err != nil {
			log.Fatalln("can't read file:", fpath)
		}
		content = string(b)
	} else if urlStr != "" {
		client := http.Client{}
		req, _ := http.NewRequest("GET", urlStr, nil)
		req.Header.Set("User-Agent", "subchecker/1.0")
		resp, err := client.Do(req)
		if err != nil {
			log.Fatalln("can't read from url:", urlStr)
		}
		body, err := ioutil.ReadAll(resp.Body)
		content = string(body)
	} else {
		log.Fatalln("please at least specify one of the u or f")
	}
	vss := parse(content)
	var s string
	var f *os.File

	if outFile != "" {
		var err error
		f, err = os.Create(outFile)
		if err != nil {
			log.Fatalln("can't open file:", outFile)
		}

	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()

	var status string

	for _, vs := range vss {
		if tcpPing(vs, timeout) {
			status = "OK "
		} else {
			status = "ERR"
		}
		if verbose {
			s = fmt.Sprintf("%s - %+v\n", status, vs)
		} else {
			s = fmt.Sprintf("%s - %s:%d @ %s\n", status, vs.Add, vs.Port, vs.Remark)
		}
		log.Print(s)
		if f != nil {
			f.WriteString(s)
		}
	}

}

func parse(data string) (vss []VServer) {
	if !strings.HasPrefix(data, "vmess://") {
		b, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			log.Fatalln("data is not a valid base64 string:", data[:10])
		}
		data = string(b)
	}
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		b64s := strings.TrimSpace(line)
		if len(b64s) > 0 {
			b64s = strings.Replace(b64s, "vmess://", "", 1)
			bj, err := base64.StdEncoding.DecodeString(b64s)
			if err != nil {
				log.Fatalln("b64s is not a valid base64 string:", b64s)
			}
			vs := VServer{}
			if err = json.Unmarshal(bj, &vs); err != nil {
				log.Println("failed to unmarshal:", string(bj)[:10])
				continue
			}
			vss = append(vss, vs)

		}

	}
	return vss
}

func tcpPing(vs VServer, timeout time.Duration) bool {
	addr := fmt.Sprintf("%s:%d", vs.Add, vs.Port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true

}
