/*
Copyright 2021-2022 The Volcano Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vsuspend

// TODO: just copied.
//  Passed.

import (
	"encoding/json"
	batchv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/batch/v1alpha1"
	busv1alpha1 "github.com/hliangzhao/volcano/pkg/apis/bus/v1alpha1"
	"github.com/spf13/cobra"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSuspendJobJob(t *testing.T) {
	responseCommand := busv1alpha1.Command{}
	responseJob := batchv1alpha1.Job{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "command") {
			w.Header().Set("Content-Type", "application/json")
			val, err := json.Marshal(responseCommand)
			if err == nil {
				w.Write(val)
			}

		} else {
			w.Header().Set("Content-Type", "application/json")
			val, err := json.Marshal(responseJob)
			if err == nil {
				w.Write(val)
			}

		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	suspendJobFlags.Master = server.URL
	suspendJobFlags.Namespace = "test"
	suspendJobFlags.JobName = "testjob"

	testCases := []struct {
		Name        string
		ExpectValue error
	}{
		{
			Name:        "SuspendJob",
			ExpectValue: nil,
		},
	}

	for i, testcase := range testCases {
		err := SuspendJob()
		if err != nil {
			t.Errorf("case %d (%s): expected: %v, got %v ", i, testcase.Name, testcase.ExpectValue, err)
		}
	}

}

func TestInitSuspendFlags(t *testing.T) {
	var cmd cobra.Command
	InitSuspendFlags(&cmd)

	if cmd.Flag("namespace") == nil {
		t.Errorf("Could not find the flag namespace")
	}
	if cmd.Flag("name") == nil {
		t.Errorf("Could not find the flag name")
	}

}
