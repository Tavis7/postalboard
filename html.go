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

func renderHTMLNode(node *htmlNode, sb *strings.Builder) error {
	if len(node.name) == 0 {
		if !node.raw {
			fmt.Fprint(sb, html.EscapeString(node.text))
			return nil
		} else {
			fmt.Fprint(sb, node.text)
			return nil
		}
	} else {
		fmt.Fprint(sb, "<")
		fmt.Fprint(sb, node.name)
		for _, attribute := range node.attributes {
			fmt.Fprint(sb, " ")
			fmt.Fprint(sb, attribute[0])
			fmt.Fprint(sb, "=\"")
			fmt.Fprint(sb, attribute[1])
			fmt.Fprint(sb, "\"")
		}
		fmt.Fprint(sb, ">")
		for _, child := range node.children {
			err := renderHTMLNode(child, sb)
			if err != nil {
				return err
			}
		}
		fmt.Fprint(sb, "</")
		fmt.Fprint(sb, node.name)
		fmt.Fprint(sb, ">")
		return nil
	}
}

func renderHTML(node htmlNode) (string, error) {
	if strings.ToLower(node.name) != "html" {
		return "", fmt.Errorf("renderHTML called with non-<html> node")
	}
	var sb strings.Builder
	fmt.Fprint(&sb, "<!doctype html>\n")
	err := renderHTMLNode(&node, &sb)
	result := sb.String()

	return result, err
}
