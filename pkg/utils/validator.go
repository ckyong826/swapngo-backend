package utils

import (
	"regexp"
	"unicode"
)

// IsValidPassword 检查密码是否满足：至少8位，包含至少1个字母，1个数字，1个特殊符号
func IsValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	var hasLetter, hasNumber, hasSymbol bool

	for _, char := range password {
		switch {
		case unicode.IsLetter(char):
			hasLetter = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSymbol = true
		}
	}

	return hasLetter && hasNumber && hasSymbol
}

// IsValidPin 检查 PIN 码是否是严格的 4 位纯数字
func IsValidPin(pin string) bool {
	// 对于固定格式的纯数字，使用正则极其简单高效
	match, _ := regexp.MatchString(`^\d{4}$`, pin)
	return match
}