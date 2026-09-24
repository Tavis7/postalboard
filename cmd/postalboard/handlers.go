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
	return result, nil
}

func getPageHeader2(pageTitle, user string) *htmlgen.Node {
	result := htmlgen.MakeNode("div", htmlgen.AttribList{{"id", "page-header"}})

	result.AppendChildren(htmlgen.MakeNode("h1", nil, htmlgen.MakeTextNode("Site Name")))

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
	))
	if len(user) > 0 {
		result.AppendChildren(htmlgen.MakeNode("p", nil,
			htmlgen.MakeTextNode("Logged in as "),
			htmlgen.MakeTextNode(user)))
	}

	result.AppendChildren(htmlgen.MakeNode("h2", nil,
		htmlgen.MakeTextNode(pageTitle)))

	return result
}

func getPageHeader(pageTitle, user string) *htmlNode {
	// @todo @delete this function
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

	htmlHead := htmlgen.MakeLeafNode("head")
	htmlBody := htmlgen.MakeLeafNode("body")

	htmlBody.AppendChildren(getPageHeader2("Debug", getUser(r)))

	htmlBody.AppendChildren(
		htmlgen.MakeNode("p", nil,
			htmlgen.MakeTextNode("Get handler says \"Hello\"")))

	htmlForm := htmlgen.MakeNode("form", htmlgen.AttribList{{"method", "post"}})
	htmlBody.AppendChildren(htmlForm)
	htmlForm.AppendChildren(htmlgen.MakeNode("div", nil,
		htmlgen.MakeNode("input",
			htmlgen.AttribList{{"type", "text"},
				{"name", "text_input"}})))
	htmlForm.AppendChildren(htmlgen.MakeNode("div", nil,
		htmlgen.MakeNode("input",
			htmlgen.AttribList{{"type", "text"},
				{"name", "text_input"}})))
	htmlForm.AppendChildren(htmlgen.MakeNode("div", nil,
		htmlgen.MakeNode("textArea",
			htmlgen.AttribList{{"name", "text_area"}})))
	htmlForm.AppendChildren(htmlgen.MakeNode("input",
		htmlgen.AttribList{{"type", "submit"},
			{"value", "Submit"}}))

	htmlResetForm := htmlgen.MakeNode("form",
		htmlgen.AttribList{{"method", "post"},
			{"action", "/reset-client"}})
	htmlBody.AppendChildren(htmlResetForm)
	htmlResetForm.AppendChildren(htmlgen.MakeNode("input",
		htmlgen.AttribList{{"type", "submit"},
			{"value", "Reset"}}))
	debugNode, err := generateDebugAsHTML(r)
	if err != nil {
		// @todo
	}

	cookies := parseCookies(r.Header.Values("cookie"))
	debugNode.AppendChildren(htmlgen.MakeNode("p", nil,
		htmlgen.MakeTextNode("Cookies:")))
	cookieListNode := htmlgen.MakeLeafNode("ul")
	for k, v := range cookies {
		cookieListNode.AppendChildren(htmlgen.MakeNode("li", nil,
			htmlgen.MakeTextNode(k),
			htmlgen.MakeTextNode(": "),
			htmlgen.MakeTextNode(v)))
	}
	debugNode.AppendChildren(cookieListNode)

	htmlBody.AppendChildren(debugNode)

	htmlRoot := htmlgen.MakeNode("html", nil, htmlHead, htmlBody)

	rendered, err := htmlgen.Render(*htmlRoot)
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

	htmlHead := htmlgen.MakeLeafNode("head")
	htmlBody := htmlgen.MakeLeafNode("body")

	htmlBody.AppendChildren(getPageHeader2(boardPath, getUser(r)))

	if len(children) > 0 {
		htmlBoardList := htmlgen.MakeNode("div", htmlgen.AttribList{{"id", "boardlist"}})
		for _, child := range children {
			htmlBoardList.AppendChildren(
				htmlgen.MakeNode("p", nil,
					htmlgen.MakeNode("a", htmlgen.AttribList{{"href", child}},
						htmlgen.MakeTextNode(child))))
		}
		htmlBody.AppendChildren(htmlgen.MakeNode("p", nil,
			htmlgen.MakeTextNode("Boards:")))

		htmlBody.AppendChildren(htmlBoardList)
	}

	if len(posts) > 0 {
		htmlBody.AppendChildren(htmlgen.MakeNode("p", nil,
			htmlgen.MakeTextNode("Posts: ")))

		for _, post := range posts {
			htmlBody.AppendChildren(htmlgen.MakeNode("p", nil,
				htmlgen.MakeTextNode(fmt.Sprintf("%v: %v", post.user, post.text))))
		}

		htmlBody.AppendChildren(
			htmlgen.MakeNode("form", htmlgen.AttribList{{"method", "post"}},
				htmlgen.MakeNode("textArea", htmlgen.AttribList{{"name", "post"}}),
				htmlgen.MakeNode("div", nil,
					htmlgen.MakeNode("input",
						htmlgen.AttribList{{"type", "submit"}, {"value", "post"}}))))
	}

	htmlRoot := htmlgen.MakeNode("html", nil, htmlHead, htmlBody)

	rendered, err := htmlgen.Render(*htmlRoot)
	if err != nil {
		// @todo
		return
	}

	fmt.Fprint(w, rendered)
}

func postLogout(w http.ResponseWriter, r *http.Request) {
	redirectTo := "/app/login"
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("location", redirectTo)
	w.Header().Add("Set-Cookie", fmt.Sprintf("username=%s; path=/; max-age=0", ""))
	if !debugRedirect {
		w.WriteHeader(http.StatusSeeOther)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	log.Printf("Logout")

	timeout := 3
	htmlHead := makeHTMLNode2("head", nil,
		makeHTMLNode("meta",
			attribList{{"http-equiv", "refresh"},
				{"content", fmt.Sprintf("%v;url=%v", timeout, redirectTo)}}))
	htmlBody := makeHTMLNode("body", nil)

	htmlBody.appendChild(getPageHeader("Logout", getUser(r)))
	// @todo Invalidate server login state
	htmlBody.appendChild(makeHTMLNode2("p", nil, makeHTMLTextNode("Logged out")))
	htmlBody.appendChild(makeHTMLNode2("p", nil, makeHTMLTextNode("Click "),
		makeHTMLNode2("a", attribList{{"href", redirectTo}},
			makeHTMLTextNode("here")),
		makeHTMLTextNode(" to continue")))

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
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
	}

	username := ""
	for key, val := range r.PostForm {
		if key == "username" {
			username = val[0]
			w.Header().Add("Set-Cookie", fmt.Sprintf("username=%s; path=/", username))
			log.Printf("Logged in as %v", val[0])
		}
	}

	redirectTo := "/"
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("location", redirectTo)
	if !debugRedirect {
		w.WriteHeader(http.StatusSeeOther)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	log.Printf("Logged In")

	timeout := 3
	htmlHead := makeHTMLNode2("head", nil,
		makeHTMLNode("meta",
			attribList{{"http-equiv", "refresh"},
				{"content", fmt.Sprintf("%v;url=%v", timeout, redirectTo)}}))
	htmlBody := makeHTMLNode("body", nil)

	htmlBody.appendChild(getPageHeader("Logged In", username))
	// @todo Invalidate server login state
	htmlBody.appendChild(makeHTMLNode2("p", nil, makeHTMLTextNode("Logged in as "),
		makeHTMLTextNode(username)))
	htmlBody.appendChild(makeHTMLNode2("p", nil, makeHTMLTextNode("Click "),
		makeHTMLNode2("a", attribList{{"href", redirectTo}},
			makeHTMLTextNode("here")),
		makeHTMLTextNode(" to continue")))

	htmlRoot := makeHTMLNode("html", nil)
	htmlRoot.appendChild(htmlHead)
	htmlRoot.appendChild(htmlBody)

	rendered, err := renderHTML(*htmlRoot)
	if err != nil {
		// @todo
	}

	fmt.Fprintf(w, rendered)
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
