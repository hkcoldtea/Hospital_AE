package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// Accident & Emergency Waiting Time
const APIURL = "http://www.ha.org.hk/opendata/aed/aedwtdata-%s.json"

// Structs for JSON decoding
type postItem struct {
	TopWait  string `json:"topWait"`
	HospName string `json:"hospName"`
}

type postsType struct {
	WaitTime   []postItem `json:"waitTime"`
	UpdateTime string     `json:"updateTime"`
}

var timezone = time.FixedZone("GMT", 8*3600)

var client = http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return nil
	},
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 10 * time.Second,
		}).DialContext,
	},
}

const UserAgent string = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.45 Safari/537.36"

var (
	BUILD string
)

func main() {
	var lang string
	var maxRetry int
	flag.IntVar(&maxRetry, "max", 2, "maximum retry")
	flag.StringVar(&lang, "lang", "tc", "Language. e.g.: en, sc, tc")
	flag.Usage = func() {
		w := flag.CommandLine.Output() // may be os.Stderr - but not necessarily
		if len(BUILD) > 0 {
			fmt.Fprintf(w, "Build: %s\n", BUILD)
		}
		fmt.Fprintf(w, "Usage of %s:\n\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(w, "\n")
	}
	flag.Parse()

	var err error
	lang, err = validlang(lang)
	if err != nil {
		fmt.Fprintln(os.Stderr, "incorrect parameter")
		return
	}

	var count int
	var resp *postsType
	var buf strings.Builder
	for count < maxRetry {
		count++
		resp, err = GetAE(lang)
		if err != nil {
			fmt.Fprintln(os.Stderr, "GetAE err: ", err)
			buf.WriteString(time.Now().In(timezone).Format("2006-01-02 15:04:05") + " 出現錯誤： " + err.Error() + "\n")
			continue
		}
		break
	}
	if err == nil {
		var formatstr string
		switch lang {
		case "en":
			fmt.Println("Accident and Emergency Waiting Time by Hospital")
			fmt.Printf("Last updated on:\t%s\n", resp.UpdateTime)
			formatstr = "%-44s\t%s\n"
		case "sc":
			fmt.Println("急症室等候时间")
			fmt.Printf("最后更新时间\t%s\n", resp.UpdateTime)
			formatstr = "%-20s\t%s\n"
		case "tc":
			fmt.Println("急症室等候時間")
			fmt.Printf("最後更新時間\t%s\n", resp.UpdateTime)
			formatstr = "%-20s\t%s\n"
		}
		for _, post := range resp.WaitTime {
			fmt.Printf(formatstr, post.HospName, post.TopWait)
		}
	} else {
		fmt.Fprintln(os.Stderr, buf.String())
	}
}

func validlang(lang string) (string, error) {
	switch lang {
	case "e":
		fallthrough
	case "en":
		return "en", nil
	case "s":
		fallthrough
	case "sc":
		return "sc", nil
	case "t":
		fallthrough
	case "tc":
		return "tc", nil
	}
	return "", errors.New("Invalid")
}

func GetAE(lang string) (*postsType, error) {
	apiURL := fmt.Sprintf(APIURL, lang)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", UserAgent)
	req.Header.Add("accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	// To compare status codes
	if resp.StatusCode != http.StatusOK {
		log.Fatal("Request was not OK: " + resp.Status)
	}

	var pt postsType
	err = json.NewDecoder(resp.Body).Decode(&pt)
	if err != nil {
		fmt.Println("error:", err)
		return &pt, err
	}

	return &pt, nil
}
