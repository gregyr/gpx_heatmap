package main

import (
	"fmt"
	"log"
	"slices"
	"strings"
)

type Node struct {
	Name       string
	Text       string
	Attributes map[string]string
	Children   []Node
}

// only valid for html
var singleTagNodes []string = []string{"area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr"}

func ParseXML(xmlString string) (Node, error) {

	// filter wrong /
	xmlString = strings.ReplaceAll(xmlString, "\\/", "/")
	xmlString = strings.Trim(xmlString, " \n")

	stringNodes := strings.Split(xmlString, "<")

	nodeHierarchy := []Node{}
	depth := 0 //not needed really
	inCommentNode := false

	isXML := false

	for _, stringNode := range stringNodes {

		if len(stringNode) <= 2 {
			continue
		}
		if stringNode[0] == '?' { // xml header
			isXML = true
			continue
		}
		//full handling of comments
		if stringNode[0] == '!' {
			if strings.Contains(stringNode, "DOCTYPE html") {
				continue // do something at somepoint maybe i dont know
			} else if strings.Split(stringNode, ">")[0][len(strings.Split(stringNode, ">")[0])-1] != '-' || !strings.Contains(stringNode, ">") {
				inCommentNode = true
			} else {
				inCommentNode = false
			}
			continue
		} else if inCommentNode {
			// catch commented out nodes within a singular comment node
			beforeLastAngleBracket := strings.Split(stringNode, ">")[len(strings.Split(stringNode, ">"))-2]
			if beforeLastAngleBracket[len(beforeLastAngleBracket)-1] == '-' || !strings.Contains(stringNode, ">") {
				inCommentNode = false
			}
			continue
		} else if stringNode[0] == '/' {

			// handle normal closing tags
			depth--
			if len(nodeHierarchy) == 1 {
				return nodeHierarchy[0], nil
			}
			nodeHierarchy[depth-1].Children = append(nodeHierarchy[depth-1].Children, nodeHierarchy[len(nodeHierarchy)-1])
			nodeHierarchy = nodeHierarchy[:len(nodeHierarchy)-1]
			// fmt.Println(stringNode, len(nodeHierarchy))

		} else {

			depth++
			text := strings.Split(stringNode, ">")[1]

			// handle names depending on wheter it has text or not
			var name string
			if len(text) > 0 {
				name = strings.Split(strings.Split(stringNode, " ")[0], ">")[0]
			} else {
				name = strings.Split(stringNode, " ")[0]
				name = strings.ReplaceAll(name, ">", "")
			}

			name = strings.Trim(name, "/")

			// handle attribute collection
			attributes := make(map[string]string)
			var lastKey string
			for _, attr := range strings.Split(strings.Split(stringNode, ">")[0], " ")[1:] {
				key_value := strings.Split(attr, "=")
				if name == "li" {
				}
				if len(key_value) > 1 {
					lastKey = key_value[0]
					attributes[key_value[0]] = strings.Trim(key_value[1], "\"")
				} else {
					attributes[lastKey] += " " + strings.Trim(key_value[0], "\"")
				}
			}
			node := Node{Name: name, Text: text, Attributes: attributes, Children: []Node{}}

			// append single tag nodes immediately to parent nodes children
			if slices.Contains(singleTagNodes, name) && !isXML {
				depth--
				nodeHierarchy[depth-1].Children = append(nodeHierarchy[depth-1].Children, node)
			} else {
				nodeHierarchy = append(nodeHierarchy, node)
			}

			// for range depth {
			// 	fmt.Print("  ")
			// }
			// fmt.Println(node.Name, node.Attributes)
		}
	}

	log.Println("Remaining depth:", depth)
	for _, n := range nodeHierarchy {
		fmt.Print(n.Name)
		fmt.Print(", ")
	}
	fmt.Print("\n")
	return Node{}, fmt.Errorf("No valid XML structure found")
}

func printNodeTree(rootNode Node, depth int) {

	for range depth {
		fmt.Print("  ")
	}
	fmt.Println(rootNode.Name, rootNode.Attributes)

	if len(rootNode.Children) >= 1 {
		for _, child := range rootNode.Children {
			printNodeTree(child, depth+1)
		}
	}
}

func EvaluateXPath(rootNode Node, xPath string) []Node {

	//ignore first empty string
	instructions := strings.Split(xPath, "/")[1:]
	validNodes := []Node{rootNode}
	// if first instruction matches rootNode, skip first step
	if rootNode.Name == instructions[0] {
		instructions = instructions[1:]
	}
	// flags whether search is "global" or "local"
	doubleSlash := false

	for _, instruction := range instructions {

		newValidNodes := []Node{}

		//split instruction into the name and the conditions
		name, conditions := createNameAndConditions(instruction)

		if len(instruction) == 0 { // since the instructions are split by "/", between double slashes there are empty strings, through these you can detect the double slashes
			doubleSlash = true
			continue
		} else if doubleSlash {

			openNodes := []Node{}

			// check the previously valid nodes, and add their children,
			// in a case of /ul//ul for example, it does not collect the first ul as valid
			for _, node := range validNodes {
				for _, child := range node.Children {
					openNodes = append(openNodes, child)
				}
			}
			// checks every possible child of every possible node from the currently valid nodes onwards
			for len(openNodes) > 0 {
				var node Node
				node, openNodes = openNodes[0], openNodes[1:] //pops off first node

				// add all children
				for _, child := range node.Children {
					openNodes = append(openNodes, child)
				}
				//check current node
				newValidNodes = addNodeIfValid(newValidNodes, node, name, conditions)
			}
			doubleSlash = false

		} else {
			for _, node := range validNodes {
				for _, child := range node.Children {

					newValidNodes = addNodeIfValid(newValidNodes, child, name, conditions)
				}
			}
		}

		validNodes = newValidNodes
	}
	return validNodes
}

// prints a list of nodes by name
func printNodes(nodes []Node) {
	for _, n := range nodes {
		fmt.Print(n.Name, " ", n.Attributes)
		fmt.Print(" ")
	}
	fmt.Print("\n")
}

// splits an XPath instruction into the node-name and the conditions
func createNameAndConditions(instruction string) (string, map[string]string) {

	conditions := make(map[string]string)
	var name string
	var conditionStrings string
	if strings.Contains(instruction, "[") { // check whethere there are conditions Present
		name, conditionStrings = strings.Split(instruction, "[")[0], strings.Split(instruction, "[")[1]
	} else {
		name = instruction
	}

	// if conditionStrings is not initialized, ignore filling condition map
	if len(conditionStrings) > 0 {
		splitConds := strings.SplitSeq(conditionStrings, "and") // uses an iterator since it is more efficient apparently
		for cond := range splitConds {
			parts := strings.Split(cond, "=")
			if len(parts) > 1 {
				conditions[strings.Trim(parts[0], " ")] = strings.Trim(parts[1], " ]") //remove whitespace and closing brackets
			} else { // handle functions when no normal condition like functions
				log.Println("XPath: Missing Condition Body")
			}
		}
	}
	return name, conditions
}

// adds a node to a node list if it fulfills the given conditions and matches the node name
func addNodeIfValid(validNodes []Node, node Node, name string, conditions map[string]string) []Node {
	// append to valid nodes if node matches searched node
	matchesConditions := true
	for key, value := range conditions {
		if key[0] == '@' && node.Attributes[key[1:]] != value {
			matchesConditions = false
		} else if key == "text()" && node.Text != strings.Trim(value, "\"") {
			matchesConditions = false
		}
	}
	if node.Name == name && matchesConditions {
		return append(validNodes, node)
	} else {
		return validNodes
	}
}
