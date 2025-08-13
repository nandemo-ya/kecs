package utils

// Ptr returns a pointer to the given value.
// This is a generic function that works with any type.
//
// Example:
//   strPtr := utils.Ptr("hello")
//   intPtr := utils.Ptr(42)
//   boolPtr := utils.Ptr(true)
func Ptr[T any](v T) *T {
	return &v
}

// Deref safely dereferences a pointer, returning the value or the zero value if the pointer is nil.
//
// Example:
//   var strPtr *string = nil
//   str := utils.Deref(strPtr) // returns ""
//   
//   strPtr = utils.Ptr("hello")
//   str = utils.Deref(strPtr) // returns "hello"
func Deref[T any](ptr *T) T {
	if ptr == nil {
		var zero T
		return zero
	}
	return *ptr
}

// DerefOrDefault safely dereferences a pointer, returning the value or a default value if the pointer is nil.
//
// Example:
//   var strPtr *string = nil
//   str := utils.DerefOrDefault(strPtr, "default") // returns "default"
//   
//   strPtr = utils.Ptr("hello")
//   str = utils.DerefOrDefault(strPtr, "default") // returns "hello"
func DerefOrDefault[T any](ptr *T, defaultValue T) T {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}