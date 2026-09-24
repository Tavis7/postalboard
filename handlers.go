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

func getUser(r *http.Request) string {
	cookies := parseCookies(r.Header.Values("cookie"))
	return cookies["username"]
}

func generateDebugAsHTML(r *http.Request) (*htmlNode, error) {
	result := makeHTMLNode("div", nil)
	acceptedMimeTypes, err := parseAccept(r.Header.Values("Accept"))
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
		for _, s := range val {
			headerListNode.appendChild(
				makeHTMLNode2("li", nil,
					makeHTMLNode2("code", nil,
						makeHTMLTextNode(fmt.Sprintf("%v: %v", header, s)))))
		}
	}

	result.appendChild(makeHTMLNode2("p", nil,
		makeHTMLTextNode(fmt.Sprintf("path: %v", r.URL.Path))))
	result.appendChild(
		makeHTMLNode2("p", nil,
			makeHTMLTextNode("Raw query: "),
			makeHTMLNode2("code", nil,
				makeHTMLTextNode(r.URL.RawQuery)),
		))
	result.appendChild(makeHTMLNode2("p", nil,
		makeHTMLTextNode("query:")))

	queryListNode := makeHTMLNode("ul", nil)
	result.appendChild(queryListNode)

	for k, v := range r.URL.Query() {
		queryListNode.appendChild(
			makeHTMLNode2("li", nil,
				makeHTMLNode2("code", nil,
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

func getPageHeader(pageTitle, user string) *htmlNode {
	result := makeHTMLNode("div", attribList{{"id", "page-header"}})

	result.appendChild(makeHTMLNode2("h1", nil, makeHTMLTextNode("Site Name")))

	result.appendChild(makeHTMLNode2("ul", nil, makeHTMLNode2("li", nil,
		makeHTMLNode2("a", attribList{{"href", "/"}}, makeHTMLTextNode("Root"))),
		makeHTMLNode2("li", nil,
			makeHTMLNode2("a", attribList{{"href", "/debug"}}, makeHTMLTextNode("Debug"))),
		makeHTMLNode2("li", nil,
			makeHTMLNode2("a", attribList{{"href", "/app"}}, makeHTMLTextNode("App"))),
		makeHTMLNode2("li", nil,
			makeHTMLNode2("a", attribList{{"href", "/app/boards"}}, makeHTMLTextNode("Boards"))),
		makeHTMLNode2("li", nil,
			makeHTMLNode2("a", attribList{{"href", "/app/login"}}, makeHTMLTextNode("Login"))),
	))
	if len(user) > 0 {
		result.appendChild(makeHTMLNode2("p", nil, makeHTMLTextNode("Logged in as "), makeHTMLTextNode(user)))
	}

	result.appendChild(makeHTMLNode2("h2", nil, makeHTMLTextNode(pageTitle)))

	return result
}

func getDebugger(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	htmlHead := makeHTMLNode("head", nil)
	htmlBody := makeHTMLNode("body", nil)

	htmlBody.appendChild(getPageHeader("Debug", getUser(r)))

	htmlBody.appendChild(
		makeHTMLNode("p", nil).appendChild(
			makeHTMLTextNode("Get handler says \"Hello\"")))

	htmlForm := makeHTMLNode("form", attribList{{"method", "post"}})
	htmlBody.appendChild(htmlForm)
	htmlForm.appendChild(makeHTMLNode("div", nil).appendChild(makeHTMLNode("input", attribList{{"type", "text"}, {"name", "text_input"}})))
	htmlForm.appendChild(makeHTMLNode("div", nil).appendChild(makeHTMLNode("input", attribList{{"type", "text"}, {"name", "text_input"}})))
	htmlForm.appendChild(makeHTMLNode("div", nil).appendChild(makeHTMLNode("textArea", attribList{{"name", "text_area"}})))
	htmlForm.appendChild(makeHTMLNode("input", attribList{{"type", "submit"}, {"value", "Submit"}}))

	htmlResetForm := makeHTMLNode("form", attribList{{"method", "post"}, {"action", "/reset-client"}})
	htmlBody.appendChild(htmlResetForm)
	htmlResetForm.appendChild(makeHTMLNode("input", attribList{{"type", "submit"}, {"value", "Reset"}}))
	debugNode, err := generateDebugAsHTML(r)
	if err != nil {
		// @todo
	}

	cookies := parseCookies(r.Header.Values("cookie"))
	debugNode.appendChild(makeHTMLNode2("p", nil, makeHTMLTextNode("Cookies:")))
	cookieListNode := makeHTMLNode("ul", nil)
	for k, v := range cookies {
		cookieListNode.appendChild(makeHTMLNode2("li", nil, makeHTMLTextNode(k), makeHTMLTextNode(": "), makeHTMLTextNode(v)))
	}
	debugNode.appendChild(cookieListNode)

	htmlBody.appendChild(debugNode)

	htmlRoot := makeHTMLNode("html", nil)
	htmlRoot.appendChildren(htmlHead, htmlBody)

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

	htmlBody.appendChild(getPageHeader(boardPath, getUser(r)))

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

	htmlRoot := makeHTMLNode("html", nil)
	htmlRoot.appendChildren(htmlHead, htmlBody)

	rendered, err := renderHTML(*htmlRoot)
	if err != nil {
		// @todo
		return
	}

	fmt.Fprint(w, rendered)
}

func postLogout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Header().Add("Set-Cookie", fmt.Sprintf("username=%s; path=/; max-age=0", ""))

	log.Printf("Logout")

	w.WriteHeader(http.StatusOK)

	htmlHead := makeHTMLNode("head", nil)
	htmlBody := makeHTMLNode("body", nil)

	htmlBody.appendChild(getPageHeader("Logout", getUser(r)))
	// @todo Invalidate server login state
	htmlBody.appendChild(makeHTMLNode2("p", nil, makeHTMLTextNode("Logged out")))

	htmlRoot := makeHTMLNode("html", nil)
	htmlRoot.appendChild(htmlHead)
	htmlRoot.appendChild(htmlBody)

	rendered, err := renderHTML(*htmlRoot)
	if err != nil {
		// @todo
	}

	fmt.Fprintf(w, rendered)
}

func httpLoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	htmlHead := makeHTMLNode("head", nil)
	htmlBody := makeHTMLNode("body", nil)

	user := getUser(r)
	htmlBody.appendChild(getPageHeader("Login", user))

	if len(user) != 0 {
		htmlBody.appendChild(makeHTMLNode2("p", nil, makeHTMLTextNode("Already logged in as "), makeHTMLTextNode(user)))
		htmlBody.appendChild(
			makeHTMLNode2("form", attribList{{"method", "post"}, {"action", "/app/logout"}},
				makeHTMLNode2("p", nil,
					makeHTMLNode("input",
						attribList{
							{"type", "submit"},
							{"value", "Log out"},
						},
					))))
	} else {
		htmlBody.appendChild(
			makeHTMLNode2("form", attribList{{"method", "post"}},
				makeHTMLNode2("label", nil,
					makeHTMLNode2("div", nil,
						makeHTMLTextNode("username"),
						makeHTMLNode("input",
							attribList{{"type", "text"},
								{"name", "username"}}))),
				makeHTMLNode2("label", attribList{{ /* @todo */ "style", "display:none"}},
					makeHTMLNode2("div", nil,
						makeHTMLTextNode("password"),
						makeHTMLNode("input",
							attribList{{"type", "password"},
								{"name", "password"}}))),
				makeHTMLNode("input",
					attribList{{"type", "submit"},
						{"value", "login"}})),
		)
	}

	htmlRoot := makeHTMLNode("html", nil)
	htmlRoot.appendChild(htmlHead)
	htmlRoot.appendChild(htmlBody)

	rendered, err := renderHTML(*htmlRoot)
	if err != nil {
		// @todo
	}

	fmt.Fprintf(w, rendered)
}

func postLogin(w http.ResponseWriter, r *http.Request) {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "Login handler says \"Hello\"\n")
	fmt.Fprintf(sb, "path: %q\n", html.EscapeString(r.URL.Path))
	fmt.Fprintf(sb, "raw query: %q\n", html.EscapeString(r.URL.RawQuery))
	fmt.Fprintf(sb, "query:\n")
	for k, v := range r.URL.Query() {
		fmt.Fprintf(sb, "    %v: %v\n", k, len(v))
		for _, val := range v {
			fmt.Fprintf(sb, "        %v\n", val)
		}
	}

	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
	}

	fmt.Fprintf(sb, "form values:\n")
	for key, val := range r.PostForm {
		fmt.Fprintf(sb, "    %v: '%v'\n", key, val)
		if key == "username" {
			w.Header().Add("Set-Cookie", fmt.Sprintf("username=%s; path=/", val[0]))
			log.Printf("Logged in as %v", val[0])
		}
	}

	fmt.Fprint(w, sb.String())

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
	if !debugRedirect {
		w.WriteHeader(http.StatusSeeOther)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	timeout := 3
	htmlHead := makeHTMLNode2("head", nil,
		makeHTMLNode("meta",
			attribList{{"http-equiv", "refresh"},
				{"content", fmt.Sprintf("%v;url=%v", timeout, r.URL.Path)}}))
	htmlBody := makeHTMLNode("body", nil)

	htmlBody.appendChild(getPageHeader("Post", getUser(r)))

	htmlBody.appendChild(makeHTMLNode2("a",
		attribList{{"href", r.URL.Path}},
		makeHTMLTextNode("Continue")))

	successNode := makeHTMLNode("div", nil)
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
	}

	postText, ok := r.PostForm["post"]
	if !ok || len(postText) != 1 {
		successNode.appendChild(makeHTMLTextNode("No post"))
	} else {
		postMessage(boardPath, "whoever", postText[0])
		successNode.appendChild(makeHTMLTextNode(fmt.Sprintf("Posted '%v'", postText[0])))
	}

	htmlBody.appendChild(successNode)

	htmlRoot := makeHTMLNode("html", nil)
	htmlRoot.appendChild(htmlHead)
	htmlRoot.appendChild(htmlBody)

	rendered, err := renderHTML(*htmlRoot)
	if err != nil {
		// @todo
	}

	fmt.Fprintf(w, rendered)
}

func getHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	htmlHead := makeHTMLNode("head", nil)
	htmlBody := makeHTMLNode("body", nil)

	htmlBody.appendChild(getPageHeader("Home", getUser(r)))

	htmlBody.appendChild(makeHTMLNode2("a",
		attribList{{"href", "/app/boards/"}},
		makeHTMLTextNode("Boards")))
	htmlBody.appendChild(makeHTMLNode2("form",
		attribList{{"method", "post"},
			{"action", "/admin/restart"}},
		makeHTMLNode("input",
			attribList{{"type", "submit"}, {"value", "Restart server"}})))

	htmlRoot := makeHTMLNode("html", nil)
	htmlRoot.appendChild(htmlHead)
	htmlRoot.appendChild(htmlBody)

	rendered, err := renderHTML(*htmlRoot)
	if err != nil {
		// @todo
	}

	fmt.Fprintf(w, rendered)
}
