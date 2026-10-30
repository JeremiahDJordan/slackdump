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
	"testing"

	"github.com/rusq/slackdump/v4/internal/structures"
)

func TestSingleChannel(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"one channel ID", []string{"C01581023"}, false},
		{"one channel URL", []string{"https://example.slack.com/archives/C01581023"}, false},
		{"channel and its thread", []string{"C01581023", "C01581023:1700000000.000100"}, false},
		{"channel with exclude", []string{"C01581023", "^C99999999"}, false},
		{"no channels", []string{}, true},
		{"only exclude", []string{"^C01581023"}, true},
		{"two channels", []string{"C01581023", "C99999999"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, err := structures.NewEntityList(tt.args)
			if err != nil {
				t.Fatalf("NewEntityList: %v", err)
			}
			err = SingleChannel(list)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SingleChannel() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, ErrSingleChannel) {
				t.Errorf("want ErrSingleChannel, got %v", err)
			}
		})
	}
	t.Run("nil list", func(t *testing.T) {
		if err := SingleChannel(nil); !errors.Is(err, ErrSingleChannel) {
			t.Errorf("want ErrSingleChannel, got %v", err)
		}
	})
}
