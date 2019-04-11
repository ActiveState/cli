workflow "New workflow" {
  on = "push"
  resolves = ["Azure/github-actions/pipelines@master"]
}

action "Azure/github-actions/pipelines@master" {
  uses = "Azure/github-actions/pipelines@master"
  secrets = ["AZURE_DEVOPS_TOKEN"]
  env = {
    AZURE_DEVOPS_URL = "https://dev.azure.com/activestate"
    AZURE_DEVOPS_PROJECT = "State Tool"
    AZURE_PIPELINE_NAME = "ActiveState.cli"
  }
}