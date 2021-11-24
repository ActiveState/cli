package p

func StrP(v string) *string {
	return &v
}

func PStr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func BoolP(v bool) *bool {
	return &v
}

func PBool(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}
