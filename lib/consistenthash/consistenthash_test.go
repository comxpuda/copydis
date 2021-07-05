package consistenthash

import (
	"fmt"
	"testing"
)

func TestHash(t *testing.T) {
	m := New(3, nil)
	m.AddNode("node1", "node2", "node3", "node4")
	fmt.Println(m.PickNode("foo"))
	fmt.Println(m.PickNode("bar"))
}
