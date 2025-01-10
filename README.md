# brew-formula-update
A GitHub action to handle multiple URLs use-case when updating a brew formula


## Try it locally

```bash
env \
  'GITHUB_API_URL=https://api.github.com' \
  'GITHUB_REPOSITORY=<repository_running_action>' \
  'INPUT_FILE=<file_to_update>' \
  'INPUT_OWNER=<owner_or_organization>' \
  'INPUT_REPO=<repository>' \
  'INPUT_VERSION=<new_version>' \
  'INPUT_SHA256=<your_sha256>' \
  'INPUT_FIELD=<your_field>' \
  'INPUT_TOKEN=<your_token>' \
  go run main.go
```
