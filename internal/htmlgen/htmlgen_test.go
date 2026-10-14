package htmlgen

import "testing"

import (
	"strings"
)

func TestRender(t *testing.T) {
	expecting := strings.Join([]string{
		"<!doctype html>",
		"<html>",
		"    <body>",
		"        <p class=\"something\" id=\"foobar\">",
		"            hello <span>world</span>",
		"        </p>",
		"    </body>",
		"</html>\n",
	}, "\n")

	testNode := MakeNode("p", AttribList{{"class", "something"}, {"id", "foobar"}})
	testNode.AppendChildren(MakeTextNode("hello "))
	testSpan := MakeLeafNode("span")
	testSpan.AppendChildren(MakeTextNode("world"))
	testNode.AppendChildren(testSpan)

	testRoot := MakeNode("html", nil,
		MakeNode("body", nil, testNode))
	rendered, err := Render(*testRoot)

	if err != nil {
		t.Errorf("render failed: %v\n", err)
	}

	if rendered != expecting {
		t.Errorf("Incorrect HTML:\n%s", rendered)
	}
}

func TestRender2(t *testing.T) {
	expecting := strings.Join([]string{
		"<!doctype html>",
		"<html>",
		"    <head>",
		"    </head>",
		"    <body>",
		"    </body>",
		"</html>\n",
	}, "\n")

	root := MakeNode("html", nil,
		MakeNode("head", nil),
		MakeNode("body", nil))

	rendered, err := Render(*root)
	if err != nil {
		t.Errorf("render failed: %v\n", err)
	}

	if rendered != expecting {
		t.Errorf("Incorrect HTML:\n%s", rendered)
	}
}
