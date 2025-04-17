Custom Tag Parsing in goccy/go-yaml
Overview of Custom Tag Support
The goccy/go-yaml library will parse YAML tags (like !Something) into its AST, but it doesn’t automatically apply custom behaviors for unrecognized tags by default​
github.com
. In other words, simply defining !uppercase foo in YAML won’t magically uppercase the string unless you add custom unmarshaling logic. The library does provide hooks to customize how values are unmarshaled, which you can leverage to handle tags such as !uppercase. These hooks include implementing interfaces or using the AST API:
Unmarshaler Interfaces: You can implement interfaces like yaml.InterfaceUnmarshaler (similar to UnmarshalYAML(func(interface{}) error) from go-yaml v2) or yaml.NodeUnmarshaler on your custom types. The NodeUnmarshaler is particularly powerful – it gives you access to the parsed AST node, including any tag, when unmarshaling​
github.com
.
Custom Unmarshaler Registration: The library allows registering custom unmarshal functions globally for a type via yaml.RegisterCustomUnmarshaler or per-call via the CustomUnmarshaler decode option​
pkg.go.dev
​
pkg.go.dev
. However, these operate by Go type, not by YAML tag name – there isn’t a built-in function to directly map a specific tag (like "!uppercase") to a custom function.
In practice, to handle !uppercase, you would typically define a custom Go type to represent uppercase strings and implement one of the unmarshaling interfaces for that type. This way, whenever the YAML parser is decoding into that type, you can intercept the value and transform it.
Implementing a !uppercase Tag via a Custom Type
One approach is to introduce a new type (e.g. UppercaseString) and give it a custom UnmarshalYAML method. This type can be used in your structs or data models wherever you expect an uppercase tag. For example:

```go
package main

import (
    "fmt"
    "strings"
    "github.com/goccy/go-yaml"
    // import the AST subpackage if using NodeUnmarshaler:
    "github.com/goccy/go-yaml/ast"
)

// Define a custom type for uppercase strings
type UppercaseString string

// Option 1: Implement NodeUnmarshaler to get full AST node (tag info, etc.)
func (u *UppercaseString) UnmarshalYAML(node ast.Node) error {
    // Convert the AST node to a Go value (e.g. string) using NodeToValue
    var raw string
    if err := yaml.NodeToValue(node, &raw); err != nil {
        return err
    }
    // Transform the value
    *u = UppercaseString(strings.ToUpper(raw))
    return nil
}

// (Alternatively, Option 2: Implement InterfaceUnmarshaler instead, which is simpler
// if you don’t need direct access to the tag. This uses the decode function to get
// the underlying value, then uppercase it.)
/*
func (u *UppercaseString) UnmarshalYAML(decode func(interface{}) error) error {
    var raw string
    if err := decode(&raw); err != nil {
        return err
    }
    *u = UppercaseString(strings.ToUpper(raw))
    return nil
}
*/
```

In the code above, the UnmarshalYAML(node ast.Node) error method will be called by goccy/go-yaml whenever it’s decoding YAML into an UppercaseString. We take the AST node and convert it to a basic Go value using yaml.NodeToValue (a helper that fills a Go variable from an AST node)​
pkg.go.dev
. This yields the actual string (in our example, "foo"), and then we set our type to the uppercased version of that string. The NodeUnmarshaler interface gives us the flexibility to inspect the node if needed (for example, we could check the node’s tag or type), but here we simply use its value. Now, you can use UppercaseString in your data structures. For example:

```go
type Config struct {
    Name UppercaseString `yaml:"name"`
}

yamlData := []byte(`name: !uppercase foo`)
var cfg Config
if err := yaml.Unmarshal(yamlData, &cfg); err != nil {
    // handle error
}
fmt.Println(cfg.Name)  // Output: FOO
```

When unmarshaled, the library sees that Config.Name is of type UppercaseString, so it invokes our UnmarshalYAML. The YAML tag !uppercase in the input doesn’t directly call a special parser, but our custom type and logic ensure that any value (in this case "foo") is transformed to uppercase ("FOO").
How it works: In this setup, the !uppercase tag is essentially a marker to remind us to use the UppercaseString type for that field. The library’s high-level decoder will not automatically uppercase anything just because of the tag; it’s our custom unmarshaler that does the work. The tag could be omitted and the code would still uppercase the value (since the logic is tied to the type, not the tag). If you needed to enforce that the tag is present, you could inspect the AST node – for example, if using NodeUnmarshaler, check if node.Type() == ast.TagType and that the tag name matches "!uppercase" before proceeding.
This pattern is similar to how one might handle custom tags with the official gopkg.in/yaml.v3 library (which uses a yaml.Node in UnmarshalYAML). Here, goccy/go-yaml’s NodeUnmarshaler interface plays a comparable role by providing the AST node to your code​
github.com
.
Alternative: Transforming Tags via the AST API
If you need a more global or dynamic solution (for example, you want to process !uppercase tags without introducing a special field type everywhere), you can use the AST and decoding API to post-process the YAML. The goccy/go-yaml package allows you to parse YAML into an AST, traverse or modify it, then convert it into Go values. A possible workflow:
Parse to AST: Use the parser to read the YAML into an AST node. For example:

```go
import "github.com/goccy/go-yaml/parser"
root, err := parser.ParseBytes(yamlData, 0)  // root will be an ast.File or ast.Node
```

(You can also use yaml.Path{}.ReadNode(...) to get an ast.Node directly​
pkg.go.dev
.)
Traverse and Transform: Walk the AST, find any nodes that are tagged with !uppercase. In the AST, a custom tag is represented as a TagNode (node of type TagType)​
github.com
. For each such node, get or decode its value (e.g., the next scalar node) and replace it with an uppercase version. You might replace the node in the AST or simply modify the scalar’s value. The ast.TagNode has methods like GetToken() (to get the tag token) and GetValue()​
pkg.go.dev
. For example, if you encounter a TagNode, you can do:

```go
if tagNode, ok := node.(*ast.TagNode); ok {
    val := yaml.NodeToValue(tagNode, new(interface{}))  // get underlying value
    // ... uppercase the string if val is string, then set it back in the AST ...
}
```

(This part requires understanding the AST structure; essentially the tag node will have a child value node in the AST. The yaml.NodeToValue helper is convenient for extracting the value​
pkg.go.dev
.)
Convert AST to Go value: After modifying the AST, convert it to your target Go structure or interface using yaml.NodeToValue. For example:

```go
var result Config
if err := yaml.NodeToValue(root, &result); err != nil {
    // handle error
}
```

This will produce a populated Config object with all transformations applied.
While this AST manipulation approach works, it’s more involved. In most cases, defining a custom type with an UnmarshalYAML is simpler and sufficient.
Summary
goccy/go-yaml does not have a one-step API to register a YAML tag (like !uppercase) with a handler function, but it provides the building blocks to achieve the effect. The recommended method is to use a custom type that implements one of the unmarshaler interfaces to intercept the value during decoding and perform the transformation. The library’s documentation highlights these extension points – for example, the ability to “Customize the Marshal/Unmarshal behavior for primitive types and third-party library types”​
github.com
(via custom interfaces or RegisterCustomUnmarshaler). By leveraging these hooks, you can support tags like !uppercase in your YAML processing. References:
goccy/go-yaml GitHub – Unmarshaler interfaces (Bytes, Interface, Node)​
github.com
goccy/go-yaml GitHub – User question about custom local tags (!Foo)​
github.com
goccy/go-yaml pkg.go.dev – yaml.NodeToValue utility description​
pkg.go.dev
(converting an AST node to a Go value)
goccy/go-yaml README – Mention of customizing unmarshal behavior
