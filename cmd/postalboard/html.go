package main

import (
	"fmt"
	"html"
	"strings"
)

type attribList [][2]string

type htmlNode struct {
	name           string
	attributes     attribList
	attributeIndex map[string][]int
	children       []*htmlNode
	text           string
	raw            bool
}

var blockish map[string]bool
var void map[string]bool

func makeHTMLNode(name string, attributes attribList) *htmlNode {
	result := htmlNode{}
	result.name = name
	result.appendAttributes(attributes...)
	return &result
}

func makeHTMLNode2(name string, attributes attribList, children ...*htmlNode) *htmlNode {
	result := htmlNode{}
	result.name = name
	result.appendAttributes(attributes...)
	result.appendChildren(children...)
	return &result
}

func makeHTMLTextNode(text string) *htmlNode {
	result := htmlNode{}
	result.text = text
	return &result
}

func (node *htmlNode) appendChildren(children ...*htmlNode) {
	node.children = append(node.children, children...)
}

func (node *htmlNode) appendChild(child *htmlNode) *htmlNode {
	node.children = append(node.children, child)
	return node
}

func (node *htmlNode) appendNode(name string, attribs attribList, children ...*htmlNode) *htmlNode {
	result := makeHTMLNode(name, attribs)
	result.appendChildren(children...)
	node.appendChild(result)
	return result
}

func (node *htmlNode) appendAttributes(attributes ...[2]string) *htmlNode {
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

func renderHTMLNode(node *htmlNode, sb *strings.Builder, indent int, newlined *bool) error {
	if len(node.name) == 0 {
		if *newlined {
			for _ = range indent {
				fmt.Fprint(sb, "    ")
			}
		}
		*newlined = false
		if !node.raw {
			fmt.Fprint(sb, html.EscapeString(node.text))
			return nil
		} else {
			fmt.Fprint(sb, node.text)
			return nil
		}
	} else {
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
				return fmt.Errorf("void element has children")
			}
			if len(node.text) > 0 {
				return fmt.Errorf("void element has text")
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
				err := renderHTMLNode(child, sb, indent+1, newlined)
				if err != nil {
					return err
				}
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
		return nil
	}
}

func renderHTML(node htmlNode) (string, error) {
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
	err := renderHTMLNode(&node, &sb, 0, &blocky)
	result := sb.String()

	return result, err
}
