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

package silaclient_test

import (
	"github.com/sila-org/sila/node"
)

var exampleNode *node.Node

// launch example server
func init() {
	config := &node.Config{
		HTTPHost: "127.0.0.1",
	}
	n, _, err := newTestBackend(config)
	if err != nil {
		panic("can't launch node: " + err.Error())
	}
	exampleNode = n
}
