package tool

import "regexp"

func IsDigit(val string) bool {
	k, _ := regexp.MatchString("^[0-9]+[.]{0,1}[0-9]*$", val)
	return k
}

func IsInteger(val string) bool {
	k, _ := regexp.MatchString("^[0-9]+$", val)
	return k
}

func IsNegativeInteger(val string) bool {
	k, _ := regexp.MatchString("^-[0-9]*$", val)
	return k
}

func IsDecimal(val string) bool {
	k, _ := regexp.MatchString("^[0-9]+.[0-9]+$", val)
	return k
}

func IsEmail(val string) bool {
	k, _ := regexp.MatchString("^\\w+@[a-z0-9]+\\.[a-z]{2,4}$", val)
	return k
}

func IsIPv4(val string) bool {
	k, _ := regexp.MatchString(`^(((\d{1,2})|(1\d{1,2})|(2[0-4]\d)|(25[0-5]))\.){3}((\d{1,2})|(1\d{1,2})|(2[0-4]\d)|(25[0-5]))$`, val)
	return k
}

func IsDate(val string) bool {
	k, _ := regexp.MatchString(`^\d{4}\-(0?[1-9]|[1][012])\-(0?[1-9]|[12][0-9]|3[01])$`, val)
	return k
}

func IsMonday(val string) bool {
	k, _ := regexp.MatchString(`^(0?[1-9]|[1][012])\-(0?[1-9]|[12][0-9]|3[01])$`, val)
	return k
}

func IsHourMin(val string) bool {
	k, _ := regexp.MatchString(`^(20|21|22|23|[0-1]\d):([0-5]\d)$`, val)
	return k
}

func IsTime(val string) bool {
	k, _ := regexp.MatchString(`^\d{4}[\-](0?[1-9]|1[012])[\-](0?[1-9]|[12][0-9]|3[01])(\s+(0?[0-9]|1[0-9]|2[0-3])\:(0?[0-9]|[1-5][0-9])\:(0?[0-9]|[1-5][0-9]))?$`, val)
	return k
}

func IsPhone(val string) bool {
	k, _ := regexp.MatchString("^1[3456789]\\d{9}$", val)
	return k
}
