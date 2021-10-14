package stringutils

import "testing"

// scenario represent each test scenario:
// d is the description,
// v is the value to be tested,
// and e is the expected output.
type scenario struct {
	d, v, e string
}

func TestMaskEmail(t *testing.T) {
	t.Run("Success scenarios", func(t *testing.T) {
		tt := []scenario{
			{
				d: "Should mask a valid email with an email id >= 4 correctly",
				v: "deliverymuch@gmail.com",
				e: "deli********@gmail.com",
			},
			{
				d: "Should mask a email with numbers",
				v: "deliverymuch0987@gmail.com",
				e: "deli************@gmail.com",
			},
			{
				d: "Should leave only one character unmasked if its an email with email id len = 4",
				v: "deli@gmail.com",
				e: "d***@gmail.com",
			},
			{
				d: "Should leave only one character unmasked if its an email with email id len = 3",
				v: "del@gmail.com",
				e: "d**@gmail.com",
			},
			{
				d: "Should care only for the first '@' when separating the email id from the address",
				v: "deliverymuch@gmail@otherthing.com",
				e: "deli********@gmail@otherthing.com",
			},
			{
				d: "Should mask normaly if the email has no address",
				v: "emailid@",
				e: "emai***@",
			},
		}

		for _, v := range tt {
			t.Run(v.d, func(t *testing.T) {
				actual := MaskEmail(v.v)
				if actual != v.e {
					t.Fatalf("failed in mask email test, expected: %s, actual: %s", v.e, actual)
				}
			})
		}
	})

	t.Run("Failure scenarios", func(t *testing.T) {
		scenarios := []scenario{
			{
				d: "Shouldn't mask if the email has no id",
				v: "@gmail.com",
				e: "@gmail.com",
			},
			{
				d: "Shouldn't mask if it's an invalid input",
				v: "thisisnotanemail.com",
				e: "thisisnotanemail.com",
			},
			{
				d: "Should return an empty string if the input is empty",
				v: "",
				e: "",
			},
		}

		for _, s := range scenarios {
			t.Run(s.d, func(t *testing.T) {
				actual := MaskEmail(s.v)
				if actual != s.e {
					t.Fatalf("failed in mask email test, expected: %s, actual: %s", s.e, actual)
				}
			})
		}
	})
}

func TestMaskString(t *testing.T) {
	scenarios := []scenario{
		{
			d: "Should mask a even string correctly",
			v: "ab11111aBc",
			e: "ab111*****",
		},
		{
			d: "Should mask a odd string correctly",
			v: "ab1111aBc",
			e: "ab11*****",
		},
		{
			d: "Should return an empty string if the input is empty",
			v: "",
			e: "",
		},
	}

	for _, s := range scenarios {
		t.Run(s.d, func(t *testing.T) {
			actual := MaskString(s.v)
			if actual != s.e {
				t.Fatalf("failed in mask string test, expected: %s, actual: %s", s.e, actual)
			}
		})
	}
}
