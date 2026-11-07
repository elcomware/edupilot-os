package main

import (
	"io"
	"testing"
)

func TestParseEdit(t *testing.T) {
	base := []string{"-tenant", "5f0a174e-999f-4c61-aeee-0a6b992fa204", "-id", "00000000-0000-4000-8000-000000000001", "-epoch", "1", "-version", "9", "-name", " Edited "}
	for _, tc := range []struct {
		kind   string
		fields []string
	}{{"organizations", nil}, {"schools", []string{"-code", " new "}}, {"campuses", []string{"-code", " new ", "-timezone", "Europe/Paris"}}} {
		args := append(append(append([]string{}, base...), "-kind", tc.kind), tc.fields...)
		got, err := parseEdit(args, io.Discard)
		if err != nil || got.Kind != tc.kind || got.Metadata.Name != " Edited " || got.Version != 9 || got.Epoch != 1 {
			t.Fatal(got, err)
		}
		if tc.kind != "organizations" && got.Metadata.Code != " new " {
			t.Fatal(got)
		}
		if tc.kind == "campuses" && got.Metadata.Timezone != "Europe/Paris" {
			t.Fatal(got)
		}
		for i := 0; i < len(args); i += 2 {
			partial := append(append([]string{}, args[:i]...), args[i+2:]...)
			if _, err := parseEdit(partial, io.Discard); err == nil {
				t.Fatal("accepted omitted required flag", args[i])
			}
		}
		if _, err := parseEdit(append(args, "positional"), io.Discard); err == nil {
			t.Fatal("positional")
		}
	}
	for _, extra := range [][]string{
		{"-kind", "unknown"}, {"-kind", "organizations", "-code", ""}, {"-kind", "organizations", "-timezone", ""}, {"-kind", "schools", "-code", "OK", "-timezone", ""},
		{"-kind", "organizations", "-id", "bad"}, {"-kind", "organizations", "-tenant", "bad"},
		{"-kind", "organizations", "-epoch", "0"}, {"-kind", "organizations", "-epoch", "18446744073709551615"}, {"-kind", "organizations", "-version", "0"}, {"-kind", "organizations", "-version", "9223372036854775807"},
	} {
		if _, err := parseEdit(append(append([]string{}, base...), extra...), io.Discard); err == nil {
			t.Fatal("accepted", extra)
		}
	}
}
