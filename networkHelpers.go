package irma

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cretz/bine/control"
	"github.com/cretz/bine/tor"
	"github.com/ipsn/go-libtor"
	"golang.org/x/net/html"
)

// private functions

func getTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" {
		var title bytes.Buffer
		if err := html.Render(&title, n.FirstChild); err != nil {
			panic(err)
		}
		return strings.TrimSpace(title.String())
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := getTitle(c); title != "" {
			return title
		}
	}
	return ""
}

// public functions

func MakeTorHttpClient(dataDir string) (*tor.Tor, func(), *http.Client) {
	// Start tor with some defaults + elevated verbosity
	fmt.Println("Starting, please wait a bit...")
	t, err := tor.Start(nil, &tor.StartConf{ProcessCreator: libtor.Creator, DebugWriter: os.Stderr, DataDir: dataDir})
	if err != nil {
		log.Panicf("Failed to start tor: %v", err)
	}
	// defer t.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	// defer cancel()

	// Make connection
	dialer, err := t.Dialer(ctx, nil)
	if err != nil {
		log.Panicf("err", err)
	}

	return t, cancel, &http.Client{Transport: &http.Transport{DialContext: dialer.DialContext}}
}

func IsClientConnectedToTor(httpClient *http.Client) bool {
	resp, err := httpClient.Get("https://check.torproject.org")
	if err != nil {
		log.Fatal("err", err)
	}
	defer resp.Body.Close()

	parsed, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatal("err", err)
	}

	if strings.Contains(getTitle(parsed), "Congratulations.") {
		return true
	} else {
		return false
	}
}

func RenewTorCircuit(tor *tor.Tor, cancel func()) (func(), *http.Client) {
	tor.Control.SetConf(control.KeyVals("DisableNetwork", "1")...)

	cancel()

	ctx := context.Background()
	tor.EnableNetwork(ctx, false)
	ctx, cancel = context.WithTimeout(ctx, 3*time.Minute)

	// Make connection // TODO: copied code
	dialer, err := tor.Dialer(ctx, nil)
	if err != nil {
		log.Panicf("err", err)
	}

	return cancel, &http.Client{Transport: &http.Transport{DialContext: dialer.DialContext}}
}

func PrintExternalIpClient(httpClient *http.Client) {
	resp, err := httpClient.Get("https://api.ipify.org?format=text")
	if err != nil {
		log.Panicf("err", err)
	}
	defer resp.Body.Close()
	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Panicf("err", err)
	}
	fmt.Printf("External IP is: %s\n", ip)
}
