# bb2todotxt

Means bitbucket (issue tasks) to [todo.txt](https://todotxt.org).

# Installation

Grab the appropriate release for your platform and install it. Homebrew instructions coming soon.

# Usage

You'll want to put your bitbucket username and app password in a json file like so:

```json
{
  "username": "ceo_of_tasks",
  "password": "secret"
}
```

> Why a json file?

I'm hoping you can rely on operating system controls to control access to your credentials. SSH does it, AWS does it.

See the [roadmap](#roadmap) for where I'm hoping to implement encryption for your credentials.

## Command Line

Call the tool like so:
```bash
$ bb2todotxt -config ~/.bb2todotxt/bitbucket.json -owner myorg -slug myrepo -id 1337 > ~/todos/todo.txt
```

## Unix Philosophy

The tool is intended to only write its intended output to stdout. If you notice the tool behaving differently, please
open an issue.

# Development

I'm using go 1.20.4, but I'm not sure that I'm using anything too complicated

# Roadmap

- Encrypted credentials file (bbcrypted interactively)
- Automatically upload to homebrew tap
- Support bitbucket datacenter edition

Any feature requests should be opened as an issue and may be added to the roadmap.
