# json-lossless #

json-lossless is a Go library that populates structs from JSON and
allows serialization back to JSON without losing fields that are
not explicitly defined in the struct.

json-lossless builds off of bit.ly's excellent
[go-simplejson](https://github.com/bitly/go-simplejson) package.

json-lossless is experimental and probably doesn't work in a lot
of cases.  Pull requests very welcome.

## API ##

Full API docs are on [GoDoc](http://godoc.org/github.com/joeshaw/json-lossless):
http://godoc.org/github.com/joeshaw/json-lossless

To get started, embed a `lossless.JSON` inside your struct:

```go
type Person struct {
        lossless.JSON `json:"-"`

	Name      string `json:"name"`
	Age       int    `json:"age"`
	Address   string
	CreatedAt time.Time
}
```

Define `MarshalJSON` and `UnmarshalJSON` methods on the type
to implement the `json.Marshaler` and `json.Unmarshaler` interfaces,
deferring the work to the `lossless.JSON` embed:

```go
func (p *Person) UnmarshalJSON(data []byte) error {
	return p.JSON.UnmarshalJSON(p, data)
}

func (p Person) MarshalJSON() ([]byte, error) {
	return p.JSON.MarshalJSON(p)
}
```

Given JSON like this:

```json
{"name": "Jack Wolfington",
 "age": 42,
 "address": "123 Fake St.",
 "CreatedAt": "2013-09-16T10:44:40.295451647-00:00",
 "Extra": {"foo": "bar"}}
```

When you decode into a struct, the `Extra` field will be kept around,
even though it's not accessible from your struct.

```go
var p Person
if err := json.Unmarshal(data, &p); err != nil {
        panic(err)
}

data, err := json.Marshal(p)
if err != nil {
        panic(err)
}

// "Extra" is still set in the marshaled JSON:
if bytes.Index(data, "Extra") == -1 {
        panic("Extra not in data!")
}

fmt.Println(string(data))

```

You can also set arbitrary key/values on your struct by calling
`Set()`:

```go
p.Set("Extra", "AgeString", "forty-two")
```

When serialized, `Extra` will look like this:

```json
{ ...
  "Extra": {"foo": "bar", "AgeString": "forty-two"}}
```

## Known issues ##

json-lossless doesn't attempt to decode arrays or simple values.
For those, just use `json/encoding` directly.

The `omitempty` setting on `json` tag is not handled.  In fact, no
parsing of the tags are done at all.

The `lossless.JSON` value needs to be tagged with `json:"-"` or
it will be marshaled to JSON.
