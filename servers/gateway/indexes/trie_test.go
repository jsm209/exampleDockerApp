package indexes

import (
	"testing"
)

func TestAllint64Set(t *testing.T) {
	// Make a new int64set
	myset := int64set{}

	// Testing add
	if !myset.add(1) {
		t.Errorf("Failed to return 'true' after attempting to successfully add to the set.")
	}

	// Checking if it's there
	if !myset.has(1) {
		t.Errorf("After adding value to set, fails to find it.")
	}

	// Testing remove
	if !myset.remove(1) {
		t.Errorf("Did not remove correctly after adding.")
	}

	// Checking it it's removed
	if myset.has(1) {
		t.Errorf("Incorrectly found previously inserted element after attempted removal.")
	}
}

func TestAllTrie(t *testing.T) {

	// Made a trie
	mytrie := NewTrieNode()

	// Check that the length of this new trie is 0
	if mytrie.Len() != 0 {
		t.Errorf("Length of newly created trie is not 0.")
	}

	// add user to trie
	id := int64(1)
	username := "gob"
	firstname := "git"
	lastname := "go"
	mytrie.Add(username, id)
	mytrie.Add(firstname, id)
	mytrie.Add(lastname, id)

	// try to find the inserted user
	findResult := mytrie.Find(username, 1)
	if findResult[0] != id {
		t.Errorf("Failed to find the user after insertion.")
	}

	// Check that the length is now 1
	if mytrie.Len() != 3 {
		t.Errorf("Length after inserting 'gob', 'git', 'go', fails to be expected value of 3.")
	}

	// delete the inserted user
	mytrie.Remove(username, id)

	// try to find the inserted user again
	findResult2 := mytrie.Find(username, 1)
	if len(findResult2) != 0 {
		t.Errorf("Incorrectly found the user after deletion.")
	}

	// insert something deep into the trie with other things
	mytrie.Add("somethingDeep", 2)
	mytrie.Add("somethingDep", 3)
	mytrie.Add("somethingDeeep", 4)
	mytrie.Add("somethingDeepp", 5)

	// attempt to find it
	findResult3 := mytrie.Find("somethingDeep", 2)
	if findResult3[0] != 2 {
		t.Errorf("Failed to find the user deeper in the tree.")
	}
}
