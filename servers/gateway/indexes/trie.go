package indexes

import (
	"sort"
	"sync"
)

//TODO: implement a trie data structure that stores
//keys of type string and values of type int64

//////////////
// INT64SET //
//////////////

// the set of int64 values
type int64set map[int64]struct{}

// will add a value to the set
// returns true if the value didn't already exist in the set (and so was added).
// returns false if the value was already present in the set (nothing added).
func (s int64set) add(value int64) bool {

	// 'ok' is true if value is within set
	_, ok := s[value]
	if ok {
		return false // signifies nothing was inserted
	}

	// empty struct creation. The value in our set is mapped
	// to an empty struct because it takes of zero bytes.
	// but at least now the value is present in the set.
	// zero bytes thing is some black magic.
	s[value] = struct{}{}
	return true
}

// will remove a value from the set
// returns true if the value was in the set (and removed)
// false if not found in the set (nothing was removed)
func (s int64set) remove(value int64) bool {

	// 'ok' is true if value is within set
	_, ok := s[value]
	if !ok {
		return false // signifies nothing was removed
	}

	// otherwise it exists so you should remove it.
	delete(s, value)
	return true
}

// will check if the value is in the set.
// returns true if contained, false otherwise.
func (s int64set) has(value int64) bool {
	_, ok := s[value]
	return ok
}

func (s int64set) all() []int64 {
	keys := make([]int64, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	return keys
}

//////////////
// TRIENODE //
//////////////

//PRO TIP: if you are having troubles and want to see
//what your trie structure looks like at various points,
//either use the debugger, or try this package:
//https://github.com/davecgh/go-spew

//Trie implements a trie data structure mapping strings to int64s
//that is safe for concurrent use.
type TrieNode struct {
	children map[rune]*TrieNode
	values   int64set
	mx       sync.RWMutex
}

//NewTrie constructs a new Trie.
func NewTrieNode() *TrieNode {
	return &TrieNode{}
}

// Len returns the number of entries in the trie.
func (t *TrieNode) Len() int {
	t.mx.RLock()
	defer t.mx.RUnlock()
	return t.LenHelper()
}

// LenHelper is a helper function for Len()
func (t *TrieNode) LenHelper() int {
	entryCount := len(t.values)
	for child := range t.children {
		entryCount += t.children[child].LenHelper()
	}
	return entryCount
}

//Add adds a key and value to the trie.
func (t *TrieNode) Add(key string, value int64) {
	t.mx.Lock()
	defer t.mx.Unlock()
	runes := []rune(key)
	t.add(runes, value)
}

func (t *TrieNode) add(key []rune, value int64) {
	// if children of the curNode does not exist, make sure it is an empty set
	if len(t.children) == 0 {
		t.children = make(map[rune]*TrieNode)
	}

	// if the child does not exist, create a new trie node and store it there
	if t.children[key[0]] == nil {
		t.children[key[0]] = NewTrieNode()
	}

	// this is the last letter in the key
	if len(key) == 1 {
		// if the children of curNode do not exit, make sure it is an empty set
		if len(t.children[key[0]].values) == 0 {
			t.children[key[0]].values = make(map[int64]struct{})
		}

		// add the value and then return
		t.children[key[0]].values.add(value)
		return
	}

	// otherwise call the add method again on the child node recursively
	t.children[key[0]].add(key[1:len(key)], value)
}

//Find finds `max` values matching `prefix`. If the trie
//is entirely empty, or the prefix is empty, or max == 0,
//or the prefix is not found, this returns a nil slice.
func (t *TrieNode) Find(prefix string, max int) []int64 {
	t.mx.RLock()
	defer t.mx.RUnlock()

	if len(t.children) == 0 || prefix == "" || max <= 0 {
		return nil
	}

	// iterate through trie until at end of prefix
	prefixRunes := []rune(prefix)
	triePointer := t
	for _, s := range prefixRunes {
		if triePointer.children[s] == nil {
			return nil
		}
		triePointer = triePointer.children[s]
	}

	// create int64 slice
	var returnSlice []int64
	triePointer.findDFS(&returnSlice, max)
	return returnSlice
}

func (t *TrieNode) findDFS(list *[]int64, max int) {
	// add all current values in node to list (or until max is reached)
	values := t.values.all()
	canGet := max - len(*list)
	if len(values) > canGet {
		*list = append(*list, values[0:canGet]...)
		return
	}
	*list = append(*list, values...)
	// if max reached or no children, just return
	if len(*list) == max || len(t.children) == 0 {
		return
	}
	// sort children
	children := make([]rune, 0, len(t.children))
	for k := range t.children {
		children = append(children, k)
	}
	sort.Slice(children, func(i, j int) bool {
		return children[i] < children[j]
	})
	// for every child, recurse and add to lsit and check for max
	for _, child := range children {
		t.children[child].findDFS(list, max)
		if len(*list) == max {
			return
		}
	}
	return
}

//Remove removes a key/value pair from the trie
//and trims branches with no values.
func (t *TrieNode) Remove(key string, value int64) {
	t.mx.Lock()
	defer t.mx.Unlock()
	runes := []rune(key)
	t.remove(runes, value)
}

func (t *TrieNode) remove(key []rune, value int64) {

	// if we reached the end of the key, remove the value here.
	if len(key) == 0 {
		t.values.remove(value)
		return
	}
	focusChild := t.children[key[0]]
	focusChild.remove(key[1:], value)
	if len(focusChild.children) == 0 && len(focusChild.values) == 0 {
		delete(t.children, key[0])
	}
}
