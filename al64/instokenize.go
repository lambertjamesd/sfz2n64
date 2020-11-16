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
	value string
	line  int
	start int
	end   int
	tType tokenType
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

func singleToken(tType tokenType) tokenizeState {
	return func(next rune) (tokenType, tokenizeState) {
		return tType, tokenizeDefaultState(next)
	}
}

func tokenizeString(next rune) (tokenType, tokenizeState) {
	if next == '\\' {
		return tokenTypeNone, tokenizeStringEscape
	} else if next == '"' {
		return tokenTypeString, tokenizeDefaultState(next)
	} else {
		return tokenTypeNone, tokenizeString
	}
}

func tokenizeStringEscape(next rune) (tokenType, tokenizeState) {
	return tokenTypeNone, tokenizeString
}

func tokenizeCommentSingle(next rune) (tokenType, tokenizeState) {
	if next == '\n' {
		return tokenTypeComment, tokenizeWhitespace
	} else {
		return tokenTypeNone, tokenizeCommentSingle
	}
}

func tokenizeCommentMulti(next rune) (tokenType, tokenizeState) {
	if next == '*' {
		return tokenTypeNone, tokenizeCommentMultiEnd
	} else {
		return tokenTypeNone, tokenizeCommentMulti
	}
}

func tokenizeCommentMultiEnd(next rune) (tokenType, tokenizeState) {
	if next == '/' {
		return tokenTypeComment, tokenizeDefaultState(next)
	} else if next == '*' {
		return tokenTypeNone, tokenizeCommentMultiEnd
	} else {
		return tokenTypeNone, tokenizeCommentMulti
	}
}

func tokenizeCommentStart(next rune) (tokenType, tokenizeState) {
	if next == '/' {
		return tokenTypeNone, tokenizeCommentSingle
	} else if next == '*' {
		return tokenTypeNone, tokenizeCommentMulti
	} else {
		return tokenTypeError, tokenizeDefaultState(next)
	}
}

func tokenizeWhitespace(next rune) (tokenType, tokenizeState) {
	if unicode.IsSpace(next) {
		return tokenTypeNone, tokenizeWhitespace
	} else {
		return tokenTypeWhitespace, tokenizeDefaultState(next)
	}
}

func tokenizeDefaultState(next rune) tokenizeState {
	if next == '{' {
		return singleToken(tokenTypeOpenCurly)
	} else if next == '}' {
		return singleToken(tokenTypeCloseCurly)
	} else if next == '(' {
		return singleToken(tokenTypeOpenParen)
	} else if next == ')' {
		return singleToken(tokenTypeCloseParen)
	} else if next == '[' {
		return singleToken(tokenTypeOpenSquare)
	} else if next == ']' {
		return singleToken(tokenTypeCloseSquare)
	} else if next == '=' {
		return singleToken(tokenTypeEqual)
	} else if next == ';' {
		return singleToken(tokenTypeSemiColon)
	} else if next == '"' {
		return tokenizeString
	} else if next == '/' {
		return tokenizeCommentStart
	} else if unicode.IsSpace(next) {
		return tokenizeWhitespace
	} else if unicode.IsLetter(next) {
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
