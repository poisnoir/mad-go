# Mad - High-Performance Serialization Library for Go

Mad is a Go serialization library that uses unsafe pointers for high-performance encoding and decoding of Go data structures. It supports various primitive types, strings, arrays, and structs with deterministic field ordering.

## Features

- **High Performance**: Uses unsafe pointers for direct memory access
- **Type Safety**: Generic interface with compile-time type checking  
- **Deterministic Encoding**: Struct fields are sorted alphabetically for consistent output
- **Zero Allocation Decoding**: Direct memory manipulation for maximum performance
- **Memory Layout**: Mad is independent of memory layout.

## Supported Types

- **Integers**: `int8`, `uint8`, `int16`, `uint16`, `int32`, `uint32`, `int64`, `uint64`
- **Floating Point**: `float32`, `float64`
- **Boolean**: `bool`
- **Strings**: `string` (with 4-byte length prefix)
- **Arrays**: Fixed-size arrays of supported types
- **Structs**: Composite types with supported field types

## Basic Usage

```go
package main

import (
    "fmt"
    "unsafe"
    "github.com/poisnoir/mad"
)

func main() {
    // Create a new encoder/decoder for int32
    m, err := mad.NewMad[int32]()
    if err != nil {
        panic(err)
    }
    
    // Encode a value
    value := int32(42)
    buffer := make([]byte, 4)
    
    err = m.Encode(&value, buffer)
    if err != nil {
        panic(err)
    }
    
    // Decode the value
    var decoded int32
    err = m.Decode(buffer, &decoded)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Original: %d, Decoded: %d\n", value, decoded)
}
```

## String Example

```go
func main() {
    m, err := mad.NewMad[string]()
    if err != nil {
        panic(err)
    }
    
    text := "Hello, 世界!"
    
    buffer := make([]byte, 30)
    
    // Encode
    err = m.Encode(&text, buffer)
    if err != nil {
        panic(err)
    }
    
    // Decode
    var decoded string
    err = m.Decode(buffer, &decoded)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Original: %s, Decoded: %s\n", text, decoded)
}
```

## Struct Example

```go
type Person struct {
    Age    int32
    Name   string
    Score  float64
    Active bool
}

func main() {
    m, err := mad.NewMad[Person]()
    if err != nil {
        panic(err)
    }
    
    person := Person{
        Age:    25,
        Name:   "Alice",
        Score:  95.5,
        Active: true,
    }
    
    // Calculate required buffer size
    size := m.sizefunc(unsafe.Pointer(&person))
    buffer := make([]byte, size)
    
    // Encode
    err = m.Encode(&person, buffer)
    if err != nil {
        panic(err)
    }
    
    // Decode
    var decoded Person
    err = m.Decode(buffer, &decoded)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Original: %+v\nDecoded:  %+v\n", person, decoded)
}
```

## Performance

Mad achieves excellent performance through unsafe pointer operations:

```
BenchmarkEncodeInt32-24     	40,738,098	    29.63 ns/op
BenchmarkDecodeInt32-24     	44,280,801	    25.03 ns/op
BenchmarkEncodeString-24    	39,330,438	    30.53 ns/op
BenchmarkDecodeString-24    	25,965,541	    42.58 ns/op
```

## Error Handling

Mad provides comprehensive error checking:

- **Buffer Size Validation**: All encoders and decoders validate buffer sizes
- **Type Safety**: Unsupported types return clear error messages
- **Bounds Checking**: Prevents buffer overflows and underflows

```go
// Example error handling
m, err := mad.NewMad[int64]()
if err != nil {
    // Handle unsupported type
}

smallBuffer := make([]byte, 4) // int64 needs 8 bytes
value := int64(12345)
err = m.Encode(&value, smallBuffer)
if err != nil {
    // Handle "output buffer too small" error
}
```

## Current Limitations
Mad decoder assumes the data is receiving is not corrupted and the only check is for buffer size.
- **Slices**: Not supported
- **Maps**: Not supported

## Field Ordering

Struct fields are encoded in alphabetical order by field name, ensuring deterministic output:

```go
type Example struct {
    Zebra string  // Encoded last
    Alpha int32   // Encoded first  
    Beta  bool    // Encoded second
}
```

## Building and Testing

```bash
# Initialize module
go mod init github.com/poisnoir/mad

# Run all tests
go test -v

# Run benchmarks
go test -bench=.

# Test specific functionality
go test -v -run TestStringEncoding
go test -v -run TestBufferValidation
```

## Installation

```bash
go get github.com/poisnoir/mad
```

