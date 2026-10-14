package main

import (
	"net/url"
	"fmt"
	"log"
	"math"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/tavis7/postalboard/internal/htmlgen"
)

func parseAccept(acceptHeader []string) ([]struct {
	mimetype string
	params   map[string]string
	q        float32
}, error) {
	accepted := make([]struct {
		mimetype string
		params   map[string]string
		q        float32
	}, 0, 8)
	var acceptHeaderList []string
	for _, a := range acceptHeader {
		acceptHeaderList = append(acceptHeaderList, strings.Split(a, ",")...)
	}
	for _, t := range acceptHeaderList {
		accepting, params, err := mime.ParseMediaType(t)
		if err != nil {
			return accepted, err
		}
		qString, ok := params["q"]
		if !ok {
			qString = "1"
		}
		q, err := strconv.ParseFloat(qString, 32)
		if err != nil {
			q = 1
		}
		q = math.Max(0.0, math.Min(1.0, q))
		accepted = append(accepted, struct {
			mimetype string
			params   map[string]string
			q        float32
		}{
			mimetype: strings.ToLower(accepting),
			params:   params,
			q:        float32(q),
		})
	}
	return accepted, nil
}

func generateDebugAsHTML(r *http.Request) (*htmlgen.Node, error) {
	result := htmlgen.MakeLeafNode("div")
	acceptedMimeTypes, err := parseAccept(r.Header.Values("Accept"))
	if err != nil {
		log.Printf("Error parsing mimetypes: %v", err)
		result.AppendChildren(
			htmlgen.MakeNode("p", nil,
				htmlgen.MakeTextNode(fmt.Sprintf("Couldn't parse mimetypes: %v", err))))
	}

	result.AppendChildren(htmlgen.MakeNode("p", nil,
		htmlgen.MakeTextNode(fmt.Sprintf("protocol: %q", r.Proto))))
	result.AppendChildren(htmlgen.MakeNode("p", nil,
		htmlgen.MakeTextNode(fmt.Sprintf("Request host: %q", r.Host))))
	result.AppendChildren(htmlgen.MakeNode("p", nil,
		htmlgen.MakeTextNode(fmt.Sprintf("Request URL host: %q", r.URL.Host))))
	result.AppendChildren(htmlgen.MakeNode("p", nil,
		htmlgen.MakeTextNode("headers:")))
	headerListNode := htmlgen.MakeLeafNode("ul")
	result.AppendChildren(headerListNode)
	for header, val := range r.Header {
		for _, s := range val {
			headerListNode.AppendChildren(
				htmlgen.MakeNode("li", nil,
					htmlgen.MakeNode("code", nil,
						htmlgen.MakeTextNode(fmt.Sprintf("%v: %v", header, s)))))
		}
	}

	result.AppendChildren(htmlgen.MakeNode("p", nil,
		htmlgen.MakeTextNode(fmt.Sprintf("path: %v", r.URL.Path))))
	result.AppendChildren(
		htmlgen.MakeNode("p", nil,
			htmlgen.MakeTextNode("Raw query: "),
			htmlgen.MakeNode("code", nil,
				htmlgen.MakeTextNode(r.URL.RawQuery))))
	result.AppendChildren(htmlgen.MakeNode("p", nil,
		htmlgen.MakeTextNode("query:")))

	queryListNode := htmlgen.MakeNode("ul", nil)
	result.AppendChildren(queryListNode)

	for k, v := range r.URL.Query() {
		queryListNode.AppendChildren(
			htmlgen.MakeNode("li", nil,
				htmlgen.MakeNode("code", nil,
					htmlgen.MakeTextNode(fmt.Sprintf("%v=%v", k, v)))))
	}

	result.AppendChildren(htmlgen.MakeNode("p", nil,
		htmlgen.MakeTextNode("accepted mimetypes:")))

	mimeListNode := htmlgen.MakeNode("ul", nil)
	result.AppendChildren(mimeListNode)

	found := -1
	for i, val := range acceptedMimeTypes {
		mimeListNode.AppendChildren(
			htmlgen.MakeNode("li", nil,
				htmlgen.MakeTextNode(fmt.Sprintf("%v", val))))
		if found == -1 {
			if val.mimetype == "text/html" ||
				val.mimetype == "text/*" ||
				val.mimetype == "*/*" {
				found = i
			}
		}
	}

	result.AppendChildren(htmlgen.MakeNode("p", nil,
		htmlgen.MakeTextNode("index: "),
		htmlgen.MakeTextNode(fmt.Sprintf("%v", found))))
	if found != -1 {
		result.AppendChildren(htmlgen.MakeNode("p", nil,
			htmlgen.MakeTextNode("Found: "),
			htmlgen.MakeTextNode(fmt.Sprintf("%v", acceptedMimeTypes[found]))))
	} else {
		result.AppendChildren(htmlgen.MakeNode("p", nil,
			htmlgen.MakeTextNode("does not contain text/html")))
		// 406 Not Acceptable
		return result, fmt.Errorf("Accept header does not contain text/html")
	}

	cookies := parseCookies(r.Header.Values("cookie"))
	result.AppendChildren(htmlgen.MakeNode("p", nil,
		htmlgen.MakeTextNode("Cookies:")))
	cookieListNode := htmlgen.MakeLeafNode("ul")
	for k, v := range cookies {
		cookieListNode.AppendChildren(htmlgen.MakeNode("li", nil,
			htmlgen.MakeTextNode(k),
			htmlgen.MakeTextNode(": "),
			htmlgen.MakeTextNode(v)))
	}
	result.AppendChildren(cookieListNode)

	return result, nil
}

func getPageHeader(pageTitle, user string) *htmlgen.Node {
	result := htmlgen.MakeNode("div", htmlgen.AttribList{{"id", "page-header"}})

	result.AppendChildren(htmlgen.MakeNode("h1", nil, htmlgen.MakeTextNode("PostalBoard")))

	result.AppendChildren(htmlgen.MakeNode("ul", nil, htmlgen.MakeNode("li", nil,
		htmlgen.MakeNode("a",
			htmlgen.AttribList{{"href", "/"}},
			htmlgen.MakeTextNode("Root"))),
		htmlgen.MakeNode("li", nil,
			htmlgen.MakeNode("a",
				htmlgen.AttribList{{"href", "/debug"}},
				htmlgen.MakeTextNode("Debug"))),
		htmlgen.MakeNode("li", nil,
			htmlgen.MakeNode("a",
				htmlgen.AttribList{{"href", "/app"}},
				htmlgen.MakeTextNode("App"))),
		htmlgen.MakeNode("li", nil,
			htmlgen.MakeNode("a",
				htmlgen.AttribList{{"href", "/app/boards"}},
				htmlgen.MakeTextNode("Boards"))),
		htmlgen.MakeNode("li", nil,
			htmlgen.MakeNode("a",
				htmlgen.AttribList{{"href", "/app/login"}},
				htmlgen.MakeTextNode("Login"))),
		htmlgen.MakeNode("li", nil,
			htmlgen.MakeNode("a",
				htmlgen.AttribList{{"href", "/app/logout"}},
				htmlgen.MakeTextNode("Logout"))),
	))
	if len(user) > 0 {
		result.AppendChildren(htmlgen.MakeNode("p", nil,
			htmlgen.MakeTextNode("Logged in as "),
			htmlgen.MakeTextNode(user)))
	}

	result.AppendChildren(htmlgen.MakeNode("h1", nil,
		htmlgen.MakeTextNode(pageTitle)))

	return result
}

func parseCookies(cookieHeader []string) map[string]string {
	log.Printf("Parsing cookies: %v", cookieHeader)
	result := map[string]string{}
	for _, h := range cookieHeader {
		for _, v := range strings.Split(h, ";") {
			cookie := strings.SplitN(v, "=", 2)
			if len(cookie) != 2 {
				log.Printf("Malformed cookie: %v", v)
				continue
			}
			key := strings.Trim(cookie[0], " ")
			val := strings.Trim(cookie[1], " ")
			_, ok := result[key]
			if ok {
				log.Printf("Multiple values for cookie: '%v': '%v' -> '%v'",
					key, result[key], val)
			}
			result[key] = val
		}
	}
	return result
}

func getRedirect(r *http.Request) string {
	refererHeaders := r.Header.Values("referer")
	redirect := ""
	if len(refererHeaders) > 0 {
		referer := refererHeaders[0]
		refererURL, err := url.Parse(referer)
		if err != nil {
			log.Printf("Error parsing referer: %v", referer)
		} else {
			// @todo Configuration whitelist
			if r.Host == refererURL.Host {
				redirect = url.QueryEscape(refererURL.Path)
			} else {
				log.Printf("Not redirecting to referer: %v is not at %v", referer, r.Host)
			}
		}
	}

	q := r.URL.Query()
	redirects, ok := q["redirect"]
	log.Printf("login redirect: '%v'", redirects)
	if ok {
		if len(redirects) != 1 {
			// @todo Include URL and other information in log outputs
			log.Printf("Warning: Too many redirect parameters: %v", redirects)
		} else {
			redirect = url.QueryEscape(redirects[0])
		}
	}
	return redirect
}
