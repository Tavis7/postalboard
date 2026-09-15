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

func generateDebugAsHTML(r *http.Request) (*htmlNode, error) {
	result := makeHTMLNode("div", nil)
	acceptedMimeTypes, err := parseAccept(r.Header.Get("Accept"))
	if err != nil {
		log.Printf("Error parsing mimetypes: %v", err)
		result.appendChild(
			makeHTMLNode("p", nil).appendChild(
				makeHTMLTextNode(fmt.Sprintf("Couldn't parse mimetypes: %v", err))))
	}

	result.appendNode("p", nil, makeHTMLTextNode(fmt.Sprintf("protocol: %q", r.Proto)))
	result.appendNode("p", nil, makeHTMLTextNode("headers:"))
	headerListNode := makeHTMLNode("ul", nil)
	result.appendChild(headerListNode)
	for header, val := range r.Header {
		headerListNode.appendNode("li", nil, makeHTMLNode2("li", nil, makeHTMLNode2("pre", attribList{{"style", "display:inline"}}, makeHTMLTextNode(fmt.Sprintf("%v: %v", header, val)))))
		/*
			fmt.Fprintf(w, "<li><pre style=\"display:inline\">%v: %v</pre></li>\n",
				html.EscapeString(fmt.Sprintf("%v", header)),
				html.EscapeString(fmt.Sprintf("%v", val)))
		*/
	}

	result.appendChild(makeHTMLNode2("p", nil,
		makeHTMLTextNode(fmt.Sprintf("path: %v", r.URL.Path))))
	result.appendChild(makeHTMLNode2("p", nil,
		makeHTMLTextNode("Raw query: "),
		makeHTMLNode2("pre", attribList{{"style", "display:inline"}}, makeHTMLTextNode(r.URL.RawQuery)),
	))
	result.appendChild(makeHTMLNode2("p", nil,
		makeHTMLTextNode("query:")))

	queryListNode := makeHTMLNode("ul", nil)
	result.appendChild(queryListNode)

	for k, v := range r.URL.Query() {
		queryListNode.appendChild(
			makeHTMLNode2("li", nil,
				makeHTMLNode2("pre", attribList{{"style", "display:inline"}},
					makeHTMLTextNode(fmt.Sprintf("%v=%v", k, v)))))
	}

	result.appendChild(makeHTMLNode2("p", nil,
		makeHTMLTextNode("accepted mimetypes:")))

	mimeListNode := makeHTMLNode("ul", nil)
	result.appendChild(mimeListNode)

	found := -1
	for i, val := range acceptedMimeTypes {
		mimeListNode.appendChild(
			makeHTMLNode2("li", nil, makeHTMLTextNode(fmt.Sprintf("%v", val))))
		if found == -1 {
			if val.mimetype == "text/html" ||
				val.mimetype == "text/*" ||
				val.mimetype == "*/*" {
				found = i
			}
		}
	}

	result.appendChild(makeHTMLNode2("p", nil,
		makeHTMLTextNode("index: "),
		makeHTMLTextNode(fmt.Sprintf("%v", found))))
	if found != -1 {
		result.appendChild(makeHTMLNode2("p", nil,
			makeHTMLTextNode("Found: "),
			makeHTMLTextNode(fmt.Sprintf("%v", acceptedMimeTypes[found]))))
	} else {
		result.appendChild(makeHTMLNode2("p", nil,
			makeHTMLTextNode("does not contain text/html")))
		// 406 Not Acceptable
		return result, fmt.Errorf("Accept header does not contain text/html")
	}
	return result, nil
}

func getDebugger(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	htmlHead := makeHTMLNode("head", nil)
	htmlBody := makeHTMLNode("body", nil)
	htmlRoot := makeHTMLNode("html", nil)
	htmlRoot.appendChildren(htmlHead, htmlBody)

	htmlBody.appendChild(makeHTMLNode2("a", attribList{{"href", "boards"}}, makeHTMLTextNode("Boards")))

	htmlBody.appendChild(
		makeHTMLNode("p", nil).appendChild(
			makeHTMLTextNode("Get handler says \"Hello\"")))

	htmlForm := makeHTMLNode("form", attribList{{"method", "post"}})
	htmlBody.appendChild(htmlForm)
	htmlForm.appendChild(makeHTMLNode("div", nil).appendChild(makeHTMLNode("input", attribList{{"type", "text"}, {"name", "text_input"}})))
	htmlForm.appendChild(makeHTMLNode("div", nil).appendChild(makeHTMLNode("input", attribList{{"type", "text"}, {"name", "text_input"}})))
	htmlForm.appendChild(makeHTMLNode("div", nil).appendChild(makeHTMLNode("textArea", attribList{{"name", "text_area"}})))
	htmlForm.appendChild(makeHTMLNode("input", attribList{{"type", "submit"}, {"value", "Submit"}}))

	debugNode, err := generateDebugAsHTML(r)
	if err != nil {
		// @todo
	}
	htmlBody.appendChild(debugNode)

	rendered, err := renderHTML(*htmlRoot)
	if err != nil {
		// @todo
	}
	fmt.Fprint(w, rendered)
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

	htmlHead := makeHTMLNode("head", nil)
	htmlBody := makeHTMLNode("body", nil)

	htmlRoot := makeHTMLNode("html", nil)
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

	rendered, err := renderHTML(*htmlRoot)
	if err != nil {
		// @todo
		return
	}

	fmt.Fprint(w, rendered)
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

	/*
		// @todo
		debugHTML, err := generateDebugAsHTML(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			fmt.Fprintf(w, "<div>")
			fmt.Fprintf(w, debugHTML)
			fmt.Fprintf(w, "</div>")
		}
	*/

	fmt.Fprintf(w, "</body></html>\n")
}
