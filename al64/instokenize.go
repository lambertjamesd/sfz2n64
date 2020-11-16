package al64

import "unicode"

type tokenType int

const (
	tokenTypeNone tokenType = iota
	tokenTypeError
	tokenTypeIdentifier
	tokenTypeDigit
	tokenTypeString
	tokenTypeOpenCurly
	tokenTypeCloseCurly
	tokenTypeOpenParen
	tokenTypeCloseParen
	tokenTypeOpenSquare
	tokenTypeCloseSquare
	tokenTypeEqual
	tokenTypeSemiColon
	tokenTypeComment
	tokenTypeWhitespace
)

type tokenizeState func(next rune) (tokenType, tokenizeState)

type tokenizer struct {
	characters   []rune
	currentState int
	state        tokenizeState
}

type token struct {
	value     string
	line      int
	start     int
	end       int
	tokenType tokenType
}

func tokenizeIdentifier(next rune) (tokenType, tokenizeState) {
	if unicode.IsLetter(next) || unicode.IsDigit(next) {
		return tokenTypeNone, tokenizeIdentifier
	} else {
		return tokenTypeIdentifier, tokenizeDefaultState(next)
	}
}

func tokenizeNumber(next rune) (tokenType, tokenizeState) {
	if unicode.IsDigit(next) {
		return tokenTypeNone, tokenizeIdentifier
	} else {
		return tokenTypeDigit, tokenizeDefaultState(next)
	}
}

func tokenizeError(next rune) (tokenType, tokenizeState) {
	return tokenTypeError, tokenizeDefaultState(next)
}

func tokenizeDefaultState(next rune) tokenizeState {
	if unicode.IsLetter(next) {
		return tokenizeIdentifier
	} else if unicode.IsDigit(next) {
		return tokenizeNumber
	} else {
		return tokenizeError
	}
}

func tokenizeInst(input string) []token {
	var result []token = nil

	return result
}
