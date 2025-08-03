package template

import (
	"encoding/json"
	"math/rand"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

// trimPrefix removes a prefix from a string (arguments reversed from strings.TrimPrefix for pipeline usage)
func trimPrefix(prefix, s string) string {
	return strings.TrimPrefix(s, prefix)
}

// sleep introduces a delay for timeout testing with context awareness
// Usage in templates: {{ sleep "200ms" }} or {{ sleep 1 }} (for 1 second)
func sleep(duration interface{}) string {
	var d time.Duration

	switch v := duration.(type) {
	case string:
		if parsed, err := time.ParseDuration(v); err == nil {
			d = parsed
		}
	case int:
		d = time.Duration(v) * time.Second
	case float64:
		d = time.Duration(v*1000) * time.Millisecond
	}

	if d > 0 {
		// Context-aware sleep that can be interrupted
		// We'll use a simple timer approach that can be cancelled
		timer := time.NewTimer(d)
		defer timer.Stop()

		// For now, this still completes the full duration
		// In a full implementation, we'd need the request context here
		<-timer.C
	}

	return "" // Return empty string so it doesn't affect template output
}

// randFloat generates a random float64 between min and max (inclusive)
// Usage in templates: {{ randFloat 1.0 10.0 }} or {{ randFloat 0 1 }}
// Takes the same parameters as sprig's randInt but returns a float64
func randFloat(min, max interface{}) float64 {
	minFloat := toFloat64(min)
	maxFloat := toFloat64(max)

	// Ensure min <= max
	if minFloat > maxFloat {
		minFloat, maxFloat = maxFloat, minFloat
	}

	// Generate random float between 0 and 1, then scale to range
	randomValue := rand.Float64()
	return minFloat + randomValue*(maxFloat-minFloat)
}

// randChoice randomly selects one value from the provided options of any type
// Usage in templates: {{ randChoice "red" "green" "blue" }} or {{ randChoice 1 2 3 }} or {{ randChoice 1.5 "text" true }}
func randChoice(choices ...interface{}) interface{} {
	if len(choices) == 0 {
		return nil
	}

	if len(choices) == 1 {
		return choices[0]
	}

	// Generate random index
	randomIndex := rand.Intn(len(choices))
	return choices[randomIndex]
}

// toFloat64 converts template-compatible numeric types to float64
// In Go templates, numeric literals are parsed as int or float64
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	default:
		// For unsupported types, return 0
		return 0
	}
}

// toJsonPretty converts any value to pretty-printed JSON with indentation
// Usage in templates: {{ .Body | toJsonPretty }} or {{ .Headers | toJsonPretty }}
func toJsonPretty(v any) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// Fake data generation functions using gofakeit

// Basic personal information
func fakeName() string           { return gofakeit.Name() }
func fakeFirstName() string      { return gofakeit.FirstName() }
func fakeLastName() string       { return gofakeit.LastName() }
func fakeEmail() string          { return gofakeit.Email() }
func fakePhone() string          { return gofakeit.Phone() }
func fakePhoneFormatted() string { return gofakeit.PhoneFormatted() }

// Business and company data
func fakeBS() string            { return gofakeit.BS() }
func fakeCompany() string       { return gofakeit.Company() }
func fakeCompanySuffix() string { return gofakeit.CompanySuffix() }
func fakeJobTitle() string      { return gofakeit.JobTitle() }
func fakeJobDescriptor() string { return gofakeit.JobDescriptor() }
func fakeJobLevel() string      { return gofakeit.JobLevel() }

// Financial data
func fakeCreditCardNumber() string       { return gofakeit.CreditCardNumber(nil) }
func fakeCreditCardType() string         { return gofakeit.CreditCardType() }
func fakeCurrency() string               { return gofakeit.Currency().Short }
func fakeCurrencyLong() string           { return gofakeit.Currency().Long }
func fakeCurrencyAbbrv() string          { return gofakeit.CurrencyShort() }
func fakeCurrencyName() string           { return gofakeit.CurrencyLong() }
func fakePrice(min, max float64) float64 { return gofakeit.Price(min, max) }

// Colors
func fakeColor() string     { return gofakeit.Color() }
func fakeHexColor() string  { return gofakeit.HexColor() }
func fakeRGBColor() []int   { return gofakeit.RGBColor() }
func fakeSafeColor() string { return gofakeit.SafeColor() }

// Product data
func fakeProduct() string            { return gofakeit.ProductName() }
func fakeProductName() string        { return gofakeit.ProductName() }
func fakeProductDescription() string { return gofakeit.ProductDescription() }
func fakeProductCategory() string    { return gofakeit.ProductCategory() }
func fakeProductFeature() string     { return gofakeit.ProductFeature() }
func fakeProductMaterial() string    { return gofakeit.ProductMaterial() }

// Person details
func fakeGender() string { return gofakeit.Gender() }
func fakeSSN() string    { return gofakeit.SSN() }
func fakeHobby() string  { return gofakeit.Hobby() }

// Authentication data
func fakeUsername() string { return gofakeit.Username() }

func fakePassword(lower, upper, numeric, special, space bool, num int) string {
	return gofakeit.Password(lower, upper, numeric, special, space, num)
}

// Address information
func fakeAddress() string      { return gofakeit.Address().Address }
func fakeStreet() string       { return gofakeit.Street() }
func fakeStreetName() string   { return gofakeit.StreetName() }
func fakeStreetNumber() string { return gofakeit.StreetNumber() }
func fakeCity() string         { return gofakeit.City() }
func fakeState() string        { return gofakeit.State() }
func fakeStateAbbrv() string   { return gofakeit.StateAbr() }
func fakeZip() string          { return gofakeit.Zip() }
func fakeCountry() string      { return gofakeit.Country() }
func fakeCountryAbbrv() string { return gofakeit.CountryAbr() }
func fakeLatitude() float64    { return gofakeit.Latitude() }
func fakeLongitude() float64   { return gofakeit.Longitude() }

// Words and text
func fakeWord() string { return gofakeit.Word() }

func fakeWords(num int) string {
	var words []string
	for range num {
		words = append(words, gofakeit.Word())
	}
	return strings.Join(words, " ")
}
func fakeSentence(wordCount int) string { return gofakeit.Sentence(wordCount) }
func fakeParagraph(paragraphCount int, sentenceCount int, wordCount int, separator string) string {
	return gofakeit.Paragraph(paragraphCount, sentenceCount, wordCount, separator)
}
func fakeLoremIpsumWord() string                  { return gofakeit.LoremIpsumWord() }
func fakeLoremIpsumSentence(wordCount int) string { return gofakeit.LoremIpsumSentence(wordCount) }
func fakeLoremIpsumParagraph(paragraphCount int, sentenceCount int, wordCount int, separator string) string {
	return gofakeit.LoremIpsumParagraph(paragraphCount, sentenceCount, wordCount, separator)
}

// Food
func fakeFood() string      { return gofakeit.Lunch() }
func fakeFruit() string     { return gofakeit.Fruit() }
func fakeVegetable() string { return gofakeit.Vegetable() }
func fakeBreakfast() string { return gofakeit.Breakfast() }
func fakeLunch() string     { return gofakeit.Lunch() }
func fakeDinner() string    { return gofakeit.Dinner() }
func fakeSnack() string     { return gofakeit.Snack() }
func fakeDessert() string   { return gofakeit.Dessert() }

// Miscellaneous
func fakeFlipACoin() string { return gofakeit.FlipACoin() }
func fakeRandomBool() bool  { return gofakeit.Bool() }
func fakeUUID() string      { return gofakeit.UUID() }

// Internet values
func fakeURL() string          { return gofakeit.URL() }
func fakeDomainName() string   { return gofakeit.DomainName() }
func fakeDomainSuffix() string { return gofakeit.DomainSuffix() }
func fakeIPv4Address() string  { return gofakeit.IPv4Address() }
func fakeIPv6Address() string  { return gofakeit.IPv6Address() }
func fakeMacAddress() string   { return gofakeit.MacAddress() }
func fakeHTTPMethod() string   { return gofakeit.HTTPMethod() }
func fakeUserAgent() string    { return gofakeit.UserAgent() }

// Date and Time
func fakeDate() time.Time                          { return gofakeit.Date() }
func fakeDateRange(start, end time.Time) time.Time { return gofakeit.DateRange(start, end) }
func fakeFuture() time.Time                        { return gofakeit.FutureDate() }
func fakePast() time.Time                          { return gofakeit.PastDate() }
func fakeWeekday() string                          { return gofakeit.WeekDay() }
func fakeMonth() int                               { return gofakeit.Month() }
func fakeMonthString() string                      { return gofakeit.MonthString() }
func fakeYear() int                                { return gofakeit.Year() }
func fakeHour() int                                { return gofakeit.Hour() }
func fakeMinute() int                              { return gofakeit.Minute() }
func fakeSecond() int                              { return gofakeit.Second() }
func fakeNanoSecond() int                          { return gofakeit.NanoSecond() }
func fakeTimeZone() string                         { return gofakeit.TimeZone() }
func fakeTimeZoneAbbrv() string                    { return gofakeit.TimeZone() }
func fakeTimeZoneFull() string                     { return gofakeit.TimeZoneFull() }
func fakeTimeZoneOffset() float32                  { return gofakeit.TimeZoneOffset() }

// Payment information
func fakeCreditCard() gofakeit.CreditCardInfo { return *gofakeit.CreditCard() }
func fakeAchRouting() string                  { return gofakeit.AchRouting() }
func fakeAchAccount() string                  { return gofakeit.AchAccount() }
func fakeBitcoinAddress() string              { return gofakeit.BitcoinAddress() }
func fakeBitcoinPrivateKey() string           { return gofakeit.BitcoinPrivateKey() }

// Animals
func fakeAnimal() string     { return gofakeit.Animal() }
func fakeAnimalType() string { return gofakeit.AnimalType() }
func fakeFarmAnimal() string { return gofakeit.FarmAnimal() }
func fakeCat() string        { return gofakeit.Cat() }
func fakeDog() string        { return gofakeit.Dog() }
func fakeBird() string       { return gofakeit.Bird() }

// Language
func fakeLanguage() string            { return gofakeit.Language() }
func fakeLanguageAbbrv() string       { return gofakeit.LanguageAbbreviation() }
func fakeProgrammingLanguage() string { return gofakeit.ProgrammingLanguage() }

// Celebrities
func fakeCelebrityActor() string    { return gofakeit.CelebrityActor() }
func fakeCelebrityBusiness() string { return gofakeit.CelebrityBusiness() }
func fakeCelebritySport() string    { return gofakeit.CelebritySport() }

// Books, Movies, and Songs
func fakeBook() string       { return gofakeit.BookTitle() }
func fakeBookTitle() string  { return gofakeit.BookTitle() }
func fakeBookAuthor() string { return gofakeit.BookAuthor() }
func fakeBookGenre() string  { return gofakeit.BookGenre() }
func fakeMovie() string      { return gofakeit.MovieName() }
func fakeMovieName() string  { return gofakeit.MovieName() }
func fakeMovieGenre() string { return gofakeit.MovieGenre() }
func fakeSong() string       { return gofakeit.SongName() }
func fakeMusicGenre() string { return gofakeit.SongGenre() }
