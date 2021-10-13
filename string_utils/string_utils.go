package stringutils

import (
	"fmt"
	"regexp"
	"strings"
)

var charRegex = regexp.MustCompile(`[A-Za-z0-9]`)

// MaskEmail masks an email string,
// leaving only the first four letters of the email id (i.e the part before the '@') and the email domain unmasked.
// if the email id has 4 or less characters, leaves only 1 character unmasked.
func MaskEmail(email string) string {
	splitedEmail := strings.SplitN(email, "@", 2)
	if len(splitedEmail) < 2 {
		return email
	}

	emailDomain := splitedEmail[1]
	emailID := splitedEmail[0]
	if len(emailID) == 0 {
		return email
	}

	nUnmasked := 4
	if len(emailID) <= 4 {
		nUnmasked = 1
	}

	unmaskedID := emailID[:nUnmasked]
	maskedID := charRegex.ReplaceAllString(emailID[nUnmasked:], `*`)

	return fmt.Sprintf("%s%s@%s", unmaskedID, maskedID, emailDomain)
}

// MaskString masks the last half of a given string, changing any letter or number character to '*'
func MaskString(str string) string {
	strLen := len(str)

	unmaskedStr := str[:strLen/2]
	maskedStr := charRegex.ReplaceAllString(str[strLen/2:], `*`)

	return fmt.Sprintf("%s%s", unmaskedStr, maskedStr)
}
