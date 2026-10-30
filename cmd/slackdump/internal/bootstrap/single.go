// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package bootstrap

import (
	"errors"
	"fmt"

	"github.com/rusq/slackdump/v4/internal/structures"
)

// ErrSingleChannel is returned when the entity list does not resolve to
// exactly one channel.
var ErrSingleChannel = errors.New("single-channel mode: specify exactly one channel ID or URL")

// SingleChannel ensures that list includes exactly one channel.  Thread links
// within that channel are allowed.
func SingleChannel(list *structures.EntityList) error {
	if list == nil {
		return fmt.Errorf("%w (got 0)", ErrSingleChannel)
	}
	channels := make(map[string]struct{})
	for id, item := range list.Index() {
		if !item.Include {
			continue
		}
		sl, err := structures.ParseLink(id)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrSingleChannel, err)
		}
		channels[sl.Channel] = struct{}{}
	}
	if len(channels) != 1 {
		return fmt.Errorf("%w (got %d)", ErrSingleChannel, len(channels))
	}
	return nil
}
