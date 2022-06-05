package goquery

// IsNull will be converted to `? IS NULL` filter.
func IsNull(val any) bool { return true }

func In[T any](val T, slice []T) bool { return true }
