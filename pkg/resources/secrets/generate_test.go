// Copyright 2018 Oracle and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package secrets

import "testing"

func TestRandomAlphanumericString(t *testing.T) {
	password := RandomAlphanumericString(10)
	if len(password) != 10 {
		t.Error("Expected password of length 10, but got ", len(password))
	}
}

func TestRandomAlphanumericStringShouldBeRandom(t *testing.T) {
	x := RandomAlphanumericString(10)
	y := RandomAlphanumericString(10)
	if x == y {
		t.Errorf("Expected two unique passwords but got %s and %s", x, y)
	}
}
