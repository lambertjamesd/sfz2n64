package al64

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type structureType interface {
	getTypeName() string
}

func (bank *ALBank) getTypeName() string {
	return "bank"
}

func (instrument *ALInstrument) getTypeName() string {
	return "instrument"
}

func (sound *ALSound) getTypeName() string {
	return "sound"
}

func (envelope *ALEnvelope) getTypeName() string {
	return "envelope"
}

func (keymap *ALKeyMap) getTypeName() string {
	return "keymap"
}

func (wavetable *ALWavetable) getTypeName() string {
	return "wavetable"
}

type ParsedIns struct {
	StructureOrder  []structureType
	StructureByName map[string]structureType
	BankFile        *ALBankFile
	TblData         []byte
}

type deferredLink struct {
	expectedType string
	forToken     *Token
	link         func(structureType) bool
}

type parseSource struct {
	source []rune
	name   string
}

type WaveTableLoader func(filename string) (*ALWavetable, error)

type parseState struct {
	source     *parseSource
	tokens     []Token
	current    int
	result     *ParsedIns
	inError    bool
	errors     []ParseError
	links      []deferredLink
	waveLoader WaveTableLoader
}

type ParseError struct {
	Token   *Token
	Message string
	source  *parseSource
}

func (source *parseSource) sourceContext(at int) (string, int) {
	var start = at
	var end = at

	for start > 0 && source.source[start-1] != '\n' {
		start--
	}

	for end < len(source.source) && source.source[end] != '\n' {
		end++
	}

	return string(source.source[start:end]), at - start
}

func (err ParseError) Error() string {
	contextString, col := err.source.sourceContext(err.Token.start)

	return fmt.Sprintf(
		"%s:%d:%d: %s\n%s\n%s^",
		err.source.name,
		int(err.Token.line+1),
		int(col+1),
		err.Message,
		contextString,
		strings.Repeat(" ", col),
	)
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
			state.errors = append(state.errors, ParseError{result, fmt.Sprintf("Expected %s but got %s", expected, result.value), state.source})
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
	return state.peek(0).tType != tokenTypeEOF
}

func (state *parseState) peek(offset int) *Token {
	if state.current+offset >= 0 && state.current+offset < len(state.tokens)-1 {
		return &state.tokens[state.current+offset]
	} else {
		return &state.tokens[len(state.tokens)-1]
	}
}

func (state *parseState) link(expectedType string, token *Token, callback func(structureType) bool) {
	state.links = append(state.links, deferredLink{
		expectedType,
		token,
		callback,
	})
}

func parseAttribute(state *parseState) (*Token, *Token, *Token) {
	var name = state.require(tokenTypeIdentifier, "attribute name")

	var index *Token = nil

	if state.optional(tokenTypeOpenSquare) != nil {
		index = state.peek(0)
		state.advance()
		state.require(tokenTypeCloseSquare, "]")
	}

	if state.optional(tokenTypeEqual) != nil {
		var value = state.peek(0)
		state.advance()
		return name, value, index
	} else if state.optional(tokenTypeOpenParen) != nil {
		var value = state.peek(0)
		state.advance()
		state.require(tokenTypeCloseParen, ")")
		return name, value, index
	} else {
		return name, nil, index
	}
}

func parseNumberValue(state *parseState, token *Token, min int64, max int64) int64 {
	asInt, err := strconv.ParseInt(token.value, 10, 64)

	if err != nil {
		state.errors = append(state.errors, ParseError{token, "Exected number value", state.source})
		return min
	} else if asInt < min || asInt > max {
		state.errors = append(state.errors, ParseError{
			token,
			fmt.Sprintf("Exected number value in the range %d and %d", int(min), int(max)),
			state.source,
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
		name, value, _ := parseAttribute(state)

		if name != nil {
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
					state.source,
				})
			}
		}

		state.optional(tokenTypeSemiColon)

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
}

func parseKeymap(state *parseState) {
	var instrumentName = state.require(tokenTypeIdentifier, "keymap name")

	state.require(tokenTypeOpenCurly, "{")

	var result ALKeyMap

	var parsing = true

	for state.hasMore() && parsing {
		name, value, _ := parseAttribute(state)

		if name != nil {
			if name.value == "velocityMin" {
				result.VelocityMin = uint8(parseNumberValue(state, value, 0, 127))
			} else if name.value == "velocityMax" {
				result.VelocityMax = uint8(parseNumberValue(state, value, 0, 127))
			} else if name.value == "keyMin" {
				result.KeyMin = uint8(parseNumberValue(state, value, 0, 127))
			} else if name.value == "keyMax" {
				result.KeyMax = uint8(parseNumberValue(state, value, 0, 127))
			} else if name.value == "keyBase" {
				result.KeyBase = uint8(parseNumberValue(state, value, 0, 127))
			} else if name.value == "detune" {
				result.Detune = uint8(parseNumberValue(state, value, -50, 50))
			} else {
				state.errors = append(state.errors, ParseError{
					name,
					fmt.Sprintf("Unrecognized attribute '%s' for keymap", name.value),
					state.source,
				})
			}
		}

		state.optional(tokenTypeSemiColon)

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
}

func parseSound(state *parseState) {
	var instrumentName = state.require(tokenTypeIdentifier, "sound name")

	state.require(tokenTypeOpenCurly, "{")

	var result ALSound

	var parsing = true

	for state.hasMore() && parsing {
		name, value, _ := parseAttribute(state)

		if name != nil {
			if name.value == "pan" {
				result.SamplePan = uint8(parseNumberValue(state, value, 0, 127))
			} else if name.value == "volume" {
				result.SampleVolume = uint8(parseNumberValue(state, value, 0, 127))
			} else if name.value == "keymap" {
				state.link("keymap", value, func(structure structureType) bool {
					asKeymap, ok := structure.(*ALKeyMap)

					if ok {
						result.KeyMap = asKeymap
					}

					return ok
				})
			} else if name.value == "envelope" {
				state.link("envelope", value, func(structure structureType) bool {
					asEnvelope, ok := structure.(*ALEnvelope)

					if ok {
						result.Envelope = asEnvelope
					}

					return ok
				})
			} else if name.value == "use" {
				var relativePath = stringFromToken(value.value)
				relativePath = strings.ReplaceAll(relativePath, "/", string(os.PathSeparator))
				relativePath = strings.ReplaceAll(relativePath, "\\", string(os.PathSeparator))
				var waveFilename = filepath.Join(filepath.Dir(state.source.name), relativePath)
				waveTable, err := state.waveLoader(waveFilename)

				if err != nil {
					state.errors = append(state.errors, ParseError{
						value,
						err.Error(),
						state.source,
					})
				} else {
					result.Wavetable = waveTable
				}
			} else {
				state.errors = append(state.errors, ParseError{
					name,
					fmt.Sprintf("Unrecognized attribute '%s' for sound", name.value),
					state.source,
				})
			}
		}

		state.optional(tokenTypeSemiColon)

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
}

func parseInstrument(state *parseState) {
	var instrumentName = state.require(tokenTypeIdentifier, "instrument name")

	state.require(tokenTypeOpenCurly, "{")

	var result ALInstrument

	var parsing = true

	for state.hasMore() && parsing {
		name, value, _ := parseAttribute(state)

		if name != nil {
			if name.value == "volume" {
				result.Volume = uint8(parseNumberValue(state, value, 0, 127))
			} else if name.value == "pan" {
				result.Pan = uint8(parseNumberValue(state, value, 0, 127))
			} else if name.value == "priority" {
				result.Priority = uint8(parseNumberValue(state, value, 0, 127))
			} else if name.value == "tremeloType" {
				result.TremType = uint8(parseNumberValue(state, value, 0, 256))
			} else if name.value == "tremeloRate" {
				result.TremRate = uint8(parseNumberValue(state, value, 0, 256))
			} else if name.value == "tremeloDepth" {
				result.TremDepth = uint8(parseNumberValue(state, value, 0, 256))
			} else if name.value == "tremeloDelay" {
				result.TremDelay = uint8(parseNumberValue(state, value, 0, 256))
			} else if name.value == "vibratoType" {
				result.VibType = uint8(parseNumberValue(state, value, 0, 256))
			} else if name.value == "vibratoRate" {
				result.VibRate = uint8(parseNumberValue(state, value, 0, 256))
			} else if name.value == "vibratoDepth" {
				result.VibDepth = uint8(parseNumberValue(state, value, 0, 256))
			} else if name.value == "vibratoDelay" {
				result.VibDelay = uint8(parseNumberValue(state, value, 0, 256))
			} else if name.value == "bendRange" {
				result.BendRange = int16(parseNumberValue(state, value, -0x8000, 0x7fff))
			} else if name.value == "sound" {
				state.link("sound", value, func(structure structureType) bool {
					asSound, ok := structure.(*ALSound)

					if ok {
						result.SoundArray = append(result.SoundArray, asSound)
					}

					return ok
				})
			} else {
				state.errors = append(state.errors, ParseError{
					name,
					fmt.Sprintf("Unrecognized attribute '%s' for instrument", name.value),
					state.source,
				})
			}
		}

		state.optional(tokenTypeSemiColon)

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
}

func parseBank(state *parseState) {
	var instrumentName = state.require(tokenTypeIdentifier, "bank name")

	state.require(tokenTypeOpenCurly, "{")

	var result ALBank

	var parsing = true

	for state.hasMore() && parsing {
		name, value, index := parseAttribute(state)

		if name != nil {
			if name.value == "sampleRate" {
				result.SampleRate = uint32(parseNumberValue(state, value, 0, 0xFFFFFFFF))
			} else if name.value == "percussionDefault" {
				state.link("instrument", value, func(structure structureType) bool {
					asInstrument, ok := structure.(*ALInstrument)

					if ok {
						result.Percussion = asInstrument
					}

					return ok
				})
			} else if name.value == "program" {
				indexAsInt, err := strconv.ParseInt(index.value, 10, 8)

				if err != nil || indexAsInt < 0 || indexAsInt > 127 {
					state.errors = append(state.errors, ParseError{
						index,
						"Expected a number between 0 and 127",
						state.source,
					})
				}

				state.link("instrument", value, func(structure structureType) bool {
					asInstrument, ok := structure.(*ALInstrument)

					if ok && indexAsInt >= 0 && indexAsInt <= 127 {
						for len(result.InstArray) <= int(indexAsInt) {
							result.InstArray = append(result.InstArray, nil)
						}

						result.InstArray[indexAsInt] = asInstrument
					}

					return ok
				})
			} else {
				state.errors = append(state.errors, ParseError{
					name,
					fmt.Sprintf("Unrecognized attribute '%s' for bank", name.value),
					state.source,
				})
			}
		}

		state.optional(tokenTypeSemiColon)

		closeParen := state.optional(tokenTypeCloseCurly)

		if closeParen != nil {
			parsing = false
			state.inError = false
		}
	}

	state.result.BankFile.BankArray = append(state.result.BankFile.BankArray, &result)

	state.result.StructureOrder = append(state.result.StructureOrder, &result)

	if instrumentName != nil {
		state.result.StructureByName[instrumentName.value] = &result
	}
}

func parseFile(state *parseState) {
	for state.hasMore() {
		var next = state.require(tokenTypeIdentifier, validStructureNames)

		if next != nil {
			if next.value == "envelope" {
				state.inError = false
				parseEnvelope(state)
			} else if next.value == "keymap" {
				state.inError = false
				parseKeymap(state)
			} else if next.value == "sound" {
				state.inError = false
				parseSound(state)
			} else if next.value == "instrument" {
				state.inError = false
				parseInstrument(state)
			} else if next.value == "bank" {
				state.inError = false
				parseBank(state)
			} else {
				state.errors = append(state.errors, ParseError{
					next,
					fmt.Sprintf("Expected %s but got %s", validStructureNames, next.value),
					state.source,
				})
			}
		}
	}
}

func ParseIns(input string, inputName string, loader WaveTableLoader) (*ParsedIns, []ParseError) {
	var characters = []rune(input)

	token := tokenizeInst(characters)

	var state = parseState{
		&parseSource{
			characters,
			inputName,
		},
		token,
		0,
		&ParsedIns{
			nil,
			make(map[string]structureType),
			&ALBankFile{nil},
			nil,
		},
		false,
		nil,
		nil,
		loader,
	}

	parseFile(&state)

	for _, link := range state.links {
		structure, has := state.result.StructureByName[link.forToken.value]

		if has {
			if !link.link(structure) {
				state.errors = append(state.errors, ParseError{
					link.forToken,
					fmt.Sprintf("Wrong type. Expected a %s got a %s", link.expectedType, structure.getTypeName()),
					state.source,
				})
			}
		} else {
			state.errors = append(state.errors, ParseError{
				link.forToken,
				fmt.Sprintf("Could not find %s", link.forToken.value),
				state.source,
			})
		}
	}

	state.result.TblData = TblFromBank(state.result.BankFile)

	return state.result, state.errors
}
