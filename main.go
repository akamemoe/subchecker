package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
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

func (v VServer) String() string {
	return fmt.Sprintf("{Add:%s Port:%d Ps:%s Path:%s Class:%d}",
		v.Add, v.Port, v.Ps, v.Path, v.Class)
}

type vsList []VServer

func (v vsList) Len() int { return len(v) }
func (v vsList) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}
func (v vsList) Less(i, j int) bool {
	if v[i].Class != v[j].Class {
		return v[i].Class > v[j].Class
	} else {
		return strings.Compare(v[i].Add, v[j].Add) < 0
	}
}

var (
	timeout time.Duration
)

func main() {
	var urlStr string
	var fpath string
	var outFile string
	var verbose bool
	var concurrency int

	flag.StringVar(&urlStr, "u", "", "the subscription url")
	flag.StringVar(&fpath, "f", "", "the subscription file")
	flag.StringVar(&outFile, "o", "", "also output to file")
	flag.BoolVar(&verbose, "v", false, "verbose mode, print node detail")
	flag.IntVar(&concurrency, "c", 1, "concurrency for per server")
	flag.DurationVar(&timeout, "t", time.Duration(2)*time.Second, "timeout of tcp connection")
	flag.Parse()
	log.SetFlags(0)
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
		body, _ := ioutil.ReadAll(resp.Body)
		content = string(body)
	} else {
		log.Fatalln("please at least specify one of the u or f")
	}
	vss := parse(content)

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
	sort.Sort(vsList(vss))
	var s string

	queues := make([]chan VServer, concurrency)
	ctx := context.Background()
	for i := 0; i < concurrency; i++ {
		go worker(ctx, queues[i])
	}
	for _, vs := range vss {
		if doPing(vs) {
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
			_, _ = f.WriteString(s)
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

func doPing(vs VServer, queues []chan VServer) {
	for i := 0; i < len(queues); i++ {
		queues[i] <- vs
	}

}

//向三个worker协程中发送任务，等待三个协程都完成，超过半数ping通则算ok
func worker(ctx context.Context, queue <-chan VServer) bool {
	select {
	case <-ctx.Done():
		return true
	case v := <-queue:
		return tcpPing(v)
	}
}

func tcpPing(vs VServer) bool {
	addr := fmt.Sprintf("%s:%d", vs.Add, vs.Port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true

}
