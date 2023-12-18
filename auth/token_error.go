package auth

import (
	"encoding/json"
	"errors"
)

type TokenError struct {
	Body         []byte `json:"-"`
	ErrorDetails struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

func (t TokenError) Error() string {
	return string(t.Body)
}

func TokenErrorFromBody(body []byte) *TokenError {
	t := &TokenError{}
	_ = json.Unmarshal(body, t)
	t.Body = body
	return t
}

func IsTokenExpiredError(err error) bool {
	if err == nil {
		return false
	}

	var te *TokenError
	if errors.As(err, &te) {
		return te.ErrorDetails.Code == 400 && te.ErrorDetails.Message == "TOKEN_EXPIRED"
	}

	return false
}
