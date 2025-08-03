# Fake Data Functions

Mockingjay now includes a set of fake data generation functions powered by [gofakeit](https://github.com/brianvoe/gofakeit). These functions can be used in your templates to generate realistic test data on the fly.

## Basic Personal Information

| Function                   | Description            | Example Output         |
| -------------------------- | ---------------------- | ---------------------- |
| `{{ fakeName }}`           | Full name              | "John Smith"           |
| `{{ fakeFirstName }}`      | First name             | "Jane"                 |
| `{{ fakeLastName }}`       | Last name              | "Doe"                  |
| `{{ fakeEmail }}`          | Email address          | "john.doe@example.com" |
| `{{ fakePhone }}`          | Phone number           | "5551234567"           |
| `{{ fakePhoneFormatted }}` | Formatted phone        | "(555) 123-4567"       |
| `{{ fakeGender }}`         | Gender                 | "Male"                 |
| `{{ fakeSSN }}`            | Social Security Number | "123-45-6789"          |
| `{{ fakeHobby }}`          | Hobby                  | "Photography"          |

## Business & Company Data

| Function                  | Description       | Example Output                 |
| ------------------------- | ----------------- | ------------------------------ |
| `{{ fakeCompany }}`       | Company name      | "Tech Solutions Inc"           |
| `{{ fakeCompanySuffix }}` | Company suffix    | "LLC"                          |
| `{{ fakeJobTitle }}`      | Job title         | "Software Engineer"            |
| `{{ fakeJobDescriptor }}` | Job descriptor    | "Senior"                       |
| `{{ fakeJobLevel }}`      | Job level         | "Executive"                    |
| `{{ fakeBS }}`            | Business buzzword | "synergize scalable solutions" |

## Financial Data

| Function                     | Description        | Example Output                       |
| ---------------------------- | ------------------ | ------------------------------------ |
| `{{ fakeCreditCardNumber }}` | Credit card number | "4532015112830366"                   |
| `{{ fakeCreditCardType }}`   | Credit card type   | "Visa"                               |
| `{{ fakeCurrency }}`         | Currency code      | "USD"                                |
| `{{ fakeCurrencyLong }}`     | Currency name      | "United States Dollar"               |
| `{{ fakePrice 10.0 100.0 }}` | Price in range     | 45.67                                |
| `{{ fakeAchRouting }}`       | ACH routing number | "123456789"                          |
| `{{ fakeAchAccount }}`       | ACH account number | "123456789012"                       |
| `{{ fakeBitcoinAddress }}`   | Bitcoin address    | "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa" |

## Colors

| Function              | Description    | Example Output |
| --------------------- | -------------- | -------------- |
| `{{ fakeColor }}`     | Color name     | "Red"          |
| `{{ fakeHexColor }}`  | Hex color      | "#FF5733"      |
| `{{ fakeSafeColor }}` | Web-safe color | "Purple"       |

## Address Information

| Function                 | Description          | Example Output   |
| ------------------------ | -------------------- | ---------------- |
| `{{ fakeAddress }}`      | Full address         | "123 Main St"    |
| `{{ fakeStreet }}`       | Street address       | "123 Oak Street" |
| `{{ fakeStreetName }}`   | Street name          | "Main Street"    |
| `{{ fakeStreetNumber }}` | Street number        | "123"            |
| `{{ fakeCity }}`         | City name            | "New York"       |
| `{{ fakeState }}`        | State name           | "California"     |
| `{{ fakeStateAbbrv }}`   | State abbreviation   | "CA"             |
| `{{ fakeZip }}`          | ZIP code             | "90210"          |
| `{{ fakeCountry }}`      | Country name         | "United States"  |
| `{{ fakeCountryAbbrv }}` | Country abbreviation | "US"             |
| `{{ fakeLatitude }}`     | Latitude             | 40.7128          |
| `{{ fakeLongitude }}`    | Longitude            | -74.0060         |

## Product Data

| Function                       | Description         | Example Output             |
| ------------------------------ | ------------------- | -------------------------- |
| `{{ fakeProductName }}`        | Product name        | "Wireless Headphones"      |
| `{{ fakeProductDescription }}` | Product description | "High-quality wireless..." |
| `{{ fakeProductCategory }}`    | Product category    | "Electronics"              |
| `{{ fakeProductFeature }}`     | Product feature     | "Waterproof"               |
| `{{ fakeProductMaterial }}`    | Product material    | "Plastic"                  |

## Text & Words

| Function                          | Description           | Example Output              |
| --------------------------------- | --------------------- | --------------------------- |
| `{{ fakeWord }}`                  | Single word           | "amazing"                   |
| `{{ fakeWords 5 }}`               | Multiple words        | "the quick brown fox jumps" |
| `{{ fakeSentence 8 }}`            | Sentence with N words | "The quick brown fox..."    |
| `{{ fakeParagraph 1 3 8 ". " }}`  | Paragraph             | "Lorem ipsum dolor..."      |
| `{{ fakeLoremIpsumWord }}`        | Lorem ipsum word      | "lorem"                     |
| `{{ fakeLoremIpsumSentence 10 }}` | Lorem ipsum sentence  | "Lorem ipsum dolor..."      |

## Food

| Function              | Description    | Example Output |
| --------------------- | -------------- | -------------- |
| `{{ fakeFood }}`      | Food item      | "Pizza"        |
| `{{ fakeFruit }}`     | Fruit          | "Apple"        |
| `{{ fakeVegetable }}` | Vegetable      | "Carrot"       |
| `{{ fakeBreakfast }}` | Breakfast food | "Pancakes"     |
| `{{ fakeLunch }}`     | Lunch food     | "Sandwich"     |
| `{{ fakeDinner }}`    | Dinner food    | "Steak"        |
| `{{ fakeSnack }}`     | Snack food     | "Chips"        |
| `{{ fakeDessert }}`   | Dessert        | "Ice Cream"    |

## Internet & Tech

| Function                 | Description       | Example Output                         |
| ------------------------ | ----------------- | -------------------------------------- |
| `{{ fakeURL }}`          | Website URL       | "https://example.com"                  |
| `{{ fakeDomainName }}`   | Domain name       | "example.com"                          |
| `{{ fakeDomainSuffix }}` | Domain suffix     | ".com"                                 |
| `{{ fakeIPv4Address }}`  | IPv4 address      | "192.168.1.1"                          |
| `{{ fakeIPv6Address }}`  | IPv6 address      | "2001:db8::1"                          |
| `{{ fakeMacAddress }}`   | MAC address       | "aa:bb:cc:dd:ee:ff"                    |
| `{{ fakeHTTPMethod }}`   | HTTP method       | "GET"                                  |
| `{{ fakeUserAgent }}`    | User agent string | "Mozilla/5.0..."                       |
| `{{ fakeUUID }}`         | UUID              | "550e8400-e29b-41d4-a716-446655440000" |

## Date & Time

| Function                | Description   | Example Output         |
| ----------------------- | ------------- | ---------------------- |
| `{{ fakeDate }}`        | Random date   | "2023-05-15T10:30:00Z" |
| `{{ fakeFuture }}`      | Future date   | "2024-12-25T15:30:00Z" |
| `{{ fakePast }}`        | Past date     | "2022-01-15T08:45:00Z" |
| `{{ fakeWeekday }}`     | Day of week   | "Monday"               |
| `{{ fakeMonth }}`       | Month number  | 7                      |
| `{{ fakeMonthString }}` | Month name    | "July"                 |
| `{{ fakeYear }}`        | Year          | 2023                   |
| `{{ fakeHour }}`        | Hour (0-23)   | 14                     |
| `{{ fakeMinute }}`      | Minute (0-59) | 30                     |
| `{{ fakeSecond }}`      | Second (0-59) | 45                     |
| `{{ fakeTimeZone }}`    | Time zone     | "America/New_York"     |

## Animals

| Function               | Description  | Example Output     |
| ---------------------- | ------------ | ------------------ |
| `{{ fakeAnimal }}`     | Animal name  | "Lion"             |
| `{{ fakeAnimalType }}` | Animal type  | "Mammal"           |
| `{{ fakeFarmAnimal }}` | Farm animal  | "Cow"              |
| `{{ fakeCat }}`        | Cat breed    | "Persian"          |
| `{{ fakeDog }}`        | Dog breed    | "Golden Retriever" |
| `{{ fakeBird }}`       | Bird species | "Eagle"            |

## Language

| Function                        | Description          | Example Output |
| ------------------------------- | -------------------- | -------------- |
| `{{ fakeLanguage }}`            | Language name        | "English"      |
| `{{ fakeLanguageAbbrv }}`       | Language code        | "en"           |
| `{{ fakeProgrammingLanguage }}` | Programming language | "Go"           |

## Entertainment

| Function                      | Description      | Example Output          |
| ----------------------------- | ---------------- | ----------------------- |
| `{{ fakeCelebrityActor }}`    | Actor name       | "Brad Pitt"             |
| `{{ fakeCelebrityBusiness }}` | Business person  | "Elon Musk"             |
| `{{ fakeCelebritySport }}`    | Sports celebrity | "Michael Jordan"        |
| `{{ fakeBookTitle }}`         | Book title       | "To Kill a Mockingbird" |
| `{{ fakeBookAuthor }}`        | Book author      | "Harper Lee"            |
| `{{ fakeBookGenre }}`         | Book genre       | "Fiction"               |
| `{{ fakeMovieName }}`         | Movie title      | "The Godfather"         |
| `{{ fakeMovieGenre }}`        | Movie genre      | "Drama"                 |
| `{{ fakeSong }}`              | Song title       | "Bohemian Rhapsody"     |
| `{{ fakeMusicGenre }}`        | Music genre      | "Rock"                  |

## Miscellaneous

| Function               | Description    | Example Output |
| ---------------------- | -------------- | -------------- |
| `{{ fakeFlipACoin }}`  | Coin flip      | "Heads"        |
| `{{ fakeRandomBool }}` | Random boolean | true           |
| `{{ fakeUsername }}`   | Username       | "user123"      |

## Usage Examples

### Simple JSON Response

```yaml
routes:
  - path: /user
    method: GET
    template: |
      {
        "id": "{{ fakeUUID }}",
        "name": "{{ fakeName }}",
        "email": "{{ fakeEmail }}",
        "company": "{{ fakeCompany }}",
        "address": {
          "street": "{{ fakeStreet }}",
          "city": "{{ fakeCity }}",
          "state": "{{ fakeState }}",
          "zip": "{{ fakeZip }}"
        }
      }
```

### Multiple Records

```yaml
routes:
  - path: /users
    method: GET
    template: |
      [
        {{- range $i := seq 1 5 }}
        {{- if gt $i 1 }},{{ end }}
        {
          "id": "{{ fakeUUID }}",
          "name": "{{ fakeName }}",
          "email": "{{ fakeEmail }}",
          "job": "{{ fakeJobTitle }}"
        }
        {{- end }}
      ]
```

### Complex Product Catalog

```yaml
routes:
  - path: /products
    method: GET
    template: |
      {
        "products": [
          {{- range $i := seq 1 10 }}
          {{- if gt $i 1 }},{{ end }}
          {
            "id": "{{ fakeUUID }}",
            "name": "{{ fakeProductName }}",
            "description": "{{ fakeProductDescription }}",
            "price": {{ fakePrice 9.99 199.99 }},
            "category": "{{ fakeProductCategory }}",
            "color": "{{ fakeColor }}",
            "material": "{{ fakeProductMaterial }}",
            "created_at": "{{ fakePast }}",
            "url": "{{ fakeURL }}"
          }
          {{- end }}
        ],
        "meta": {
          "total": 10,
          "generated_at": "{{ now }}"
        }
      }
```

## Function Parameters

Some functions accept parameters to customize their behavior:

- `fakeWords N` - Generate N words
- `fakeSentence N` - Generate sentence with N words
- `fakeParagraph paragraphCount sentenceCount wordCount separator` - Generate paragraph
- `fakePrice min max` - Generate price between min and max
- `fakePassword lower upper numeric special space length` - Generate password with criteria

Example:
```yaml
template: |
  {
    "password": "{{ fakePassword true true true false false 12 }}",
    "description": "{{ fakeSentence 15 }}",
    "price": {{ fakePrice 10.00 500.00 }}
  }
```

## Tips

1. **Consistent Data**: Each template execution generates new random data. If you need consistent data across multiple calls, consider using a fixed seed or caching mechanism.

2. **JSON Escaping**: When using fake data in JSON, be aware that some generated text might contain characters that need escaping. The template engine handles most cases automatically.

3. **Performance**: Fake data generation is fast, but for large datasets (hundreds of records), consider the performance impact.

4. **Combining Functions**: You can combine fake functions with Sprig template functions for more complex scenarios:
   ```yaml
   template: |
     {
       "name": "{{ fakeName | upper }}",
       "slug": "{{ fakeProductName | lower | replace " " "-" }}"
     }
   ```

## Example Files

Check out the `examples/fake-data-demo.yaml` file for a demonstration of all available fake data functions.
