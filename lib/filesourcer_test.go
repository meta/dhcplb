/**
 * Copyright (c) Facebook, Inc. and its affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package dhcplb

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetServersFromTierAddressFamily(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts")
	if err := os.WriteFile(path, []byte("localhost\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, version := range []int{4, 6} {
		fs := &FileSourcer{version: version}
		servers, err := fs.GetServersFromTier(path)
		if err != nil {
			t.Fatal(err)
		}
		// localhost may not have an address of both families, but any
		// server that is returned must use the configured family.
		for _, s := range servers {
			if s.Address == nil || (s.Address.To4() != nil) != (version == 4) {
				t.Errorf("version %d: localhost resolved to %v", version, s.Address)
			}
		}
	}
}
