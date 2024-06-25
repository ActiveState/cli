package buildscript

import "fmt"

var atTime = "2000-01-01T00:00:00.000Z"

var basicBuildScript = []byte(fmt.Sprintf(
	`at_time = "%s"
runtime = state_tool_artifacts(
	build_flags = [
		{
			name = "foo",
			value = "bar"
		}
	],
	src = sources
)
sources = solve(
	at_time = at_time,
	platforms = [
		"12345",
		"67890"
	],
	requirements = [
		Req(name = "python", namespace = "language", version = Eq(value = "3.10.10"))
	]
)

main = runtime`, atTime))

var basicBuildExpression = []byte(`{
  "let": {
    "in": "$runtime",
    "runtime": {
      "state_tool_artifacts": {
        "build_flags": [
          {
            "name": "foo",
            "value": "bar"
          }
        ],
        "src": "$sources"
      }
    },
    "sources": {
      "solve": {
        "at_time": "$at_time",
        "platforms": [
          "12345",
          "67890"
        ],
        "requirements": [
          {
            "name": "python",
            "namespace": "language",
            "version_requirements": [
              {
                "comparator": "eq",
                "version": "3.10.10"
              }
            ]
          }
        ]
      }
    }
  }
}`)
