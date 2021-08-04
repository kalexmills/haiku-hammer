package dict

type TrieNode struct {
	isWord bool
	children [26]*TrieNode
}

func (n *TrieNode) insert(word string) {
	if len(word) == 0 {
		n.isWord = true
		return
	}

	idx := word[0] - 'A'
	if idx < 0 || idx > 26 {
		return
	}

	if child := n.children[idx]; child == nil {
		n.children[idx] = &TrieNode{}
	}
	n.children[idx].insert(word[1:])
}

func (n *TrieNode) HasPrefix(str string) bool {
	if n == nil {
		return false
	}
	if len(str) == 0 {
		return n.isWord
	}

	return n.Child(str[0]).HasPrefix(str[1:])
}

func (n *TrieNode) Child(ch byte) *TrieNode {
	idx := ch - 'A'
	if idx < 0 || idx > 26 {
		return nil
	}
	return n.children[idx]
}

func (n *TrieNode) IsWord() bool {
	return n.isWord
}