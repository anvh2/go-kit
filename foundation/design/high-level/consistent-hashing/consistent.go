package consistenthashing

import (
	"hash/fnv"
	"sort"
)

type IConsistentHashing interface {
	AddNode(node Node)
	RemoveNode(node Node)
	RebalanceNodes()
	GetNode(key string) Node
}

type Node struct {
	IP   string
	Name string
	Keys []string
}

type ConsistentHashing struct {
	Nodes       map[uint32]Node
	SortedNodes []uint32
}

func NewConsistentHashing() *ConsistentHashing {
	return &ConsistentHashing{
		Nodes: make(map[uint32]Node),
	}
}

func (c *ConsistentHashing) AddNode(node Node) {
	id := hash(node.IP)
	c.Nodes[id] = node
	c.SortedNodes = append(c.SortedNodes, id)
	sort.Slice(c.SortedNodes, func(i, j int) bool {
		return c.SortedNodes[i] < c.SortedNodes[j]
	})
	c.RebalanceNodes()
}

func (c *ConsistentHashing) RemoveNode(node Node) {
	id := hash(node.IP)
	for i, v := range c.SortedNodes {
		if v == id {
			c.SortedNodes = append(c.SortedNodes[:i], c.SortedNodes[i+1:]...)
			break
		}
	}
	c.RebalanceNodes()
	delete(c.Nodes, id)
}

func (c *ConsistentHashing) RebalanceNodes() {
	for _, id := range c.SortedNodes {
		node := c.Nodes[id]
		for _, key := range node.Keys {
			c.GetNode(key)
		}
		node.Keys = nil
	}
}

func (c *ConsistentHashing) GetNode(key string) Node {
	if len(c.SortedNodes) == 0 {
		return Node{}
	}
	id := hash(key)
	idx := sort.Search(len(c.SortedNodes), func(i int) bool {
		return c.SortedNodes[i] >= id
	})
	if idx == len(c.SortedNodes) {
		idx = 0
	}
	node := c.Nodes[c.SortedNodes[idx]]
	node.Keys = append(node.Keys, key)
	return node
}

// Hash function using FNV-1a
func hash(ip string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(ip))
	return h.Sum32()
}

// // Hash function using MD5
// func hash(ip string) int {
// 	h := md5.Sum([]byte(ip))
// 	return int(h[0])<<24 | int(h[1])<<16 | int(h[2])<<8 | int(h[3])
// }
