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
