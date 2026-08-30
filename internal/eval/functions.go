package eval

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"time"
	"uuid"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

// standardFunctions returns the global function registry for HCL evaluation.
func standardFunctions() map[string]function.Function {
	return map[string]function.Function{
		// System
		"env":     envFunc,
		"uuid":    uuidV4Func,
		"uuid_v4": uuidV4Func,
		"uuid_v7": uuidV7Func,
		"now":     nowFunc,

		// Encoding
		"json_encode":   stdlib.JSONEncodeFunc,
		"json_decode":   stdlib.JSONDecodeFunc,
		"base64_encode": base64EncodeFunc,
		"base64_decode": base64DecodeFunc,
		"url_encode":    urlEncodeFunc,
		"url_decode":    urlDecodeFunc,

		// Strings
		"lower":       stdlib.LowerFunc,
		"upper":       stdlib.UpperFunc,
		"trim_space":  stdlib.TrimSpaceFunc,
		"trim":        stdlib.TrimFunc,
		"trim_prefix": stdlib.TrimPrefixFunc,
		"trim_suffix": stdlib.TrimSuffixFunc,
		"split":       stdlib.SplitFunc,
		"join":        stdlib.JoinFunc,
		"replace":     stdlib.ReplaceFunc,
		"format":      stdlib.FormatFunc,

		// Collections & Objects
		"coalesce": coalesceFunc,
		"length":   lengthFunc,
		"merge":    stdlib.MergeFunc,
		"lookup":   stdlib.LookupFunc,
		"keys":     stdlib.KeysFunc,
		"values":   stdlib.ValuesFunc,
		"contains": stdlib.ContainsFunc,

		// Cryptography
		"sha256":      sha256Func,
		"md5":         md5Func,
		"hmac_sha256": hmacSHA256Func,

		// Math
		"min":       stdlib.MinFunc,
		"max":       stdlib.MaxFunc,
		"abs":       stdlib.AbsoluteFunc,
		"ceil":      stdlib.CeilFunc,
		"floor":     stdlib.FloorFunc,
		"parse_int": stdlib.ParseIntFunc,
	}
}

// envFunc reads the value of an environment variable from the host operating system.
var envFunc = function.New(&function.Spec{
	Description: "Reads the value of an environment variable from the host operating system.",
	Params: []function.Parameter{
		{
			Name:        "name",
			Type:        cty.String,
			Description: "The environment variable name to look up",
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		name := args[0].AsString()
		return cty.StringVal(os.Getenv(name)), nil
	},
})

// uuidV4Func generates a cryptographically secure random UUID version 4 string.
var uuidV4Func = function.New(&function.Spec{
	Description: "Generates a cryptographically secure random UUID version 4 string.",
	Type:        function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return cty.StringVal(uuid.NewV4().String()), nil
	},
})

// uuidV7Func generates a time-ordered UUID version 7 string (RFC 9562).
var uuidV7Func = function.New(&function.Spec{
	Description: "Generates a time-ordered UUID version 7 string (RFC 9562) optimized for database primary keys.",
	Type:        function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return cty.StringVal(uuid.NewV7().String()), nil
	},
})

// nowFunc returns the current system timestamp in UTC formatted according to RFC 3339.
var nowFunc = function.New(&function.Spec{
	Description: "Returns the current system timestamp in UTC formatted according to RFC 3339.",
	Type:        function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		return cty.StringVal(time.Now().UTC().Format(time.RFC3339)), nil
	},
})

// base64EncodeFunc encodes a string using standard RFC 4648 Base64 encoding.
var base64EncodeFunc = function.New(&function.Spec{
	Description: "Encodes a string using standard RFC 4648 Base64 encoding.",
	Params: []function.Parameter{
		{
			Name:        "str",
			Type:        cty.String,
			Description: "The string to encode",
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		str := args[0].AsString()
		encoded := base64.StdEncoding.EncodeToString([]byte(str))
		return cty.StringVal(encoded), nil
	},
})

// base64DecodeFunc decodes a standard Base64 encoded string back into plain text.
var base64DecodeFunc = function.New(&function.Spec{
	Description: "Decodes a standard Base64 encoded string back into plain text.",
	Params: []function.Parameter{
		{
			Name:        "str",
			Type:        cty.String,
			Description: "The Base64 encoded string to decode",
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		str := args[0].AsString()
		decoded, err := base64.StdEncoding.DecodeString(str)
		if err != nil {
			return cty.NilVal, fmt.Errorf("failed to decode base64 string: %w", err)
		}
		return cty.StringVal(string(decoded)), nil
	},
})

// urlEncodeFunc escapes characters in a string to make it safe for inclusion inside a URL query parameter.
var urlEncodeFunc = function.New(&function.Spec{
	Description: "Escapes characters in a string to make it safe for inclusion inside a URL query parameter.",
	Params: []function.Parameter{
		{
			Name:        "str",
			Type:        cty.String,
			Description: "The string to escape",
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		str := args[0].AsString()
		return cty.StringVal(url.QueryEscape(str)), nil
	},
})

// urlDecodeFunc unescapes a URL percent-encoded string.
var urlDecodeFunc = function.New(&function.Spec{
	Description: "Unescapes a URL percent-encoded string.",
	Params: []function.Parameter{
		{
			Name:        "str",
			Type:        cty.String,
			Description: "The URL encoded string",
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		str := args[0].AsString()
		decoded, err := url.QueryUnescape(str)
		if err != nil {
			return cty.NilVal, fmt.Errorf("failed to decode url string: %w", err)
		}
		return cty.StringVal(decoded), nil
	},
})

// coalesceFunc evaluates arguments sequentially and returns the first argument that is not null and not an empty string.
var coalesceFunc = function.New(&function.Spec{
	Description: "Evaluates arguments sequentially and returns the first argument that is not null and not an empty string.",
	VarParam: &function.Parameter{
		Name:             "vals",
		Type:             cty.DynamicPseudoType,
		AllowNull:        true,
		AllowDynamicType: true,
	},
	Type: function.StaticReturnType(cty.DynamicPseudoType),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		if len(args) == 0 {
			return cty.NilVal, fmt.Errorf("at least one argument is required")
		}
		for _, arg := range args {
			if !arg.IsKnown() || arg.IsNull() {
				continue
			}
			if arg.Type().Equals(cty.String) && arg.AsString() == "" {
				continue
			}
			return arg, nil
		}
		return cty.NullVal(cty.DynamicPseudoType), nil
	},
})

// lengthFunc returns the total number of elements in a list, keys in a map, or characters in a string.
var lengthFunc = function.New(&function.Spec{
	Description: "Returns the total number of elements in a list, keys in a map, or characters in a string.",
	Params: []function.Parameter{
		{
			Name:             "collection",
			Type:             cty.DynamicPseudoType,
			AllowDynamicType: true,
		},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		val := args[0]
		if !val.IsKnown() || val.IsNull() {
			return cty.NumberIntVal(0), nil
		}

		ty := val.Type()
		switch {
		case ty.Equals(cty.String):
			return cty.NumberIntVal(int64(len([]rune(val.AsString())))), nil
		case ty.IsListType() || ty.IsSetType() || ty.IsTupleType() || ty.IsMapType() || ty.IsObjectType():
			return cty.NumberIntVal(int64(val.LengthInt())), nil
		default:
			return cty.NilVal, fmt.Errorf("length requires a string, list, tuple, set, or map; got %s", ty.FriendlyName())
		}
	},
})

// sha256Func computes the SHA-256 cryptographic hash of a string, returning a lowercase 64-character hexadecimal digest.
var sha256Func = function.New(&function.Spec{
	Description: "Computes the SHA-256 cryptographic hash of a string, returning a lowercase 64-character hexadecimal digest.",
	Params: []function.Parameter{
		{
			Name:        "str",
			Type:        cty.String,
			Description: "The string to hash",
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		str := args[0].AsString()
		hash := sha256.Sum256([]byte(str))
		return cty.StringVal(hex.EncodeToString(hash[:])), nil
	},
})

// md5Func computes the MD5 checksum of a string, returning a 32-character hexadecimal digest.
var md5Func = function.New(&function.Spec{
	Description: "Computes the MD5 checksum of a string, returning a 32-character hexadecimal digest.",
	Params: []function.Parameter{
		{
			Name:        "str",
			Type:        cty.String,
			Description: "The string to hash",
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		str := args[0].AsString()
		hash := md5.Sum([]byte(str))
		return cty.StringVal(hex.EncodeToString(hash[:])), nil
	},
})

// hmacSHA256Func computes an HMAC-SHA256 signature for a payload using a secret key. Returns a lowercase hexadecimal string.
var hmacSHA256Func = function.New(&function.Spec{
	Description: "Computes an HMAC-SHA256 signature for a payload using a secret key. Returns a lowercase hexadecimal string.",
	Params: []function.Parameter{
		{
			Name:        "key",
			Type:        cty.String,
			Description: "The secret key used for signing",
		},
		{
			Name:        "message",
			Type:        cty.String,
			Description: "The message payload to sign",
		},
	},
	Type: function.StaticReturnType(cty.String),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		key := args[0].AsString()
		message := args[1].AsString()

		mac := hmac.New(sha256.New, []byte(key))
		mac.Write([]byte(message))

		return cty.StringVal(hex.EncodeToString(mac.Sum(nil))), nil
	},
})
