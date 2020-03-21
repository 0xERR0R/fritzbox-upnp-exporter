package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"crypto/md5"
	"crypto/rand"
	"encoding/hex"

	//digestRequest "github.com/delphinus/go-digest-request"

	log "github.com/sirupsen/logrus"
	//	dac "github.com/xinsnake/go-http-digest-auth-client"
)

var dummy = http.Response{
	Body: ioutil.NopCloser(bytes.NewBufferString("")),
}

func newRequest(method, uri, body string) *http.Request {
	request, _ := http.NewRequest(method, uri, strings.NewReader(body))
	//request.SetBasicAuth("admin", "!jryN18VD!")

	return request
}

func do(dr *http.Request) io.ReadCloser {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	//	dr.SetBasicAuth("admin", "!jryN18VD!")
	//t := digest.NewTransport("admin", "!jryN18VD!")

	//	resp, err := t.RoundTrip(dr)

	resp := digestPost(dr)
	var err error

	//resp, err := http.DefaultClient.Do(dr)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			log.Warn(fmt.Sprintf("Timeout on calling URL %s", dr.URL))
			return dummy.Body
		} else {
			log.Fatalln(err)
		}
	}
	if resp.StatusCode != http.StatusOK {
		log.Warn(fmt.Sprintf("Failed to call URL %s - status code was %d", dr.URL, resp.StatusCode))
		return dummy.Body
	}
	return resp.Body
}

func digestPost(req *http.Request) *http.Response {
	//   url := host + uri
	//   method := "POST"
	//   req, err := http.NewRequest(method, url, nil)
	//   req.Header.Set("Content-Type", "application/json")

	//	fmt.Printf("request 1 %s", formatRequest(req))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		log.Printf("Recieved status code '%v' auth skipped", resp.StatusCode)
		return resp
	}

	// Authorization: Digest username="admin", realm="HTTPS Access", nonce="BF7D02122AC0FBDF", uri="/upnp/control/lanconfigsecurity", response="31b4b07ab071959270b6870ddad66cd3", qop=auth, nc=00000003, cnonce="f4f08e537623900e", algorithm=MD5
	defer resp.Body.Close()
	digestParts := digestParts(resp)
	digestParts["uri"] = req.URL.Path
	digestParts["method"] = req.Method
	//digestParts["realm"] = "F!Box SOAP-Auth"
	digestParts["username"] = "admin"
	digestParts["password"] = "!jryN18VD!"
	body, _ := req.GetBody()
	l := req.ContentLength
	action := req.Header.Get("soapaction")
	req, err = http.NewRequest(req.Method, req.URL.String(), body)
	req.ContentLength = l

	req.Header.Set("Authorization", getDigestAuthrization(digestParts))
	req.Header.Set("Content-Type", "text/xml")
	req.Header.Set("Charset", "utf-8")
	req.Header.Set("soapaction", action)

	resp, err = client.Do(req)
	if err != nil {
		panic(err)
	}

	//	htmlData, err := ioutil.ReadAll(resp.Body)

	//	fmt.Printf("request 2 %s", formatRequest(req))
	//	fmt.Printf("reqponse 2 %s", htmlData)

	//   defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		log.Println("response body: ", string(body))
		return resp
	}
	return resp
}

func formatRequest(r *http.Request) string {
	// Create return string
	var request []string
	// Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}
	// Return the request as a string
	return strings.Join(request, "\n")
}

func digestParts(resp *http.Response) map[string]string {
	result := map[string]string{}
	if len(resp.Header["Www-Authenticate"]) > 0 {
		wantedHeaders := []string{"nonce", "realm", "qop"}
		responseHeaders := strings.Split(resp.Header["Www-Authenticate"][0], ",")
		for _, r := range responseHeaders {
			for _, w := range wantedHeaders {
				if strings.Contains(r, w) {
					result[w] = strings.Split(r, `"`)[1]
				}
			}
		}
	}
	return result
}

func getMD5(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func getCnonce() string {
	b := make([]byte, 8)
	io.ReadFull(rand.Reader, b)
	return fmt.Sprintf("%x", b)[:16]
}

func getDigestAuthrization(digestParts map[string]string) string {
	d := digestParts
	ha1 := getMD5(d["username"] + ":" + d["realm"] + ":" + d["password"])
	ha2 := getMD5(d["method"] + ":" + d["uri"])
	nonceCount := 00000000
	cnonce := getCnonce()
	// Authorization: Digest username="admin", realm="HTTPS Access", nonce="BF7D02122AC0FBDF", uri="/upnp/control/lanconfigsecurity", response="31b4b07ab071959270b6870ddad66cd3", qop=auth, nc=00000003, cnonce="f4f08e537623900e", algorithm=MD5
	response := getMD5(fmt.Sprintf("%s:%s:%v:%s:%s:%s", ha1, d["nonce"], nonceCount, cnonce, d["qop"], ha2))
	authorization := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", response="%s", qop="%s", nc="%v", cnonce="%s", algorithm=MD5`,
		d["username"], d["realm"], d["nonce"], d["uri"], response, d["qop"], nonceCount, cnonce)
	return authorization
}
