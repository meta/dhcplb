/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package dhcplb

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddressForVersion(t *testing.T) {
	tests := []struct {
		name    string
		addrs   []string
		version int
		want    net.IP
	}{
		{"V4SkipsIPv6", []string{"::1", "127.0.0.1"}, 4, net.ParseIP("127.0.0.1")},
		{"V6SkipsIPv4", []string{"127.0.0.1", "::1"}, 6, net.ParseIP("::1")},
		{"V4NoIPv4", []string{"::1"}, 4, nil},
		{"V6NoIPv6", []string{"127.0.0.1"}, 6, nil},
		{"V6SkipsUnparseable", []string{"fe80::1%eth0", "::1"}, 6, net.ParseIP("::1")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, addressForVersion(tt.addrs, tt.version))
		})
	}
}

func TestGetServersFromTierSkipsHostsWithoutAddress(t *testing.T) {
	// LookupHost returns a zoned literal like fe80::1%eth0 unchanged and ParseIP rejects it.
	tests := []struct {
		name    string
		version int
		lines   string
		want    []*DHCPServer
	}{
		{
			name:    "V4Mixed",
			version: 4,
			lines:   "10.0.0.1\nfe80::1%eth0\n10.0.0.2:6767\n",
			want: []*DHCPServer{
				NewDHCPServer("10.0.0.1", net.ParseIP("10.0.0.1"), 67),
				NewDHCPServer("10.0.0.2", net.ParseIP("10.0.0.2"), 6767),
			},
		},
		{
			name:    "V6Mixed",
			version: 6,
			lines:   "2001:db8::1\nfe80::1%eth0\n[2001:db8::2]:5470\n",
			want: []*DHCPServer{
				NewDHCPServer("2001:db8::1", net.ParseIP("2001:db8::1"), 547),
				NewDHCPServer("2001:db8::2", net.ParseIP("2001:db8::2"), 5470),
			},
		},
		{name: "V4NoneUsable", version: 4, lines: "fe80::1%eth0\n"},
		{name: "V6NoneUsable", version: 6, lines: "fe80::1%eth0\n[fe80::2%eth0]:547\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "servers")
			require.NoError(t, os.WriteFile(path, []byte(tt.lines), 0o600))
			fs := &FileSourcer{version: tt.version}
			servers, err := fs.GetServersFromTier(path)
			require.NoError(t, err)
			require.Equal(t, tt.want, servers)
		})
	}
}
