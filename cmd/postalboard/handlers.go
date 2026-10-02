package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/lib/pq"

	"github.com/tavis7/postalboard/internal/database"
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

func authenticate(context context.Context, username string) (database.User, error) {
	log.Printf("Authenticating as %v", username)
	user, err := dbQueries.GetUserByUsername(context, username)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("sql error: %v", err)
		}
		pqErr := new(pq.Error)
		log.Printf("authentication error")
		if errors.As(err, &pqErr) {
			log.Printf("pq error: %v", pqErr.Code.Name())
		}
		return user, err
	}
	return user, nil
}

func getUser(r *http.Request) (string, error) {
	cookies := parseCookies(r.Header.Values("cookie"))
	username := cookies["username"]

	if len(username) == 0 {
		return "", nil
	}

	user, err := authenticate(r.Context(), username)

	return user.Username, err
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

func getPageHeader(pageTitle, user string) *htmlgen.Node {
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

func getDebugger(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	htmlHead := htmlgen.MakeLeafNode("head")
	htmlBody := htmlgen.MakeLeafNode("body")

	username, err := getUser(r)
	if err != nil {
		// @todo Respond with error / user not found
		log.Printf("Error getting user: %v", err)
	}
	htmlBody.AppendChildren(getPageHeader("Debug", username))

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
		log.Printf("Error generating debug html: %v", err)
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
		log.Printf("Error rendering html: %v", err)
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

	username, err := getUser(r)
	if err != nil {
		// @todo Respond with error
		log.Printf("Error getting user: %v", err)
	}
	htmlBody.AppendChildren(getPageHeader(boardPath, username))

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

		if len(username) > 0 {
			htmlBody.AppendChildren(
				htmlgen.MakeNode("form", htmlgen.AttribList{{"method", "post"}},
					htmlgen.MakeNode("textArea", htmlgen.AttribList{{"name", "post"}}),
					htmlgen.MakeNode("div", nil,
						htmlgen.MakeNode("input",
							htmlgen.AttribList{{"type", "submit"}, {"value", "post"}}))))
		} else {
			redirect := url.QueryEscape(r.URL.Path)
			loginURL := fmt.Sprintf("/app/login?redirect=%v", redirect)
			registerURL := fmt.Sprintf("/app/register?redirect=%v", redirect)
			htmlBody.AppendChildren(
				htmlgen.MakeNode("p", nil,
					htmlgen.MakeNode("a",
						htmlgen.AttribList{{"href", loginURL}},
						htmlgen.MakeTextNode("Log in")),
					htmlgen.MakeTextNode(" or "),
					htmlgen.MakeNode("a",
						htmlgen.AttribList{{"href", registerURL}},
						htmlgen.MakeTextNode("register")),
					htmlgen.MakeTextNode(" to post")))
		}
	}

	htmlRoot := htmlgen.MakeNode("html", nil, htmlHead, htmlBody)

	rendered, err := htmlgen.Render(*htmlRoot)
	if err != nil {
		// @todo
		log.Printf("Error rendering html: %v", err)
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
	htmlHead := htmlgen.MakeNode("head", nil,
		htmlgen.MakeNode("meta",
			htmlgen.AttribList{{"http-equiv", "refresh"},
				{"content", fmt.Sprintf("%v;url=%v", timeout, redirectTo)}}))
	htmlBody := htmlgen.MakeLeafNode("body")

	username, err := getUser(r)
	if err != nil {
		// @todo Respond with error
		log.Printf("Error getting user: %v", err)
	}
	htmlBody.AppendChildren(getPageHeader("Logout", username))
	// @todo Invalidate server login state
	htmlBody.AppendChildren(htmlgen.MakeNode("p", nil, htmlgen.MakeTextNode("Logged out")))
	htmlBody.AppendChildren(htmlgen.MakeNode("p", nil, htmlgen.MakeTextNode("Click "),
		htmlgen.MakeNode("a", htmlgen.AttribList{{"href", redirectTo}},
			htmlgen.MakeTextNode("here")),
		htmlgen.MakeTextNode(" to continue")))

	htmlRoot := htmlgen.MakeNode("html", nil, htmlHead, htmlBody)

	rendered, err := htmlgen.Render(*htmlRoot)
	if err != nil {
		// @todo
		log.Printf("Error rendering html: %v", err)
	}

	fmt.Fprint(w, rendered)
}

func httpRegisterPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	logoutURL := "/app/logout"
	registerURL := "/app/register"

	q := r.URL.Query()
	redirects, ok := q["redirect"]
	log.Printf("register redirect: '%v'", redirects)
	if ok {
		if len(redirects) != 1 {
			// @todo Include URL and other information in log outputs
			log.Printf("Warning: Too many redirect parameters: %v", redirects)
		} else {
			redirect := url.QueryEscape(redirects[0])
			registerURL = strings.Join([]string{registerURL, "?redirect=", redirect}, "")
		}
	}

	htmlHead := htmlgen.MakeLeafNode("head")
	htmlBody := htmlgen.MakeLeafNode("body")

	username, err := getUser(r)
	if err != nil {
		// @todo Respond with error
		log.Printf("Error getting user: %v", err)
	}
	htmlBody.AppendChildren(getPageHeader("Register", username))

	if len(username) != 0 {
		htmlBody.AppendChildren(htmlgen.MakeNode("p", nil, htmlgen.MakeTextNode("Already logged in as "), htmlgen.MakeTextNode(username)))
		htmlBody.AppendChildren(
			htmlgen.MakeNode("form",
				htmlgen.AttribList{{"method", "post"},
					{"action", logoutURL}},
				htmlgen.MakeNode("p", nil,
					htmlgen.MakeNode("input",
						htmlgen.AttribList{
							{"type", "submit"},
							{"value", "Log out"},
						},
					))))
	} else {
		htmlBody.AppendChildren(
			htmlgen.MakeNode("form",
				htmlgen.AttribList{{"method", "post"},
					{"action", registerURL}},
				htmlgen.MakeNode("label", nil,
					htmlgen.MakeNode("div", nil,
						htmlgen.MakeTextNode("username"),
						htmlgen.MakeNode("input",
							htmlgen.AttribList{{"type", "text"},
								{"name", "username"}}))),
				htmlgen.MakeNode("label",
					htmlgen.AttribList{{ /* @todo */ "style", "display:none"}},
					htmlgen.MakeNode("div", nil,
						htmlgen.MakeTextNode("password"),
						htmlgen.MakeNode("input",
							htmlgen.AttribList{{"type", "password"},
								{"name", "password"}}))),
				htmlgen.MakeNode("input",
					htmlgen.AttribList{{"type", "submit"},
						{"value", "register"}})),
		)
	}

	htmlRoot := htmlgen.MakeNode("html", nil, htmlHead, htmlBody)

	rendered, err := htmlgen.Render(*htmlRoot)
	if err != nil {
		// @todo
		log.Printf("Error rendering html: %v", err)
	}

	fmt.Fprint(w, rendered)
}

func httpLoginPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	logoutURL := "/app/logout"
	loginURL := "/app/login"

	q := r.URL.Query()
	redirects, ok := q["redirect"]
	log.Printf("login redirect: '%v'", redirects)
	if ok {
		if len(redirects) != 1 {
			// @todo Include URL and other information in log outputs
			log.Printf("Warning: Too many redirect parameters: %v", redirects)
		} else {
			redirect := url.QueryEscape(redirects[0])
			logoutURL = strings.Join([]string{logoutURL, "?redirect=", redirect}, "")
			loginURL = strings.Join([]string{loginURL, "?redirect=", redirect}, "")
		}
	}

	htmlHead := htmlgen.MakeLeafNode("head")
	htmlBody := htmlgen.MakeLeafNode("body")

	username, err := getUser(r)
	if err != nil {
		// @todo Respond with error
		log.Printf("Error getting user: %v", err)
	}

	htmlBody.AppendChildren(getPageHeader("Login", username))

	if len(username) != 0 {
		htmlBody.AppendChildren(htmlgen.MakeNode("p", nil, htmlgen.MakeTextNode("Already logged in as "), htmlgen.MakeTextNode(username)))
		htmlBody.AppendChildren(
			htmlgen.MakeNode("form",
				htmlgen.AttribList{{"method", "post"},
					{"action", logoutURL}},
				htmlgen.MakeNode("p", nil,
					htmlgen.MakeNode("input",
						htmlgen.AttribList{
							{"type", "submit"},
							{"value", "Log out"},
						},
					))))
	} else {
		htmlBody.AppendChildren(
			htmlgen.MakeNode("form",
				htmlgen.AttribList{{"method", "post"},
					{"action", loginURL}},
				htmlgen.MakeNode("label", nil,
					htmlgen.MakeNode("div", nil,
						htmlgen.MakeTextNode("username"),
						htmlgen.MakeNode("input",
							htmlgen.AttribList{{"type", "text"},
								{"name", "username"}}))),
				htmlgen.MakeNode("label",
					htmlgen.AttribList{{ /* @todo */ "style", "display:none"}},
					htmlgen.MakeNode("div", nil,
						htmlgen.MakeTextNode("password"),
						htmlgen.MakeNode("input",
							htmlgen.AttribList{{"type", "password"},
								{"name", "password"}}))),
				htmlgen.MakeNode("input",
					htmlgen.AttribList{{"type", "submit"},
						{"value", "login"}})),
		)
	}

	htmlRoot := htmlgen.MakeNode("html", nil, htmlHead, htmlBody)

	rendered, err := htmlgen.Render(*htmlRoot)
	if err != nil {
		// @todo
		log.Printf("Error rendering html: %v", err)
	}

	fmt.Fprint(w, rendered)
}

func respondWithError(w http.ResponseWriter, r *http.Request, statusCode int) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(statusCode)

	htmlHead := htmlgen.MakeLeafNode("head")
	htmlBody := htmlgen.MakeLeafNode("body")

	username, err := getUser(r)
	if err != nil {
		// @todo Respond with error
		log.Printf("Error getting user: %v", err)
	}
	htmlBody.AppendChildren(getPageHeader("Home", username))

	htmlBody.AppendChildren(htmlgen.MakeNode("h1", nil, htmlgen.MakeTextNode(fmt.Sprintf("%v", statusCode))))

	htmlRoot := htmlgen.MakeNode("html", nil, htmlHead, htmlBody)

	rendered, err := htmlgen.Render(*htmlRoot)
	if err != nil {
		// @todo
		log.Printf("Error rendering html: %v", err)
	}

	fmt.Fprint(w, rendered)
}

func postRegister(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
	}

	username := ""
	for key, val := range r.PostForm {
		if key == "username" {
			username = val[0]
		}
	}

	redirectTo := "/"

	q := r.URL.Query()
	redirects, ok := q["redirect"]
	if ok {
		if len(redirects) != 1 {
			// @todo Include URL and other information in log outputs
			log.Printf("Warning: Too many redirect parameters: %v", redirects)
		}
		redirectTo = redirects[0]
	}

	users, err := dbQueries.CreateUser(r.Context(),
		database.CreateUserParams{uuid.New(), username,
			fmt.Sprintf("%s@localhost" /* @todo */, username)})
	if err != nil {
		// @todo Detect user already exists
		log.Printf("Error registering user: %v", err)
		respondWithError(w, r, http.StatusInternalServerError)
		return
	}

	log.Printf("%v", users)

	if username != users.Username {
		// @todo
		log.Printf("Error: username doesn't match created user's username: %v != %v",
			username, users.Username)
		respondWithError(w, r, http.StatusInternalServerError)
		return
	}

	w.Header().Add("Set-Cookie", fmt.Sprintf("username=%s; path=/", username))
	log.Printf("Logged in as %v", username)

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("location", redirectTo)
	if !debugRedirect {
		// @todo StatusCreated instead of redirect
		w.WriteHeader(http.StatusSeeOther)
	} else {
		// @todo StatusCreated instead of redirect
		w.WriteHeader(http.StatusOK)
	}

	log.Printf("Registered as %s", users.Username)

	timeout := 3
	htmlHead := htmlgen.MakeNode("head", nil,
		htmlgen.MakeNode("meta",
			htmlgen.AttribList{{"http-equiv", "refresh"},
				{"content", fmt.Sprintf("%v;url=%v", timeout, redirectTo)}}))
	htmlBody := htmlgen.MakeLeafNode("body")

	htmlBody.AppendChildren(getPageHeader("Logged In", username))
	// @todo Invalidate server login state
	htmlBody.AppendChildren(htmlgen.MakeNode("p", nil,
		htmlgen.MakeTextNode("Logged in as "),
		htmlgen.MakeTextNode(username)))
	htmlBody.AppendChildren(htmlgen.MakeNode("p", nil,
		htmlgen.MakeTextNode("Click "),
		htmlgen.MakeNode("a", htmlgen.AttribList{{"href", redirectTo}},
			htmlgen.MakeTextNode("here")),
		htmlgen.MakeTextNode(" to continue")))

	htmlRoot := htmlgen.MakeNode("html", nil, htmlHead, htmlBody)

	rendered, err := htmlgen.Render(*htmlRoot)
	if err != nil {
		// @todo
		log.Printf("Error rendering html: %v", err)
	}

	fmt.Fprint(w, rendered)
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
		}
	}
	user, err := authenticate(r.Context(), username)
	if err != nil {
		// @todo Return error
		log.Printf("postLogin(): Error getting user: %v", err)
		return
	}

	if user.Username != username {
		// @todo Return error
		log.Printf("Error: username doesn't match user: %v != %v", username, user)
		return
	}

	w.Header().Add("Set-Cookie", fmt.Sprintf("username=%s; path=/", username))
	log.Printf("Logged in as %v", username)

	redirectTo := "/"

	q := r.URL.Query()
	redirects, ok := q["redirect"]
	if ok {
		if len(redirects) != 1 {
			// @todo Include URL and other information in log outputs
			log.Printf("Warning: Too many redirect parameters: %v", redirects)
		}
		redirectTo = redirects[0]
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("location", redirectTo)
	if !debugRedirect {
		// @todo StatusCreated instead of redirect
		w.WriteHeader(http.StatusSeeOther)
	} else {
		// @todo StatusCreated
		w.WriteHeader(http.StatusOK)
	}

	log.Printf("Logged In")

	timeout := 3
	htmlHead := htmlgen.MakeNode("head", nil,
		htmlgen.MakeNode("meta",
			htmlgen.AttribList{{"http-equiv", "refresh"},
				{"content", fmt.Sprintf("%v;url=%v", timeout, redirectTo)}}))
	htmlBody := htmlgen.MakeLeafNode("body")

	htmlBody.AppendChildren(getPageHeader("Logged In", username))
	// @todo Invalidate server login state
	htmlBody.AppendChildren(htmlgen.MakeNode("p", nil,
		htmlgen.MakeTextNode("Logged in as "),
		htmlgen.MakeTextNode(username)))
	htmlBody.AppendChildren(htmlgen.MakeNode("p", nil,
		htmlgen.MakeTextNode("Click "),
		htmlgen.MakeNode("a", htmlgen.AttribList{{"href", redirectTo}},
			htmlgen.MakeTextNode("here")),
		htmlgen.MakeTextNode(" to continue")))

	htmlRoot := htmlgen.MakeNode("html", nil, htmlHead, htmlBody)

	rendered, err := htmlgen.Render(*htmlRoot)
	if err != nil {
		// @todo
		log.Printf("Error rendering html: %v", err)
	}

	fmt.Fprint(w, rendered)
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
	htmlHead := htmlgen.MakeNode("head", nil,
		htmlgen.MakeNode("meta",
			htmlgen.AttribList{{"http-equiv", "refresh"},
				{"content", fmt.Sprintf("%v;url=%v", timeout, r.URL.Path)}}))
	htmlBody := htmlgen.MakeLeafNode("body")

	username, err := getUser(r)
	if err != nil {
		// @todo Respond with error
		log.Printf("Error getting user: %v", err)
	}
	htmlBody.AppendChildren(getPageHeader("Post", username))

	htmlBody.AppendChildren(htmlgen.MakeNode("a",
		htmlgen.AttribList{{"href", r.URL.Path}},
		htmlgen.MakeTextNode("Continue")))

	successNode := htmlgen.MakeLeafNode("div")
	err = r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
	}

	postText, ok := r.PostForm["post"]
	if !ok || len(postText) != 1 {
		successNode.AppendChildren(htmlgen.MakeTextNode("No post"))
	} else {
		postMessage(boardPath, username, postText[0])
		successNode.AppendChildren(htmlgen.MakeTextNode(fmt.Sprintf("Posted '%v'", postText[0])))
	}

	htmlBody.AppendChildren(successNode)

	htmlRoot := htmlgen.MakeNode("html", nil, htmlHead, htmlBody)

	rendered, err := htmlgen.Render(*htmlRoot)
	if err != nil {
		// @todo
		log.Printf("Error rendering html: %v", err)
	}

	fmt.Fprint(w, rendered)
}

func getHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	htmlHead := htmlgen.MakeLeafNode("head")
	htmlBody := htmlgen.MakeLeafNode("body")

	username, err := getUser(r)
	if err != nil {
		// @todo Respond with error
		log.Printf("Error getting user: %v", err)
	}
	htmlBody.AppendChildren(getPageHeader("Home", username))

	htmlBody.AppendChildren(htmlgen.MakeNode("a",
		htmlgen.AttribList{{"href", "/app/boards/"}},
		htmlgen.MakeTextNode("Boards")))
	htmlBody.AppendChildren(htmlgen.MakeNode("form",
		htmlgen.AttribList{{"method", "post"},
			{"action", "/admin/restart"}},
		htmlgen.MakeNode("input",
			htmlgen.AttribList{{"type", "submit"}, {"value", "Restart server"}})))

	htmlRoot := htmlgen.MakeNode("html", nil, htmlHead, htmlBody)

	rendered, err := htmlgen.Render(*htmlRoot)
	if err != nil {
		// @todo
		log.Printf("Error rendering html: %v", err)
	}

	fmt.Fprint(w, rendered)
}
