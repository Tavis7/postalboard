package main

import ()

type htmlNode struct {
	name           string
	attributes     [][2]string
	attributeIndex map[string][]int
	children       []htmlNode
}

type attribList [][2]string

func makeHtmlNode(name string, attributes attribList) htmlNode {
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
