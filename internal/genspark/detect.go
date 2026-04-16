package genspark

import "strings"

func IsRateLimit(body string) bool {
	body = strings.TrimSpace(body)
	b := strings.ToLower(body)
	return strings.Contains(b, "rate limit exceeded") || strings.Contains(b, "too many requests")
}

func IsFreeLimit(body string) bool {
	return strings.Contains(body, "free usage limit") || strings.Contains(body, "quota exceeded")
}

func IsNotLogin(body string) bool {
	b := strings.ToLower(body)
	return strings.Contains(b, `"not login"`) || strings.Contains(b, `"status":-5`) || strings.Contains(b, "login required")
}

func IsServerError(body string) bool {
	body = strings.TrimSpace(body)
	b := strings.ToLower(body)
	return b == "internal server error" || strings.Contains(b, "an error occurred with the current request")
}

func IsServerOverloaded(body string) bool {
	b := strings.ToLower(body)
	return strings.Contains(b, "server overloaded") || strings.Contains(b, "please try again later")
}

func IsRetiredCopilot(body string) bool {
	b := strings.ToLower(body)
	return strings.Contains(b, "this feature has been retired") && strings.Contains(b, "ai chat")
}

func IsServiceUnavailablePage(body string) bool {
	b := strings.ToLower(body)
	if !strings.Contains(b, "service unavailable") {
		return false
	}
	if !strings.Contains(b, "<title>genspark</title>") {
		return false
	}
	return true
}

func IsCloudflareChallenge(body string) bool {
	b := strings.ToLower(body)
	return strings.Contains(b, "just a moment") || strings.Contains(b, "_cf_chl_opt") || strings.Contains(b, "challenge-platform")
}

func IsCloudflareBlocked(body string) bool {
	return strings.Contains(body, "Sorry, you have been blocked")
}
