package template

import (
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// Engine handles template compilation and execution with custom function maps
type Engine struct {
	funcMap        template.FuncMap
	leftDelimiter  string
	rightDelimiter string
}

// NewEngine creates a new template engine with all available functions and default delimiters
func NewEngine() *Engine {
	return NewEngineWithDelimiters("{{", "}}")
}

// NewEngineWithDelimiters creates a new template engine with custom delimiters
func NewEngineWithDelimiters(leftDelim, rightDelim string) *Engine {
	engine := &Engine{
		funcMap:        createFuncMap(),
		leftDelimiter:  leftDelim,
		rightDelimiter: rightDelim,
	}
	return engine
}

// createFuncMap builds the complete function map combining Sprig functions with custom ones
func createFuncMap() template.FuncMap {
	// Start with sprig functions (provides 100+ utility functions)
	funcMap := sprig.FuncMap()

	// Add our custom template functions
	customFuncs := template.FuncMap{
		"trimPrefix":   trimPrefix,
		"sleep":        sleep,
		"randFloat":    randFloat,
		"randChoice":   randChoice,
		"toJsonPretty": toJsonPretty,

		// Basic personal information
		"fakeName":           fakeName,
		"fakeFirstName":      fakeFirstName,
		"fakeLastName":       fakeLastName,
		"fakeEmail":          fakeEmail,
		"fakePhone":          fakePhone,
		"fakePhoneFormatted": fakePhoneFormatted,

		// Business and company data
		"fakeBS":            fakeBS,
		"fakeCompany":       fakeCompany,
		"fakeCompanySuffix": fakeCompanySuffix,
		"fakeJobTitle":      fakeJobTitle,
		"fakeJobDescriptor": fakeJobDescriptor,
		"fakeJobLevel":      fakeJobLevel,

		// Financial data
		"fakeCreditCardNumber": fakeCreditCardNumber,
		"fakeCreditCardType":   fakeCreditCardType,
		"fakeCurrency":         fakeCurrency,
		"fakeCurrencyLong":     fakeCurrencyLong,
		"fakeCurrencyAbbrv":    fakeCurrencyAbbrv,
		"fakeCurrencyName":     fakeCurrencyName,
		"fakePrice":            fakePrice,

		// Colors
		"fakeColor":     fakeColor,
		"fakeHexColor":  fakeHexColor,
		"fakeRGBColor":  fakeRGBColor,
		"fakeSafeColor": fakeSafeColor,

		// Product data
		"fakeProduct":            fakeProduct,
		"fakeProductName":        fakeProductName,
		"fakeProductDescription": fakeProductDescription,
		"fakeProductCategory":    fakeProductCategory,
		"fakeProductFeature":     fakeProductFeature,
		"fakeProductMaterial":    fakeProductMaterial,

		// Person details
		"fakeGender": fakeGender,
		"fakeSSN":    fakeSSN,
		"fakeHobby":  fakeHobby,

		// Authentication data
		"fakeUsername": fakeUsername,
		"fakePassword": fakePassword,

		// Address information
		"fakeAddress":      fakeAddress,
		"fakeStreet":       fakeStreet,
		"fakeStreetName":   fakeStreetName,
		"fakeStreetNumber": fakeStreetNumber,
		"fakeCity":         fakeCity,
		"fakeState":        fakeState,
		"fakeStateAbbrv":   fakeStateAbbrv,
		"fakeZip":          fakeZip,
		"fakeCountry":      fakeCountry,
		"fakeCountryAbbrv": fakeCountryAbbrv,
		"fakeLatitude":     fakeLatitude,
		"fakeLongitude":    fakeLongitude,

		// Words and text
		"fakeWord":                fakeWord,
		"fakeWords":               fakeWords,
		"fakeSentence":            fakeSentence,
		"fakeParagraph":           fakeParagraph,
		"fakeLoremIpsumWord":      fakeLoremIpsumWord,
		"fakeLoremIpsumSentence":  fakeLoremIpsumSentence,
		"fakeLoremIpsumParagraph": fakeLoremIpsumParagraph,

		// Food
		"fakeFood":      fakeFood,
		"fakeFruit":     fakeFruit,
		"fakeVegetable": fakeVegetable,
		"fakeBreakfast": fakeBreakfast,
		"fakeLunch":     fakeLunch,
		"fakeDinner":    fakeDinner,
		"fakeSnack":     fakeSnack,
		"fakeDessert":   fakeDessert,

		// Miscellaneous
		"fakeFlipACoin":  fakeFlipACoin,
		"fakeRandomBool": fakeRandomBool,
		"fakeUUID":       fakeUUID,

		// Internet values
		"fakeURL":          fakeURL,
		"fakeDomainName":   fakeDomainName,
		"fakeDomainSuffix": fakeDomainSuffix,
		"fakeIPv4Address":  fakeIPv4Address,
		"fakeIPv6Address":  fakeIPv6Address,
		"fakeMacAddress":   fakeMacAddress,
		"fakeHTTPMethod":   fakeHTTPMethod,
		"fakeUserAgent":    fakeUserAgent,

		// Date and Time
		"fakeDate":           fakeDate,
		"fakeDateRange":      fakeDateRange,
		"fakeFuture":         fakeFuture,
		"fakePast":           fakePast,
		"fakeWeekday":        fakeWeekday,
		"fakeMonth":          fakeMonth,
		"fakeMonthString":    fakeMonthString,
		"fakeYear":           fakeYear,
		"fakeHour":           fakeHour,
		"fakeMinute":         fakeMinute,
		"fakeSecond":         fakeSecond,
		"fakeNanoSecond":     fakeNanoSecond,
		"fakeTimeZone":       fakeTimeZone,
		"fakeTimeZoneAbbrv":  fakeTimeZoneAbbrv,
		"fakeTimeZoneFull":   fakeTimeZoneFull,
		"fakeTimeZoneOffset": fakeTimeZoneOffset,

		// Payment information
		"fakeCreditCard":        fakeCreditCard,
		"fakeAchRouting":        fakeAchRouting,
		"fakeAchAccount":        fakeAchAccount,
		"fakeBitcoinAddress":    fakeBitcoinAddress,
		"fakeBitcoinPrivateKey": fakeBitcoinPrivateKey,

		// Animals
		"fakeAnimal":     fakeAnimal,
		"fakeAnimalType": fakeAnimalType,
		"fakeFarmAnimal": fakeFarmAnimal,
		"fakeCat":        fakeCat,
		"fakeDog":        fakeDog,
		"fakeBird":       fakeBird,

		// Language
		"fakeLanguage":            fakeLanguage,
		"fakeLanguageAbbrv":       fakeLanguageAbbrv,
		"fakeProgrammingLanguage": fakeProgrammingLanguage,

		// Celebrities
		"fakeCelebrityActor":    fakeCelebrityActor,
		"fakeCelebrityBusiness": fakeCelebrityBusiness,
		"fakeCelebritySport":    fakeCelebritySport,

		// Books, Movies, and Songs
		"fakeBook":       fakeBook,
		"fakeBookTitle":  fakeBookTitle,
		"fakeBookAuthor": fakeBookAuthor,
		"fakeBookGenre":  fakeBookGenre,
		"fakeMovie":      fakeMovie,
		"fakeMovieName":  fakeMovieName,
		"fakeMovieGenre": fakeMovieGenre,
		"fakeSong":       fakeSong,
		"fakeMusicGenre": fakeMusicGenre,
	}

	// Merge custom functions into the sprig function map
	maps.Copy(funcMap, customFuncs)

	return funcMap
}

// CompileInlineTemplate compiles an inline template string with the engine's function map
func (e *Engine) CompileInlineTemplate(name, content string) (*template.Template, error) {
	if strings.TrimSpace(name) == "" {
		return nil, NewCompilationError("inline", "template name cannot be empty", nil)
	}

	if strings.TrimSpace(content) == "" {
		return nil, NewCompilationError("inline", "template content cannot be empty", nil)
	}

	tmpl, err := template.New(name).Delims(e.leftDelimiter, e.rightDelimiter).Funcs(e.funcMap).Parse(content)
	if err != nil {
		return nil, NewCompilationError("inline", fmt.Sprintf("failed to parse template: %v", err), err)
	}

	return tmpl, nil
}

// CompileFileTemplate compiles a template from a file with the engine's function map
func (e *Engine) CompileFileTemplate(filename string) (*template.Template, error) {
	if strings.TrimSpace(filename) == "" {
		return nil, NewCompilationError(filename, "filename cannot be empty", nil)
	}

	tmpl, err := template.New("").Delims(e.leftDelimiter, e.rightDelimiter).Funcs(e.funcMap).ParseFiles(filename)
	if err != nil {
		return nil, NewCompilationError(filename, fmt.Sprintf("failed to parse template file: %v", err), err)
	}

	return tmpl, nil
}

// ExecuteTemplate executes a template with the given context and writes the result to the writer
func (e *Engine) ExecuteTemplate(tmpl *template.Template, w io.Writer, ctx *TemplateContext) error {
	if tmpl == nil {
		return NewExecutionError("", "template is nil", nil)
	}

	if w == nil {
		return NewExecutionError(tmpl.Name(), "writer is nil", nil)
	}

	if ctx == nil {
		return NewExecutionError(tmpl.Name(), "context is nil", nil)
	}

	// Execute the template
	err := tmpl.Execute(w, ctx)
	if err != nil {
		return NewExecutionError(tmpl.Name(), fmt.Sprintf("template execution failed: %v", err), err)
	}

	return nil
}

// GetFuncMap returns a copy of the engine's function map
func (e *Engine) GetFuncMap() template.FuncMap {
	// Return a copy to prevent external modification
	funcMapCopy := make(template.FuncMap)
	for k, v := range e.funcMap {
		funcMapCopy[k] = v
	}
	return funcMapCopy
}

// BuildTemplateContext creates a complete template context from an HTTP request and route parameters
// This is a convenience function that wraps the existing NewTemplateContext function
func (e *Engine) BuildTemplateContext(req *http.Request, params map[string]string) (*TemplateContext, error) {
	if req == nil {
		return nil, NewContextError("request", "HTTP request cannot be nil", nil)
	}

	// Use the existing context builder which already handles all the complex parsing
	ctx, err := NewTemplateContext(req, params)
	if err != nil {
		return nil, NewContextError("context", "failed to build template context", err)
	}

	return ctx, nil
}

// CompileAndExecute is a convenience function that compiles a template and executes it in one step
func (e *Engine) CompileAndExecute(templateSource, templateContent string, w io.Writer, req *http.Request, params map[string]string) error {
	// Build the template context
	ctx, err := e.BuildTemplateContext(req, params)
	if err != nil {
		return fmt.Errorf("failed to build template context: %w", err)
	}

	// Determine if this is a file template or inline template
	var tmpl *template.Template
	if strings.TrimSpace(templateContent) != "" {
		// Inline template
		tmpl, err = e.CompileInlineTemplate(templateSource, templateContent)
		if err != nil {
			return fmt.Errorf("failed to compile inline template: %w", err)
		}
	} else {
		// File template
		tmpl, err = e.CompileFileTemplate(templateSource)
		if err != nil {
			return fmt.Errorf("failed to compile file template: %w", err)
		}
	}

	// Execute the template
	err = e.ExecuteTemplate(tmpl, w, ctx)
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}
