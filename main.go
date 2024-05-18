package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/miekg/dns"
)

func main() {
	listenAddr := flag.String("addr", "localhost:53", "Listen address")
	dohserver := flag.String("dohserver", "mozilla.cloudflare-dns.com", "DNS Over HTTPS server address")
	proxy := flag.String("proxy", "", "Http proxy")
	timeout := flag.Duration("timeout", 10*time.Second, "timeout (default 10s)")
	debug := flag.Bool("debug", false, "print debug logs")
	flag.Parse()

	if *debug {
		log.SetFlags(log.Lshortfile)
	} else {
		log.SetFlags(0)
	}

	var httpclient *http.Client
	if *proxy != "" {
		proxyUrl, err := url.Parse(*proxy)
		if err != nil {
			log.Fatalln("bad proxy url")
		}
		httpclient = &http.Client{
			Transport: &http.Transport{
				Proxy:           http.ProxyURL(proxyUrl),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	} else {
		httpclient = http.DefaultClient
	}

	dns.HandleFunc(".", func(w dns.ResponseWriter, req *dns.Msg) {
		msg := new(dns.Msg)
		msg.SetReply(req)
		msg.Authoritative = true
		for _, question := range req.Question {

			log.Println(question.Name)

			ctx, cancel := context.WithTimeout(context.Background(), *timeout)
			defer cancel()

			var b64 []byte

			origID := req.Id

			// Set DNS ID as zero accoreding to RFC8484 (cache friendly)
			req.Id = 0
			buf, err := req.Pack()
			b64 = make([]byte, base64.RawURLEncoding.EncodedLen(len(buf)))
			if err != nil {
				log.Println(err)
				continue
			}
			base64.RawURLEncoding.Encode(b64, buf)

			// No need to use hreq.URL.Query()
			hreq, _ := http.NewRequestWithContext(ctx, "GET", "https://"+*dohserver+"/dns-query?dns="+string(b64), nil)
			hreq.Header.Add("Accept", "application/dns-message")
			resp, err := httpclient.Do(hreq)
			if err != nil {
				log.Println(err)
				continue
			}
			defer resp.Body.Close()

			content, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
				continue
			}
			if resp.StatusCode != http.StatusOK {
				log.Println(errors.New("DoH query failed: " + string(content)))
				continue
			}

			r := new(dns.Msg)
			err = r.Unpack(content)
			if err != nil {
				log.Println(err)
				continue
			}
			r.Id = origID

			// log.Println(r.Answer)

			msg.Answer = append(msg.Answer, r.Answer...)
		}

		err := w.WriteMsg(msg)
		if err != nil {
			log.Println("Failed to write message:", err)
		}
	})

	if err := (&dns.Server{Addr: *listenAddr, Net: "udp"}).ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}
