let:
	runtime = state_tool_artifacts_v1(
		build_flags = [
		],
		camel_flags = [
		],
		src = "$sources"
	)
	sources = solve(
		at_time = "2023-10-16T22:20:29.000000Z",
		platforms = [
			"78977bc8-0f32-519d-80f3-9043f059398c",
			"7c998ec2-7491-4e75-be4d-8885800ef5f2",
			"96b7e6f2-bebf-564c-bc1c-f04482398f38"
		],
		requirements = [
			{
				name = "python",
				namespace = "language",
				version_requirements = [
					{
						comparator = "eq",
						version = "3.10.11"
					}
				]
			},
			{
				name = "requests",
				namespace = "language/python"
			}
		],
		solver_version = null
	)

in:
	runtime