package main

import (
	"fmt"
	"html"
	"log"
	"math"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

func parseAccept(acceptHeader string) ([]struct {
	mimetype string
	params   map[string]string
	q        float32
}, error) {
	accepted := make([]struct {
		mimetype string
		params   map[string]string
		q        float32
	}, 0, 8)
	acceptHeaderList := strings.Split(acceptHeader, ",")
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

func generateDebugAsHTML(r *http.Request) (string, error) {
	w := &strings.Builder{}
	acceptedMimeTypes, err := parseAccept(r.Header.Get("Accept"))
	if err != nil {
		log.Printf("Error parsing mimetypes: %v", err)
		fmt.Fprintf(w, "<p>Couldn't parse mimetypes: %v</p>", err)
	}

	fmt.Fprintf(w, "<p>protocol: %q</p>\n", html.EscapeString(r.Proto))
	fmt.Fprintf(w, "<p>headers:</p>")
	fmt.Fprintf(w, "<ul>")
	for header, val := range r.Header {
		fmt.Fprintf(w, "<li><pre style=\"display:inline\">%v: %v</pre></li>\n",
			html.EscapeString(fmt.Sprintf("%v", header)),
			html.EscapeString(fmt.Sprintf("%v", val)))
	}
	fmt.Fprintf(w, "</ul>")
	fmt.Fprintf(w, "<p>path: %q</p>\n", html.EscapeString(r.URL.Path))
	fmt.Fprintf(w, "<p>raw query: <pre style=\"display:inline\">%q</pre></p>\n",
		html.EscapeString(r.URL.RawQuery))
	fmt.Fprintf(w, "<p>query:</p>\n")
	fmt.Fprintf(w, "<ul>\n")
	for k, v := range r.URL.Query() {
		fmt.Fprintf(w, "<li>\n")
		fmt.Fprintf(w, "<pre style=\"display:inline\">%v=%v</pre>",
			html.EscapeString(fmt.Sprintf("%v", k)),
			html.EscapeString(fmt.Sprintf("%v", v)))
		fmt.Fprintf(w, "</li>\n")
	}
	fmt.Fprintf(w, "</ul>\n")
	fmt.Fprintf(w, "<p>accepted mimetypes:</p>\n")
	fmt.Fprintf(w, "<ul>")
	found := -1
	for i, val := range acceptedMimeTypes {
		fmt.Fprintf(w, "<li>%v</li>\n", html.EscapeString(fmt.Sprintf("%v", val)))
		if found == -1 {
			if val.mimetype == "text/html" ||
				val.mimetype == "text/*" ||
				val.mimetype == "*/*" {
				found = i
			}
		}
	}
	fmt.Fprintf(w, "</ul>")
	fmt.Fprintf(w, "<p>index: %v</p>\n",
		html.EscapeString(fmt.Sprintf("%v", found)))
	if found != -1 {
		fmt.Fprintf(w, "<p>Found: %v</p>\n",
			html.EscapeString(fmt.Sprintf("%v", acceptedMimeTypes[found])))
	} else {
		fmt.Fprintf(w, "does not contain text/html\n")
		// 406 Not Acceptable
		return "", fmt.Errorf("Accept header does not contain text/html")
	}
	return w.String(), nil
}

func getDebugger(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "<html><head></head><body>\n")
	fmt.Fprintf(w, "<p>Get handler says \"Hello\"</p>\n")

	fmt.Fprintf(w, "<form method=\"post\">\n")
	fmt.Fprintf(w, "<div>\n")
	fmt.Fprintf(w, "<input type=\"text\" name=\"text_input\"/>\n")
	fmt.Fprintf(w, "</div>\n")
	fmt.Fprintf(w, "<div>\n")
	fmt.Fprintf(w, "<input type=\"text\" name=\"text_input\"/>\n")
	fmt.Fprintf(w, "</div>\n")
	fmt.Fprintf(w, "<div>\n")
	fmt.Fprintf(w, "<textarea name=\"text_area\">\n")
	fmt.Fprintf(w, "</textarea>\n")
	fmt.Fprintf(w, "</div>\n")
	fmt.Fprintf(w, "<input type=\"submit\" value=\"Submit\" />\n")
	fmt.Fprintf(w, "</form>\n")

	debugHTML, err := generateDebugAsHTML(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		fmt.Fprintf(w, "<div>")
		fmt.Fprintf(w, debugHTML)
		fmt.Fprintf(w, "</div>")
	}

	fmt.Fprintf(w, "</body></html>\n")
}

func httpGetBoard(w http.ResponseWriter, r *http.Request) {
	boardPath := strings.TrimSuffix(r.PathValue("board"), "/")

	posts, err := getMessages(boardPath)
	if err != nil {
		// @todo
		log.Printf("Failed getting messages from %v: %v", boardPath, err)
	}

	children, err := getChildrenBoards(boardPath)
	if err != nil {
		// @todo
		log.Printf("Failed getting children from %v: %v", boardPath, err)
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	htmlRoot := makeHTMLNode("html", nil)
	htmlHead := makeHTMLNode("head", nil)
	htmlBody := makeHTMLNode("body", nil)

	htmlRoot.appendChildren(htmlHead, htmlBody)

	htmlBoardList := makeHTMLNode("p", nil)
	for _, child := range children {
		htmlBoardList.appendChild(
			makeHTMLNode("p", nil).appendChild(
				makeHTMLNode("a", attribList{{"href", child}}).appendChild(
					makeHTMLTextNode(child))))
	}

	htmlBody.appendChild(makeHTMLNode("p", nil).appendChild(
		makeHTMLTextNode("Boards:"),
	))

	htmlBody.appendChild(htmlBoardList)

	if len(posts) > 0 {
		htmlBody.appendChild(makeHTMLNode("p", nil).appendChild(
			makeHTMLTextNode("Posts: "),
		))
		for _, post := range posts {
			htmlBody.appendChild(makeHTMLNode("p", nil).appendChild(
				makeHTMLTextNode(fmt.Sprintf("%v: %v", post.user, post.text)),
			))
		}
		htmlBody.appendChild(
			makeHTMLNode("form", attribList{{"method", "post"}}).appendChild(
				makeHTMLNode("textArea", attribList{{"name", "post"}})).appendChild(
				makeHTMLNode("div", nil).appendChild(
					makeHTMLNode("input",
						attribList{{"type", "submit"}, {"value", "post"}}))))
	}

	fmt.Fprintf(w, "%s", renderHTMLNode(htmlRoot))
}

func postDebugger(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Post handler says \"Hello\"\n")
	fmt.Fprintf(w, "path: %q\n", html.EscapeString(r.URL.Path))
	fmt.Fprintf(w, "raw query: %q\n", html.EscapeString(r.URL.RawQuery))
	fmt.Fprintf(w, "query:\n")
	for k, v := range r.URL.Query() {
		fmt.Fprintf(w, "    %v: %v\n", k, len(v))
		for _, val := range v {
			fmt.Fprintf(w, "        %v\n", val)
		}
	}

	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
	}

	fmt.Fprintf(w, "form values:\n")
	for key, val := range r.PostForm {
		fmt.Fprintf(w, "    %v: '%v'\n", key, val)
	}
}

func postBoardPost(w http.ResponseWriter, r *http.Request) {
	boardPath := strings.TrimSuffix(r.PathValue("board"), "/")
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("location", r.URL.Path)
	w.WriteHeader(http.StatusSeeOther)
	fmt.Fprintf(w, "<html>")
	fmt.Fprintf(w, "<head>")
	timeout := 3
	fmt.Fprintf(w, "<meta http-equiv=\"refresh\" content=\"%v;url=%q\" />",
		timeout,
		html.EscapeString(r.URL.Path))
	fmt.Fprintf(w, "</head>")
	fmt.Fprintf(w, "<body>\n")
	fmt.Fprintf(w, "<a href=%v>Continue</a>\n",
		html.EscapeString(r.URL.Path))

	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
	}

	postText, ok := r.PostForm["post"]
	if !ok || len(postText) != 1 {
		fmt.Fprintf(w, "<div>")
		fmt.Fprintf(w, "No post")
		fmt.Fprintf(w, "</div>")
	} else {
		postMessage(boardPath, "whoever", postText[0])
		fmt.Fprintf(w, "<div>")
		fmt.Fprintf(w, "Posted '%v'", postText[0])
		fmt.Fprintf(w, "</div>")
	}

	fmt.Fprintf(w, "</body></html>\n")
}

func getHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "<html><head></head><body>\n")
	fmt.Fprintf(w, "<h1>Home</h1>\n")

	fmt.Fprintf(w, "<a href=/app/boards/>Boards</a>\n")
	fmt.Fprintf(w, "<form method=\"post\" action=\"/admin/restart\">")
	fmt.Fprintf(w, "<input type=\"submit\" value=\"Restart server\" /a>\n")
	fmt.Fprintf(w, "</form>")

	debugHTML, err := generateDebugAsHTML(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		fmt.Fprintf(w, "<div>")
		fmt.Fprintf(w, debugHTML)
		fmt.Fprintf(w, "</div>")
	}

	fmt.Fprintf(w, "</body></html>\n")
}
