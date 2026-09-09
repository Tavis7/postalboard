package main

import (
	"strings"
)

type htmlNode struct {
	name           string
	attributes     [][2]string
	attributeIndex map[string][]int
	children       []htmlNode
	text           string
}

type attribList [][2]string

func makeHTMLNode(name string, attributes attribList) htmlNode {
	result := htmlNode{}
	result.name = name
	result.appendAttributes(attributes...)
	return result
}

func (node *htmlNode) appendChildren(children ...htmlNode) {
	node.children = append(node.children, children...)
}

func (node *htmlNode) appendAttributes(attributes ...[2]string) {
	node.attributes = append(node.attributes, attributes...)
	oldLen := len(node.attributes)
	if node.attributeIndex == nil {
		node.attributeIndex = make(map[string][]int)
	}
	for k, v := range node.attributes[oldLen:] {
		node.attributeIndex[v[0]] = append(node.attributeIndex[v[0]], k)
	}
}

func renderHTMLNode(node htmlNode) string {
	if len(node.name) == 0 {
		return node.text
	} else {
		text := []string{}
		text = append(text, "<")
		text = append(text, node.name)
		for _, attribute := range node.attributes {
			text = append(text, " ")
			text = append(text, attribute[0])
			text = append(text, "=\"")
			text = append(text, attribute[1])
			text = append(text, "\"")
		}
		text = append(text, ">\n")
		for _, child := range node.children {
			text = append(text, renderHTMLNode(child))
			text = append(text, "\n")
		}
		// @todo render children
		text = append(text, "</")
		text = append(text, node.name)
		text = append(text, ">")
		return strings.Join(text, "")
	}
}
