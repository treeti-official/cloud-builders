/*
Copyright 2019 Google, Inc. All rights reserved.
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
package resource

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/GoogleCloudPlatform/cloud-builders/gke-deploy/services"
	"github.com/GoogleCloudPlatform/cloud-builders/gke-deploy/testservices"
)

func TestEncoder(t *testing.T) {
	testDeploymentFile := "testing/deployment.yaml"
	testServiceFile := "testing/service.yaml"

	tests := []struct {
		name string

		obj *Object

		want string
	}{{
		name: "Encode deployment",

		obj: newObjectFromFile(t, testDeploymentFile),

		want: string(fileContents(t, testDeploymentFile)),
	}, {
		name: "Encode service",

		obj: newObjectFromFile(t, testServiceFile),

		want: string(fileContents(t, testServiceFile)),
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := runtime.Encode(encoder, tc.obj); string(got) != tc.want || err != nil {
				t.Errorf("Encode(encoder, %v) = %v, %v; want %v, <nil>", tc.obj, string(got), err, tc.want)
			}
		})
	}
}

func TestParseYaml(t *testing.T) {
	ctx := context.Background()

	testDeploymentFile := "testing/deployment.yaml"
	testServiceFile := "testing/service.yaml"

	tests := []struct {
		name string

		yaml []byte

		want *Object
	}{{
		name: "Decode deployment",

		yaml: fileContents(t, testDeploymentFile),

		want: &Object{
			&unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "extensions/v1beta1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "test-app",
						},
						"name": "test-app",
					},
					"spec": map[string]interface{}{
						"replicas": int64(1),
						"selector": map[string]interface{}{
							"matchLabels": map[string]interface{}{
								"app": "test-app",
							},
						},
						"template": map[string]interface{}{
							"metadata": map[string]interface{}{
								"labels": map[string]interface{}{
									"app": "test-app",
								},
							},
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"image": "gcr.io/cbd-test/test-app:latest",
										"name":  "test-app",
									},
								},
							},
						},
					},
				},
			},
		},
	}, {
		name: "Decode service",

		yaml: fileContents(t, testServiceFile),

		want: &Object{
			&unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "test-app",
						},
						"name": "test-app",
					},
					"spec": map[string]interface{}{
						"ports": []interface{}{
							map[string]interface{}{
								"port":       int64(80),
								"protocol":   "TCP",
								"targetPort": int64(8080),
							},
						},
						"selector": map[string]interface{}{
							"app": "test-app",
						},
						"type": "LoadBalancer",
					},
				},
			},
		},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := DecodeFromYAML(ctx, tc.yaml); !reflect.DeepEqual(got, tc.want) || err != nil {
				t.Errorf("DecodeFromYAML(ctx, %s) = %v, %v; want %v, <nil>", tc.yaml, got, err, tc.want)
			}
		})
	}
}

func TestSaveAsConfigs(t *testing.T) {
	ctx := context.Background()

	testDeploymentFile := "testing/deployment.yaml"
	testServiceFile := "testing/service.yaml"

	outputDir := "path/to/output"

	tests := []struct {
		name string

		objs         Objects
		outputDir    string
		lineComments map[string]string
		oss          services.OSService

		want Objects
	}{{
		name: "Zero objects",

		outputDir:    outputDir,
		objs:         Objects{},
		lineComments: nil,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				outputDir: {
					Res: nil,
					Err: os.ErrNotExist,
				},
			},
			MkdirAllResponse: map[string]error{
				outputDir: nil,
			},
			WriteFileResponse: map[string]error{
				filepath.Join(outputDir, AggregatedFilename): nil,
			},
		},
	}, {
		name: "Non-zero objects",

		outputDir: outputDir,
		objs: Objects{
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testServiceFile),
		},
		lineComments: nil,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				outputDir: {
					Res: nil,
					Err: os.ErrNotExist,
				},
			},
			MkdirAllResponse: map[string]error{
				outputDir: nil,
			},
			WriteFileResponse: map[string]error{
				filepath.Join(outputDir, AggregatedFilename): nil,
			},
		},
	}, {
		name:      "Output directory exists and is empty",
		outputDir: outputDir,
		objs: Objects{
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testServiceFile),
		},
		lineComments: nil,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				outputDir: {
					Res: nil,
					Err: os.ErrNotExist,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				outputDir: {
					Res: []os.FileInfo{},
					Err: nil,
				},
			},
			MkdirAllResponse: map[string]error{
				outputDir: nil,
			},
			WriteFileResponse: map[string]error{
				filepath.Join(outputDir, AggregatedFilename): nil,
			},
		},
	}, {
		name: "Non-zero objects",

		outputDir: outputDir,
		objs: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
		lineComments: map[string]string{
			"unfound":                                "abc",
			"image: gcr.io/cbd-test/test-app:latest": "comment 123",
		},
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				outputDir: {
					Res: nil,
					Err: os.ErrNotExist,
				},
			},
			MkdirAllResponse: map[string]error{
				outputDir: nil,
			},
			WriteFileResponse: map[string]error{
				filepath.Join(outputDir, AggregatedFilename): nil,
			},
		},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := SaveAsConfigs(ctx, tc.objs, tc.outputDir, tc.lineComments, tc.oss); err != nil {
				t.Errorf("SaveAsConfigs(ctx, %v, %s, %v, oss) = %v; want <nil>", tc.objs, tc.outputDir, tc.lineComments, err)
			}
		})
	}
}

func TestSaveAsConfigsErrors(t *testing.T) {
	ctx := context.Background()

	testDeploymentFile := "testing/deployment.yaml"

	outputDir := "path/to/output"

	tests := []struct {
		name string

		objs         Objects
		outputDir    string
		lineComments map[string]string
		oss          services.OSService

		want Objects
	}{{
		name: "Failed to make directory",

		outputDir:    outputDir,
		lineComments: nil,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				outputDir: {
					Res: nil,
					Err: os.ErrNotExist,
				},
			},
			MkdirAllResponse: map[string]error{
				outputDir: fmt.Errorf("failed to make directory"),
			},
		},
	}, {
		name: "Failed to write file",

		outputDir:    outputDir,
		lineComments: nil,
		objs: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				outputDir: {
					Res: nil,
					Err: os.ErrNotExist,
				},
			},
			MkdirAllResponse: map[string]error{
				outputDir: nil,
			},
			WriteFileResponse: map[string]error{
				filepath.Join(outputDir, AggregatedFilename): fmt.Errorf("failed to write file"),
			},
		},
	}, {
		name: "Failed to stat output directory",

		outputDir:    outputDir,
		lineComments: nil,
		objs: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				outputDir: {
					Res: nil,
					Err: fmt.Errorf("failed to stat file"),
				},
			},
		},
	}, {
		name:         "Output directory exists and is not empty",
		outputDir:    outputDir,
		lineComments: nil,
		objs: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				outputDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: true,
					},
					Err: nil,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				outputDir: {
					Res: []os.FileInfo{
						&testservices.TestFileInfo{
							BaseName:    "existing.txt",
							IsDirectory: false,
						},
					},
					Err: nil,
				},
			},
		},
	}, {
		name:         "Failed to read output directory",
		outputDir:    outputDir,
		lineComments: nil,
		objs: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				outputDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: true,
					},
					Err: nil,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				outputDir: {
					Res: nil,
					Err: fmt.Errorf("failed to read directory"),
				},
			},
		},
	}, {
		name:         "Output directory exists and is a file",
		outputDir:    outputDir,
		lineComments: nil,
		objs: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				outputDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: false,
					},
					Err: nil,
				},
			},
		},
	}, {
		name: "Line to add comment to contains newline character",

		outputDir: outputDir,
		objs: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
		lineComments: map[string]string{
			"asdf\nasdf": "asdf",
		},
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				outputDir: {
					Res: nil,
					Err: os.ErrNotExist,
				},
			},
			MkdirAllResponse: map[string]error{
				outputDir: nil,
			},
		},
	}, {
		name: "Comment to add contains newline character",

		outputDir: outputDir,
		objs: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
		lineComments: map[string]string{
			"asdf": "asdf\nasdf",
		},
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				outputDir: {
					Res: nil,
					Err: os.ErrNotExist,
				},
			},
			MkdirAllResponse: map[string]error{
				outputDir: nil,
			},
		},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := SaveAsConfigs(ctx, tc.objs, tc.outputDir, tc.lineComments, tc.oss); err == nil {
				t.Errorf("SaveAsConfigs(ctx, %v, %s, %v, oss) = <nil>; want error", tc.objs, tc.outputDir, tc.lineComments)
			}
		})
	}
}

func TestParseConfigs(t *testing.T) {
	ctx := context.Background()

	testDeploymentFile := "testing/deployment.yaml"
	testServiceFile := "testing/service.yaml"
	testMultiResourceFile := "testing/multi-resource.yaml"
	testMultiResourceWithWhitespaceFile := "testing/multi-resource-with-whitespace.yaml"
	testWhitespaceAndCommentsFile := "testing/whitespace-and-comments.yaml"

	configsDir := "path/to/configs"
	deploymentYaml := "deployment.yaml"
	deploymentYml := "deployment.yml"
	serviceYaml := "service.yaml"
	multiResourceYaml := "multi-resource.yaml"
	multiResourceWithWhitespaceYaml := "multi-resource-with-whitespace.yaml"

	tests := []struct {
		name string

		configs string
		oss     services.OSService

		want Objects
	}{{
		name: "Configs is a directory with single .yaml file",

		configs: configsDir,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				configsDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: true,
					},
					Err: nil,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				configsDir: {
					Res: []os.FileInfo{
						&testservices.TestFileInfo{
							BaseName:    deploymentYaml,
							IsDirectory: false,
						},
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				filepath.Join(configsDir, deploymentYaml): {
					Res: fileContents(t, testDeploymentFile),
					Err: nil,
				},
			},
		},

		want: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
	}, {
		name: "Configs is a directory with single .yml file",

		configs: configsDir,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				configsDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: true,
					},
					Err: nil,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				configsDir: {
					Res: []os.FileInfo{
						&testservices.TestFileInfo{
							BaseName:    deploymentYml,
							IsDirectory: false,
						},
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				filepath.Join(configsDir, deploymentYml): {
					Res: fileContents(t, testDeploymentFile),
					Err: nil,
				},
			},
		},

		want: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
	}, {
		name: "Configs is a directory with multiple .yaml files",

		configs: configsDir,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				configsDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: true,
					},
					Err: nil,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				configsDir: {
					Res: []os.FileInfo{
						&testservices.TestFileInfo{
							BaseName:    deploymentYaml,
							IsDirectory: false,
						},
						&testservices.TestFileInfo{
							BaseName:    serviceYaml,
							IsDirectory: false,
						},
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				filepath.Join(configsDir, deploymentYaml): {
					Res: fileContents(t, testDeploymentFile),
					Err: nil,
				},
				filepath.Join(configsDir, serviceYaml): {
					Res: fileContents(t, testServiceFile),
					Err: nil,
				},
			},
		},

		want: Objects{
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testServiceFile),
		},
	}, {
		name: "Configs is a directory containing a multi-resource .yaml file",

		configs: configsDir,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				configsDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: true,
					},
					Err: nil,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				configsDir: {
					Res: []os.FileInfo{
						&testservices.TestFileInfo{
							BaseName:    multiResourceYaml,
							IsDirectory: false,
						},
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				filepath.Join(configsDir, multiResourceYaml): {
					Res: fileContents(t, testMultiResourceFile),
					Err: nil,
				},
			},
		},

		want: Objects{
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testServiceFile),
		},
	}, {
		name: "Configs is a directory containing a multi-resource .yaml file and single-resource .yaml file",

		configs: configsDir,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				configsDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: true,
					},
					Err: nil,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				configsDir: {
					Res: []os.FileInfo{
						&testservices.TestFileInfo{
							BaseName:    multiResourceYaml,
							IsDirectory: false,
						},
						&testservices.TestFileInfo{
							BaseName:    deploymentYaml,
							IsDirectory: false,
						},
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				filepath.Join(configsDir, multiResourceYaml): {
					Res: fileContents(t, testMultiResourceFile),
					Err: nil,
				},
				filepath.Join(configsDir, deploymentYaml): {
					Res: fileContents(t, testDeploymentFile),
					Err: nil,
				},
			},
		},

		want: Objects{
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testServiceFile),
			newObjectFromFile(t, testDeploymentFile),
		},
	}, {
		name: "Configs is a directory containing two multi-resource .yaml files",

		configs: configsDir,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				configsDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: true,
					},
					Err: nil,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				configsDir: {
					Res: []os.FileInfo{
						&testservices.TestFileInfo{
							BaseName:    multiResourceYaml,
							IsDirectory: false,
						},
						&testservices.TestFileInfo{
							BaseName:    "multi-resource-2.yaml",
							IsDirectory: false,
						},
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				filepath.Join(configsDir, multiResourceYaml): {
					Res: fileContents(t, testMultiResourceFile),
					Err: nil,
				},
				filepath.Join(configsDir, "multi-resource-2.yaml"): {
					Res: fileContents(t, testMultiResourceFile),
					Err: nil,
				},
			},
		},

		want: Objects{
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testServiceFile),
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testServiceFile),
		},
	}, {
		name: "Configs is a directory containing a multi-resource .yaml file with whitespace",

		configs: configsDir,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				configsDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: true,
					},
					Err: nil,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				configsDir: {
					Res: []os.FileInfo{
						&testservices.TestFileInfo{
							BaseName:    multiResourceWithWhitespaceYaml,
							IsDirectory: false,
						},
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				filepath.Join(configsDir, multiResourceWithWhitespaceYaml): {
					Res: fileContents(t, testMultiResourceWithWhitespaceFile),
					Err: nil,
				},
			},
		},

		want: Objects{
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testServiceFile),
		},
	}, {
		name: "Configs is .yaml file",

		configs: deploymentYaml,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				deploymentYaml: {
					Res: &testservices.TestFileInfo{
						IsDirectory: false,
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				deploymentYaml: {
					Res: fileContents(t, testDeploymentFile),
					Err: nil,
				},
			},
		},

		want: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
	}, {
		name: "Configs is .yml file",

		configs: deploymentYml,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				deploymentYml: {
					Res: &testservices.TestFileInfo{
						IsDirectory: false,
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				deploymentYml: {
					Res: fileContents(t, testDeploymentFile),
					Err: nil,
				},
			},
		},

		want: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
	}, {
		name: "Configs is a multi-resource .yaml file",

		configs: multiResourceYaml,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				multiResourceYaml: {
					Res: &testservices.TestFileInfo{
						IsDirectory: false,
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				multiResourceYaml: {
					Res: fileContents(t, testMultiResourceFile),
					Err: nil,
				},
			},
		},

		want: Objects{
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testServiceFile),
		},
	}, {
		name: "Configs is a directory containing files that lead to collisions",

		configs: configsDir,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				configsDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: true,
					},
					Err: nil,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				configsDir: {
					Res: []os.FileInfo{
						&testservices.TestFileInfo{
							BaseName:    "multi-resource-deployment-test-app.yaml",
							IsDirectory: false,
						},
						&testservices.TestFileInfo{
							BaseName:    "multi-resource-service-test-app.yaml",
							IsDirectory: false,
						},
						&testservices.TestFileInfo{
							BaseName:    multiResourceYaml,
							IsDirectory: false,
						},
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				filepath.Join(configsDir, "multi-resource-deployment-test-app.yaml"): {
					Res: fileContents(t, testDeploymentFile),
					Err: nil,
				},
				filepath.Join(configsDir, "multi-resource-service-test-app.yaml"): {
					Res: fileContents(t, testServiceFile),
					Err: nil,
				},
				filepath.Join(configsDir, multiResourceYaml): {
					Res: fileContents(t, testMultiResourceFile),
					Err: nil,
				},
			},
		},

		want: Objects{
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testServiceFile),
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testServiceFile),
		},
	}, {
		name: "Configs is stdin with single object",

		configs: "-",
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				"-": {
					Res: &testservices.TestFileInfo{
						IsDirectory: false,
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				"-": {
					Res: fileContents(t, testDeploymentFile),
					Err: nil,
				},
			},
		},

		want: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
	}, {
		name: "Configs is stdin with multiple objects",

		configs: "-",
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				"-": {
					Res: &testservices.TestFileInfo{
						IsDirectory: false,
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				"-": {
					Res: fileContents(t, testMultiResourceFile),
					Err: nil,
				},
			},
		},

		want: Objects{
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testServiceFile),
		},
	}, {
		name: "Do not parse file with only comments and whitespace",

		configs: "file.yaml",
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				"file.yaml": {
					Res: &testservices.TestFileInfo{
						IsDirectory: false,
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				filepath.Join("file.yaml"): {
					Res: fileContents(t, testWhitespaceAndCommentsFile),
					Err: nil,
				},
			},
		},

		want: Objects{},
	}, {
		name: "Do not parse file in dir with only comments and whitespace",

		configs: configsDir,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				configsDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: true,
					},
					Err: nil,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				configsDir: {
					Res: []os.FileInfo{
						&testservices.TestFileInfo{
							BaseName:    "file.yaml",
							IsDirectory: false,
						},
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				filepath.Join(configsDir, "file.yaml"): {
					Res: fileContents(t, testWhitespaceAndCommentsFile),
					Err: nil,
				},
			},
		},

		want: Objects{},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := ParseConfigs(ctx, tc.configs, tc.oss); !reflect.DeepEqual(got, tc.want) || err != nil {
				t.Errorf("ParseConfigs(ctx, %s, oss) = %v, %v; want %v, <nil>", tc.configs, got, err, tc.want)
			}
		})
	}
}

func TestParseConfigsErrors(t *testing.T) {
	ctx := context.Background()

	configsDir := "path/to/configs"
	deploymentYaml := "deployment.yaml"
	txtFile := "file.txt"

	tests := []struct {
		name string

		configs string
		oss     services.OSService
	}{{
		name: "Failed to get file info",

		configs: configsDir,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				configsDir: {
					Res: nil,
					Err: fmt.Errorf("failed to call stat"),
				},
			},
		},
	}, {
		name: "Failed to read directory",

		configs: configsDir,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				configsDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: true,
					},
					Err: nil,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				configsDir: {
					Res: nil,
					Err: fmt.Errorf("failed to read directory"),
				},
			},
		},
	}, {
		name: "Failed to read file",

		configs: deploymentYaml,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				deploymentYaml: {
					Res: &testservices.TestFileInfo{
						IsDirectory: false,
					},
					Err: nil,
				},
			},
			ReadFileResponse: map[string]testservices.ReadFileResponse{
				deploymentYaml: {
					Res: nil,
					Err: fmt.Errorf("failed to read file"),
				},
			},
		},
	}, {
		name: "Configs is a directory with no files",

		configs: configsDir,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				configsDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: true,
					},
					Err: nil,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				configsDir: {
					Res: []os.FileInfo{},
					Err: nil,
				},
			},
		},
	}, {
		name: "Configs is a file that does not end in .yaml or .yaml",

		configs: txtFile,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				txtFile: {
					Res: &testservices.TestFileInfo{
						IsDirectory: false,
					},
					Err: nil,
				},
			},
		},
	}, {
		name: "Configs is a directory with no .yaml or .yml files",

		configs: configsDir,
		oss: &testservices.TestOS{
			StatResponse: map[string]testservices.StatResponse{
				configsDir: {
					Res: &testservices.TestFileInfo{
						IsDirectory: true,
					},
					Err: nil,
				},
			},
			ReadDirResponse: map[string]testservices.ReadDirResponse{
				configsDir: {
					Res: []os.FileInfo{
						&testservices.TestFileInfo{
							BaseName: txtFile,
						},
					},
					Err: nil,
				},
			},
		},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := ParseConfigs(ctx, tc.configs, tc.oss); got != nil || err == nil {
				t.Errorf("ParseConfigs(ctx, %s, oss) = %v, <nil>; want <nil>, error", tc.configs, got)
			}
		})
	}
}

func TestUpdateMatchingContainerImage(t *testing.T) {
	ctx := context.Background()

	testCronjobFile := "testing/cronjob.yaml"
	testCronjobUpdatedFile := "testing/cronjob-updated.yaml"
	testDaemonsetFile := "testing/daemonset.yaml"
	testDaemonsetUpdatedFile := "testing/daemonset-updated.yaml"
	testDeploymentFile := "testing/deployment.yaml"
	testDeploymentUpdatedFile := "testing/deployment-updated.yaml"
	testJobFile := "testing/job.yaml"
	testJobUpdatedFile := "testing/job-updated.yaml"
	testPodFile := "testing/pod.yaml"
	testPodUpdatedFile := "testing/pod-updated.yaml"
	testReplicasetFile := "testing/replicaset.yaml"
	testReplicasetUpdatedFile := "testing/replicaset-updated.yaml"
	testReplicationcontrollerFile := "testing/replicationcontroller.yaml"
	testReplicationcontrollerUpdatedFile := "testing/replicationcontroller-updated.yaml"
	testStatefulsetFile := "testing/statefulset.yaml"
	testStatefulsetUpdatedFile := "testing/statefulset-updated.yaml"
	testDeployment2File := "testing/deployment-2.yaml"
	testDeployment3File := "testing/deployment-3.yaml"
	testDeploymentUpdated3File := "testing/deployment-updated-2.yaml"

	imageName := "gcr.io/cbd-test/test-app"
	replace := "REPLACED"

	tests := []struct {
		name string

		objs Objects

		beforeUpdate Objects
		want         Objects
	}{{
		name: "Empty objects",

		objs: Objects{},

		beforeUpdate: Objects{},
		want:         Objects{},
	}, {
		name: "Update objects",

		objs: Objects{
			newObjectFromFile(t, testCronjobFile),
			newObjectFromFile(t, testDaemonsetFile),
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testJobFile),
			newObjectFromFile(t, testPodFile),
			newObjectFromFile(t, testReplicasetFile),
			newObjectFromFile(t, testReplicationcontrollerFile),
			newObjectFromFile(t, testStatefulsetFile),
		},

		beforeUpdate: Objects{
			newObjectFromFile(t, testCronjobFile),
			newObjectFromFile(t, testDaemonsetFile),
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testJobFile),
			newObjectFromFile(t, testPodFile),
			newObjectFromFile(t, testReplicasetFile),
			newObjectFromFile(t, testReplicationcontrollerFile),
			newObjectFromFile(t, testStatefulsetFile),
		},
		want: Objects{
			newObjectFromFile(t, testCronjobUpdatedFile),
			newObjectFromFile(t, testDaemonsetUpdatedFile),
			newObjectFromFile(t, testDeploymentUpdatedFile),
			newObjectFromFile(t, testJobUpdatedFile),
			newObjectFromFile(t, testPodUpdatedFile),
			newObjectFromFile(t, testReplicasetUpdatedFile),
			newObjectFromFile(t, testReplicationcontrollerUpdatedFile),
			newObjectFromFile(t, testStatefulsetUpdatedFile),
		},
	}, {
		name: "Nothing to update",

		objs: Objects{
			newObjectFromFile(t, testDeployment2File),
		},

		beforeUpdate: Objects{
			newObjectFromFile(t, testDeployment2File),
		},
		want: Objects{
			newObjectFromFile(t, testDeployment2File),
		},
	}, {
		name: "Second image is substring of first",

		objs: Objects{
			newObjectFromFile(t, testDeployment3File),
		},

		beforeUpdate: Objects{
			newObjectFromFile(t, testDeployment3File),
		},
		want: Objects{
			newObjectFromFile(t, testDeploymentUpdated3File),
		},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := UpdateMatchingContainerImage(ctx, tc.objs, imageName, replace); !reflect.DeepEqual(tc.objs, tc.want) || err != nil {
				t.Errorf("UpdateMatchingContainerImage(ctx, %v, %s, %s) = %v, %v; want <nil>, %v", tc.beforeUpdate, imageName, replace, err, tc.objs, tc.want)
			}
		})
	}
}

func TestAddLabel(t *testing.T) {
	ctx := context.Background()

	testDeploymentFile := "testing/deployment.yaml"
	testDeploymentUpdatedLabelFile := "testing/deployment-updated-label.yaml"
	testDeploymentUpdatedLabel2File := "testing/deployment-updated-label-2.yaml"
	testCronjobFile := "testing/cronjob.yaml"
	testCronjobUpdatedLabelFile := "testing/cronjob-updated-label.yaml"
	testDaemonsetFile := "testing/daemonset.yaml"
	testDaemonsetUpdatedLabelFile := "testing/daemonset-updated-label.yaml"
	testDaemonsetUpdatedLabel2File := "testing/daemonset-updated-label-2.yaml"
	testJobFile := "testing/job.yaml"
	testJobUpdatedLabelFile := "testing/job-updated-label.yaml"
	testReplicasetFile := "testing/replicaset.yaml"
	testReplicasetUpdatedLabelFile := "testing/replicaset-updated-label.yaml"
	testReplicationcontrollerFile := "testing/replicationcontroller.yaml"
	testReplicationcontrollerUpdatedLabelFile := "testing/replicationcontroller-updated-label.yaml"
	testStatefulsetFile := "testing/statefulset.yaml"
	testStatefulsetUpdatedLabelFile := "testing/statefulset-updated-label.yaml"

	tests := []struct {
		name string

		obj      *Object
		key      string
		value    string
		override bool

		beforeUpdate *Object
		want         *Object
	}{{
		name: "Override key",

		obj:      newObjectFromFile(t, testDeploymentFile),
		key:      "app",
		value:    "OVERRIDDEN",
		override: true,

		beforeUpdate: newObjectFromFile(t, testDeploymentFile),
		want:         newObjectFromFile(t, testDeploymentUpdatedLabelFile),
	}, {
		name: "Does not override key",

		obj:      newObjectFromFile(t, testDeploymentFile),
		key:      "app",
		value:    "OVERRIDDEN",
		override: false,

		beforeUpdate: newObjectFromFile(t, testDeploymentFile),
		want:         newObjectFromFile(t, testDeploymentFile),
	}, {
		name: "Normal case",

		obj:      newObjectFromFile(t, testDeploymentFile),
		key:      "foo",
		value:    "bar",
		override: false,

		beforeUpdate: newObjectFromFile(t, testDeploymentFile),
		want:         newObjectFromFile(t, testDeploymentUpdatedLabel2File),
	}, {
		name: "No existing labels field",

		obj:      newObjectFromFile(t, testCronjobFile),
		key:      "foo",
		value:    "bar",
		override: false,

		beforeUpdate: newObjectFromFile(t, testCronjobFile),
		want:         newObjectFromFile(t, testCronjobUpdatedLabelFile),
	}, {
		name: "DaemonSet nested template",

		obj:      newObjectFromFile(t, testDaemonsetFile),
		key:      "foo",
		value:    "bar",
		override: false,

		beforeUpdate: newObjectFromFile(t, testDaemonsetFile),
		want:         newObjectFromFile(t, testDaemonsetUpdatedLabelFile),
	}, {
		name:     "DaemonSet nested template, no override",
		obj:      newObjectFromFile(t, testDaemonsetFile),
		key:      "app",
		value:    "hi",
		override: false,

		beforeUpdate: newObjectFromFile(t, testDaemonsetFile),
		want:         newObjectFromFile(t, testDaemonsetUpdatedLabel2File),
	}, {
		name: "Job nested template",

		obj:      newObjectFromFile(t, testJobFile),
		key:      "foo",
		value:    "bar",
		override: false,

		beforeUpdate: newObjectFromFile(t, testJobFile),
		want:         newObjectFromFile(t, testJobUpdatedLabelFile),
	}, {
		name: "ReplicaSet nested template",

		obj:      newObjectFromFile(t, testReplicasetFile),
		key:      "foo",
		value:    "bar",
		override: false,

		beforeUpdate: newObjectFromFile(t, testReplicasetFile),
		want:         newObjectFromFile(t, testReplicasetUpdatedLabelFile),
	}, {
		name: "ReplicationController nested template",

		obj:      newObjectFromFile(t, testReplicationcontrollerFile),
		key:      "foo",
		value:    "bar",
		override: false,

		beforeUpdate: newObjectFromFile(t, testReplicationcontrollerFile),
		want:         newObjectFromFile(t, testReplicationcontrollerUpdatedLabelFile),
	}, {
		name: "StatefulSet nested template",

		obj:      newObjectFromFile(t, testStatefulsetFile),
		key:      "foo",
		value:    "bar",
		override: false,

		beforeUpdate: newObjectFromFile(t, testStatefulsetFile),
		want:         newObjectFromFile(t, testStatefulsetUpdatedLabelFile),
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := AddLabel(ctx, tc.obj, tc.key, tc.value, tc.override); !reflect.DeepEqual(tc.obj, tc.want) || err != nil {
				t.Errorf("AddLabel(ctx, %v, %s, %s, %t) = %v, %v; want <nil>, %v", tc.beforeUpdate, tc.key, tc.value, tc.override, err, tc.obj, tc.want)
			}
		})
	}
}

func TestAddLabelErrors(t *testing.T) {
	ctx := context.Background()

	testDeploymentFile := "testing/deployment.yaml"

	tests := []struct {
		name string

		obj   *Object
		key   string
		value string
	}{{
		name: "Empty key",

		obj:   newObjectFromFile(t, testDeploymentFile),
		key:   "",
		value: "bar",
	}, {
		name: "Empty value",

		obj:   newObjectFromFile(t, testDeploymentFile),
		key:   "foo",
		value: "",
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := AddLabel(ctx, tc.obj, tc.key, tc.value, false); err == nil {
				t.Errorf("AddLabel(ctx, %v, %s, %s, false) = <nil>; want error", tc.obj, tc.key, tc.value)
			}
		})
	}
}

func TestAddAnnotation(t *testing.T) {
	testDeploymentFile := "testing/deployment.yaml"
	testDeploymentUpdatedAnnotationFile := "testing/deployment-updated-annotation.yaml"
	testCronjobFile := "testing/cronjob.yaml"
	testCronjobUpdatedAnnotationFile := "testing/cronjob-updated-annotation.yaml"
	testDaemonsetFile := "testing/daemonset.yaml"
	testDaemonsetUpdatedAnnotationFile := "testing/daemonset-updated-annotation.yaml"
	testJobFile := "testing/job.yaml"
	testJobUpdatedAnnotationFile := "testing/job-updated-annotation.yaml"
	testReplicasetFile := "testing/replicaset.yaml"
	testReplicasetUpdatedAnnotationFile := "testing/replicaset-updated-annotation.yaml"
	testReplicationcontrollerFile := "testing/replicationcontroller.yaml"
	testReplicationcontrollerUpdatedAnnotationFile := "testing/replicationcontroller-updated-annotation.yaml"
	testStatefulsetFile := "testing/statefulset.yaml"
	testStatefulsetUpdatedAnnotationFile := "testing/statefulset-updated-annotation.yaml"

	tests := []struct {
		name string

		obj   *Object
		key   string
		value string

		beforeUpdate *Object
		want         *Object
	}{{
		name: "Normal case",

		obj:   newObjectFromFile(t, testDeploymentFile),
		key:   "foo",
		value: "bar",

		beforeUpdate: newObjectFromFile(t, testDeploymentFile),
		want:         newObjectFromFile(t, testDeploymentUpdatedAnnotationFile),
	}, {
		name: "No existing annotations field",

		obj:   newObjectFromFile(t, testCronjobFile),
		key:   "foo",
		value: "bar",

		beforeUpdate: newObjectFromFile(t, testCronjobFile),
		want:         newObjectFromFile(t, testCronjobUpdatedAnnotationFile),
	}, {
		name: "DaemonSet nested template",

		obj:   newObjectFromFile(t, testDaemonsetFile),
		key:   "foo",
		value: "bar",

		beforeUpdate: newObjectFromFile(t, testDaemonsetFile),
		want:         newObjectFromFile(t, testDaemonsetUpdatedAnnotationFile),
	}, {
		name: "Job nested template",

		obj:   newObjectFromFile(t, testJobFile),
		key:   "foo",
		value: "bar",

		beforeUpdate: newObjectFromFile(t, testJobFile),
		want:         newObjectFromFile(t, testJobUpdatedAnnotationFile),
	}, {
		name: "ReplicaSet nested template",

		obj:   newObjectFromFile(t, testReplicasetFile),
		key:   "foo",
		value: "bar",

		beforeUpdate: newObjectFromFile(t, testReplicasetFile),
		want:         newObjectFromFile(t, testReplicasetUpdatedAnnotationFile),
	}, {
		name: "ReplicationController nested template",

		obj:   newObjectFromFile(t, testReplicationcontrollerFile),
		key:   "foo",
		value: "bar",

		beforeUpdate: newObjectFromFile(t, testReplicationcontrollerFile),
		want:         newObjectFromFile(t, testReplicationcontrollerUpdatedAnnotationFile),
	}, {
		name: "StatefulSet nested template",

		obj:   newObjectFromFile(t, testStatefulsetFile),
		key:   "foo",
		value: "bar",

		beforeUpdate: newObjectFromFile(t, testStatefulsetFile),
		want:         newObjectFromFile(t, testStatefulsetUpdatedAnnotationFile),
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := AddAnnotation(tc.obj, tc.key, tc.value); !reflect.DeepEqual(tc.obj, tc.want) || err != nil {
				t.Errorf("AddAnnotation(%v, %s, %s) = %v, %v; want <nil>, %v", tc.beforeUpdate, tc.key, tc.value, err, tc.obj, tc.want)
			}
		})
	}
}

func TestAddAnnotationErrors(t *testing.T) {
	testDeploymentFile := "testing/deployment.yaml"

	tests := []struct {
		name string

		obj   *Object
		key   string
		value string
	}{{
		name: "Empty key",

		obj:   newObjectFromFile(t, testDeploymentFile),
		key:   "",
		value: "bar",
	}, {
		name: "Empty value",

		obj:   newObjectFromFile(t, testDeploymentFile),
		key:   "foo",
		value: "",
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := AddAnnotation(tc.obj, tc.key, tc.value); err == nil {
				t.Errorf("AddAnnotation(%v, %s, %s) = <nil>; want error", tc.obj, tc.key, tc.value)
			}
		})
	}
}

func TestUpdateNamespace(t *testing.T) {
	ctx := context.Background()

	testHpaFile := "testing/hpa.yaml"
	testHpaUpdatedNamespacefile := "testing/hpa-updated-namespace.yaml"
	testDeploymentFile := "testing/deployment.yaml"
	testDeploymentUpdatedNamespacefile := "testing/deployment-updated-namespace.yaml"

	tests := []struct {
		name string

		objs    Objects
		replace string

		beforeUpdate Objects
		want         Objects
	}{{
		name: "Updates namespace",

		objs: Objects{
			newObjectFromFile(t, testHpaFile),
		},
		replace: "REPLACED",

		beforeUpdate: Objects{
			newObjectFromFile(t, testHpaFile),
		},
		want: Objects{
			newObjectFromFile(t, testHpaUpdatedNamespacefile),
		},
	}, {
		name: "No namespace field",

		objs: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
		replace: "REPLACED",

		beforeUpdate: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
		want: Objects{
			newObjectFromFile(t, testDeploymentUpdatedNamespacefile),
		},
	}, {
		name: "Same namespace",

		objs: Objects{
			newObjectFromFile(t, testHpaFile),
		},
		replace: "default",

		beforeUpdate: Objects{
			newObjectFromFile(t, testHpaFile),
		},
		want: Objects{
			newObjectFromFile(t, testHpaFile),
		},
	}, {
		name: "Empty objects",

		objs:    Objects{},
		replace: "REPLACED",

		beforeUpdate: Objects{},
		want:         Objects{},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := UpdateNamespace(ctx, tc.objs, tc.replace); !reflect.DeepEqual(tc.objs, tc.want) || err != nil {
				t.Errorf("UpdateNamespace(ctx, %v, %s) = %v, %v; want <nil>, %v", tc.beforeUpdate, tc.replace, err, tc.objs, tc.want)
			}
		})
	}
}

func TestAddNamespaceIfMissing(t *testing.T) {
	testDeploymentFile := "testing/deployment.yaml"
	testDeploymentUpdatedNamespacefile := "testing/deployment-updated-namespace.yaml"
	testHpaFile := "testing/hpa.yaml"

	tests := []struct {
		name string

		objs    Objects
		replace string

		beforeUpdate Objects
		want         Objects
	}{{
		name: "No namespace field, adds namespace",

		objs: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
		replace: "REPLACED",

		beforeUpdate: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
		want: Objects{
			newObjectFromFile(t, testDeploymentUpdatedNamespacefile),
		},
	}, {
		name: "Does not update namespace",

		objs: Objects{
			newObjectFromFile(t, testHpaFile),
		},
		replace: "REPLACED",

		beforeUpdate: Objects{
			newObjectFromFile(t, testHpaFile),
		},
		want: Objects{
			newObjectFromFile(t, testHpaFile),
		},
	}, {
		name: "Empty objects",

		objs:    Objects{},
		replace: "REPLACED",

		beforeUpdate: Objects{},
		want:         Objects{},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := AddNamespaceIfMissing(tc.objs, tc.replace); !reflect.DeepEqual(tc.objs, tc.want) || err != nil {
				t.Errorf("AddNamespaceIfMissing(ctx, %v, %s) = %v, %v; want <nil>, %v", tc.beforeUpdate, tc.replace, err, tc.objs, tc.want)
			}
		})
	}
}

func TestHasObject(t *testing.T) {
	ctx := context.Background()

	testDeploymentFile := "testing/deployment.yaml"
	testHpaFile := "testing/hpa.yaml"

	tests := []struct {
		name string

		objs    Objects
		kind    string
		objName string

		want bool
	}{{
		name: "Has object",

		objs: Objects{
			newObjectFromFile(t, testDeploymentFile),
		},
		kind:    "Deployment",
		objName: "test-app",

		want: true,
	}, {
		name: "Does not have object",

		objs: Objects{
			newObjectFromFile(t, testHpaFile),
		},
		kind:    "Deployment",
		objName: "test-app",

		want: false,
	}, {
		name: "Empty objects",

		objs:    Objects{},
		kind:    "Deployment",
		objName: "test-app",

		want: false,
	}, {
		name: "Duplicate objects",

		objs: Objects{
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testDeploymentFile),
			newObjectFromFile(t, testHpaFile),
		},
		kind:    "Deployment",
		objName: "test-app",

		want: true,
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := HasObject(ctx, tc.objs, tc.kind, tc.objName); got != tc.want || err != nil {
				t.Errorf("HasObject(ctx, %v, %s, %s) = %t, %v; want %t, <nil>", tc.objs, tc.kind, tc.objName, got, err, tc.want)
			}
		})
	}
}

func TestCreateDeploymentObject(t *testing.T) {
	ctx := context.Background()

	testDeploymentFile := "testing/deployment-4.yaml"

	objName := "test-app"
	selectorValue := "foo"
	image := "bar"

	want := newObjectFromFile(t, testDeploymentFile)

	if got, err := CreateDeploymentObject(ctx, objName, selectorValue, image); !reflect.DeepEqual(got, want) || err != nil {
		t.Errorf("CreateDeploymentObject(ctx, %s  %s, %s) = %v, %v; want %v, <nil>", objName, selectorValue, image, got, err, want)
	}
}

func TestCreateHorizontalPodAutoscalerObject(t *testing.T) {
	ctx := context.Background()

	testHorizontalPodAutoscalerFile := "testing/hpa-2.yaml"

	objName := "test-app-hpa"
	deploymentName := "test-app"

	want := newObjectFromFile(t, testHorizontalPodAutoscalerFile)

	if got, err := CreateHorizontalPodAutoscalerObject(ctx, objName, deploymentName); !reflect.DeepEqual(got, want) || err != nil {
		t.Errorf("CreateHorizontalPodAutoscalerObject(ctx, %s, %s) = %v, %v; want %v, <nil>", objName, deploymentName, got, err, want)
	}
}

func TestCreateNamespaceObject(t *testing.T) {
	ctx := context.Background()

	testNamespaceFile := "testing/namespace.yaml"

	objName := "foobar"

	want := newObjectFromFile(t, testNamespaceFile)

	if got, err := CreateNamespaceObject(ctx, objName); !reflect.DeepEqual(got, want) || err != nil {
		t.Errorf("CreateNamespaceObject(ctx, %s) = %v, %v; want %v, <nil>", objName, got, err, want)
	}
}

func TestCreateNamespaceObjectErrors(t *testing.T) {
	ctx := context.Background()

	objName := "default"

	if got, err := CreateNamespaceObject(ctx, objName); got != nil || err == nil {
		t.Errorf("CreateNamespaceObject(ctx, %s) = %v, %v; want <nil>, error", objName, got, err)
	}
}

func TestCreateServiceObject(t *testing.T) {
	ctx := context.Background()

	testServiceFile := "testing/service-2.yaml"

	objName := "test-app-service"
	selectorKey := "app"
	selectorValue := "test-app"
	port := 100

	want := newObjectFromFile(t, testServiceFile)

	if got, err := CreateServiceObject(ctx, objName, selectorKey, selectorValue, port); !reflect.DeepEqual(got, want) || err != nil {
		t.Errorf("CreateServiceObject(ctx, %s, %s, %s, %d) = %v, %v; want %v, <nil>", objName, selectorKey, selectorValue, port, got, err, want)
	}
}

func TestDeploySummary(t *testing.T) {
	ctx := context.Background()

	testCronjobReadyFile := "testing/cronjob.yaml"
	testDaemonsetReadyFile := "testing/daemonset-ready.yaml"
	testDeploymentReadyFile := "testing/deployment-ready.yaml"
	testNamespaceReadyFile := "testing/namespace.yaml"
	testReplicationcontrollerReadyFile := "testing/replicationcontroller-ready.yaml"
	testLoadBalancerServiceReadyFile := "testing/service-ready.yaml"
	testLoadBalancerServiceUnreadyFile := "testing/service-unready.yaml"
	testExternalNameServiceReadyFile := "testing/service-ready-4.yaml"
	testStatefulsetUnreadyFile := "testing/statefulset-unready.yaml"

	tests := []struct {
		name string

		objs Objects

		want string
	}{{
		name: "Simple deploy summary",

		objs: Objects{
			newObjectFromFile(t, testCronjobReadyFile),
			newObjectFromFile(t, testDaemonsetReadyFile),
			newObjectFromFile(t, testDeploymentReadyFile),
			newObjectFromFile(t, testNamespaceReadyFile),
			newObjectFromFile(t, testReplicationcontrollerReadyFile),
			newObjectFromFile(t, testLoadBalancerServiceReadyFile),
			newObjectFromFile(t, testExternalNameServiceReadyFile),
			newObjectFromFile(t, testStatefulsetUnreadyFile),
		},

		want: `NAMESPACE                KIND                     NAME                              READY    
default                  CronJob                  test-cron-job                     Yes      
default                  DaemonSet                test-app-daemonset                Yes      
foobar                   Deployment               test-app                          Yes      
default                  Namespace                foobar                            Yes      
test-local-deploy-all    ReplicationController    test-app-replicationcontroller    Yes      
foobar                   Service                  test-app                          Yes      34.74.85.152
foobar                   Service                  test-app-service-externalname     Yes      test-app.example.com
default                  StatefulSet              test-app-statefulset              No       
`,
	}, {
		name: "LoadBalancer Service not ready",

		objs: Objects{
			newObjectFromFile(t, testCronjobReadyFile),
			newObjectFromFile(t, testDaemonsetReadyFile),
			newObjectFromFile(t, testDeploymentReadyFile),
			newObjectFromFile(t, testNamespaceReadyFile),
			newObjectFromFile(t, testReplicationcontrollerReadyFile),
			newObjectFromFile(t, testLoadBalancerServiceUnreadyFile),
			newObjectFromFile(t, testExternalNameServiceReadyFile),
			newObjectFromFile(t, testStatefulsetUnreadyFile),
		},

		want: `NAMESPACE                KIND                     NAME                              READY    
default                  CronJob                  test-cron-job                     Yes      
default                  DaemonSet                test-app-daemonset                Yes      
foobar                   Deployment               test-app                          Yes      
default                  Namespace                foobar                            Yes      
test-local-deploy-all    ReplicationController    test-app-replicationcontroller    Yes      
foobar                   Service                  test-app                          No       
foobar                   Service                  test-app-service-externalname     Yes      test-app.example.com
default                  StatefulSet              test-app-statefulset              No       
`,
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := DeploySummary(ctx, tc.objs); got != tc.want || err != nil {
				t.Errorf("DeploySummary(ctx, %v) = %s, %v; want %v, <nil>", tc.objs, got, err, tc.want)
			}
		})
	}
}

func TestSortObjectsByKindAndName(t *testing.T) {
	testCronjobFile := "testing/cronjob.yaml"
	testCronjob2File := "testing/cronjob-updated.yaml"
	testDaemonsetFile := "testing/daemonset.yaml"
	testDaemonset2File := "testing/daemonset-updated.yaml"
	testDeploymentFile := "testing/deployment.yaml"
	testDeployment2File := "testing/deployment-updated.yaml"
	testJobFile := "testing/job.yaml"
	testJob2File := "testing/job-updated.yaml"
	testPodFile := "testing/pod.yaml"
	testPod2File := "testing/pod-updated.yaml"

	objs := []*Object{
		newObjectFromFile(t, testPod2File),
		newObjectFromFile(t, testCronjobFile),
		newObjectFromFile(t, testJob2File),
		newObjectFromFile(t, testDeployment2File),
		newObjectFromFile(t, testDaemonsetFile),
		newObjectFromFile(t, testDeploymentFile),
		newObjectFromFile(t, testPodFile),
		newObjectFromFile(t, testJobFile),
		newObjectFromFile(t, testCronjob2File),
		newObjectFromFile(t, testDaemonset2File),
	}

	beforeUpdate := []*Object{
		newObjectFromFile(t, testPod2File),
		newObjectFromFile(t, testCronjobFile),
		newObjectFromFile(t, testJob2File),
		newObjectFromFile(t, testDeployment2File),
		newObjectFromFile(t, testDaemonsetFile),
		newObjectFromFile(t, testDeploymentFile),
		newObjectFromFile(t, testPodFile),
		newObjectFromFile(t, testJobFile),
		newObjectFromFile(t, testCronjob2File),
		newObjectFromFile(t, testDaemonset2File),
	}
	want := []*Object{
		newObjectFromFile(t, testCronjobFile),
		newObjectFromFile(t, testCronjob2File),
		newObjectFromFile(t, testDaemonsetFile),
		newObjectFromFile(t, testDaemonset2File),
		newObjectFromFile(t, testDeployment2File),
		newObjectFromFile(t, testDeploymentFile),
		newObjectFromFile(t, testJob2File),
		newObjectFromFile(t, testJobFile),
		newObjectFromFile(t, testPod2File),
		newObjectFromFile(t, testPodFile),
	}

	if sortObjectsByKindAndName(objs); !reflect.DeepEqual(objs, want) {
		t.Errorf("sortObjectsByKindAndName(%v) = %v; want %v", beforeUpdate, objs, want)
	}
}

func TestAddCommentsToLines(t *testing.T) {
	tests := []struct {
		name string

		s            string
		lineComments map[string]string

		want string
	}{{
		name: "Add comments to lines",

		s: "abc\n123\nhithere\nbyehere",
		lineComments: map[string]string{
			"12":   "first comment",
			"here": "multiple comments",
		},

		want: "abc\n123  # first comment\nhithere  # multiple comments\nbyehere  # multiple comments",
	}, {
		name: "No comments to add",

		s:            "abc\n123\nhithere\nbyehere",
		lineComments: nil,

		want: "abc\n123\nhithere\nbyehere",
	}, {
		name: "No matching lines",

		s: "red\nfish\nblue\nfish",
		lineComments: map[string]string{
			"12":   "first comment",
			"here": "multiple comments",
		},
		want: "red\nfish\nblue\nfish",
	}, {
		name: "Empty string",

		s: "",
		lineComments: map[string]string{
			"12":   "first comment",
			"here": "multiple comments",
		},
		want: "",
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := addCommentsToLines(tc.s, tc.lineComments); got != tc.want || err != nil {
				t.Errorf("addCommentsToLines(%s, %v) = %s, %v; want %s, <nil>", tc.s, tc.lineComments, got, err, tc.want)
			}
		})
	}
}

func newObjectFromFile(t *testing.T, filename string) *Object {
	contents := fileContents(t, filename)
	obj, err := DecodeFromYAML(nil, contents)
	if err != nil {
		t.Fatalf("failed to decode resource from file %s", filename)
	}
	return obj
}

func fileContents(t *testing.T, filename string) []byte {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read file %s", filename)
	}
	return contents
}
