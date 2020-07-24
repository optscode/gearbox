package gearbox

import (
	"strings"
)

type nodeType uint8

const (
	static nodeType = iota
	root
	parama
	catchAll
)

type node struct {
	path     string
	param    *node
	children map[string]*node
	nType    nodeType
	handlers handlersChain
}

func (n *node) addRoute(path string, handlers handlersChain) {
	currentNode := n
	originalPath := path
	path = path[1:]

	paramNames := make(map[string]bool, 0)

walk:
	for {
		pathLen := len(path)
		if pathLen == 0 {
			if currentNode.handlers != nil {
				panic("handlers are already registered for path '" + originalPath + "'")
			}
			currentNode.handlers = handlers
			break
		}

		segmentDelimter := strings.Index(path, "/")
		if segmentDelimter == -1 {
			segmentDelimter = pathLen
		}

		pathSegment := path[:segmentDelimter]
		if pathSegment[0] == ':' || pathSegment[0] == '*' {
			// Parameter
			if len(currentNode.children) > 0 {
				panic("parameter " + pathSegment + " conflicts with existing static children in path '" + originalPath + "'")
			}

			if currentNode.param != nil {
				if currentNode.param.path[0] == '*' {
					panic("parameter " + pathSegment + " conflicts with catch all (*) route in path '" + originalPath + "'")
				} else if currentNode.param.path != pathSegment {
					panic("parameter " + pathSegment + " in new path '" + originalPath +
						"' conflicts with existing wildcard '" + currentNode.param.path)
				}
			}

			if currentNode.param == nil {
				var nType nodeType
				if pathSegment[0] == '*' {
					nType = catchAll
					if pathLen > 1 {
						panic("catch all (*) routes are only allowed at the end of the path in path '" + originalPath + "'")
					}
				} else {
					nType = parama
					if _, ok := paramNames[pathSegment]; ok {
						panic("parameter " + pathSegment + " must be unique in path '" + originalPath + "'")
					} else {
						paramNames[pathSegment] = true
					}
				}

				currentNode.param = &node{
					path:     pathSegment,
					nType:    nType,
					children: make(map[string]*node, 0),
				}
			}
			currentNode = currentNode.param
		} else {
			// Static
			if currentNode.param != nil {
				panic(pathSegment + "' conflicts with existing parameter " +
					currentNode.param.path + " in path '" + originalPath + "'")
			}
			if child, ok := currentNode.children[pathSegment]; ok {
				currentNode = child

			} else {
				child = &node{
					path:     pathSegment,
					nType:    static,
					children: make(map[string]*node, 0),
				}
				currentNode.children[pathSegment] = child
				currentNode = child
			}
		}

		if pathLen > segmentDelimter {
			segmentDelimter++
		}
		path = path[segmentDelimter:]
		continue walk
	}
}

func (n *node) matchRoute(path string, ctx *context) handlersChain {
	pathLen := len(path)
	if pathLen > 0 && path[0] != '/' {
		return nil
	}

	currentNode := n
	path = path[1:]

walk:
	for {
		pathLen = len(path)

		if pathLen == 0 || currentNode.nType == catchAll {
			return currentNode.handlers
		}
		segmentDelimter := strings.Index(path, "/")
		if segmentDelimter == -1 {
			segmentDelimter = pathLen
		}
		pathSegment := path[:segmentDelimter]

		if pathLen > segmentDelimter {
			segmentDelimter++
		}
		path = path[segmentDelimter:]

		if currentNode.param != nil {
			currentNode = currentNode.param
			ctx.paramValues[currentNode.path[1:]] = pathSegment
			continue walk
		}

		if child, ok := currentNode.children[pathSegment]; ok {
			currentNode = child
			continue walk
		}

		return nil
	}
}

// createRootNode creates an instance of node with root type
func createRootNode() *node {
	return &node{
		nType:    root,
		path:     "/",
		children: make(map[string]*node, 0),
	}
}
