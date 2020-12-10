// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package security

import (
	"bytes"
	"log"
	"net/http"
	"os"
)

// MakeGetRequestFixture is an HTTP Cloud Function to facilitate testing security snippets.
func MakeGetRequestFixture(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("TARGET_URL") == "" {
		w.Write([]byte("Success!"))
		return
	}

	var b bytes.Buffer
	if err := makeGetRequest(&b, os.Getenv("TARGET_URL")); err != nil {
		log.Printf("makeGetRequest: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Write(b.Bytes())
}
