// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatrix2SetString(t *testing.T) {
	tests := []struct {
		str     string
		wantErr bool
		want    Matrix2
	}{
		{
			str:     "none",
			wantErr: false,
			want:    Identity2(),
		},
		{
			str:     "matrix(1, 2, 3, 4, 5, 6)",
			wantErr: false,
			want:    Matrix2{1, 2, 3, 4, 5, 6},
		},
		{
			str:     "translate(1, 2)",
			wantErr: false,
			want:    Matrix2{XX: 1, YX: 0, XY: 0, YY: 1, X0: 1, Y0: 2},
		},
		{
			str:     "invalid(1, 2)",
			wantErr: true,
			want:    Identity2(),
		},
	}

	for _, tt := range tests {
		a := &Matrix2{}
		err := a.SetString(tt.str)
		if tt.wantErr {
			assert.Error(t, err, tt.str)
		} else {
			assert.NoError(t, err, tt.str)
		}
		assert.Equal(t, tt.want, *a, tt.str)
	}
}
