// Copyright 2022 The sila Authors
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

package node

import (
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/sila-org/sila/rpc"
)

// NewJWTAuth creates an rpc client authentication provider that uses JWT. The
// secret MUST be 32 bytes (256 bits) as defined by the SilaEngine-API authentication spec.
//
// See https://github.com/sila-org/execution-apis/blob/main/src/silaEngine/authentication.md
// for more details about this authentication scheme.
func NewJWTAuth(jwtsecret [32]byte) rpc.HTTPAuth {
	return func(h http.Header) error {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"iat": &jwt.NumericDate{Time: time.Now()},
		})
		s, err := token.SignedString(jwtsecret[:])
		if err != nil {
			return fmt.Errorf("failed to create JWT token: %w", err)
		}
		h.Set("Authorization", "Bearer "+s)
		return nil
	}
}
