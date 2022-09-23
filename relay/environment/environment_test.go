package environment_test

import (
	"testing"

	"github.com/fullstorydev/relay-core/relay/environment"
)

type TestProvider struct {
	values map[string]string
}

func (provider *TestProvider) Lookup(key string) (string, bool) {
	val, ok := provider.values[key]
	return val, ok
}

func TestSubstituteVarsIntoYaml(t *testing.T) {
	testCases := []struct {
		desc     string
		env      map[string]string
		input    string
		expected string
	}{
		{
			desc:     `Missing variables result in a null value`,
			env:      map[string]string{},
			input:    `foo: ${MISSING}`,
			expected: `foo: `,
		},
		{
			desc: `Null values are preserved`,
			env: map[string]string{
				`VALUE1`: `null`,
				`VALUE2`: `NULL`,
				`VALUE3`: `~`,
				`VALUE4`: ``,
			},
			input:    `foo: ${VALUE1} ${VALUE2} ${VALUE3} ${VALUE4}`,
			expected: `foo: null NULL ~ `,
		},
		{
			desc: `Boolean values are preserved`,
			env: map[string]string{
				`VALUE1`: `true`,
				`VALUE2`: `True`,
				`VALUE3`: `false`,
				`VALUE4`: `FALSE`,
			},
			input:    `foo: ${VALUE1} ${VALUE2} ${VALUE3} ${VALUE4}`,
			expected: `foo: true True false FALSE`,
		},
		{
			desc: `Int values are preserved`,
			env: map[string]string{
				`VALUE1`: `0`,
				`VALUE2`: `0o7`,
				`VALUE3`: `0x3A`,
				`VALUE4`: `-19`,
			},
			input:    `foo: ${VALUE1} ${VALUE2} ${VALUE3} ${VALUE4}`,
			expected: `foo: 0 0o7 0x3A -19`,
		},
		{
			desc: `Float values are preserved`,
			env: map[string]string{
				`VALUE1`: `0.`,
				`VALUE2`: `-0.0`,
				`VALUE3`: `.5`,
				`VALUE4`: `+12e03`,
				`VALUE5`: `-2E+05`,
			},
			input:    `foo: ${VALUE1} ${VALUE2} ${VALUE3} ${VALUE4} ${VALUE5}`,
			expected: `foo: 0. -0.0 .5 +12e03 -2E+05`,
		},
		{
			desc: `Special float values are preserved`,
			env: map[string]string{
				`VALUE1`: `.inf`,
				`VALUE2`: `-.Inf`,
				`VALUE3`: `+.INF`,
				`VALUE4`: `.NAN`,
			},
			input:    `foo: ${VALUE1} ${VALUE2} ${VALUE3} ${VALUE4}`,
			expected: `foo: .inf -.Inf +.INF .NAN`,
		},
		{
			desc: `String values are preserved and escaped`,
			env: map[string]string{
				`VALUE1`: `bar`,
				`VALUE2`: `Two words.`,
				`VALUE3`: `"double" 'single'`,
				`VALUE4`: `🛑`,
			},
			input:    `foo: ${VALUE1} ${VALUE2} ${VALUE3} ${VALUE4}`,
			expected: `foo: bar Two words. '"double" ''single''' "\U0001F6D1"`,
		},
		{
			desc:     `Default values are used if present`,
			env:      map[string]string{},
			input:    `foo: ${MISSING:bar}`,
			expected: `foo: bar`,
		},
		{
			desc:     `Primitive default values are preserved`,
			env:      map[string]string{},
			input:    `foo: ${MISSING1:null} ${MISSING2:true} ${MISSING3:100} ${MISSING4:-0.5} ${MISSING5:.NAN}`,
			expected: `foo: null true 100 -0.5 .NAN`,
		},
		{
			desc:     `String default values are preserved and escaped`,
			env:      map[string]string{},
			input:    `foo: ${MISSING1:bar} ${MISSING2:Two words.} ${MISSING3:"double" 'single'} ${MISSING4:🛑}`,
			expected: `foo: bar Two words. '"double" ''single''' "\U0001F6D1"`,
		},
		{
			desc: `Unescaped substitutions work`,
			env: map[string]string{
				`VALUE1`: `bar`,
				`VALUE2`: `"double" 'single'`,
				`VALUE3`: `🛑`,
				`VALUE4`: `[1, 2, 3, 4]`,
			},
			input:    `foo: $(VALUE1) $(VALUE2) $(VALUE3) $(VALUE4)`,
			expected: `foo: bar "double" 'single' 🛑 [1, 2, 3, 4]`,
		},
		{
			desc:     `Default values in unescaped substitutions are used if present`,
			env:      map[string]string{},
			input:    `foo: $(MISSING:bar)`,
			expected: `foo: bar`,
		},
		{
			desc:     `Default values in unescaped substitutions are not escaped`,
			env:      map[string]string{},
			input:    `foo: $(MISSING1:bar) $(MISSING2:Two words.) $(MISSING3:"double" 'single') $(MISSING4:🛑)`,
			expected: `foo: bar Two words. "double" 'single' 🛑`,
		},
		{
			desc:     `Empty variable names always result in the default`,
			env:      map[string]string{},
			input:    `foo: ${:bar} $(:baz)`,
			expected: `foo: bar baz`,
		},
		{
			desc:     `Escaping substitution syntax is possible`,
			env:      map[string]string{},
			input:    `foo: $${}{bar} $$()(baz)`,
			expected: `foo: ${bar} $(baz)`,
		},
	}

	for _, testCase := range testCases {
		provider := &TestProvider{
			values: testCase.env,
		}
		env := environment.NewMap(provider)

		actual := env.SubstituteVarsIntoYaml(testCase.input)
		if actual != testCase.expected {
			t.Errorf(
				"Test '%v': Expected '%s' but got '%s'",
				testCase.desc,
				testCase.expected,
				actual,
			)
			return
		}
	}
}
