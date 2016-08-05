package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// JSONTestCaseLoader loads test cases stored as a JSON files.
// It travers all directories under the RootDir and tries to
// parse files that has a test suite shape.
type JSONTestCaseLoader struct {
	// starting point
	RootDir string

	suits []TestSuite
}

// NewJSONTestCaseLoader creates new json test case loader
// for a given directory.
func NewJSONTestCaseLoader(dir string) *JSONTestCaseLoader {
	return &JSONTestCaseLoader{RootDir: dir}
}

// Load all test cases.
func (s *JSONTestCaseLoader) Load() ([]TestSuite, error) {
	err := filepath.Walk(s.RootDir, s.loadFile)
	if err != nil {
		return nil, err
	}

	return s.suits, nil
}

func (s *JSONTestCaseLoader) loadFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		return nil
	}

	if info.IsDir() {
		return nil
	}

	if !strings.HasSuffix(info.Name(), ".json") {
		return nil
	}

	ok := isSuite(path)
	if !ok {
		return nil
	}

	err = validateSuite(path)
	if err != nil {
		fmt.Printf("Invalid suite file: %s\n%s\n", path, err.Error())
		return nil
	}

	debugMsgF("Process file: %s\n", info.Name())
	content, e := ioutil.ReadFile(path)

	if e != nil {
		debugMsgF("File error: %v\n", e)
		return filepath.SkipDir
	}

	var testCases []TestCase
	err = json.Unmarshal(content, &testCases)
	if err != nil {
		debugMsgF("Parse error: %v\n", err)
		return nil
	}

	dir, _ := filepath.Rel(s.RootDir, filepath.Dir(path))
	su := TestSuite{
		Name:  strings.TrimSuffix(info.Name(), filepath.Ext(info.Name())),
		Dir:   dir,
		Cases: testCases,
	}
	s.suits = append(s.suits, su)
	return nil
}

func isSuite(path string) bool {
	schemaLoader := gojsonschema.NewStringLoader(suiteShapeSchema)

	path, _ = filepath.Abs(path)
	documentLoader := gojsonschema.NewReferenceLoader("file:///" + path)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return false
	}

	return result.Valid()
}

func validateSuite(path string) error {
	schemaLoader := gojsonschema.NewStringLoader(suiteDetailedSchema)

	path, _ = filepath.Abs(path)
	documentLoader := gojsonschema.NewReferenceLoader("file:///" + path)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		var msg string
		for _, desc := range result.Errors() {
			msg = fmt.Sprintf(msg+"%s\n", desc)
		}
		return errors.New(msg)
	}

	return nil
}

// used to detect suite
const suiteShapeSchema = `
{
  "$schema": "http://json-schema.org/draft-04/schema#",
  "type": "array",
  "items": {
    "type": "object",
    "properties": {
      "name": {
        "type": "string"
      },
      "calls": {
        "type": "array"
      }
    },
    "required": [
      "name",
      "calls"
    ]
  }
}
`

// used to validate suite
const suiteDetailedSchema = `
{
	"$schema": "http://json-schema.org/draft-04/schema#",
	"type": "array",
	"items": {
		"type": "object",
		"properties": {
			"name": {
				"type": "string"
			},
			"calls": {
				"type": "array",
				"items": {
					"type": "object",
					"properties": {
						"on": {
							"type": "object",
							"properties": {
								"method": {
									"type": "string"
								},
								"url": {
									"type": "string"
								},
								"headers": {
									"type": "object"
								},
								"params": {
									"type": "object"
								}
							},
							"required": [
								"method",
								"url"
							]
						},
						"expect": {
							"type": "object",
							"properties": {
								"statusCode": {
									"type": "integer"
								},
								"contentType": {
									"type": "string"
								},
								"body": {
									"type": "object"
								}
							}
						}
					},
					"required": [
						"on",
						"expect"
					]
				}
			}
		},
		"required": [
			"calls"
		]
	}
}
`
