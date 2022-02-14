/*
Copyright 2022 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package print

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

type Node struct {
	Name        string
	OutVertices []*Node
	InVertices  []*Node
}

func NewNode(name string) *Node {
	return &Node{
		Name:        name,
		OutVertices: []*Node{},
		InVertices:  []*Node{},
	}
}
func (n *Node) VertLen() int {
	return len(n.OutVertices)
}

func (n *Node) Insert(name string) *Node {
	v := NewNode(name)
	n.OutVertices = append(n.OutVertices, v)
	return v
}

func (n Node) String() string {
	return fmt.Sprintf("| %s |", n.Name)
}
func PPrint(n *Node) {
	fmt.Println(SPPrint(n))
}
func SPPrint(n *Node) string {
	var b strings.Builder
	output := toAscii(n)
	if len(n.OutVertices) > 0 {
		children := [][]string{}
		for _, v := range n.OutVertices {
			children = append(children, toAscii(v))
		}
		m := int(math.Floor(float64(len(children) / 2)))
		vertical := "  "
		for i, n := range children {
			printNode := false
			if i == m {
				printNode = true
			}
			for j, l := range n {
				left := func() string {
					r := utf8.RuneCountInString(output[j])
					left := strings.Repeat(" ", r)
					if printNode {
						left = output[j]
					}
					return left
				}()
				mid := func() string {
					// ┗ ┏
					if j == 1 {
						switch {
						case len(children) == 1:
							return "━━━━▶"
						case len(children) == 2:
							if i == 0 {
								vertical = " ┃"
								return " ┏━━▶"
							} else {
								vertical = "  "
								return "━┻━━▶"
							}
						case i == 0:
							vertical = " ┃"
							return " ┏━━▶"
						case i == m:
							vertical = " ┃"
							return "━╋━━▶"
						case i < len(children)-1:
							vertical = " ┃"
							return " ┣━━▶"
						default:
							vertical = "  "
							return " ┗━━▶"
						}
					} else {
						return fmt.Sprintf("%s   ", vertical)
					}
				}()
				fmt.Fprintf(&b, "%s%s%s\n", left, mid, l)
			}
		}

	} else {
		for _, l := range output {
			fmt.Fprintf(&b, l)
		}
	}
	return b.String()
}

func toAscii(g *Node) []string {
	output := make([]string, 0, 0)
	//if len(g.OutVertices) == 0 {
	output = append(output, NewCellTop(g.Name))
	//output = append(output, "┃      ┃")
	output = append(output, NewCellText(g.Name))
	//output = append(output, "┃      ┃")
	output = append(output, NewCellLower(g.Name))
	//}
	return output
}
func NewCellLower(text string) string {
	s := strings.Repeat("━", len(text)+4)
	return fmt.Sprintf("┗%s┛", s)
}
func NewCellText(text string) string {
	return fmt.Sprintf("┃  %s  ┃", text)
}

func NewCellTop(text string) string {
	s := strings.Repeat("━", len(text)+4)
	return fmt.Sprintf("┏%s┓", s)
}
