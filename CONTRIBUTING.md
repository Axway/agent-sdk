# Contributing

Welcome to the Amplify Agents SDK. Contributions are welcome from anyone. All work on this project happens directly on GitHub. Both core team members and external contributors send pull requests which go through the same review process.

# Prerequisites

Before submitting code to this project you must first sign the Axway Contributors License Agreement (CLA).

# Semantic Versioning

The Amplify Agents SDK follow semantic versioning. We release patch versions for critical bugfixes, minor versions for new features or non-essential changes, and major versions for any breaking changes. When we make breaking changes, we also introduce deprecation warnings in a minor version so that our users learn about the upcoming changes and migrate their code in advance.

Every significant change is documented in the changelog file.

# Branch Organization

Submit all changes directly to the main branch. We don’t use separate branches for development or for upcoming releases.

If you are merging a breaking change to main, then a new major release must be made.

Patch and minor releases should be made for all other bug and feature enhancements that do not introduce a breaking change.

# Features, Enhancements & Bugs

We use GitHub Issues for all of our feature requests, enhancements, and bugs.

## Where to Find Known Issues

If you are experiencing an issue you can check our GitHub Issues. All issues related to known bugs will be labeled as 'bug'. We keep a close eye on this and try to make it clear when we have an internal fix in progress.

## Reporting New Issues

Before filing a new task, try to make sure your problem does not already exist by looking through the known issues. If you are experiencing a problem that you think is not documented, create an issue and attach the 'bug' label.

Before fixing a bug we need to reproduce and confirm it. We require that you provide a reproducible scenario. Having a minimal reproducible scenario gives us important information without going back and forth to you with additional questions.

Submit a feature request [here](https://github.com/Axway/agent-sdk/issues/new?assignees=&labels=enhancement&template=feature_request.md&title=)

Report a bug [here](https://github.com/Axway/agent-sdk/issues/new?assignees=&labels=bug&template=bug_report.md&title=)

## Security Issues

If you have encountered a security vulnerability, then create an issue and attach the 'security' label.

The Axway security team and associated development organizations will use reasonable efforts to:

* Respond promptly in acknowledging the receipt of your vulnerability report
* Work with you to understand the scope and severity of the vulnerability
* Provide an estimated time frame for addressing the vulnerability
* Update you when the vulnerability has been fixed

# Proposing a Change

If you intend to make any non-trivial changes to the implementation, we recommend filing an issue. This lets us reach an agreement on your proposal before you put significant effort into it.

If you’re only fixing a bug, it’s fine to submit a pull request right away, but we still recommend that you file an issue detailing what you’re fixing. This is helpful in case we don’t accept that specific fix but want to keep track of the issue.

# Documentation

When a change is made please update the documentation found in the `README.md` files accordingly so that the documentation reflects the code.

# Submitting a pull request

The core team is monitoring for pull requests. We will review your pull request and either merge it, request changes to it, or close it with an explanation. We’ll do our best to provide updates and feedback throughout the process.

## Before Submitting

Please make sure the following is done before submitting a pull request:

1. Fork the repository and create your branch from main.
2. If you’ve fixed a bug or added code that should be tested, then add tests.
3. Ensure the test suite passes by running `make test`.
4. Format your code with `make format`.
5. If you haven’t already, complete the CLA.
6. Make sure your pull request describes the issue you are fixing, or the feature you are adding. The description should also have a comment specifying which issue the pull request will resolve. For example, if the issue you are working on is #100, then please leave a comment saying 'Resolves #100'. This will cause the issue to be closed automatically when the pull request is closed.

# Development Prerequisites

* You have Go 1.18 or newer installed
* Install goimports - `go install golang.org/x/tools/cmd/goimports`

# Development Workflow

After cloning the Amplify Agents SDK, run `make download` to download all the project dependencies.

* `make format` formats your code.
* `make test` runs all the unit tests with the `-race` flag to check for race conditions.

## Starting local development

Two sample stubs are provided within the Amplify Agents SDK source code:

* Discovery agent: <https://github.com/Axway/agent-sdk/raw/main/samples/apic_discovery_agent.zip>
* Traceability agent: <https://github.com/Axway/agent-sdk/raw/main/samples/apic_traceability_agent.zip>

Refer to <https://github.com/Axway/agent-sdk#sample-projects>

# Axway Contributors

You may create your branches directly within the repo. You do not need to fork the project.

Please make sure the following is done when you open a pull request.

1. Labels are added to the pull request. These can be the same labels that are found on the issue.
2. Assign the pull request to the 'Axway Agent SDK' project. This is the board for tracking work in progress.
3. Link the pull request back to your issue.

All of these steps can be taken care of after opening your pull request.

Reviewers will automatically be added to your pull request. Assign the pull request to one of the core maintainers when you are ready to merge your branch. You may merge the branch once it has been approved.

## Accepting pull requests

As an Axway contributor who reviews incoming pull requests please ensure the following are met on each request.

1. Labels are added to the pull request.
2. It has been assigned to the project board.
3. It has been added to the sprint milestone.
4. It is linked to the issue.
5. The description has a reference to what issue it will close, such as 'closes #100'.
6. The description outlines what has changed.
7. The title accurately reflects the new changes.
8. The pipeline is passing.

## Board

The project board has three columns

* To do - Issues you plan to work on during the sprint should be moved here.
* In progress - Any issue or pull request that is actively being worked on.
* Done - Any issue or pull request that has been completed, rejected, blocked, or closed.

When you open a pull request and link it to the issue and the board, a task for a pull request will automatically be placed in the 'In progress' column. When the pull request is merged, the task for the pull request will automatically be moved to the 'Done' column, and it will be closed.

# License

By contributing to the Axway MuleSoft Agents, you agree that your contributions will be licensed under the Apache 2.0 license.
