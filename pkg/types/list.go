package types

import (
	"fmt"
	"reflect"
	"unsafe"
)

// ListHead represents a linked list head
type ListHead struct {
	Prev *ListHead
	Next *ListHead
}

// InitListHead initializes a list head
func InitListHead(list *ListHead) {
	list.Prev = list
	list.Next = list
}

// NewListHead creates a new initialized list head
func NewListHead() *ListHead {
	head := &ListHead{}
	InitListHead(head)
	return head
}

// ListAdd adds a new entry after the specified head
func ListAdd(entry *ListHead, head *ListHead) {
	__ListAdd(entry, head, head.Next)
}

// ListAddTail adds a new entry before the specified head (at the end of the list)
func ListAddTail(entry *ListHead, head *ListHead) {
	__ListAdd(entry, head.Prev, head)
}

// __ListAdd inserts a new entry between two known consecutive entries
func __ListAdd(entry *ListHead, prev *ListHead, next *ListHead) {
	entry.Prev = prev
	entry.Next = next
	prev.Next = entry
	next.Prev = entry
}

// ListDel deletes an entry from the list
func ListDel(entry *ListHead) {
	__ListDel(entry.Prev, entry.Next)
	entry.Prev = nil
	entry.Next = nil
}

// __ListDel deletes entry by connecting prev and next entries
func __ListDel(prev *ListHead, next *ListHead) {
	prev.Next = next
	next.Prev = prev
}

// ListIsLast checks if an entry is the last one
func ListIsLast(list, head *ListHead) bool {
	return list.Next == head
}

// IsListEmpty checks if a list is empty
func IsListEmpty(list *ListHead) bool {
	return list.Next == list
}

// __ListSplice joins two lists
func __ListSplice(list *ListHead, prev *ListHead, next *ListHead) {
	first := list.Next
	last := list.Prev

	first.Prev = prev
	prev.Next = first

	last.Next = next
	next.Prev = last
}

// ListSpliceTail joins two lists at the end
func ListSpliceTail(list *ListHead, head *ListHead) {
	if !IsListEmpty(list) {
		__ListSplice(list, head.Prev, head)
	}
}

// ListEntry is a macro replacement to get the struct containing a list head
func ListEntry(ptr *ListHead, container interface{}, member string) interface{} {
	return ContainerOf(ptr, container, member)
}

// ListFirstEntry gets the first entry in the list
func ListFirstEntry(head *ListHead, container interface{}, member string) interface{} {
	return ListEntry(head.Next, container, member)
}

// ListLastEntry gets the last entry in the list
func ListLastEntry(head *ListHead, container interface{}, member string) interface{} {
	return ListEntry(head.Prev, container, member)
}

// ListNextEntry gets the next entry in the list
func ListNextEntry(pos interface{}, member string) interface{} {
	// Reflection to get the member field (ListHead) from pos
	posValue := reflect.ValueOf(pos).Elem()
	memberField := posValue.FieldByName(member)

	if !memberField.IsValid() || memberField.Type() != reflect.TypeOf(&ListHead{}) {
		panic("ListNextEntry: invalid member field")
	}

	// Get the next list head
	nextHead := memberField.Interface().(*ListHead).Next

	// Get the container that has this list head
	return ListEntry(nextHead, pos, member)
}

// PPS::> TESTING
//
//	func ListNextEntryBB(pos *BufferBlock) *BufferBlock {
//		return ContainerOf(pos.List.Next, BufferBlock{}, "List").(*BufferBlock)
//	}
//
// Then use these directly
func ListNextEntryBB(pos *BufferBlock) *BufferBlock {
	if pos.List.Next == nil {
		return nil
	}
	return BufferBlockFromList(pos.List.Next)
}

func BufferBlockFromList(list *ListHead) *BufferBlock {
	if list == nil {
		return nil
	}
	offset := unsafe.Offsetof(BufferBlock{}.List)
	containerAddr := uintptr(unsafe.Pointer(list)) - offset
	return (*BufferBlock)(unsafe.Pointer(containerAddr))
}

// Helper functions for clean type conversions
// func BufferHeadFromList(list *ListHead) *BufferHead {
// 	if list == nil {
// 		return nil
// 	}
// 	offset := unsafe.Offsetof(BufferHead{}.List)
// 	containerAddr := uintptr(unsafe.Pointer(list)) - offset
// 	return (*BufferHead)(unsafe.Pointer(containerAddr))
// }

func BufferHeadFromList(list *ListHead) *BufferHead {
	if list == nil {
		return nil
	}

	offset := unsafe.Offsetof(BufferHead{}.List)
	containerAddr := uintptr(unsafe.Pointer(list)) - offset

	if containerAddr == 0 {
		return nil
	}

	return (*BufferHead)(unsafe.Pointer(containerAddr))
}

// ListPrevEntry gets the previous entry in the list
func ListPrevEntry(pos interface{}, member string) interface{} {
	// Reflection to get the member field (ListHead) from pos
	posValue := reflect.ValueOf(pos).Elem()
	memberField := posValue.FieldByName(member)

	if !memberField.IsValid() || memberField.Type() != reflect.TypeOf(&ListHead{}) {
		panic("ListPrevEntry: invalid member field")
	}

	// Get the prev list head
	prevHead := memberField.Interface().(*ListHead).Prev

	// Get the container that has this list head
	return ListEntry(prevHead, pos, member)
}

// ForEachInList is a helper for the list_for_each macro
func ForEachInList(f func(*ListHead) bool, head *ListHead) {
	for pos := head.Next; pos != head; pos = pos.Next {
		if !f(pos) {
			break
		}
	}
}

// ForEachInListSafe is a helper for the list_for_each_safe macro
func ForEachInListSafe(f func(*ListHead, *ListHead) bool, head *ListHead) {
	var n *ListHead
	for pos := head.Next; pos != head; pos = n {
		n = pos.Next
		if !f(pos, n) {
			break
		}
	}
}

// ForEachEntry is a helper for list_for_each_entry
// Usage requires type assertions since Go doesn't support C-style macros directly
// Example usage is shown in the package test cases
func ForEachEntry(head *ListHead, container interface{}, member string, f func(interface{}) bool) {
	pos := ListFirstEntry(head, container, member)

	for {
		posValue := reflect.ValueOf(pos).Elem()
		memberField := posValue.FieldByName(member)

		if !memberField.IsValid() {
			panic("ForEachEntry: invalid member field")
		}

		listHead := memberField.Interface().(*ListHead)
		if listHead == head {
			break
		}

		if !f(pos) {
			break
		}

		pos = ListNextEntry(pos, member)
	}
}

// ForEachEntryReverse iterates the list in reverse order
func ForEachEntryReverse(head *ListHead, container interface{}, member string, f func(interface{}) bool) {
	pos := ListLastEntry(head, container, member)

	for {
		posValue := reflect.ValueOf(pos).Elem()
		memberField := posValue.FieldByName(member)

		if !memberField.IsValid() {
			panic("ForEachEntryReverse: invalid member field")
		}

		listHead := memberField.Interface().(*ListHead)
		if listHead == head {
			break
		}

		if !f(pos) {
			break
		}

		pos = ListPrevEntry(pos, member)
	}
}

// ForEachEntrySafe iterates the list safely (allows removal during iteration)
func ForEachEntrySafe(head *ListHead, container interface{}, member string, f func(interface{}, interface{}) bool) {
	pos := ListFirstEntry(head, container, member)

	for {
		posValue := reflect.ValueOf(pos).Elem()
		memberField := posValue.FieldByName(member)

		if !memberField.IsValid() {
			panic("ForEachEntrySafe: invalid member field")
		}

		listHead := memberField.Interface().(*ListHead)
		if listHead == head {
			break
		}

		n := ListNextEntry(pos, member)

		if !f(pos, n) {
			break
		}

		pos = n
	}
}

// ContainerOf is a helper to get the containing struct from a member
func ContainerOf(ptr interface{}, containerType interface{}, memberName string) interface{} {
	// Get the pointer value
	ptrVal := reflect.ValueOf(ptr)
	if ptrVal.Kind() != reflect.Ptr && ptrVal.Kind() != reflect.UnsafePointer {
		panic("ContainerOf: ptr must be a pointer")
	}

	// Get the type of the container
	containerTypeVal := reflect.TypeOf(containerType)
	if containerTypeVal.Kind() == reflect.Ptr {
		containerTypeVal = containerTypeVal.Elem()
	}

	// Find the field offset
	var fieldOffset uintptr
	var found bool
	for i := 0; i < containerTypeVal.NumField(); i++ {
		field := containerTypeVal.Field(i)
		if field.Name == memberName {
			fieldOffset = field.Offset
			found = true
			break
		}
	}

	if !found {
		panic(fmt.Sprintf("ContainerOf: member '%s' not found in type %v",
			memberName, containerTypeVal))
	}

	// Calculate the container address
	var ptrAddr uintptr
	if ptrVal.Kind() == reflect.Ptr {
		ptrAddr = ptrVal.Pointer()
	} else {
		// For UnsafePointer
		ptrAddr = uintptr(ptrVal.Interface().(unsafe.Pointer))
	}
	containerAddr := ptrAddr - fieldOffset

	// Create a new container and set its address
	container := reflect.New(containerTypeVal).Interface()

	// Copy the memory from the calculated address to the new container
	dst := unsafe.Pointer(reflect.ValueOf(container).Pointer())
	*(*uintptr)(dst) = containerAddr

	return container
}

// ForEachEntrySafeWithPos is a more direct equivalent to list_for_each_entry_safe
// It iterates the list safely while providing direct access to pos and n
// This allows for a syntax closer to the original C macro
func ForEachEntrySafeWithPos(head *ListHead, container interface{}, member string) func() (interface{}, interface{}, bool) {
	// Initialize pos and n
	pos := ListFirstEntry(head, container, member)

	// Check if list is empty
	posValue := reflect.ValueOf(pos).Elem()
	memberField := posValue.FieldByName(member)
	listHead := memberField.Interface().(*ListHead)
	if listHead == head {
		return func() (interface{}, interface{}, bool) {
			return nil, nil, false
		}
	}

	n := ListNextEntry(pos, member)

	// Return an iterator function
	return func() (interface{}, interface{}, bool) {
		// If we've reached the end of the list
		posValue := reflect.ValueOf(pos).Elem()
		memberField := posValue.FieldByName(member)
		listHead := memberField.Interface().(*ListHead)
		if listHead == head {
			return nil, nil, false
		}

		// Save current pos and n
		curPos := pos
		curN := n

		// Update for next iteration
		pos = n

		// Check if we're at the end of the list
		posValue = reflect.ValueOf(pos).Elem()
		memberField = posValue.FieldByName(member)
		listHead = memberField.Interface().(*ListHead)
		if listHead == head {
			return curPos, curN, false
		}

		n = ListNextEntry(pos, member)

		return curPos, curN, true
	}
}
