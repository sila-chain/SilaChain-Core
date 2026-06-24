// Copyright 2024 The sila Authors
// This file is part of the sila library.
//
// The sila library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The sila library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the sila library. If not, see <http://www.gnu.org/licenses/>.

package rawdb

import (
	"testing"

	"github.com/sila-org/sila/core/rawdb/ancienttest"
	"github.com/sila-org/sila/siladb"
)

func TestMemoryFreezer(t *testing.T) {
	ancienttest.TestAncientSuite(t, func(kinds []string) siladb.AncientStore {
		tables := make(map[string]freezerTableConfig)
		for _, kind := range kinds {
			tables[kind] = freezerTableConfig{
				noSnappy:  true,
				tailGroup: ancienttest.TailGroup,
			}
		}
		return NewMemoryFreezer(false, tables)
	})
	ancienttest.TestResettableAncientSuite(t, func(kinds []string) siladb.ResettableAncientStore {
		tables := make(map[string]freezerTableConfig)
		for _, kind := range kinds {
			tables[kind] = freezerTableConfig{
				noSnappy:  true,
				tailGroup: ancienttest.TailGroup,
			}
		}
		return NewMemoryFreezer(false, tables)
	})
}
