package flowpilot

import (
	"github.com/teamhanko/hanko/backend/flowpilot/jsonmanager"
	"regexp"
)

// InputType represents the type of the input field.
type InputType string

// Input types enumeration.
const (
	StringType   InputType = "string"
	EmailType    InputType = "email"
	NumberType   InputType = "number"
	PasswordType InputType = "password"
	JSONType     InputType = "json"
)

// Input defines the interface for input fields.
type Input interface {
	MinLength(minLength int) Input
	MaxLength(maxLength int) Input
	Required(b bool) Input
	Hidden(b bool) Input
	Preserve(b bool) Input
	Persist(b bool) Input
	ConditionalIncludeOnState(states ...StateName) Input
	CompareWithStash(b bool) Input

	setValue(value interface{}) Input
	setError(errType *ErrorType)
	getName() string
	shouldPersist() bool
	shouldPreserve() bool
	isIncludedOnState(stateName StateName) bool
	validate(stateName StateName, inputData jsonmanager.ReadOnlyJSONManager, stashData jsonmanager.JSONManager) bool
	toPublicInput() *PublicInput
}

// defaultExtraInputOptions holds additional input field options.
type defaultExtraInputOptions struct {
	preserveValue    bool
	persistValue     bool
	includeOnStates  []StateName
	compareWithStash bool
}

// DefaultInput represents an input field with its options.
type DefaultInput struct {
	name      string
	dataType  InputType
	value     interface{}
	minLength *int
	maxLength *int
	required  *bool
	hidden    *bool
	errorType *ErrorType

	defaultExtraInputOptions
}

// PublicInput represents an input field for public exposure.
type PublicInput struct {
	Name      string      `json:"name"`
	Type      InputType   `json:"type"`
	Value     interface{} `json:"value,omitempty"`
	MinLength *int        `json:"min_length,omitempty"`
	MaxLength *int        `json:"max_length,omitempty"`
	Required  *bool       `json:"required,omitempty"`
	Hidden    *bool       `json:"hidden,omitempty"`
	Error     *ErrorType  `json:"error,omitempty"`
}

// newInput creates a new DefaultInput instance with provided parameters.
func newInput(name string, t InputType, persistValue bool) Input {
	return &DefaultInput{
		name:     name,
		dataType: t,
		defaultExtraInputOptions: defaultExtraInputOptions{
			preserveValue:    false,
			persistValue:     persistValue,
			includeOnStates:  []StateName{},
			compareWithStash: false,
		},
	}
}

// StringInput creates a new input field of string type.
func StringInput(name string) Input {
	return newInput(name, StringType, true)
}

// EmailInput creates a new input field of email type.
func EmailInput(name string) Input {
	return newInput(name, EmailType, true)
}

// NumberInput creates a new input field of number type.
func NumberInput(name string) Input {
	return newInput(name, NumberType, true)
}

// PasswordInput creates a new input field of password type.
func PasswordInput(name string) Input {
	return newInput(name, PasswordType, false)
}

// JSONInput creates a new input field of JSON type.
func JSONInput(name string) Input {
	return newInput(name, JSONType, false)
}

// MinLength sets the minimum length for the input field.
func (i *DefaultInput) MinLength(minLength int) Input {
	i.minLength = &minLength
	return i
}

// MaxLength sets the maximum length for the input field.
func (i *DefaultInput) MaxLength(maxLength int) Input {
	i.maxLength = &maxLength
	return i
}

// Required sets whether the input field is required.
func (i *DefaultInput) Required(b bool) Input {
	i.required = &b
	return i
}

// Hidden sets whether the input field is hidden.
func (i *DefaultInput) Hidden(b bool) Input {
	i.hidden = &b
	return i
}

// Preserve sets whether the input field value should be preserved, so that the value is included in the response
// instead of being blanked out.
func (i *DefaultInput) Preserve(b bool) Input {
	i.preserveValue = b
	return i
}

// Persist sets whether the input field value should be persisted.
func (i *DefaultInput) Persist(b bool) Input {
	i.persistValue = b
	return i
}

// ConditionalIncludeOnState sets the states where the input field is included.
func (i *DefaultInput) ConditionalIncludeOnState(stateNames ...StateName) Input {
	i.includeOnStates = stateNames
	return i
}

// isIncludedOnState check if a conditional input field is included according to the given stateName.
func (i *DefaultInput) isIncludedOnState(stateName StateName) bool {
	if len(i.includeOnStates) == 0 {
		return true
	}

	for _, s := range i.includeOnStates {
		if s == stateName {
			return true
		}
	}

	return false
}

// CompareWithStash sets whether the input field is compared with stash values.
func (i *DefaultInput) CompareWithStash(b bool) Input {
	i.compareWithStash = b
	return i
}

// setValue sets the value for the input field for the current response.
func (i *DefaultInput) setValue(value interface{}) Input {
	i.value = &value
	return i
}

// getName returns the name of the input field.
func (i *DefaultInput) getName() string {
	return i.name
}

// setError sets an error to the given input field.
func (i *DefaultInput) setError(errType *ErrorType) {
	i.errorType = errType
}

// shouldPersist indicates the value should be persisted.
func (i *DefaultInput) shouldPersist() bool {
	return i.persistValue
}

// shouldPersist indicates the value should be preserved.
func (i *DefaultInput) shouldPreserve() bool {
	return i.preserveValue
}

// validate performs validation on the input field.
func (i *DefaultInput) validate(stateName StateName, inputData jsonmanager.ReadOnlyJSONManager, stashData jsonmanager.JSONManager) bool {
	// TODO: Replace with more structured validation logic.

	var inputValue *string
	var stashValue *string

	if v := inputData.Get(i.name); v.Exists() {
		inputValue = &v.Str
	}

	if v := stashData.Get(i.name); v.Exists() {
		stashValue = &v.Str
	}

	if len(i.includeOnStates) > 0 && !i.isIncludedOnState(stateName) {
		// skip validation
		return true
	}

	if i.required != nil && *i.required && (inputValue == nil || len(*inputValue) <= 0) {
		i.errorType = ValueMissingError
		return false
	}

	if i.compareWithStash && inputValue != nil && stashValue != nil && *inputValue != *stashValue {
		i.errorType = ValueInvalidError
		return false
	}

	if i.dataType == JSONType {
		// skip further validation
		return true
	}

	if i.minLength != nil {
		if len(*inputValue) < *i.minLength {
			i.errorType = ValueTooShortError
			return false
		}
	}

	if i.maxLength != nil {
		if len(*inputValue) > *i.maxLength {
			i.errorType = ValueTooLongError
			return false
		}
	}

	if i.dataType == EmailType {
		pattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if matched := pattern.MatchString(*inputValue); !matched {
			i.errorType = EmailInvalidError
			return false
		}
	}

	return true
}

// toPublicInput converts the DefaultInput to a PublicInput for public exposure.
func (i *DefaultInput) toPublicInput() *PublicInput {
	return &PublicInput{
		Name:      i.name,
		Type:      i.dataType,
		Value:     i.value,
		MinLength: i.minLength,
		MaxLength: i.maxLength,
		Required:  i.required,
		Hidden:    i.hidden,
		Error:     i.errorType,
	}
}
