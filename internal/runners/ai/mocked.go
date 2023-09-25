package ai

import "encoding/json"

const mocked = `[
    {
      "name": "openai",
      "description": "OpenAI API. It exposes methods to interact with OpenAI's models, like GPT-3.",
      "advantages": ["Highly scalable", "Easy access to GPT-3", "Robust API"],
      "disadvantages": ["Requires API key", "Rate limitations"],
      "example": "import openai\n\nopenai.api_key = 'your-api-key'\n\nresponse = openai.Completion.create(\n  engine='text-davinci-002',\n  prompt='Translate the following English text to French: {},'\n  max_tokens=60\n)",
      "projects": ["GPT-3", "OpenAI Codex"],
      "rank": 1
    },
    {
      "name": "openai-gym",
      "description": "A toolkit for developing and comparing reinforcement learning algorithms.",
      "advantages": ["Extensive set of environments", "Simple, consistent API", "Great for beginners and experts alike"],
      "disadvantages": ["Could use better documentation", "Requires underlying software for some environments"],
      "example": "import gym\n\nenv = gym.make('CartPole-v0')\n\nevent.operation(env.reset())",
      "projects": ["CartPole, MountainCar", "Atari Games"],
      "rank": 2
    },
    {
      "name": "openai-gpt",
      "description": "A Python interface to OpenAI’s GPT transformer text generation model.",
      "advantages": ["Leverages GPT models", "Direct, easy-to-use interface"],
      "disadvantages": ["Deprecation potential as new models are released", "Limited to text generation"],
      "example": "from openai import gpt\n\ngpt = gpt.GPT()\n\noutput = gpt.generate('What is the meaning of life?')",
      "projects": ["GPT-2", "GPT-3"],
      "rank": 3
    },
    {
      "name": "openai-baselines",
      "description": "A set of implementations of reinforcement learning algorithms provided by OpenAI.",
      "advantages": ["Wide variety of BASICS algorithms", "Common interface"],
      "disadvantages": ["Could use better documentation", "Requires extensive computing resources"],
      "example": "import baselines\n\nbaselines.run.main()\n\n",
      "projects": ["DQN, A3C", "TRPO"],
      "rank": 4
    },
    {
      "name": "openai-universe",
      "description": "A software platform for measuring and training an AI's general intelligence across a wide variety of games, websites and other applications.",
      "advantages": ["Extensive set of environments", "Broad test bed for general AI capabilities"],
      "disadvantages": ["Project is no longer actively maintained", "Complex setup"],
      "example": "import universe\n\nenv = universe.make('flashgames.DuskDrive-v0')",
      "projects": ["Any game that can be accessed via a web browser", "DuskDrive"],
      "rank": 5
    }
  ]`

func mockedPackages() []*Package {
	result := []*Package{}
	err := json.Unmarshal([]byte(mocked), &result)
	if err != nil {
		panic(err)
	}
	return result
}
