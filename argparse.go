package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type ArgParameter interface {
	Names() []string
	ArgCount() int
	HelpMessage() string
	ValidateAndParse(usedName string, args []string) (interface{}, error)
	DefaultValue() interface{}
}

func NewArgs(defaultUsageExample string) Args {
	return Args{
		nil,
		defaultUsageExample,
		make(map[string]ArgParameter),
	}
}

type Args struct {
	args                []ArgParameter
	defaultUsageExample string
	flagNameToArg       map[string]ArgParameter
}

func (args *Args) AddStringArg(names []string, helpMessage string, defaultValue string) {
	var arg = &stringArg{names, helpMessage, defaultValue}
	args.args = append(args.args, arg)
	for _, name := range names {
		args.flagNameToArg[name] = arg
	}
}

func (args *Args) AddIntegerArg(names []string, helpMessage string, defaultValue int64, minValue int64, maxValue int64) {
	var arg = &integerArg{names, helpMessage, defaultValue, minValue, maxValue}
	args.args = append(args.args, arg)
	for _, name := range names {
		args.flagNameToArg[name] = arg
	}
}

func (args *Args) AddFloatArg(names []string, helpMessage string, defaultValue float64, minValue float64, maxValue float64) {
	var arg = &floatArg{names, helpMessage, defaultValue, minValue, maxValue}
	args.args = append(args.args, arg)
	for _, name := range names {
		args.flagNameToArg[name] = arg
	}
}

func (args *Args) AddFlagArg(names []string, helpMessage string) {
	var arg = &flagArg{names, helpMessage}
	args.args = append(args.args, arg)
	for _, name := range names {
		args.flagNameToArg[name] = arg
	}
}

func (args *Args) CreateHelpMessage() string {
	var result = []string{args.defaultUsageExample}

	result = append(result, "")

	for _, arg := range args.args {
		result = append(result, fmt.Sprintf("    %s %s", strings.Join(arg.Names(), ", "), arg.HelpMessage()))
	}

	return strings.Join(result, "\n")
}

func (args *Args) Parse(stringArgs []string) (map[string]interface{}, []string, []error) {
	var namedArgs = make(map[string]interface{})
	var listArguments []string = nil
	var errs []error = nil

	for index := 0; index < len(stringArgs); {
		var current = stringArgs[index]
		index++

		argParam, ok := args.flagNameToArg[current]

		if ok {
			var maxActualArgs = len(stringArgs) - index
			if maxActualArgs >= argParam.ArgCount() {
				value, err := argParam.ValidateAndParse(current, stringArgs[index:index+argParam.ArgCount()])

				if err != nil {
					errs = append(errs, err)
				} else {
					for _, name := range argParam.Names() {
						namedArgs[name] = value
					}
				}
			} else {
				errs = append(errs, errors.New(fmt.Sprintf("%s expects %d args, got %d", current, argParam.ArgCount(), maxActualArgs)))
			}
		} else if current[0] == '-' {
			errs = append(errs, errors.New(fmt.Sprintf("Unknown parameter %s", current)))
		} else {
			listArguments = append(listArguments, current)
		}
	}

	for name, arg := range args.flagNameToArg {
		if _, has := namedArgs[name]; !has {
			namedArgs[name] = arg.DefaultValue()
		}
	}

	return namedArgs, listArguments, errs
}

type stringArg struct {
	names        []string
	helpMessage  string
	defaultValue string
}

func (arg *stringArg) Names() []string {
	return arg.names
}

func (arg *stringArg) ArgCount() int {
	return 1
}

func (arg *stringArg) HelpMessage() string {
	return arg.helpMessage
}

func (arg *stringArg) ValidateAndParse(usedName string, args []string) (interface{}, error) {
	return args[0], nil
}

func (arg *stringArg) DefaultValue() interface{} {
	return arg.defaultValue
}

type integerArg struct {
	names        []string
	helpMessage  string
	minValue     int64
	maxValue     int64
	defaultValue int64
}

func (arg *integerArg) Names() []string {
	return arg.names
}

func (arg *integerArg) ArgCount() int {
	return 1
}

func (arg *integerArg) HelpMessage() string {
	return arg.helpMessage
}

func (arg *integerArg) ValidateAndParse(usedName string, args []string) (interface{}, error) {
	asInt, err := strconv.ParseInt(args[0], 10, 64)

	if err != nil || asInt < arg.minValue || asInt > arg.maxValue {
		return nil, errors.New(fmt.Sprintf("%s should be an integer in the range [%d, %d]", usedName, arg.minValue, arg.maxValue))
	}

	return asInt, nil
}

func (arg *integerArg) DefaultValue() interface{} {
	return arg.defaultValue
}

type floatArg struct {
	names        []string
	helpMessage  string
	minValue     float64
	maxValue     float64
	defaultValue float64
}

func (arg *floatArg) Names() []string {
	return arg.names
}

func (arg *floatArg) ArgCount() int {
	return 1
}

func (arg *floatArg) HelpMessage() string {
	return arg.helpMessage
}

func (arg *floatArg) ValidateAndParse(usedName string, args []string) (interface{}, error) {
	asInt, err := strconv.ParseFloat(args[0], 64)

	if err != nil || asInt < arg.minValue || asInt > arg.maxValue {
		return nil, errors.New(fmt.Sprintf("%s should be an number in the range [%g, %g]", usedName, arg.minValue, arg.maxValue))
	}

	return asInt, nil
}

func (arg *floatArg) DefaultValue() interface{} {
	return arg.defaultValue
}

type flagArg struct {
	names       []string
	helpMessage string
}

func (arg *flagArg) Names() []string {
	return arg.names
}

func (arg *flagArg) ArgCount() int {
	return 0
}

func (arg *flagArg) HelpMessage() string {
	return arg.helpMessage
}

func (arg *flagArg) ValidateAndParse(usedName string, args []string) (interface{}, error) {
	return true, nil
}

func (arg *flagArg) DefaultValue() interface{} {
	return false
}
