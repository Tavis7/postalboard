package htmlgen

import (
	"fmt"
	"html"
	"log"
	"strings"
)

type attribList [][2]string
type AttribList = attribList

type Node struct {
	name           string           // name of tag
	attributes     attribList       // id, class, etc.
	attributeIndex map[string][]int // used to check for duplicates
	children       []*Node          // child nodes
	text           string           // text for nodes named ""
	raw            bool             // disables escaping text
}

var blockish map[string]bool
var void map[string]bool

func MakeLeafNode(name string) *Node {
	result := Node{}
	result.name = name
	return &result
}

func MakeNode(name string, attributes attribList, children ...*Node) *Node {
	result := Node{}
	result.name = name
	for _, child := range children {
		if child == nil {
			log.Printf("Nil child when making %v: %v", result, children)
		}
	}
	result.appendAttributes(attributes...)
	result.AppendChildren(children...)
	return &result
}

func MakeTextNode(text string) *Node {
	result := Node{}
	result.text = text
	return &result
}

func (node *Node) AppendChildren(children ...*Node) {
	node.children = append(node.children, children...)
}

func (node *Node) appendAttributes(attributes ...[2]string) *Node {
	node.attributes = append(node.attributes, attributes...)
	oldLen := len(node.attributes)
	if node.attributeIndex == nil {
		node.attributeIndex = make(map[string][]int)
	}
	for k, v := range node.attributes[oldLen:] {
		node.attributeIndex[v[0]] = append(node.attributeIndex[v[0]], k)
	}
	return node
}

func renderHTMLNode(node *Node, sb *strings.Builder, indent int, newlined *bool) {
	if node == nil {
		log.Printf("Warning: node is nil: %v", node)
		log.Printf("Rendered so far: '%v'", sb.String())
		return
	}
	if len(node.name) == 0 {
		if len(node.attributes) > 0 {
			log.Printf("Warning: text element has attributes");
		}
		if len(node.children) > 0 {
			log.Printf("Warning: text element has children");
		}
		if *newlined {
			for _ = range indent {
				fmt.Fprint(sb, "    ")
			}
		}
		*newlined = false
		if !node.raw {
			fmt.Fprint(sb, html.EscapeString(node.text))
			return
		} else {
			fmt.Fprint(sb, node.text)
			return
		}
	} else {
		if len(node.text) > 0 {
			log.Printf("Warning: named element has text");
		}
		// @todo Turn off indentation for <pre> tags
		blocky := blockish[node.name]
		if blocky && !*newlined {
			fmt.Fprint(sb, "\n")
		}
		if blocky || *newlined {
			for _ = range indent {
				fmt.Fprint(sb, "    ")
			}
		}
		*newlined = blocky
		fmt.Fprint(sb, "<")
		fmt.Fprint(sb, node.name)
		for _, attribute := range node.attributes {
			fmt.Fprint(sb, " ")
			fmt.Fprint(sb, html.EscapeString(attribute[0]))
			fmt.Fprint(sb, "=\"")
			fmt.Fprint(sb, html.EscapeString(attribute[1]))
			fmt.Fprint(sb, "\"")
		}
		voidish := void[node.name]
		if voidish {
			if len(node.children) > 0 {
				log.Printf("Warning: void element has children");
			}
			fmt.Fprint(sb, " /")
		}
		fmt.Fprint(sb, ">")
		if blocky {
			fmt.Fprint(sb, "\n")
		}
		*newlined = blocky
		if !voidish {
			children := false
			for _, child := range node.children {
				children = true
				renderHTMLNode(child, sb, indent+1, newlined)
			}
			if blocky && !*newlined && children {
				fmt.Fprint(sb, "\n")
			}
			if blocky {
				for _ = range indent {
					fmt.Fprint(sb, "    ")
				}
			}
			fmt.Fprint(sb, "</")
			fmt.Fprint(sb, node.name)
			fmt.Fprint(sb, ">")
			if blocky {
				fmt.Fprint(sb, "\n")
				*newlined = true
			}
		}
	}
}

func Render(node Node) (string, error) {
	blockish = map[string]bool{
		"html": true,
		"head": true,
		"body": true,
		"div":  true,
		"p":    true,
		"ul":   true,
		"ol":   true,
		"li":   true,
		"form": true,
	}

	void = map[string]bool{
		"area":   true,
		"base":   true,
		"br":     true,
		"col":    true,
		"embed":  true,
		"hr":     true,
		"img":    true,
		"input":  true,
		"link":   true,
		"meta":   true,
		"param":  true,
		"source": true,
		"track":  true,
		"wbr":    true,
	}

	if strings.ToLower(node.name) != "html" {
		return "", fmt.Errorf("renderHTML called with non-<html> node")
	}
	var sb strings.Builder
	fmt.Fprint(&sb, "<!doctype html>\n")
	blocky := true
	renderHTMLNode(&node, &sb, 0, &blocky)
	result := sb.String()

	return result, nil
}
