package al64

import (
	"fmt"
	"strconv"
)

type ParsedIns struct {
	StructureOrder  []interface{}
	StructureByName map[string]interface{}
	BankFile        *ALBankFile
}

type deferredLink struct {
	forToken *Token
	link     func(interface{})
}

type parseState struct {
	tokens  []Token
	current int
	result  *ParsedIns
	inError bool
	errors  []ParseError
	links   []deferredLink
}

type ParseError struct {
	Token   *Token
	Message string
}

const validStructureNames = "keymap, envelope, sound, instrument, or bank"

func (state *parseState) optional(tokenType tokenType) *Token {
	var result = state.peek(0)

	if result.tType == tokenType {
		state.advance()
		return result
	} else {
		return nil
	}
}

func (state *parseState) require(tokenType tokenType, expected string) *Token {
	var result = state.peek(0)

	if result.tType == tokenType {
		state.advance()
		return result
	} else {
		if !state.inError {
			state.errors = append(state.errors, ParseError{result, fmt.Sprintf("Expected %s but got %s", expected, result.value)})
			state.inError = true
		}
		state.advance()
		return nil
	}
}

func (state *parseState) advance() {
	state.current = state.current + 1
}

func (state *parseState) hasMore() bool {
	return state.current < len(state.tokens)
}

func (state *parseState) peek(offset int) *Token {
	if state.current+offset >= 0 && state.current+offset < len(state.tokens) {
		return &state.tokens[state.current+offset]
	} else {
		return &Token{
			"EOF",
			0,
			0,
			0,
			tokenTypeEOF,
		}
	}
}

func (state *parseState) link(token *Token, callback func(interface{})) {
	state.links = append(state.links, deferredLink{
		token,
		callback,
	})
}

func parseAttribute(state *parseState) (*Token, *Token) {
	var name = state.require(tokenTypeIdentifier, "attribute name")

	if state.optional(tokenTypeEqual) != nil {
		var value = state.peek(0)
		state.advance()
		return name, value
	} else if state.optional(tokenTypeOpenParen) != nil {
		var value = state.peek(0)
		state.advance()
		state.require(tokenTypeCloseParen, ")")
		return name, value
	} else {
		return name, nil
	}
}

func parseNumberValue(state *parseState, token *Token, min int64, max int64) int64 {
	asInt, err := strconv.ParseInt(token.value, 10, 64)

	if err != nil {
		state.errors = append(state.errors, ParseError{token, "Exected number value"})
		return min
	} else if asInt < min || asInt > max {
		state.errors = append(state.errors, ParseError{
			token,
			fmt.Sprintf("Exected number value in the range %d and %d", int(min), int(max)),
		})

		if asInt < min {
			return min
		} else {
			return max
		}
	} else {
		return asInt
	}
}

func parseEnvelope(state *parseState) {
	var instrumentName = state.require(tokenTypeIdentifier, "envelope name")

	state.require(tokenTypeOpenCurly, "{")

	var result ALEnvelope

	var parsing = true

	for state.hasMore() && parsing {
		name, value := parseAttribute(state)

		if name.value == "attackTime" {
			result.AttackTime = int32(parseNumberValue(state, value, 0, 2147483647))
		} else if name.value == "attackVolume" {
			result.AttackVolume = uint8(parseNumberValue(state, value, 0, 127))
		} else if name.value == "decayTime" {
			result.DecayTime = int32(parseNumberValue(state, value, 0, 2147483647))
		} else if name.value == "decayVolume" {
			result.DecayVolume = uint8(parseNumberValue(state, value, 0, 127))
		} else if name.value == "releaseTime" {
			result.ReleaseTime = int32(parseNumberValue(state, value, 0, 2147483647))
		} else {
			state.errors = append(state.errors, ParseError{
				name,
				fmt.Sprintf("Unrecognized attribute '%s' for envelope", name.value),
			})
		}

		closeParen := state.optional(tokenTypeCloseCurly)

		if closeParen != nil {
			parsing = false
			state.inError = false
		}
	}

	state.result.StructureOrder = append(state.result.StructureOrder, &result)

	if instrumentName != nil {
		state.result.StructureByName[instrumentName.value] = &result
	}

	state.require(tokenTypeCloseCurly, "}")
}

func parseFile(state *parseState) {
	for state.hasMore() {
		var next = state.require(tokenTypeIdentifier, validStructureNames)

		if next != nil {
			if next.value == "envelope" {

			} else if next.value == "keymap" {

			} else if next.value == "sound" {

			} else if next.value == "instrument" {

			} else if next.value == "bank" {

			} else {
				state.errors = append(state.errors, ParseError{
					next,
					fmt.Sprintf("Expected %s but got %s", validStructureNames, next.value),
				})
			}
		}
	}
}

func ParseIns(input string) (*ParsedIns, []ParseError) {
	token := tokenizeInst(input)

	var state = parseState{
		token,
		0,
		&ParsedIns{
			nil,
			make(map[string]interface{}),
			&ALBankFile{nil},
		},
		false,
		nil,
		nil,
	}

	parseFile(&state)

	for _, link := range state.links {
		structure, has := state.result.StructureByName[link.forToken.value]

		if has {
			link.link(structure)
		} else {
			state.errors = append(state.errors, ParseError{
				link.forToken,
				fmt.Sprintf("Could not find %s", link.forToken.value),
			})
		}
	}

	return state.result, state.errors
}
