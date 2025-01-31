# brew-formula-update

![GitHub Release](https://img.shields.io/github/v/release/yodamad-actions/brew-formula-update)
![GitHub last commit](https://img.shields.io/github/last-commit/yodamad-actions/brew-formula-update)
![GitHub License](https://img.shields.io/github/license/yodamad-actions/brew-formula-update)

A GitHub action to handle multiple URLs use-case when updating a brew formula

An example of `.rb` file to update : 

```ruby
class Slidesk < Formula
    desc "Speaker companion"
    homepage "https://github.com/gouz/homebrew-tools"

    version "0.11"
    BASE_URL = "https://github.com/yodamad/slidesk-fork/releases/download/#{version}"

    MAC_ARM_SHA = "bd198c80cf98f591f10c03ab177b2600690863e0bb55afd1ae239c60660b86d6"
    MAC_AMD_SHA = "738efe5ab5753d56b35b0562fb4c0a807f7bda3dde0e72a5ec3911d12944356d"

    on_macos do
        on_arm do
            @@file_name = "slidesk_mac"
            sha256 MAC_ARM_SHA
        end
        on_intel do
            @@file_name = "slidesk_mac_intel"
            sha256 MAC_AMD_SHA
        end
    end
    
    url "#{BASE_URL}/#{@@file_name}"

    def install
        bin.install "#{@@file_name}" => "slidesk"
    end
end
```

## How to use it

In your GitHub actions workflow, you need to add this job 

```yaml
  update_brew_formula:
    name: Update formula
    runs-on: ubuntu-latest
    needs: build_release
    steps:
      - name: Update formula
        uses: yodamad-actions/brew-formula-update@0.2
        with:
          file: slidesk.rb
          owner: yodamad
          repo: homebrew-tools
          version: ${{ github.ref_name }}
          fields: ${{ toJSON(needs.build_release.outputs) }}
          token: ${{ secrets.BREW_TOKEN }}
          auto_merge: true
```

Inputs required to use the action

| Option   | Description                          | Example |
|----------|--------------------------------------|---------|
| file     | Formula file to update               | slidesk.rb |
| owner    | Owner or organization hosting repository | yodamad |
| repo     | Repository containing file to update | homebrew-tools |
| version  | New version for formula              | 1.0.0 |
| fields   | Field to update with sha256 value    | JSON output (see below) |
| token    | GitHub token to access repository    | A GitHub token with write access to the `repo` |
 | auto_merge | Automatically merge pull request    | true |

## How to build `fields` input

Fields input is a JSON string that looks like this

```json
{
  "output_mac_amd": "MAC_AMD_SHA-b328dc00b258eaa7aeadca0d3b4e3994f6f1ca5f1aaddc16f46faff040bf60b6",
  "output_mac_arm": "MAC_ARM_SHA-bd198c80cf98f591f10c03ab177b2600690863e0bb55afd1ae239c60660b86d6"
}
```

The keys (`output_xxx`) of the json is not very important, but are necessary to pass data from job to another in GitHub Action.
The values **must** be the concatenation of 
- the `field` in the formula file 
- the `sha` value to update associated to the `field`

An example to create this :

```yml
  build_release:
    name: Build Release
    needs: create_release
    environment: build
    outputs:
      output_mac_amd: ${{ steps.release_package.outputs.output_MAC_AMD_SHA }}
      output_mac_arm: ${{ steps.release_package.outputs.output_MAC_ARM_SHA }}
    strategy:
      matrix:
        include:
          - os: macos-latest
            release_suffix: mac
            compile_options: --target=bun-darwin-arm64
            sha_field: MAC_ARM_SHA
          - os: macos-15
            release_suffix: mac_intel
            platform: x64
            compile_options: --target=bun-darwin-x64
            sha_field: MAC_AMD_SHA
    runs-on: ${{ matrix.os }}
    steps:
      - uses: oven-sh/setup-bun@v1
      - name: Checkout code
        uses: actions/checkout@v4
      - run: bun install
      - run: bun make:exe ${{ matrix.compile_options }}
      - name: Rename release package
        id: release_package
        run: |
          cat ./exe/slidesk > slidesk_${{ matrix.release_suffix }}
          sha=($(shasum -a 256 ./exe/slidesk))
          sha_value="${{ matrix.sha_field }}"
          echo "output_${sha_value}=${{ matrix.sha_field }}-$sha" >> "$GITHUB_OUTPUT"
```

This is not very convinient, but GitHub Action are quite restricted when you have a matrix and to pass data from one job to another. 

## Try it locally

```bash
env \
  'GITHUB_API_URL=https://api.github.com' \
  'GITHUB_REPOSITORY=<repository_running_action>' \
  'INPUT_FILE=<file_to_update>' \
  'INPUT_OWNER=<owner_or_organization>' \
  'INPUT_REPO=<repository>' \
  'INPUT_VERSION=<new_version>' \
  'INPUT_FIELDS=<json_input>' \
  'INPUT_TOKEN=<your_token>' \
  'INPUT_AUTO_MERGE=true' \
  go run main.go
```
