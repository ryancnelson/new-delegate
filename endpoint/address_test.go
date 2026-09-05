package endpoint

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  Address
	}{
		{
			name:  "TCP listener",
			input: "TCP-LISTEN:127.0.0.1:18080",
			want:  Address{Kind: TCPListen, Host: "127.0.0.1", Port: 18080},
		},
		{
			name:  "TCP connector",
			input: "TCP-CONNECT:backend.example:8080",
			want:  Address{Kind: TCPConnect, Host: "backend.example", Port: 8080},
		},
		{
			name:  "bracketed IPv6 connector",
			input: "TCP-CONNECT:[::1]:8080",
			want:  Address{Kind: TCPConnect, Host: "::1", Port: 8080},
		},
		{
			name:  "Tailcat listener",
			input: "TAILCAT-LISTEN:8080",
			want:  Address{Kind: TailcatListen, Port: 8080},
		},
		{
			name:  "Tailcat connector reads handoff",
			input: "TAILCAT-CONNECT:@stdin",
			want:  Address{Kind: TailcatConnect, Input: Stdin},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseAddress(test.input)
			if err != nil {
				t.Fatalf("ParseAddress(%q) error = %v", test.input, err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("ParseAddress(%q) = %#v, want %#v", test.input, got, test.want)
			}
		})
	}
}

func TestParseAddressRejectsAmbiguousOrUnsupportedForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		wantErr string
	}{
		{"", "address is required"},
		{"tcp-listen:127.0.0.1:8080", "unknown address type"},
		{"UDP-LISTEN:127.0.0.1:8080", "unknown address type"},
		{"TCP-LISTEN::8080", "host is required"},
		{"TCP-CONNECT:127.0.0.1", "host:port"},
		{"TCP-CONNECT:127.0.0.1:0", "invalid port"},
		{"TCP-CONNECT:host name:8080", "invalid host"},
		{"TCP-LISTEN:127.0.0.1:8080,reuseaddr", "options are not supported"},
		{"TAILCAT-LISTEN:0", "invalid port"},
		{"TAILCAT-LISTEN:127.0.0.1:8080", "invalid port"},
		{"TAILCAT-CONNECT:tc-example", "must be @stdin"},
		{"TAILCAT-CONNECT:@stdin,keepalive", "options are not supported"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()
			_, err := ParseAddress(test.input)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("ParseAddress(%q) error = %v, want error containing %q", test.input, err, test.wantErr)
			}
		})
	}
}

func TestParseRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want Route
	}{
		{
			name: "ordinary TCP relay",
			args: []string{"TCP-LISTEN:127.0.0.1:18080", "TCP-CONNECT:127.0.0.1:8080"},
			want: Route{
				Ingress: Address{Kind: TCPListen, Host: "127.0.0.1", Port: 18080},
				Egress:  Address{Kind: TCPConnect, Host: "127.0.0.1", Port: 8080},
			},
		},
		{
			name: "right geographic half",
			args: []string{"TAILCAT-LISTEN:8080", "TCP-CONNECT:127.0.0.1:8080"},
			want: Route{
				Ingress: Address{Kind: TailcatListen, Port: 8080},
				Egress:  Address{Kind: TCPConnect, Host: "127.0.0.1", Port: 8080},
			},
		},
		{
			name: "left geographic half",
			args: []string{"TCP-LISTEN:127.0.0.1:18080", "TAILCAT-CONNECT:@stdin"},
			want: Route{
				Ingress: Address{Kind: TCPListen, Host: "127.0.0.1", Port: 18080},
				Egress:  Address{Kind: TailcatConnect, Input: Stdin},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseRoute(test.args)
			if err != nil {
				t.Fatalf("ParseRoute(%q) error = %v", test.args, err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("ParseRoute(%q) = %#v, want %#v", test.args, got, test.want)
			}
		})
	}
}

func TestParseRouteRequiresOneListenerAndOneConnector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"missing both", nil, "exactly two addresses"},
		{"missing connector", []string{"TCP-LISTEN:127.0.0.1:8080"}, "exactly two addresses"},
		{"extra address", []string{"TCP-LISTEN:127.0.0.1:8080", "TCP-CONNECT:127.0.0.1:80", "TCP-CONNECT:127.0.0.1:81"}, "exactly two addresses"},
		{"connector on left", []string{"TCP-CONNECT:127.0.0.1:8080", "TCP-CONNECT:127.0.0.1:80"}, "first address must listen"},
		{"listener on right", []string{"TCP-LISTEN:127.0.0.1:8080", "TAILCAT-LISTEN:80"}, "second address must connect"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseRoute(test.args)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("ParseRoute(%q) error = %v, want error containing %q", test.args, err, test.wantErr)
			}
		})
	}
}

func FuzzParseAddress(f *testing.F) {
	for _, seed := range []string{
		"",
		"TCP-LISTEN:127.0.0.1:8080",
		"TCP-CONNECT:[::1]:443",
		"TAILCAT-LISTEN:8080",
		"TAILCAT-CONNECT:@stdin",
		"TCP-LISTEN:127.0.0.1:8080,reuseaddr",
		"tcp",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		address, err := ParseAddress(input)
		if err != nil {
			return
		}
		if !address.listens() && !address.connects() {
			t.Fatalf("ParseAddress(%q) returned address with no role: %#v", input, address)
		}
	})
}
